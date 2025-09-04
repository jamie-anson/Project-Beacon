package websocket

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/internal/metrics"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from localhost for development
		return true
	},
}

// Message represents a WebSocket message
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Client represents a WebSocket client
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
	// optional correlation id captured at handshake
	requestID string
}

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	l := logging.L()
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			l2 := l.With().Str("request_id", client.requestID).Logger()
			l2.Info().Int("clients", len(h.clients)).Msg("ws client connected")
			metrics.WebSocketConnections.Inc()

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mutex.Unlock()
			l2 := l.With().Str("request_id", client.requestID).Logger()
			l2.Info().Int("clients", len(h.clients)).Msg("ws client disconnected")
			metrics.WebSocketConnections.Dec()

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
					// delivered to client buffer
				default:
					// client's buffer full -> drop and unregister
					metrics.WebSocketMessagesDroppedTotal.Inc()
					delete(h.clients, client)
					close(client.send)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// BroadcastMessage sends a message to all connected clients
func (h *Hub) BroadcastMessage(msgType string, data interface{}) {
	l := logging.L()
	message := Message{Type: msgType, Data: data}
	jsonData, err := json.Marshal(message)
	if err != nil {
		l.Error().Err(err).Msg("ws marshal message error")
		return
	}

	select {
	case h.broadcast <- jsonData:
		metrics.WebSocketMessagesBroadcastTotal.Inc()
	default:
		metrics.WebSocketMessagesDroppedTotal.Inc()
		l.Error().Msg("ws broadcast channel full, dropping message")
	}
}

// BroadcastMessageWithRequestID sends a message including request_id in the payload
func (h *Hub) BroadcastMessageWithRequestID(requestID string, msgType string, data interface{}) {
	l := logging.L()
	payload := map[string]interface{}{"type": msgType, "data": data}
	if requestID != "" {
		payload["request_id"] = requestID
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		l.Error().Err(err).Str("request_id", requestID).Msg("ws marshal message error")
		return
	}
	select {
	case h.broadcast <- jsonData:
		metrics.WebSocketMessagesBroadcastTotal.Inc()
	default:
		metrics.WebSocketMessagesDroppedTotal.Inc()
		l.Error().Str("request_id", requestID).Msg("ws broadcast channel full, dropping message")
	}
}

// ServeWS handles WebSocket requests from clients
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	l := logging.L()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		l.Error().Err(err).Msg("ws upgrade error")
		return
	}

	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
		requestID: r.Header.Get("X-Request-ID"),
	}

	client.hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	// Set read deadline and pong handler for keepalive
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			// Only log unexpected close errors
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				l := logging.L()
				l.Error().Err(err).Str("request_id", c.requestID).Msg("ws read error")
			}
			break
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
