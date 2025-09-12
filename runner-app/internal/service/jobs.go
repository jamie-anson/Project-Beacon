package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/cache"
	"github.com/jamie-anson/project-beacon-runner/internal/constants"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"

	// OpenTelemetry
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// JobsService orchestrates validation + persistence + outbox
type JobsService struct {
	DB             *sql.DB
	JobsRepo       *store.JobsRepo
	ExecutionsRepo *store.ExecutionsRepo
	DiffsRepo      *store.DiffsRepo
	Outbox         *store.OutboxRepo
	IdempotencyRepo *store.IdempotencyRepo
	QueueName      string
	Cache          cache.Cache
}

// IdempotentCreateJob checks an idempotency key first; if it already maps to a job,
// it returns the existing job ID without creating or enqueuing. Otherwise it creates
// the job, records the key->job mapping, and enqueues in a single transaction.
// Returns (jobID, reusedExisting, error).
func (s *JobsService) IdempotentCreateJob(ctx context.Context, idemKey string, spec *models.JobSpec, jobspecJSON []byte) (string, bool, error) {
    tracer := otel.Tracer("runner/service/jobs")
    ctx, span := tracer.Start(ctx, "JobsService.IdempotentCreateJob", oteltrace.WithAttributes(
        attribute.String("job.id", spec.ID),
        attribute.String("idempotency.key", idemKey),
    ))
    defer span.End()
    if s.DB == nil {
        return "", false, errors.New("database not initialized")
    }
    // Fast path: if key already exists, return its job ID
    if s.IdempotencyRepo != nil && idemKey != "" {
        if existingID, ok, _ := s.IdempotencyRepo.GetByKey(ctx, idemKey); ok && existingID != "" {
            return existingID, true, nil
        }
    }

    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
    if err != nil {
        return "", false, err
    }
    defer func() {
        if err != nil {
            _ = tx.Rollback()
        }
    }()

    if err = s.JobsRepo.UpsertJobTx(ctx, tx, spec.ID, jobspecJSON, "created"); err != nil {
        return "", false, err
    }
    if s.IdempotencyRepo != nil && idemKey != "" {
        if err = s.IdempotencyRepo.PutTx(ctx, tx, idemKey, spec.ID); err != nil {
            return "", false, err
        }
    }

    // Outbox envelope (as in CreateJob)
    var requestID string
    if v := ctx.Value("request_id"); v != nil {
        if s, ok := v.(string); ok {
            requestID = s
        }
    }
    envelope := map[string]interface{}{
        "id":          spec.ID,
        "enqueued_at": time.Now().UTC(),
        "attempt":     0,
        "request_id":  requestID,
    }
    payload, mErr := json.Marshal(envelope)
    if mErr != nil {
        return "", false, mErr
    }
    if err = s.Outbox.InsertTx(ctx, tx, s.QueueName, payload); err != nil {
        return "", false, err
    }

    if err = tx.Commit(); err != nil {
        return "", false, err
    }
    return spec.ID, false, nil
}

// NewJobsServiceWithQueue creates a JobsService with an explicit queue name.
func NewJobsServiceWithQueue(db *sql.DB, queueName string) *JobsService {
    return &JobsService{
        DB:             db,
        JobsRepo:       store.NewJobsRepo(db),
        ExecutionsRepo: store.NewExecutionsRepo(db),
        DiffsRepo:      store.NewDiffsRepo(db),
        Outbox:         store.NewOutboxRepo(db),
        IdempotencyRepo: store.NewIdempotencyRepo(db),
        QueueName:      queueName,
    }
}

// NewJobsService creates a JobsService using the default queue name.
func NewJobsService(db *sql.DB) *JobsService {
	return NewJobsServiceWithQueue(db, constants.JobsQueueName)
}

// SetCache allows wiring a cache after construction
func (s *JobsService) SetCache(c cache.Cache) {
	s.Cache = c
}

