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
	
	// Add panic recovery for crash detection
	defer func() {
		if r := recover(); r != nil {
			l.Error().Interface("panic", r).Msg("outbox publisher crashed with panic")
			metrics.OutboxPublishErrorsTotal.Inc()
		}
		l.Warn().Msg("outbox publisher exiting")
	}()
	
	backoff := time.Second
	consecutiveErrors := 0
	
	for {
		select {
		case <-ctx.Done():
			l.Warn().Err(ctx.Err()).Msg("outbox publisher stopping due to context cancellation")
			return
		default:
		}

		l.Debug().Msg("outbox publisher fetching unpublished entries")
		rows, err := p.Outbox.FetchUnpublished(ctx, 100)
		if err != nil {
			consecutiveErrors++
			l.Error().Err(err).Int("consecutive_errors", consecutiveErrors).Msg("outbox fetch error")
			metrics.OutboxPublishErrorsTotal.Inc()
			
			// Exponential backoff for consecutive errors
			backoffDuration := time.Duration(consecutiveErrors) * backoff
			if backoffDuration > 30*time.Second {
				backoffDuration = 30 * time.Second
			}
			l.Warn().Dur("backoff", backoffDuration).Msg("outbox publisher backing off due to errors")
			time.Sleep(backoffDuration)
			continue
		}
		
		// Reset error counter on successful fetch
		if consecutiveErrors > 0 {
			l.Info().Int("recovered_from_errors", consecutiveErrors).Msg("outbox publisher recovered from errors")
			consecutiveErrors = 0
		}

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
			// Update metrics for unpublished outbox items
			p.updateOutboxMetrics(ctx)
			// idle sleep
			time.Sleep(500 * time.Millisecond)
		} else {
			l.Info().Int("rows_published", rowCount).Msg("outbox publisher completed batch")
		}
	}
}

// enqueueWithRetry sends the payload directly as JobRunner expects it with Redis connection resilience
func (p *OutboxPublisher) enqueueWithRetry(ctx context.Context, topic string, payload []byte, outboxID int64) error {
	l := logging.FromContext(ctx)
	
	// The JobRunner expects the original envelope format directly from JobsService.
	// Validate that the payload is a proper job envelope before sending.
	var envelope map[string]interface{}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		l.Error().Err(err).Int64("outbox_id", outboxID).Msg("outbox payload JSON unmarshal failed")
		return fmt.Errorf("invalid JSON payload: %w", err)
	}
	
	// Ensure required fields are present for JobRunner.handleEnvelope()
	requiredFields := []string{"id", "enqueued_at", "attempt"}
	for _, field := range requiredFields {
		if _, ok := envelope[field]; !ok {
			l.Error().Str("missing_field", field).Int64("outbox_id", outboxID).Msg("outbox payload missing required field")
			return fmt.Errorf("outbox payload missing required '%s' field", field)
		}
	}

	// Debug logging to identify envelope format issue
	l.Debug().
		Str("envelope_id", fmt.Sprintf("%v", envelope["id"])).
		Str("envelope_json", string(payload)).
		Int64("outbox_id", outboxID).
		Msg("outbox publisher sending envelope")

	// Implement Redis connection retry with exponential backoff
	maxRetries := 3
	baseDelay := 100 * time.Millisecond
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Send the original envelope directly - JobRunner expects this exact format
		err := p.Queue.Enqueue(ctx, topic, payload)
		if err == nil {
			if attempt > 0 {
				l.Info().Int("retry_attempt", attempt).Int64("outbox_id", outboxID).Msg("Redis enqueue succeeded after retry")
			}
			return nil
		}
		
		// Log Redis connection error with details
		l.Error().
			Err(err).
			Int("attempt", attempt+1).
			Int("max_retries", maxRetries).
			Int64("outbox_id", outboxID).
			Str("topic", topic).
			Msg("Redis enqueue failed")
		
		// Don't retry on last attempt
		if attempt == maxRetries-1 {
			l.Error().Err(err).Int64("outbox_id", outboxID).Msg("Redis enqueue failed after all retries")
			return fmt.Errorf("Redis enqueue failed after %d retries: %w", maxRetries, err)
		}
		
		// Exponential backoff
		delay := time.Duration(1<<attempt) * baseDelay
		l.Warn().Dur("delay", delay).Int("next_attempt", attempt+2).Msg("Redis enqueue retry backoff")
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next retry
		}
	}
	
	return fmt.Errorf("Redis enqueue failed after %d retries", maxRetries)
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
