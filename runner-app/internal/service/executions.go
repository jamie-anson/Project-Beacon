package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/internal/store"

	// OpenTelemetry
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// ExecutionService handles execution-related operations
type ExecutionService struct {
	DB           *sql.DB
	JobsRepo     *store.JobsRepo
	ExecRepo     *store.ExecutionsRepo
}

// NewExecutionService creates a new ExecutionService
func NewExecutionService(db *sql.DB) *ExecutionService {
	return &ExecutionService{
		DB:       db,
		JobsRepo: store.NewJobsRepo(db),
		ExecRepo: store.NewExecutionsRepo(db),
	}
}

// RecordEarlyFailure attempts to persist a failed execution row before dispatch
// and marks the job as failed. Region is required by schema; use "unknown" when not applicable.
func (s *ExecutionService) RecordEarlyFailure(ctx context.Context, jobID string, reason error, region string, extras map[string]any) error {
	tracer := otel.Tracer("runner/service/executions")
	ctx, span := tracer.Start(ctx, "ExecutionService.RecordEarlyFailure", oteltrace.WithAttributes(
		attribute.String("job.id", jobID),
		attribute.String("region", region),
		attribute.String("error", reason.Error()),
	))
	defer span.End()

	l := logging.FromContext(ctx)
	if jobID == "" {
		return nil
	}
	if region == "" {
		region = "unknown"
	}

	// Build output JSON with error and any extras
	out := map[string]any{"error": reason.Error()}
	for k, v := range extras {
		out[k] = v
	}
	outJSON, _ := json.Marshal(out)

	startedAt := time.Now().UTC()
	completedAt := startedAt

	// Insert failed execution; ignore errors (e.g., job not found)
	execID, insErr := s.ExecRepo.InsertExecution(ctx, jobID, "", region, "failed", startedAt, completedAt, outJSON, nil)
	if insErr != nil {
		l.Error().Err(insErr).Str("job_id", jobID).Msg("early failure: failed to insert execution record")
		span.RecordError(insErr)
	} else {
		l.Info().Str("job_id", jobID).Int64("execution_id", execID).Msg("early failure: execution record created")
		span.SetAttributes(attribute.Int64("execution.id", execID))
	}

	// Best-effort: mark job as failed
	if uerr := s.JobsRepo.UpdateJobStatus(ctx, jobID, "failed"); uerr != nil {
		l.Error().Err(uerr).Str("job_id", jobID).Msg("early failure: failed to mark job as failed")
		span.RecordError(uerr)
		return uerr
	} else {
		l.Info().Str("job_id", jobID).Msg("early failure: job marked as failed")
	}

	return insErr
}
