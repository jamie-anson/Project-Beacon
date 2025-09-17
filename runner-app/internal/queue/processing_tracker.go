package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ProcessingTracker manages job processing state and stale job recovery
type ProcessingTracker struct {
	queueName         string
	client            *redis.Client
	visibilityTimeout time.Duration
	retryHandler      *RetryHandler
}

// NewProcessingTracker creates a new ProcessingTracker
func NewProcessingTracker(queueName string, client *redis.Client, retryHandler *RetryHandler) *ProcessingTracker {
	return &ProcessingTracker{
		queueName:         queueName,
		client:            client,
		visibilityTimeout: 10 * time.Minute,
		retryHandler:      retryHandler,
	}
}

// MarkAsProcessing marks a job as being processed
func (p *ProcessingTracker) MarkAsProcessing(ctx context.Context, message *JobMessage) error {
	processingKey := fmt.Sprintf("%s:processing:%s", p.queueName, message.ID)
	messageJSON, _ := json.Marshal(message)
	
	if err := p.client.SetEX(ctx, processingKey, messageJSON, p.visibilityTimeout).Err(); err != nil {
		log.Printf("Warning: failed to mark job %s as processing: %v", message.ID, err)
		return err
	}
	return nil
}

// MarkAsCompleted removes a job from the processing set
func (p *ProcessingTracker) MarkAsCompleted(ctx context.Context, message *JobMessage) error {
	tracer := otel.Tracer("runner/queue/processing")
	ctx, span := tracer.Start(ctx, "ProcessingTracker.MarkAsCompleted", trace.WithAttributes(
		attribute.String("queue.name", p.queueName),
		attribute.String("job.id", message.ID),
		attribute.String("jobspec.id", message.JobSpecID),
		attribute.Int("job.attempts", message.Attempts),
	))
	defer span.End()

	processingKey := fmt.Sprintf("%s:processing:%s", p.queueName, message.ID)
	
	// Remove from processing set
	if err := p.client.Del(ctx, processingKey).Err(); err != nil {
		log.Printf("Warning: failed to remove completed job %s from processing: %v", message.ID, err)
		return err
	}

	log.Printf("Completed job %s for JobSpec %s", message.ID, message.JobSpecID)
	return nil
}

// MarkAsFailed removes a job from the processing set when it fails
func (p *ProcessingTracker) MarkAsFailed(ctx context.Context, message *JobMessage) error {
	processingKey := fmt.Sprintf("%s:processing:%s", p.queueName, message.ID)
	
	// Remove from processing set
	if err := p.client.Del(ctx, processingKey).Err(); err != nil {
		log.Printf("Warning: failed to remove failed job %s from processing: %v", message.ID, err)
		return err
	}
	return nil
}

// RecoverStaleJobs recovers jobs that have been processing too long
func (p *ProcessingTracker) RecoverStaleJobs(ctx context.Context) error {
	tracer := otel.Tracer("runner/queue/processing")
	ctx, span := tracer.Start(ctx, "ProcessingTracker.RecoverStaleJobs", trace.WithAttributes(
		attribute.String("queue.name", p.queueName),
	))
	defer span.End()

	pattern := fmt.Sprintf("%s:processing:*", p.queueName)
	keys, err := p.client.Keys(ctx, pattern).Result()
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to get processing keys: %w", err)
	}

	recovered := 0
	for _, key := range keys {
		// Check if key has expired (TTL <= 0)
		ttl, err := p.client.TTL(ctx, key).Result()
		if err != nil {
			continue
		}

		if ttl <= 0 {
			// Job has expired, recover it
			messageJSON, err := p.client.Get(ctx, key).Result()
			if err != nil {
				continue
			}

			var message JobMessage
			if err := json.Unmarshal([]byte(messageJSON), &message); err != nil {
				continue
			}

			// Remove from processing
			p.client.Del(ctx, key)

			// Re-queue for retry using retry handler
			if err := p.retryHandler.HandleFailure(ctx, &message, fmt.Errorf("job processing timeout")); err != nil {
				log.Printf("Failed to recover stale job %s: %v", message.ID, err)
				continue
			}

			recovered++
			span.AddEvent("recovered_stale_job", trace.WithAttributes(
				attribute.String("job.id", message.ID),
				attribute.String("jobspec.id", message.JobSpecID),
			))
		}
	}

	if recovered > 0 {
		log.Printf("Recovered %d stale jobs", recovered)
	}

	span.SetAttributes(attribute.Int("recovered.count", recovered))
	return nil
}

// GetProcessingCount returns the number of jobs currently being processed
func (p *ProcessingTracker) GetProcessingCount(ctx context.Context) (int64, error) {
	pattern := fmt.Sprintf("%s:processing:*", p.queueName)
	keys, err := p.client.Keys(ctx, pattern).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get processing keys: %w", err)
	}
	return int64(len(keys)), nil
}
