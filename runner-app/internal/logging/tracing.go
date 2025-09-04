package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/gin-gonic/gin"
)

// TraceContext holds tracing information
type TraceContext struct {
	TraceID   string
	SpanID    string
	RequestID string
	UserID    string
	StartTime time.Time
}

// NewTraceContext creates a new trace context
func NewTraceContext() *TraceContext {
	return &TraceContext{
		TraceID:   generateID(),
		SpanID:    generateID(),
		RequestID: generateID(),
		StartTime: time.Now(),
	}
}

// generateID generates a random hex ID
func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// TracingMiddleware adds tracing context to requests
func TracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create or extract trace context
		traceCtx := NewTraceContext()
		
		// Check for existing trace headers
		if traceID := c.GetHeader("X-Trace-ID"); traceID != "" {
			traceCtx.TraceID = traceID
		}
		if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
			traceCtx.RequestID = requestID
		}

		// Add to context
		ctx := context.WithValue(c.Request.Context(), "trace_id", traceCtx.TraceID)
		ctx = context.WithValue(ctx, "span_id", traceCtx.SpanID)
		ctx = context.WithValue(ctx, "request_id", traceCtx.RequestID)
		c.Request = c.Request.WithContext(ctx)

		// Add response headers
		c.Header("X-Trace-ID", traceCtx.TraceID)
		c.Header("X-Request-ID", traceCtx.RequestID)

		// Log request start
		logger := NewStructuredLogger("http").WithContext(ctx)
		logger.Info("request started",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"remote_addr", c.ClientIP(),
			"user_agent", c.Request.UserAgent())

		start := time.Now()
		c.Next()
		duration := time.Since(start)

		// Log request completion
		logger.Info("request completed",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", duration.Milliseconds(),
			"bytes_written", c.Writer.Size())
	}
}

// Span represents a tracing span
type Span struct {
	TraceID     string
	SpanID      string
	ParentSpanID string
	Operation   string
	StartTime   time.Time
	EndTime     *time.Time
	Tags        map[string]interface{}
	Logs        []SpanLog
	logger      *StructuredLogger
}

// SpanLog represents a log entry within a span
type SpanLog struct {
	Timestamp time.Time
	Level     string
	Message   string
	Fields    map[string]interface{}
}

// NewSpan creates a new tracing span
func NewSpan(ctx context.Context, operation string) *Span {
	traceID := getStringFromContext(ctx, "trace_id")
	parentSpanID := getStringFromContext(ctx, "span_id")
	
	span := &Span{
		TraceID:      traceID,
		SpanID:       generateID(),
		ParentSpanID: parentSpanID,
		Operation:    operation,
		StartTime:    time.Now(),
		Tags:         make(map[string]interface{}),
		Logs:         make([]SpanLog, 0),
		logger:       NewStructuredLogger("tracer"),
	}

	span.logger.Debug("span started",
		"trace_id", span.TraceID,
		"span_id", span.SpanID,
		"parent_span_id", span.ParentSpanID,
		"operation", operation)

	return span
}

// SetTag adds a tag to the span
func (s *Span) SetTag(key string, value interface{}) {
	s.Tags[key] = value
}

// LogInfo adds an info log to the span
func (s *Span) LogInfo(message string, fields map[string]interface{}) {
	s.addLog("info", message, fields)
}

// LogError adds an error log to the span
func (s *Span) LogError(message string, err error, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["error"] = err.Error()
	s.addLog("error", message, fields)
}

// addLog adds a log entry to the span
func (s *Span) addLog(level, message string, fields map[string]interface{}) {
	log := SpanLog{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    fields,
	}
	s.Logs = append(s.Logs, log)

	// Also log to structured logger
	logFields := map[string]interface{}{
		"trace_id": s.TraceID,
		"span_id":  s.SpanID,
		"operation": s.Operation,
	}
	for k, v := range fields {
		logFields[k] = v
	}

	switch level {
	case "error":
		s.logger.WithFields(logFields).Error(message)
	case "warn":
		s.logger.WithFields(logFields).Warn(message)
	case "debug":
		s.logger.WithFields(logFields).Debug(message)
	default:
		s.logger.WithFields(logFields).Info(message)
	}
}

// Finish completes the span
func (s *Span) Finish() {
	now := time.Now()
	s.EndTime = &now
	duration := now.Sub(s.StartTime)

	s.logger.Debug("span finished",
		"trace_id", s.TraceID,
		"span_id", s.SpanID,
		"operation", s.Operation,
		"duration_ms", duration.Milliseconds(),
		"tags", s.Tags)
}

// FinishWithError completes the span with an error
func (s *Span) FinishWithError(err error) {
	s.SetTag("error", true)
	s.LogError("span failed", err, nil)
	s.Finish()
}

// Context creates a new context with this span
func (s *Span) Context(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, "span_id", s.SpanID)
	ctx = context.WithValue(ctx, "trace_id", s.TraceID)
	return ctx
}

// JobTracer provides job-specific tracing
type JobTracer struct {
	jobID  string
	logger *StructuredLogger
	spans  map[string]*Span
}

// NewJobTracer creates a tracer for job operations
func NewJobTracer(jobID string) *JobTracer {
	return &JobTracer{
		jobID:  jobID,
		logger: NewStructuredLogger("job-tracer").WithFields(map[string]interface{}{"job_id": jobID}),
		spans:  make(map[string]*Span),
	}
}

// StartSpan starts a new span for job operations
func (jt *JobTracer) StartSpan(ctx context.Context, operation string) *Span {
	span := NewSpan(ctx, operation)
	span.SetTag("job_id", jt.jobID)
	jt.spans[span.SpanID] = span
	return span
}

// TraceJobExecution traces the entire job execution
func (jt *JobTracer) TraceJobExecution(ctx context.Context, fn func(context.Context, *Span) error) error {
	span := jt.StartSpan(ctx, "job_execution")
	defer span.Finish()

	span.SetTag("job_id", jt.jobID)
	span.LogInfo("job execution started", map[string]interface{}{
		"timestamp": time.Now().UTC(),
	})

	err := fn(span.Context(ctx), span)
	if err != nil {
		span.FinishWithError(err)
		return err
	}

	span.LogInfo("job execution completed", map[string]interface{}{
		"timestamp": time.Now().UTC(),
	})
	return nil
}

// GetSpanSummary returns a summary of all spans
func (jt *JobTracer) GetSpanSummary() map[string]interface{} {
	summary := map[string]interface{}{
		"job_id":     jt.jobID,
		"span_count": len(jt.spans),
		"spans":      make([]map[string]interface{}, 0, len(jt.spans)),
	}

	for _, span := range jt.spans {
		spanData := map[string]interface{}{
			"span_id":   span.SpanID,
			"operation": span.Operation,
			"start_time": span.StartTime,
			"tags":      span.Tags,
			"log_count": len(span.Logs),
		}
		if span.EndTime != nil {
			spanData["end_time"] = *span.EndTime
			spanData["duration_ms"] = span.EndTime.Sub(span.StartTime).Milliseconds()
		}
		summary["spans"] = append(summary["spans"].([]map[string]interface{}), spanData)
	}

	return summary
}

// Helper function to get string from context
func getStringFromContext(ctx context.Context, key string) string {
	if val := ctx.Value(key); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return generateID()
}
