package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/internal/metrics"
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
	l := logging.FromContext(ctx)
	l.Info().Msg("outbox publisher started")
	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			l.Warn().Err(ctx.Err()).Msg("outbox publisher stopping")
			return
		default:
		}

		rows, err := p.Outbox.FetchUnpublished(ctx, 100)
		if err != nil {
			l.Error().Err(err).Msg("outbox fetch error")
			// Count fetch errors towards publish error budget
			metrics.OutboxPublishErrorsTotal.Inc()
			time.Sleep(backoff)
			continue
		}

		var publishedAny bool
		for rows.Next() {
			var id int64
			var topic string
			var payload []byte
			if err := rows.Scan(&id, &topic, &payload); err != nil {
				l.Error().Err(err).Msg("outbox scan error")
				metrics.OutboxPublishErrorsTotal.Inc()
				continue
			}
			// ensure payload is valid JSON
			var tmp map[string]any
			if err := json.Unmarshal(payload, &tmp); err != nil {
				l.Error().Err(err).Int64("outbox_id", id).Msg("outbox payload invalid JSON")
				metrics.OutboxPublishErrorsTotal.Inc()
				continue
			}

			// publish to Redis using advanced queue for retry support
			if err := p.enqueueWithRetry(ctx, topic, payload, id); err != nil {
				l.Error().Err(err).Int64("outbox_id", id).Str("topic", topic).Msg("outbox enqueue error")
				metrics.OutboxPublishErrorsTotal.Inc()
				continue
			}
			if err := p.Outbox.MarkPublished(ctx, id); err != nil {
				l.Error().Err(err).Int64("outbox_id", id).Msg("outbox mark published error")
				metrics.OutboxPublishErrorsTotal.Inc()
				continue
			}
			// Successful publish
			metrics.OutboxPublishedTotal.Inc()
			publishedAny = true
		}
		_ = rows.Close()

		if !publishedAny {
			// Update metrics for unpublished outbox items
			p.updateOutboxMetrics(ctx)
			// idle sleep
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// enqueueWithRetry wraps the payload in a job message and enqueues with retry support
func (p *OutboxPublisher) enqueueWithRetry(ctx context.Context, topic string, payload []byte, outboxID int64) error {
	// The advanced RedisQueue expects a JobMessage JSON with string fields.
	// Extract the JobSpec ID from the outbox payload, then build a proper message.
	var envelope map[string]interface{}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return err
	}
	jsID, _ := envelope["id"].(string)
	if jsID == "" {
		return fmt.Errorf("outbox payload missing jobspec id")
	}

	message := map[string]interface{}{
		"id":           fmt.Sprintf("%s:%d", jsID, time.Now().UnixNano()),
		"jobspec_id":   jsID,
		"action":       "run",
		"payload":      envelope, // keep original envelope for handler
		"attempts":     0,
		"max_retries":  3,
		"enqueued_at":  time.Now().UTC(),
	}

	jobData, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Use the simple queue client which now delegates to advanced queue for dequeue
	return p.Queue.Enqueue(ctx, topic, jobData)
}

// updateOutboxMetrics collects and updates Prometheus metrics for outbox monitoring
func (p *OutboxPublisher) updateOutboxMetrics(ctx context.Context) {
	count, oldestAge, err := p.Outbox.GetUnpublishedStats(ctx)
	if err != nil {
        l := logging.FromContext(ctx)
        l.Error().Err(err).Msg("failed to get outbox stats")
        return
    }
	
	metrics.OutboxUnpublishedCount.Set(float64(count))
	metrics.OutboxOldestUnpublishedAge.Set(oldestAge)
}
