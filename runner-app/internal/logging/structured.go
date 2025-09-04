package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// StructuredLogger provides structured logging with consistent fields
type StructuredLogger struct {
	logger   *slog.Logger
	zerolog  zerolog.Logger
	metadata map[string]interface{}
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(service string) *StructuredLogger {
	// Configure slog for structured logging
	opts := &slog.HandlerOptions{
		Level: getLogLevel(),
	}

	var handler slog.Handler
	if isProductionMode() {
		// JSON format for production
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		// Text format for development
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)

	// Also maintain zerolog for compatibility
	zerologger := Init()

	return &StructuredLogger{
		logger:  logger,
		zerolog: zerologger,
		metadata: map[string]interface{}{
			"service":     service,
			"version":     os.Getenv("APP_VERSION"),
			"environment": getEnvironment(),
			"region":      os.Getenv("FLY_REGION"),
		},
	}
}

// WithContext adds context-specific fields
func (sl *StructuredLogger) WithContext(ctx context.Context) *StructuredLogger {
	newLogger := &StructuredLogger{
		logger:   sl.logger,
		zerolog:  sl.zerolog,
		metadata: make(map[string]interface{}),
	}

	// Copy existing metadata
	for k, v := range sl.metadata {
		newLogger.metadata[k] = v
	}

	// Add context fields
	if requestID := ctx.Value("request_id"); requestID != nil {
		newLogger.metadata["request_id"] = requestID
	}
	if userID := ctx.Value("user_id"); userID != nil {
		newLogger.metadata["user_id"] = userID
	}
	if jobID := ctx.Value("job_id"); jobID != nil {
		newLogger.metadata["job_id"] = jobID
	}
	if traceID := ctx.Value("trace_id"); traceID != nil {
		newLogger.metadata["trace_id"] = traceID
	}

	return newLogger
}

// WithFields adds arbitrary fields to the logger
func (sl *StructuredLogger) WithFields(fields map[string]interface{}) *StructuredLogger {
	newLogger := &StructuredLogger{
		logger:   sl.logger,
		zerolog:  sl.zerolog,
		metadata: make(map[string]interface{}),
	}

	// Copy existing metadata
	for k, v := range sl.metadata {
		newLogger.metadata[k] = v
	}

	// Add new fields
	for k, v := range fields {
		newLogger.metadata[k] = v
	}

	return newLogger
}

// Info logs an info message
func (sl *StructuredLogger) Info(msg string, args ...interface{}) {
	sl.log(slog.LevelInfo, msg, args...)
}

// Warn logs a warning message
func (sl *StructuredLogger) Warn(msg string, args ...interface{}) {
	sl.log(slog.LevelWarn, msg, args...)
}

// Error logs an error message
func (sl *StructuredLogger) Error(msg string, args ...interface{}) {
	sl.log(slog.LevelError, msg, args...)
}

// Debug logs a debug message
func (sl *StructuredLogger) Debug(msg string, args ...interface{}) {
	sl.log(slog.LevelDebug, msg, args...)
}

// Fatal logs a fatal message and exits
func (sl *StructuredLogger) Fatal(msg string, args ...interface{}) {
	sl.log(slog.LevelError, msg, args...)
	os.Exit(1)
}

// log is the internal logging method
func (sl *StructuredLogger) log(level slog.Level, msg string, args ...interface{}) {
	// Build slog attributes from metadata and args
	attrs := make([]slog.Attr, 0, len(sl.metadata)+len(args)/2)

	// Add metadata
	for k, v := range sl.metadata {
		attrs = append(attrs, slog.Any(k, v))
	}

	// Add args as key-value pairs
	for i := 0; i < len(args)-1; i += 2 {
		if key, ok := args[i].(string); ok {
			attrs = append(attrs, slog.Any(key, args[i+1]))
		}
	}

	sl.logger.LogAttrs(context.Background(), level, msg, attrs...)
}

// JobLogger provides job-specific logging
type JobLogger struct {
	*StructuredLogger
	jobID string
}

// NewJobLogger creates a logger for job operations
func NewJobLogger(jobID string) *JobLogger {
	sl := NewStructuredLogger("job-processor")
	return &JobLogger{
		StructuredLogger: sl.WithFields(map[string]interface{}{
			"job_id": jobID,
		}),
		jobID: jobID,
	}
}

// LogJobStart logs job execution start
func (jl *JobLogger) LogJobStart(region string, provider string) {
	jl.Info("job execution started",
		"region", region,
		"provider", provider,
		"timestamp", time.Now().UTC())
}

// LogJobComplete logs job completion
func (jl *JobLogger) LogJobComplete(duration time.Duration, status string) {
	jl.Info("job execution completed",
		"duration_ms", duration.Milliseconds(),
		"status", status,
		"timestamp", time.Now().UTC())
}

// LogJobError logs job errors with context
func (jl *JobLogger) LogJobError(err error, stage string, context map[string]interface{}) {
	fields := map[string]interface{}{
		"error":     err.Error(),
		"stage":     stage,
		"timestamp": time.Now().UTC(),
	}
	for k, v := range context {
		fields[k] = v
	}
	
	jl.Error("job execution failed", 
		"error", err.Error(),
		"stage", stage,
		"context", context)
}

// AuditLogger provides audit trail logging
type AuditLogger struct {
	*StructuredLogger
}

// NewAuditLogger creates an audit logger
func NewAuditLogger() *AuditLogger {
	sl := NewStructuredLogger("audit")
	return &AuditLogger{
		StructuredLogger: sl,
	}
}

// LogJobSubmission logs job submissions for audit
func (al *AuditLogger) LogJobSubmission(jobID, userID, signature string) {
	al.Info("job submitted",
		"job_id", jobID,
		"user_id", userID,
		"has_signature", signature != "",
		"timestamp", time.Now().UTC(),
		"event_type", "job_submission")
}

// LogJobStatusChange logs job status changes
func (al *AuditLogger) LogJobStatusChange(jobID, oldStatus, newStatus, reason string) {
	al.Info("job status changed",
		"job_id", jobID,
		"old_status", oldStatus,
		"new_status", newStatus,
		"reason", reason,
		"timestamp", time.Now().UTC(),
		"event_type", "status_change")
}

// LogSecurityEvent logs security-related events
func (al *AuditLogger) LogSecurityEvent(eventType, userID, details string) {
	al.Warn("security event",
		"event_type", eventType,
		"user_id", userID,
		"details", details,
		"timestamp", time.Now().UTC(),
		"category", "security")
}

// Helper functions
func getLogLevel() slog.Level {
	levelStr := strings.ToLower(os.Getenv("LOG_LEVEL"))
	switch levelStr {
	case "debug":
		return slog.LevelDebug
	case "info", "":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func isProductionMode() bool {
	return os.Getenv("GIN_MODE") == "release" || 
		   os.Getenv("ENVIRONMENT") == "production" ||
		   os.Getenv("LOG_FORMAT") == "json"
}

func getEnvironment() string {
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		return env
	}
	if os.Getenv("GIN_MODE") == "release" {
		return "production"
	}
	return "development"
}
