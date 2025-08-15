package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/golem"
	"github.com/jamie-anson/project-beacon-runner/internal/jobspec"
	"github.com/jamie-anson/project-beacon-runner/internal/queue"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
)

// JobRunner consumes job envelopes from Redis and executes single-region runs
 type JobRunner struct {
	DB        *sql.DB
	Queue     *queue.Client
	JobsRepo  *store.JobsRepo
	ExecRepo  *store.ExecutionsRepo
	Golem     *golem.Service
}

func NewJobRunner(db *sql.DB, q *queue.Client, gsvc *golem.Service) *JobRunner {
	return &JobRunner{
		DB:       db,
		Queue:    q,
		JobsRepo: store.NewJobsRepo(db),
		ExecRepo: store.NewExecutionsRepo(db),
		Golem:    gsvc,
	}
}

// Start begins consuming from the jobs queue and processing each job
func (w *JobRunner) Start(ctx context.Context) {
	log.Printf("job runner started")
	w.Queue.StartWorker(ctx, queue.JobsQueue, func(payload []byte) error {
		return w.handleEnvelope(ctx, payload)
	})
}

type jobEnvelope struct {
	ID         string     `json:"id"`
	EnqueuedAt time.Time  `json:"enqueued_at"`
	Attempt    int        `json:"attempt"`
}

func (w *JobRunner) handleEnvelope(ctx context.Context, payload []byte) error {
	// Parse envelope
	var env jobEnvelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return fmt.Errorf("invalid envelope: %w", err)
	}
	if env.ID == "" {
		return fmt.Errorf("missing job id in envelope")
	}

	// Load stored JobSpec JSON
	_, _, jobspecJSON, _, _, err := w.JobsRepo.GetJob(ctx, env.ID)
	if err != nil {
		return fmt.Errorf("get job: %w", err)
	}
	if len(jobspecJSON) == 0 {
		return fmt.Errorf("empty jobspec JSON for %s", env.ID)
	}

	// Validate/parse JobSpec
	validator := jobspec.NewValidator()
	spec, err := validator.ValidateJobSpec(jobspecJSON)
	if err != nil {
		return fmt.Errorf("jobspec validate: %w", err)
	}

	// Choose first region (MVP single-region)
	if len(spec.Constraints.Regions) == 0 {
		return fmt.Errorf("no regions in job constraints")
	}
	region := spec.Constraints.Regions[0]

	// Execute single region
	res, err := golem.ExecuteSingleRegion(ctx, w.Golem, spec, region)
	if err != nil {
		log.Printf("execution error for job %s: %v", env.ID, err)
		// Persist failed execution row with error details in output
		out := map[string]any{"error": err.Error()}
		outJSON, _ := json.Marshal(out)
		_, insErr := w.ExecRepo.InsertExecution(ctx, spec.ID, res.ProviderID, region, "failed", time.Now().UTC(), time.Now().UTC(), outJSON, nil)
		return insErr
	}

	// Marshal output and receipt
	outJSON, _ := json.Marshal(res.Execution.Output)
	recJSON, _ := json.Marshal(res.Receipt)

	status := "completed"
	if res.Execution != nil {
		status = res.Execution.Status
	}

	_, err = w.ExecRepo.InsertExecution(ctx, spec.ID, res.ProviderID, region, status, res.Execution.StartedAt, res.Execution.CompletedAt, outJSON, recJSON)
	if err != nil {
		return fmt.Errorf("insert execution: %w", err)
	}

	return nil
}
