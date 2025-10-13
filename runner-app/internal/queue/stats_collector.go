package queue

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// StatsCollector collects queue statistics
type StatsCollector struct {
	queueName         string
	retryQueue        string
	deadQueue         string
	client            *redis.Client
	processingTracker *ProcessingTracker
}

// NewStatsCollector creates a new StatsCollector
func NewStatsCollector(queueName string, client *redis.Client, processingTracker *ProcessingTracker) *StatsCollector {
	return &StatsCollector{
		queueName:         queueName,
		retryQueue:        queueName + ":retry",
		deadQueue:         queueName + ":dead",
		client:            client,
		processingTracker: processingTracker,
	}
}

// GetQueueStats returns statistics about the queue
func (s *StatsCollector) GetQueueStats(ctx context.Context) (map[string]int64, error) {
	tracer := otel.Tracer("runner/queue/stats")
	ctx, span := tracer.Start(ctx, "StatsCollector.GetQueueStats", trace.WithAttributes(
		attribute.String("queue.name", s.queueName),
		attribute.String("queue.retry", s.retryQueue),
		attribute.String("queue.dead", s.deadQueue),
	))
	defer span.End()

	pipe := s.client.Pipeline()
	
	mainLen := pipe.LLen(ctx, s.queueName)
	retryLen := pipe.ZCard(ctx, s.retryQueue)
	deadLen := pipe.LLen(ctx, s.deadQueue)

	_, err := pipe.Exec(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}


	// Get processing count from tracker (nil-safe)
	var processingCount int64
	if s.processingTracker != nil {
		pc, err := s.processingTracker.GetProcessingCount(ctx)
		if err != nil {
			span.RecordError(err)
			processingCount = 0
		} else {
			processingCount = pc
		}
	} else {
		processingCount = 0
	}

	stats := map[string]int64{
		"main":       mainLen.Val(),
		"retry":      retryLen.Val(),
		"dead":       deadLen.Val(),
		"processing": processingCount,
	}

	span.SetAttributes(
		attribute.Int64("stats.main", stats["main"]),
		attribute.Int64("stats.retry", stats["retry"]),
		attribute.Int64("stats.dead", stats["dead"]),
		attribute.Int64("stats.processing", stats["processing"]),
	)

	return stats, nil
}
