package service

import (
	"context"
	"database/sql"
	"errors"
	"encoding/json"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/store"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// JobsService orchestrates validation + persistence + outbox
 type JobsService struct {
	DB        *sql.DB
	JobsRepo  *store.JobsRepo
	Outbox    *store.OutboxRepo
 }

func NewJobsService(db *sql.DB) *JobsService {
	return &JobsService{
		DB:       db,
		JobsRepo: store.NewJobsRepo(db),
		Outbox:   store.NewOutboxRepo(db),
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
