package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
)

// AdminJobsHandler handles job-related admin operations
type AdminJobsHandler struct {
	jobsService *service.JobsService
}

// NewAdminJobsHandler creates a new AdminJobsHandler
func NewAdminJobsHandler(jobsService *service.JobsService) *AdminJobsHandler {
	return &AdminJobsHandler{jobsService: jobsService}
}

// RepublishJobByID republishes a specific job to the outbox queue.
// Body: {"job_id": "<jobspec_id>"}
func (h *AdminJobsHandler) RepublishJobByID(c *gin.Context) {
	if h.jobsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "jobs service not available"})
		return
	}
	var req struct{ JobID string `json:"job_id"` }
	body, err := c.GetRawData()
	if err != nil || len(body) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_body"})
		return
	}
	if err := json.Unmarshal(body, &req); err != nil || req.JobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing_job_id"})
		return
	}
	if err := h.jobsService.RepublishJob(c.Request.Context(), req.JobID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "republish_failed", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "republished", "job_id": req.JobID})
}

// RepublishStuckJobs finds jobs in "created" status and republishes them to the outbox queue
func (h *AdminJobsHandler) RepublishStuckJobs(c *gin.Context) {
	if h.jobsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "jobs service not available"})
		return
	}

	ctx := c.Request.Context()
	
	// Find jobs stuck in "created" status
	stuckJobs, err := h.jobsService.JobsRepo.ListJobsByStatus(ctx, "created", 100)
	if err != nil {
		log.Printf("Failed to find stuck jobs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find stuck jobs"})
		return
	}

	if len(stuckJobs) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "no stuck jobs found",
			"republished": 0,
		})
		return
	}

	republished := 0
	for _, job := range stuckJobs {
		// Republish job to outbox
		err := h.jobsService.RepublishJob(ctx, job.ID)
		if err != nil {
			log.Printf("Failed to republish stuck job %s: %v", job.ID, err)
			continue
		}
		republished++
		log.Printf("Republished stuck job: %s", job.ID)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "republished stuck jobs",
		"total_found": len(stuckJobs),
		"republished": republished,
	})
}

// RepairStuckJobsHandler handles job repair requests
func (h *AdminJobsHandler) RepairStuckJobsHandler(c *gin.Context) {
	if h.jobsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Jobs service not available",
		})
		return
	}

	// Parse max age parameter (default: 30 minutes)
	maxAgeStr := c.DefaultQuery("max_age", "30m")
	maxAge, err := time.ParseDuration(maxAgeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid max_age parameter",
			"details": err.Error(),
		})
		return
	}

	// Create repair service and run repair
	repairService := service.NewJobRepairService(h.jobsService)
	summary, err := repairService.RepairStuckJobs(c.Request.Context(), maxAge)
	if err != nil {
		log.Printf("Failed to repair stuck jobs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to repair stuck jobs",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Job repair completed",
		"summary": summary,
	})
}

// GetStuckJobsStats returns statistics about potentially stuck jobs
func (h *AdminJobsHandler) GetStuckJobsStats(c *gin.Context) {
	if h.jobsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Jobs service not available",
		})
		return
	}

	repairService := service.NewJobRepairService(h.jobsService)
	stats, err := repairService.GetStuckJobsStats(c.Request.Context())
	if err != nil {
		log.Printf("Failed to get stuck jobs stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get stuck jobs stats",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
		"timestamp": time.Now().UTC(),
	})
}
