package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/golem"
	"github.com/jamie-anson/project-beacon-runner/internal/ipfs"
	"github.com/jamie-anson/project-beacon-runner/internal/jobspec"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/internal/queue"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
	"github.com/jamie-anson/project-beacon-runner/internal/metrics"
)

// JobRunner consumes job envelopes from Redis and executes single-region runs
type JobRunner struct {
	DB        *sql.DB
	Queue     *queue.Client
	JobsRepo  *store.JobsRepo
	ExecRepo  *store.ExecutionsRepo
	Golem     *golem.Service
	Bundler   *ipfs.Bundler
	QueueName string
}

func NewJobRunner(db *sql.DB, q *queue.Client, gsvc *golem.Service, bundler *ipfs.Bundler) *JobRunner {
	return &JobRunner{
		DB:       db,
		Queue:    q,
		JobsRepo: store.NewJobsRepo(db),
		ExecRepo: store.NewExecutionsRepo(db),
		Golem:    gsvc,
		Bundler:  bundler,
	}
}

// NewJobRunnerWithQueue allows specifying the queue name explicitly.
func NewJobRunnerWithQueue(db *sql.DB, q *queue.Client, gsvc *golem.Service, bundler *ipfs.Bundler, queueName string) *JobRunner {
	jr := NewJobRunner(db, q, gsvc, bundler)
	jr.QueueName = queueName
	return jr
}

// Start begins consuming from the jobs queue and processing each job
func (w *JobRunner) Start(ctx context.Context) {
	l := logging.FromContext(ctx)
	l.Info().Msg("job runner started")
	qName := w.QueueName
	if qName == "" {
		qName = queue.JobsQueue
	}
	w.Queue.StartWorker(ctx, qName, func(payload []byte) error {
		return w.handleEnvelope(ctx, payload)
	})
}

type jobEnvelope struct {
	ID         string     `json:"id"`
	EnqueuedAt time.Time  `json:"enqueued_at"`
	Attempt    int        `json:"attempt"`
	RequestID  string     `json:"request_id,omitempty"`
}

func (w *JobRunner) handleEnvelope(ctx context.Context, payload []byte) error {
	l := logging.FromContext(ctx)
	// Parse envelope
	var env jobEnvelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return fmt.Errorf("invalid envelope: %w", err)
	}
	if env.ID == "" {
		return fmt.Errorf("missing job id in envelope")
	}

	// Inject request_id into context for downstream correlation (DB logs, tracing, etc.)
	if env.RequestID != "" {
		ctx = context.WithValue(ctx, "request_id", env.RequestID)
		l = logging.FromContext(ctx)
	}
	l.Info().Str("job_id", env.ID).Int("attempt", env.Attempt).Msg("job envelope received")

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

	// Queue latency metric if we have enqueued_at
	if !env.EnqueuedAt.IsZero() {
		latency := time.Since(env.EnqueuedAt).Seconds()
		metrics.QueueLatencySeconds.WithLabelValues(region).Observe(latency)
	}

	// Execute single region
	execStart := time.Now()
	res, err := golem.ExecuteSingleRegion(ctx, w.Golem, spec, region)
	if err != nil {
		l.Error().Err(err).Str("job_id", env.ID).Str("region", region).Msg("execution error")
		// Persist failed execution row with error details in output
		out := map[string]any{"error": err.Error()}
		outJSON, _ := json.Marshal(out)
		_, insErr := w.ExecRepo.InsertExecution(ctx, spec.ID, res.ProviderID, region, "failed", time.Now().UTC(), time.Now().UTC(), outJSON, nil)
		// Metrics: failed execution
		metrics.ExecutionDurationSeconds.WithLabelValues(region, "failed").Observe(time.Since(execStart).Seconds())
		metrics.JobsFailedTotal.Inc()
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

	// Metrics: successful/finished execution
	metrics.ExecutionDurationSeconds.WithLabelValues(region, status).Observe(time.Since(execStart).Seconds())
	if status == "failed" {
		metrics.JobsFailedTotal.Inc()
	} else {
		metrics.JobsProcessedTotal.Inc()
	}

	// Best-effort: trigger IPFS bundling and CID persistence after success
	if w.Bundler != nil {
		go func(jid string) {
			ctx2, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if _, berr := w.Bundler.StoreBundle(ctx2, jid); berr != nil {
				l2 := logging.FromContext(ctx)
				l2.Error().Err(berr).Str("job_id", jid).Msg("bundler error")
			}
		}(spec.ID)
	}

	return nil
}
