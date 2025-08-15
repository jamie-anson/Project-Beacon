package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/queue"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
)

// OutboxPublisher reads rows from outbox and publishes to Redis queues
 type OutboxPublisher struct {
	DB     *sql.DB
	Outbox *store.OutboxRepo
	Queue  *queue.Client
 }

func NewOutboxPublisher(db *sql.DB, q *queue.Client) *OutboxPublisher {
	return &OutboxPublisher{DB: db, Outbox: store.NewOutboxRepo(db), Queue: q}
}

// Start begins publishing in a loop until context is cancelled.
func (p *OutboxPublisher) Start(ctx context.Context) {
	log.Printf("outbox publisher started")
	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			log.Printf("outbox publisher stopping: %v", ctx.Err())
			return
		default:
		}

		rows, err := p.Outbox.FetchUnpublished(ctx, 100)
		if err != nil {
			log.Printf("outbox fetch error: %v", err)
			time.Sleep(backoff)
			continue
		}

		var publishedAny bool
		for rows.Next() {
			var id int64
			var topic string
			var payload []byte
			if err := rows.Scan(&id, &topic, &payload); err != nil {
				log.Printf("outbox scan error: %v", err)
				continue
			}
			// ensure payload is valid JSON
			var tmp map[string]any
			if err := json.Unmarshal(payload, &tmp); err != nil {
				log.Printf("outbox payload invalid JSON for id=%d: %v", id, err)
				continue
			}

			// publish to Redis list named by topic
			if err := p.Queue.Enqueue(ctx, topic, payload); err != nil {
				log.Printf("outbox enqueue error (id=%d, topic=%s): %v", id, topic, err)
				continue
			}
			if err := p.Outbox.MarkPublished(ctx, id); err != nil {
				log.Printf("outbox mark published error (id=%d): %v", id, err)
				continue
			}
			publishedAny = true
		}
		_ = rows.Close()

		if !publishedAny {
			// idle sleep
			time.Sleep(500 * time.Millisecond)
		}
	}
}