// CreateJob validates and persists a job, and writes an outbox message transactionally.
// jobspecJSON is the canonical JSON to store and publish.
func (s *JobsService) CreateJob(ctx context.Context, spec *models.JobSpec, jobspecJSON []byte) error {
	tracer := otel.Tracer("runner/service/jobs")
	ctx, span := tracer.Start(ctx, "JobsService.CreateJob", oteltrace.WithAttributes(
		attribute.String("job.id", spec.ID),
		attribute.Int("questions.count", len(spec.Questions)),
	))
	defer span.End()
	
	// Log questions persistence for debugging
	if len(spec.Questions) > 0 {
		span.SetAttributes(attribute.StringSlice("questions.list", spec.Questions))
		fmt.Printf("SERVICE: CreateJob %s persisting %d questions: %v\n", spec.ID, len(spec.Questions), spec.Questions)
	} else {
		fmt.Printf("SERVICE: CreateJob %s has no questions\n", spec.ID)
	}
	
	if s.DB == nil {
		return errors.New("database not initialized")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = s.JobsRepo.UpsertJobTx(ctx, tx, spec.ID, jobspecJSON, "created"); err != nil {
		return err
	}

	// Outbox payload: minimal envelope referring to job by ID
	// Try to propagate request ID from context if present
	var requestID string
	if v := ctx.Value("request_id"); v != nil {
		if s, ok := v.(string); ok {
			requestID = s
		}
	}
	envelope := map[string]interface{}{
		"id":          spec.ID,
		"enqueued_at": time.Now().UTC(),
		"attempt":     0,
		"request_id":  requestID,
	}
	payload, mErr := json.Marshal(envelope)
	if mErr != nil {
		return mErr
	}
	if err = s.Outbox.InsertTx(ctx, tx, s.QueueName, payload); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// GetJob retrieves a job by ID with its current status
func (s *JobsService) GetJob(ctx context.Context, jobspecID string) (*models.JobSpec, string, error) {
	tracer := otel.Tracer("runner/service/jobs")
	ctx, span := tracer.Start(ctx, "JobsService.GetJob", oteltrace.WithAttributes(
		attribute.String("job.id", jobspecID),
	))
	defer span.End()
	// Try cache first
	if s.Cache != nil {
		if b, ok, _ := s.Cache.Get(ctx, "job:"+jobspecID); ok {
			var cached struct {
				Spec   models.JobSpec `json:"spec"`
				Status string         `json:"status"`
			}
			if err := json.Unmarshal(b, &cached); err == nil {
				return &cached.Spec, cached.Status, nil
			}
		}
	}

	spec, status, err := s.JobsRepo.GetJobByID(ctx, jobspecID)
	if err != nil {
		return nil, "", err
	}
	if s.Cache != nil && spec != nil {
		payload, _ := json.Marshal(struct {
			Spec   *models.JobSpec `json:"spec"`
			Status string         `json:"status"`
		}{Spec: spec, Status: status})
		_ = s.Cache.Set(ctx, "job:"+jobspecID, payload, 30*time.Second)
	}
	return spec, status, nil
}

// RecordExecution records a completed execution with its receipt
func (s *JobsService) RecordExecution(ctx context.Context, jobspecID string, receipt *models.Receipt) error {
	tracer := otel.Tracer("runner/service/jobs")
	ctx, span := tracer.Start(ctx, "JobsService.RecordExecution", oteltrace.WithAttributes(
		attribute.String("job.id", jobspecID),
		attribute.String("execution.status", receipt.ExecutionDetails.Status),
	))
	defer span.End()
	// Verify the receipt signature
	if err := receipt.VerifySignature(); err != nil {
		return fmt.Errorf("receipt signature verification failed: %w", err)
	}

	// Create execution record
	_, err := s.ExecutionsRepo.CreateExecution(ctx, jobspecID, receipt)
	if err != nil {
		return fmt.Errorf("failed to record execution: %w", err)
	}

	// Update job status based on execution results
	var newStatus string
	switch receipt.ExecutionDetails.Status {
	case "completed":
		newStatus = "running" // Still running if we have partial results
	case "failed":
		newStatus = "failed"
	default:
		newStatus = "running"
	}

	if err := s.JobsRepo.UpdateJobStatus(ctx, jobspecID, newStatus); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Best-effort cache invalidation via short TTL; explicit delete if available
	if s.Cache != nil {
		_ = s.Cache.Set(ctx, "job:"+jobspecID, nil, 1*time.Nanosecond)
		_ = s.Cache.Set(ctx, "job:"+jobspecID+":latest_receipt", nil, 1*time.Nanosecond)
	}
	return nil
}

// GetLatestReceiptCached returns the latest receipt using cache when possible
func (s *JobsService) GetLatestReceiptCached(ctx context.Context, jobspecID string) (*models.Receipt, error) {
	tracer := otel.Tracer("runner/service/jobs")
	ctx, span := tracer.Start(ctx, "JobsService.GetLatestReceiptCached", oteltrace.WithAttributes(
		attribute.String("job.id", jobspecID),
	))
	defer span.End()
	if s.Cache != nil {
		if b, ok, _ := s.Cache.Get(ctx, "job:"+jobspecID+":latest_receipt"); ok {
			var rec models.Receipt
			if err := json.Unmarshal(b, &rec); err == nil {
				return &rec, nil
			}
		}
	}
	rec, err := s.ExecutionsRepo.GetReceiptByJobSpecID(ctx, jobspecID)
	if err != nil {
		return nil, err
	}
	if s.Cache != nil && rec != nil {
		b, _ := json.Marshal(rec)
		_ = s.Cache.Set(ctx, "job:"+jobspecID+":latest_receipt", b, 15*time.Second)
	}
	return rec, nil
}

// RepublishJob republishes a job to the outbox queue (for stuck jobs)
func (s *JobsService) RepublishJob(ctx context.Context, jobspecID string) error {
	tracer := otel.Tracer("runner/service/jobs")
	ctx, span := tracer.Start(ctx, "JobsService.RepublishJob", oteltrace.WithAttributes(
		attribute.String("job.id", jobspecID),
	))
	defer span.End()
	
	if s.DB == nil {
		return errors.New("database not initialized")
	}
	
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Create outbox envelope for the job
	var requestID string
	if v := ctx.Value("request_id"); v != nil {
		if s, ok := v.(string); ok {
			requestID = s
		}
	}
	envelope := map[string]interface{}{
		"id":          jobspecID,
		"enqueued_at": time.Now().UTC(),
		"attempt":     0,
		"request_id":  requestID,
	}
	payload, mErr := json.Marshal(envelope)
	if mErr != nil {
		return mErr
	}
	
	// Insert into outbox
	if err = s.Outbox.InsertTx(ctx, tx, s.QueueName, payload); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}
