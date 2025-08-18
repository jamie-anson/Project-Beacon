package service

import (
	"context"
	"database/sql"
	"errors"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/store"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// JobsService orchestrates validation + persistence + outbox
type JobsService struct {
	DB             *sql.DB
	JobsRepo       *store.JobsRepo
	ExecutionsRepo *store.ExecutionsRepo
	DiffsRepo      *store.DiffsRepo
	Outbox         *store.OutboxRepo
}

func NewJobsService(db *sql.DB) *JobsService {
	return &JobsService{
		DB:             db,
		JobsRepo:       store.NewJobsRepo(db),
		ExecutionsRepo: store.NewExecutionsRepo(db),
		DiffsRepo:      store.NewDiffsRepo(db),
		Outbox:         store.NewOutboxRepo(db),
	}
}

// CreateJob validates and persists a job, and writes an outbox message transactionally.
// jobspecJSON is the canonical JSON to store and publish.
func (s *JobsService) CreateJob(ctx context.Context, spec *models.JobSpec, jobspecJSON []byte) error {
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
	envelope := map[string]interface{}{
		"id":          spec.ID,
		"enqueued_at": time.Now().UTC(),
		"attempt":     0,
	}
	payload, mErr := json.Marshal(envelope)
	if mErr != nil {
		return mErr
	}
	if err = s.Outbox.InsertTx(ctx, tx, "jobs", payload); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// GetJob retrieves a job by ID with its current status
func (s *JobsService) GetJob(ctx context.Context, jobspecID string) (*models.JobSpec, string, error) {
	return s.JobsRepo.GetJobByID(ctx, jobspecID)
}

// RecordExecution records a completed execution with its receipt
func (s *JobsService) RecordExecution(ctx context.Context, jobspecID string, receipt *models.Receipt) error {
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

	return nil
}
