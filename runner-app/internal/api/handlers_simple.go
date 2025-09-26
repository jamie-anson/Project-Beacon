package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/jamie-anson/project-beacon-runner/internal/api/processors"
	"github.com/jamie-anson/project-beacon-runner/internal/api/security"
	"github.com/jamie-anson/project-beacon-runner/internal/config"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/internal/metrics"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
	legacysecurity "github.com/jamie-anson/project-beacon-runner/internal/security"
)

// JobsHandler handles job-related requests
type JobsHandler struct {
	jobsService       *service.JobsService
	cfg               *config.Config
	jobSpecProcessor  *processors.JobSpecProcessor
	securityPipeline  *security.SecurityPipeline
}

// NewJobsHandler creates a new jobs handler
func NewJobsHandler(jobsService *service.JobsService, cfg *config.Config, redisClient *redis.Client) *JobsHandler {
	// Create JobSpec processor
	jobSpecProcessor := processors.NewJobSpecProcessor()
	
	// Create legacy security components for the security pipeline
	var replayProtection *legacysecurity.ReplayProtection
	var rateLimiter *legacysecurity.RateLimiter
	
	if redisClient != nil {
		replayProtection = legacysecurity.NewReplayProtection(redisClient, cfg.TimestampMaxAge)
		rateLimiter = legacysecurity.NewRateLimiter(redisClient)
	}
	
	// Create security pipeline with all components
	securityPipeline := security.NewSecurityPipeline(cfg, replayProtection, rateLimiter)
	
	return &JobsHandler{
		jobsService:      jobsService,
		cfg:             cfg,
		jobSpecProcessor: jobSpecProcessor,
		securityPipeline: securityPipeline,
	}
}

