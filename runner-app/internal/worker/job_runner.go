package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/golem"
	"github.com/jamie-anson/project-beacon-runner/internal/hybrid"
	"github.com/jamie-anson/project-beacon-runner/internal/ipfs"
	"github.com/jamie-anson/project-beacon-runner/internal/jobspec"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/internal/queue"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
	"github.com/jamie-anson/project-beacon-runner/internal/metrics"
	"github.com/jamie-anson/project-beacon-runner/internal/negotiation"
	"github.com/jamie-anson/project-beacon-runner/internal/geoip"
	models "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// JobRunner consumes job envelopes from Redis and executes single-region runs
// Small interfaces for testability
type jobsRepoIface interface {
	GetJob(ctx context.Context, id string) (idOut string, status string, data []byte, createdAt, updatedAt sql.NullTime, err error)
}

type execRepoIface interface {
	InsertExecution(ctx context.Context, jobID string, providerID string, region string, status string, startedAt time.Time, completedAt time.Time, outputJSON []byte, receiptJSON []byte) (int64, error)
	UpdateRegionVerification(ctx context.Context, executionID int64, regionClaimed sql.NullString, regionObserved sql.NullString, regionVerified sql.NullBool, verificationMethod sql.NullString, evidenceRef sql.NullString) error
}

// ProbeFactory constructs a preflight probe; injectable for tests.
type ProbeFactory func() negotiation.PreflightProbe

type JobRunner struct {
	DB           *sql.DB
	Queue        *queue.Client
	JobsRepo     jobsRepoIface
	ExecRepo     execRepoIface
	Golem        *golem.Service
	Hybrid       *hybrid.Client
	Bundler      *ipfs.Bundler
	QueueName    string
	ProbeFactory ProbeFactory
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
	// Debug logging to identify envelope format issue
	l.Debug().
		Str("payload_json", string(payload)).
		Msg("job runner received envelope")
	
	// Parse envelope
	var env jobEnvelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return fmt.Errorf("invalid envelope: %w", err)
	}
	
	// Debug log parsed envelope
	l.Debug().
		Str("parsed_id", env.ID).
		Time("parsed_enqueued_at", env.EnqueuedAt).
		Int("parsed_attempt", env.Attempt).
		Str("parsed_request_id", env.RequestID).
		Msg("job runner parsed envelope")
	
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
	// Prefer Hybrid Router if configured
	if w.Hybrid != nil {
		prompt := extractPrompt(spec)
		model := extractModel(spec)
		regionPref := mapRegionToRouter(region)
		req := hybrid.InferenceRequest{
			Model:            model,
			Prompt:           prompt,
			Temperature:      0.1,
			MaxTokens:        128,
			RegionPreference: regionPref,
			CostPriority:     true,
		}
		hre, herr := w.Hybrid.RunInference(ctx, req)
		if herr != nil || hre == nil || !hre.Success {
			l.Error().Err(herr).Str("job_id", env.ID).Str("region", regionPref).Msg("hybrid router inference error")
			out := map[string]any{"error": fmt.Sprintf("hybrid error: %v", herr)}
			if hre != nil && hre.Error != "" { out["router_error"] = hre.Error }
			outJSON, _ := json.Marshal(out)
			startedAt := time.Now().UTC()
			completedAt := startedAt
			_, insErr := w.ExecRepo.InsertExecution(ctx, spec.ID, "", regionPref, "failed", startedAt, completedAt, outJSON, nil)
			metrics.ExecutionDurationSeconds.WithLabelValues(regionPref, "failed").Observe(time.Since(execStart).Seconds())
			metrics.JobsFailedTotal.Inc()
			return insErr
		}
		// Success via Hybrid
		out := map[string]any{
			"response": hre.Response,
			"provider": hre.ProviderUsed,
			"metadata": hre.Metadata,
		}
		outJSON, _ := json.Marshal(out)
		startedAt := execStart.UTC()
		completedAt := time.Now().UTC()
		execID, err := w.ExecRepo.InsertExecution(ctx, spec.ID, hre.ProviderUsed, regionPref, "completed", startedAt, completedAt, outJSON, nil)
		if err != nil {
			return fmt.Errorf("insert execution: %w", err)
		}
		metrics.ExecutionDurationSeconds.WithLabelValues(regionPref, "completed").Observe(time.Since(execStart).Seconds())
		metrics.JobsProcessedTotal.Inc()
		if w.Bundler != nil {
			go func(jid string) {
				ctx2, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				_, _ = w.Bundler.StoreBundle(ctx2, jid)
			}(spec.ID)
		}
		if execID > 0 {
			go w.verifyRegionAsync(context.Background(), execID, regionPref)
		}
		return nil
	}

	res, err := golem.ExecuteSingleRegion(ctx, w.Golem, spec, region)
	if err != nil {
		l.Error().Err(err).Str("job_id", env.ID).Str("region", region).Msg("execution error")
		// Persist failed execution row with error details in output
		out := map[string]any{"error": err.Error()}
		outJSON, _ := json.Marshal(out)
		// res may be nil on error; avoid dereference and use safe defaults
		providerID := ""
		startedAt := time.Now().UTC()
		completedAt := startedAt
		_, insErr := w.ExecRepo.InsertExecution(ctx, spec.ID, providerID, region, "failed", startedAt, completedAt, outJSON, nil)
		// Metrics: failed execution
		metrics.ExecutionDurationSeconds.WithLabelValues(region, "failed").Observe(time.Since(execStart).Seconds())
		metrics.JobsFailedTotal.Inc()
		return insErr
	}

	// Marshal output and receipt
	outJSON, _ := json.Marshal(res.Execution.Output)
	recJSON, _ := json.Marshal(res.Receipt)

	status := "completed"
	var startedAt, completedAt time.Time
	if res.Execution != nil {
		status = res.Execution.Status
		startedAt = res.Execution.StartedAt
		completedAt = res.Execution.CompletedAt
	}

	execID, err := w.ExecRepo.InsertExecution(ctx, spec.ID, res.ProviderID, region, status, startedAt, completedAt, outJSON, recJSON)
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

	// Preflight region verification (best-effort, non-fatal)
	// Only attempt if we have an execution row id
	if execID > 0 {
		go func(executionID int64, claimed string) {
			// Short timeout to avoid blocking worker
			pctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			var probe negotiation.PreflightProbe
			if w.ProbeFactory != nil {
				probe = w.ProbeFactory()
			} else {
				probe = negotiation.NewPreflightProbe(negotiation.DefaultHTTPIPFetcher(5*time.Second), geoip.NewResolver())
			}
			observed, _, verr := probe.Verify(pctx, "")
			if verr != nil {
				// Log and continue; do not fail the job
				l2 := logging.FromContext(ctx)
				l2.Warn().Err(verr).Int64("execution_id", executionID).Msg("preflight verification skipped")
				return
			}
			verified := (observed == claimed)
			// Persist verification fields
			_ = w.ExecRepo.UpdateRegionVerification(context.Background(), executionID,
				sql.NullString{String: claimed, Valid: true},
				sql.NullString{String: observed, Valid: true},
				sql.NullBool{Bool: verified, Valid: true},
				sql.NullString{String: "preflight-geoip", Valid: true},
				sql.NullString{}, // evidence ref optional, not stored yet
			)
		}(execID, region)
	}

	return nil
}

