package logging

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
)

// GenerateTraceID generates a new trace ID for distributed tracing
func GenerateTraceID() uuid.UUID {
	return uuid.New()
}

// DBTracer provides database persistence for distributed tracing
type DBTracer struct {
	db      *sql.DB
	enabled bool
}

// NewDBTracer creates a new database tracer
// Respects ENABLE_DB_TRACING environment variable (default: false)
func NewDBTracer(db *sql.DB) *DBTracer {
	enabled := os.Getenv("ENABLE_DB_TRACING") == "true"
	
	// Debug logging
	envValue := os.Getenv("ENABLE_DB_TRACING")
	if enabled {
		println("🔍 DBTracer: ENABLED (ENABLE_DB_TRACING=" + envValue + ")")
	} else {
		println("⚠️  DBTracer: DISABLED (ENABLE_DB_TRACING=" + envValue + ")")
	}
	
	return &DBTracer{
		db:      db,
		enabled: enabled,
	}
}

// RecordSpan inserts a new span into the database
// Non-blocking: errors are logged but don't crash the application
func (dt *DBTracer) RecordSpan(ctx context.Context, span *DBSpan) error {
	if !dt.enabled || dt.db == nil {
		return nil // Tracing disabled, skip silently
	}

	// Convert metadata to JSONB
	metadataJSON, err := json.Marshal(span.Metadata)
	if err != nil {
		// Log but don't fail
		metadataJSON = []byte("{}")
	}

	// Non-blocking INSERT
	_, err = dt.db.ExecContext(ctx, `
		INSERT INTO trace_spans 
		(trace_id, span_id, parent_span_id, service, operation, 
		 started_at, completed_at, duration_ms, status, 
		 job_id, execution_id, model_id, region, metadata, 
		 error_message, error_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`,
		span.TraceID,
		span.SpanID,
		span.ParentSpanID,
		span.Service,
		span.Operation,
		span.StartedAt,
		span.CompletedAt,
		span.DurationMs,
		span.Status,
		span.JobID,
		span.ExecutionID,
		span.ModelID,
		span.Region,
		metadataJSON,
		span.ErrorMessage,
		span.ErrorType,
	)

	if err != nil {
		// Log error but DON'T crash - tracing is optional
		logger := NewStructuredLogger("db-tracer")
		logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"trace_id":   span.TraceID,
			"span_id":    span.SpanID,
			"service":    span.Service,
			"operation":  span.Operation,
		}).Error("failed to persist trace span to database")
	}

	return err
}

// UpdateSpan updates an existing span with completion data
// Used to set completed_at, duration_ms, status, and error info
func (dt *DBTracer) UpdateSpan(ctx context.Context, spanID uuid.UUID, completed time.Time, status string, errorMsg *string, errorType *string) error {
	if !dt.enabled || dt.db == nil {
		return nil
	}

	_, err := dt.db.ExecContext(ctx, `
		UPDATE trace_spans 
		SET completed_at = $2,
		    duration_ms = EXTRACT(EPOCH FROM ($2 - started_at)) * 1000,
		    status = $3,
		    error_message = $4,
		    error_type = $5
		WHERE span_id = $1
	`, spanID, completed, status, errorMsg, errorType)

	if err != nil {
		logger := NewStructuredLogger("db-tracer")
		logger.WithFields(map[string]interface{}{
			"error":   err.Error(),
			"span_id": spanID,
		}).Error("failed to update trace span")
	}

	return err
}

// DBSpan represents a span stored in the database
type DBSpan struct {
	TraceID       uuid.UUID
	SpanID        uuid.UUID
	ParentSpanID  *uuid.UUID
	Service       string
	Operation     string
	StartedAt     time.Time
	CompletedAt   *time.Time
	DurationMs    *int
	Status        string // 'started', 'completed', 'failed', 'timeout'
	JobID         *string
	ExecutionID   *int64
	ModelID       *string
	Region        *string
	Metadata      map[string]interface{}
	ErrorMessage  *string
	ErrorType     *string
}

// StartSpan creates a new span in started state
// Returns the span which can be completed later
func (dt *DBTracer) StartSpan(ctx context.Context, traceID uuid.UUID, parentSpanID *uuid.UUID, service, operation string, metadata map[string]interface{}) (*DBSpan, error) {
	// Debug logging
	println("🔍 StartSpan called: service=" + service + ", operation=" + operation + ", enabled=" + fmt.Sprint(dt.enabled))
	
	span := &DBSpan{
		TraceID:      traceID,
		SpanID:       uuid.New(),
		ParentSpanID: parentSpanID,
		Service:      service,
		Operation:    operation,
		StartedAt:    time.Now(),
		Status:       "started",
		Metadata:     metadata,
	}

	// Record immediately in started state
	err := dt.RecordSpan(ctx, span)
	if err != nil {
		println("❌ StartSpan RecordSpan error: " + err.Error())
	} else {
		println("✅ StartSpan RecordSpan success")
	}
	return span, err
}

// CompleteSpan marks a span as completed
func (dt *DBTracer) CompleteSpan(ctx context.Context, span *DBSpan, status string) error {
	if span == nil {
		return nil
	}

	completedAt := time.Now()
	duration := int(completedAt.Sub(span.StartedAt).Milliseconds())
	
	span.CompletedAt = &completedAt
	span.DurationMs = &duration
	span.Status = status

	return dt.UpdateSpan(ctx, span.SpanID, completedAt, status, nil, nil)
}

// CompleteSpanWithError marks a span as failed with error details
func (dt *DBTracer) CompleteSpanWithError(ctx context.Context, span *DBSpan, err error, errorType string) error {
	if span == nil {
		return nil
	}

	completedAt := time.Now()
	duration := int(completedAt.Sub(span.StartedAt).Milliseconds())
	errorMsg := err.Error()
	
	span.CompletedAt = &completedAt
	span.DurationMs = &duration
	span.Status = "failed"
	span.ErrorMessage = &errorMsg
	span.ErrorType = &errorType

	return dt.UpdateSpan(ctx, span.SpanID, completedAt, "failed", &errorMsg, &errorType)
}

// SetExecutionContext adds execution context to a span
// Used to link trace spans with executions table
func (span *DBSpan) SetExecutionContext(jobID string, executionID int64, modelID, region string) {
	span.JobID = &jobID
	span.ExecutionID = &executionID
	span.ModelID = &modelID
	span.Region = &region
}

// IsEnabled returns whether database tracing is enabled
func (dt *DBTracer) IsEnabled() bool {
	return dt.enabled
}
