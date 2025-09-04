package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// tryReadMessage attempts to read a single message with a short timeout.
func tryReadMessage(conn *websocket.Conn, timeout time.Duration) ([]byte, error) {
	conn.SetReadDeadline(time.Now().Add(timeout))
	_, msg, err := conn.ReadMessage()
	return msg, err
}

// broadcastUntilReceived keeps broadcasting until a message is received or timeout.
func broadcastUntilReceived(t *testing.T, hub *Hub, conn *websocket.Conn, send func()) []byte {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for time.Now().Before(deadline) {
		// send a broadcast (non-blocking in hub)
		send()
		if msg, err := tryReadMessage(conn, 50*time.Millisecond); err == nil {
			return msg
		}
		<-ticker.C
	}
	t.Fatalf("did not receive message before timeout")
	return nil
}

// waitForMessage reads a single message from the websocket conn with a timeout.
func waitForMessage(t *testing.T, conn *websocket.Conn, timeout time.Duration) []byte {
	t.Helper()
	deadline := time.Now().Add(timeout)
	require.NoError(t, conn.SetReadDeadline(deadline))
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)
	return msg
}

func TestWebSocketHub_BroadcastAndReceive(t *testing.T) {
	hub := NewHub()
	// Run hub
	go hub.Run()

	// Start test server that upgrades to websocket
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWS(w, r)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Convert http://127.0.0.1:xxxxx to ws://127.0.0.1:xxxxx
	wsURL := "ws" + ts.URL[len("http"): ] + "/ws"

	head := http.Header{}
	head.Set("X-Request-ID", "req-123")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, head)
	require.NoError(t, err)
	defer conn.Close()

	// Nudge server goroutines and ensure registration path runs
	_ = conn.WriteMessage(websocket.TextMessage, []byte("hello"))
	time.Sleep(50 * time.Millisecond)

	// Retry broadcast until we receive it due to non-blocking hub broadcast
	msg := broadcastUntilReceived(t, hub, conn, func() {
		hub.BroadcastMessage("test_event", map[string]string{"hello": "world"})
	})

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(msg, &payload))

	require.Equal(t, "test_event", payload["type"]) // from Message.Type
	data, ok := payload["data"].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "world", data["hello"])
}

func TestWebSocketHub_BroadcastWithRequestID(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWS(w, r)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + ts.URL[len("http"): ] + "/ws"

	head := http.Header{}
	head.Set("X-Request-ID", "abc-req")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, head)
	require.NoError(t, err)
	defer conn.Close()

	_ = conn.WriteMessage(websocket.TextMessage, []byte("hello2"))
	time.Sleep(50 * time.Millisecond)

	msg := broadcastUntilReceived(t, hub, conn, func() {
		hub.BroadcastMessageWithRequestID("abc-req", "evt", map[string]int{"n": 42})
	})

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(msg, &payload))

	require.Equal(t, "evt", payload["type"]) // uses custom payload with request_id
	require.Equal(t, "abc-req", payload["request_id"]) // present when provided
	data, ok := payload["data"].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, float64(42), data["n"]) // JSON numbers are float64
}
