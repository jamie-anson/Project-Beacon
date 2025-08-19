package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/internal/metrics"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// JobsHandler handles job-related requests
type JobsHandler struct {
	jobsService *service.JobsService
}

// NewJobsHandler creates a new jobs handler
func NewJobsHandler(jobsService *service.JobsService) *JobsHandler {
	return &JobsHandler{
		jobsService: jobsService,
	}
}

// CreateJob handles job creation requests
func (h *JobsHandler) CreateJob(c *gin.Context) {
	l := logging.FromContext(c.Request.Context())
	l.Info().Msg("api: CreateJob request")
	// Parse incoming JobSpec
	var spec models.JobSpec
	if err := c.ShouldBindJSON(&spec); err != nil {
		l.Error().Err(err).Msg("invalid JSON")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}

	// Validate spec
	validator := models.NewJobSpecValidator()
	if err := validator.ValidateAndVerify(&spec); err != nil {
		l.Error().Err(err).Str("job_id", spec.ID).Msg("jobspec validation failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Marshal canonical JSON for persistence/outbox
	jobspecJSON, err := json.Marshal(&spec)
	if err != nil {
		l.Error().Err(err).Str("job_id", spec.ID).Msg("failed to marshal jobspec")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal jobspec"})
		return
	}

	if h.jobsService == nil || h.jobsService.DB == nil {
		l.Error().Msg("persistence unavailable")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "persistence unavailable"})
		return
	}

	if err := h.jobsService.CreateJob(c.Request.Context(), &spec, jobspecJSON); err != nil {
		l.Error().Err(err).Str("job_id", spec.ID).Msg("CreateJob service error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	l.Info().Str("job_id", spec.ID).Msg("job enqueued")
	metrics.JobsEnqueuedTotal.Inc()
	c.JSON(http.StatusAccepted, gin.H{"id": spec.ID, "status": "enqueued"})
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
				rec, lerr := h.jobsService.ExecutionsRepo.GetReceiptByJobSpecID(c.Request.Context(), jobID)
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
	if h.jobsService == nil || h.jobsService.JobsRepo == nil {
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