// CreateJob handles job creation requests
func (h *JobsHandler) CreateJob(c *gin.Context) {
	l := logging.FromContext(c.Request.Context())
	l.Info().Msg("DIAGNOSTIC: CreateJob request started")
	
	// Parse and process JobSpec using the processor (includes validation and ID generation)
	spec, raw, err := h.jobSpecProcessor.ProcessJobSpec(c)
	if err != nil {
		l.Error().Err(err).Msg("DIAGNOSTIC: JobSpec processing failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	l.Info().
		Str("job_id", spec.ID).
		Str("request_body_length", strconv.Itoa(len(raw))).
		Msg("DIAGNOSTIC: JobSpec processed successfully with ID")
	
	// Run security pipeline (trust evaluation, rate limiting, replay protection, signature verification)
	clientIP := c.ClientIP()
	
	if err := h.securityPipeline.ValidateJobSpec(c.Request.Context(), spec, raw, clientIP); err != nil {
		l.Error().Err(err).Str("job_id", spec.ID).Msg("DIAGNOSTIC: Security validation failed")
		// Check if it's a structured error with specific response
		if validationErr, ok := err.(*security.ValidationError); ok {
			status := http.StatusBadRequest
			switch validationErr.ErrorCode {
			case "rate_limit_exceeded":
				status = http.StatusTooManyRequests
			case "protection_unavailable:replay":
				status = http.StatusServiceUnavailable
			}
			l.Error().
				Str("error_code", validationErr.ErrorCode).
				Str("error_message", validationErr.Message).
				Int("http_status", status).
				Msg("DIAGNOSTIC: Returning structured error response")
			c.JSON(status, gin.H{"error": validationErr.Message, "error_code": validationErr.ErrorCode})
		} else {
			l.Error().
				Str("error_message", err.Error()).
				Int("http_status", http.StatusBadRequest).
				Msg("DIAGNOSTIC: Returning generic error response")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	
	l.Info().Str("job_id", spec.ID).Msg("DIAGNOSTIC: Security validation passed successfully")

	
	// Marshal canonical JSON for persistence/outbox
	jobspecJSON, err := json.Marshal(spec)
	if err != nil {
		l.Error().Err(err).Str("job_id", spec.ID).Msg("DIAGNOSTIC: Failed to marshal jobspec")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal jobspec"})
		return
	}
	
	l.Info().Str("job_id", spec.ID).Int("json_length", len(jobspecJSON)).Msg("DIAGNOSTIC: JobSpec marshaled successfully")

	if h.jobsService == nil || h.jobsService.DB == nil {
		l.Error().Msg("DIAGNOSTIC: Persistence unavailable")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "persistence unavailable"})
		return
	}
	
	// Idempotency support
	idemKey, hasKey := GetIdempotencyKey(c)
	l.Info().Bool("has_idempotency_key", hasKey).Str("idempotency_key", idemKey).Msg("DIAGNOSTIC: Checking idempotency")
	
	if hasKey && h.jobsService != nil {
		l.Info().Str("job_id", spec.ID).Msg("DIAGNOSTIC: Using idempotent path")
		jobID, reused, ierr := h.jobsService.IdempotentCreateJob(c.Request.Context(), idemKey, spec, jobspecJSON)
		if ierr != nil {
			l.Error().Err(ierr).Str("job_id", spec.ID).Msg("DIAGNOSTIC: IdempotentCreateJob error")
			c.JSON(http.StatusInternalServerError, gin.H{"error": ierr.Error()})
			return
		}
		if reused {
			l.Info().Str("job_id", jobID).Bool("idempotent", true).Msg("DIAGNOSTIC: Idempotent key reused; returning existing job")
			response := gin.H{"id": jobID, "idempotent": true}
			l.Info().Interface("response", response).Int("status", http.StatusOK).Msg("DIAGNOSTIC: Sending idempotent response")
			c.JSON(http.StatusOK, response)
			return
		}
		l.Info().Str("job_id", jobID).Msg("DIAGNOSTIC: Job enqueued (idempotent create)")
		metrics.JobsEnqueuedTotal.Inc()
		response := gin.H{"id": jobID, "status": "enqueued"}
		l.Info().Interface("response", response).Int("status", http.StatusAccepted).Msg("DIAGNOSTIC: Sending idempotent success response")
		c.JSON(http.StatusAccepted, response)
		return
	}

	// Non-idempotent path
	l.Info().Str("job_id", spec.ID).Msg("DIAGNOSTIC: Using non-idempotent path")
	if err := h.jobsService.CreateJob(c.Request.Context(), spec, jobspecJSON); err != nil {
		l.Error().Err(err).Str("job_id", spec.ID).Msg("DIAGNOSTIC: CreateJob service error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	l.Info().Str("job_id", spec.ID).Msg("DIAGNOSTIC: Job enqueued successfully")
	metrics.JobsEnqueuedTotal.Inc()
	response := gin.H{"id": spec.ID, "status": "enqueued"}
	l.Info().Interface("response", response).Int("status", http.StatusAccepted).Msg("DIAGNOSTIC: Sending final success response")
	c.JSON(http.StatusAccepted, response)
}

// GetJob handles job retrieval requests
func (h *JobsHandler) GetJob(c *gin.Context) {
	l := logging.FromContext(c.Request.Context())
	jobID := c.Param("id")
	if jobID == "" {
		l.Error().Msg("missing job id")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing job id"})
		return
	}
	if h.jobsService == nil || h.jobsService.DB == nil {
		l.Error().Msg("persistence unavailable")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "persistence unavailable"})
		return
	}
	spec, status, err := h.jobsService.GetJob(c.Request.Context(), jobID)
	if err != nil {
		// Map not-found to 404; keep other errors as 500
		if strings.Contains(err.Error(), "job not found") {
			l.Info().Str("job_id", jobID).Msg("job not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		l.Error().Err(err).Str("job_id", jobID).Msg("GetJob service error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if spec == nil {
		l.Info().Str("job_id", jobID).Msg("job not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	// Optional includes
	include := strings.ToLower(c.Query("include"))
	if include == "executions" || include == "all" || include == "latest" {
		if h.jobsService.ExecutionsRepo != nil {
			if include == "latest" {
                rec, lerr := h.jobsService.GetLatestReceiptCached(c.Request.Context(), jobID)
                if lerr != nil {
                    // If no receipt yet, still return job with empty executions
                    l.Info().Str("job_id", jobID).Msg("no latest receipt yet")
                    c.JSON(http.StatusOK, gin.H{"job": spec, "status": status, "executions": []interface{}{}})
                    return
                }
                l.Info().Str("job_id", jobID).Msg("returning latest receipt")
                c.JSON(http.StatusOK, gin.H{"job": spec, "status": status, "executions": []interface{}{rec}})
                return
            }

			// Pagination params for executions list
			execLimit := 20
			if v := c.Query("exec_limit"); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
					execLimit = n
				}
			}
			execOffset := 0
			if v := c.Query("exec_offset"); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n >= 0 {
					execOffset = n
				}
			}

			// Prefer paginated method if available
			recs, lerr := h.jobsService.ExecutionsRepo.ListExecutionsByJobSpecIDPaginated(c.Request.Context(), jobID, execLimit, execOffset)
			if lerr != nil {
				// Fallback to non-paginated list
				recs, lerr2 := h.jobsService.ExecutionsRepo.ListExecutionsByJobSpecID(c.Request.Context(), jobID)
				if lerr2 != nil {
					l.Error().Err(lerr2).Str("job_id", jobID).Msg("list executions error")
					c.JSON(http.StatusInternalServerError, gin.H{"error": lerr2.Error()})
					return
				}
				l.Info().Str("job_id", jobID).Int("count", len(recs)).Msg("returning executions (fallback)")
				c.JSON(http.StatusOK, gin.H{"job": spec, "status": status, "executions": recs})
				return
			}
			l.Info().Str("job_id", jobID).Int("count", len(recs)).Msg("returning executions (paginated)")
			c.JSON(http.StatusOK, gin.H{"job": spec, "status": status, "executions": recs})
			return
		}
	}
	l.Info().Str("job_id", jobID).Msg("returning job without executions")
	c.JSON(http.StatusOK, gin.H{"job": spec, "status": status})
}

// ListJobs handles job listing requests
func (h *JobsHandler) ListJobs(c *gin.Context) {
	l := logging.FromContext(c.Request.Context())
	if h.jobsService == nil || h.jobsService.DB == nil || h.jobsService.JobsRepo == nil {
		l.Error().Msg("persistence unavailable")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "persistence unavailable"})
		return
	}

	// Parse limit (default 50)
	limitStr := c.Query("limit")
	limit := 50
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}

	rows, err := h.jobsService.JobsRepo.ListRecentJobs(c.Request.Context(), limit)
	if err != nil {
		l.Error().Err(err).Int("limit", limit).Msg("ListRecentJobs error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type item struct {
		ID        string    `json:"id"`
		Status    string    `json:"status"`
		CreatedAt time.Time `json:"created_at"`
	}
	var out []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.ID, &it.Status, &it.CreatedAt); err != nil {
			l.Error().Err(err).Msg("rows scan error")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		l.Error().Err(err).Msg("rows iteration error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	l.Info().Int("count", len(out)).Msg("returning recent jobs")
	c.JSON(http.StatusOK, gin.H{"jobs": out})
}