// Helper: extract a prompt string from JobSpec input (fallback to generic)
func extractPrompt(spec *models.JobSpec) string {
    if spec != nil && spec.Benchmark.Input.Type == "prompt" {
        if v, ok := spec.Benchmark.Input.Data["prompt"].(string); ok && v != "" {
            return v
        }
    }
    return "Who are you? Describe yourself in 2-3 sentences."
}

// Helper: choose a model name (can be extended later)
func extractModel(spec *models.JobSpec) string {
    // In future, derive from spec.Metadata or Benchmark.Name
    return "llama-3.2-1b"
}

// Helper: map runner regions (US, EU, APAC) to router regions
func mapRegionToRouter(r string) string {
	switch r {
	case "US":
		return "us-east"
	case "EU":
		return "eu-west"
	case "APAC":
		return "asia-pacific"
	default:
		return "eu-west"
	}
}

// Helper: spawn async verification using existing probe logic
func (w *JobRunner) verifyRegionAsync(ctx context.Context, executionID int64, claimed string) {
	// Short timeout to avoid blocking worker
	pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var probe negotiation.PreflightProbe
	if w.ProbeFactory != nil {
		probe = w.ProbeFactory()
	} else {
		probe = negotiation.NewPreflightProbe(negotiation.DefaultHTTPIPFetcher(5*time.Second), geoip.NewResolver())
	}
	observed, _, verr := probe.Verify(pctx, "")
	if verr != nil {
		return
	}
	verified := (observed == claimed)
	_ = w.ExecRepo.UpdateRegionVerification(context.Background(), executionID,
		sql.NullString{String: claimed, Valid: true},
		sql.NullString{String: observed, Valid: true},
		sql.NullBool{Bool: verified, Valid: true},
		sql.NullString{String: "preflight-geoip", Valid: true},
		sql.NullString{},
	)
}
