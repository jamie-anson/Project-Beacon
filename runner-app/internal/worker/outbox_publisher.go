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
		
		// Debug: Log fetch attempt
		l.Debug().Msg("outbox publisher checking for unpublished entries")

		var publishedAny bool
		var rowCount int
		for rows.Next() {
			rowCount++
			var id int64
			var topic string
			var payload []byte
			if err := rows.Scan(&id, &topic, &payload); err != nil {
				l.Error().Err(err).Msg("outbox scan error")
				metrics.OutboxPublishErrorsTotal.Inc()
				continue
			}
			
			// Debug: Log found entry
			l.Info().Int64("outbox_id", id).Str("topic", topic).Msg("outbox publisher found unpublished entry")
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
			// Debug: Log no entries found
			l.Debug().Int("rows_found", rowCount).Msg("outbox publisher found no unpublished entries")
			// Update metrics for unpublished outbox items
			p.updateOutboxMetrics(ctx)
			// idle sleep
			time.Sleep(500 * time.Millisecond)
		} else {
			l.Info().Int("rows_published", rowCount).Msg("outbox publisher completed batch")
		}
	}
}

// enqueueWithRetry sends the payload directly as JobRunner expects it
func (p *OutboxPublisher) enqueueWithRetry(ctx context.Context, topic string, payload []byte, outboxID int64) error {
	// The JobRunner expects the original envelope format directly from JobsService.
	// Validate that the payload is a proper job envelope before sending.
	var envelope map[string]interface{}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return err
	}
	
	// Ensure required fields are present for JobRunner.handleEnvelope()
	if _, ok := envelope["id"]; !ok {
		return fmt.Errorf("outbox payload missing required 'id' field")
	}
	if _, ok := envelope["enqueued_at"]; !ok {
		return fmt.Errorf("outbox payload missing required 'enqueued_at' field")
	}
	if _, ok := envelope["attempt"]; !ok {
		return fmt.Errorf("outbox payload missing required 'attempt' field")
	}

	// Debug logging to identify envelope format issue
	l := logging.FromContext(ctx)
	l.Debug().
		Str("envelope_id", fmt.Sprintf("%v", envelope["id"])).
		Str("envelope_json", string(payload)).
		Int64("outbox_id", outboxID).
		Msg("outbox publisher sending envelope")

	// Send the original envelope directly - JobRunner expects this exact format
	return p.Queue.Enqueue(ctx, topic, payload)
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
