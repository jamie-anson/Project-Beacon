package api

import (
	"net/http"
	"strconv"
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/queue"
)

// Missing API handler methods - stub implementations

func (s *APIServer) getJob(c *gin.Context) {
	if s.jobsRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "job store not available"})
		return
	}
	id := c.Param("id")
	jobspec, status, err := s.jobsRepo.GetJobByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "job not found",
			"details": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"job_id":  id,
		"status":  status,
		"jobspec": jobspec,
	})
}

func (s *APIServer) listJobs(c *gin.Context) {
	if s.jobsRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "job store not available"})
		return
	}
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}
	rows, err := s.jobsRepo.ListRecentJobs(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to list jobs",
			"details": err.Error(),
		})
		return
	}
	defer rows.Close()

	jobs := make([]map[string]interface{}, 0, limit)
	for rows.Next() {
		var id string
		var status string
		var createdAt interface{}
		if err := rows.Scan(&id, &status, &createdAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "failed to scan job row",
				"details": err.Error(),
			})
			return
		}
		jobs = append(jobs, map[string]interface{}{
			"id":         id,
			"status":     status,
			"created_at": createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "error iterating jobs",
			"details": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"count": len(jobs),
		"jobs":  jobs,
	})
}

func (s *APIServer) deleteJob(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "deleteJob not implemented yet"})
}

func (s *APIServer) listDiffs(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "listDiffs not implemented yet"})
}

func (s *APIServer) getDiff(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "getDiff not implemented yet"})
}

func (s *APIServer) executeJob(c *gin.Context) {
	if s.jobsRepo == nil || s.q == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "dependencies not available"})
		return
	}
	id := c.Param("id")
	// ensure job exists
	if _, _, err := s.jobsRepo.GetJobByID(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found", "details": err.Error()})
		return
	}

	// Build envelope expected by worker.job_runner (fields: id, enqueued_at, attempt)
	env := map[string]interface{}{
		"id":          id,
		"enqueued_at": time.Now().UTC(),
		"attempt":     0,
	}
	payload, _ := json.Marshal(env)
	if err := s.q.Enqueue(c.Request.Context(), queue.JobsQueue, payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enqueue execution", "details": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "execution enqueued", "job_id": id})
}

func (s *APIServer) getExecution(c *gin.Context) {
	if s.execsRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "execution store not available"})
		return
	}
	// Treat :id as JobSpec ID and return latest execution
	jobspecID := c.Param("id")
	id, providerID, region, status, startedAt, completedAt, outputJSON, receiptJSON, createdAt, err := s.execsRepo.GetLatestByJobSpecID(c.Request.Context(), jobspecID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "execution not found", "details": err.Error()})
		return
	}

	var startedPtr *time.Time
	if startedAt.Valid {
		t := startedAt.Time
		startedPtr = &t
	}
	var completedPtr *time.Time
	if completedAt.Valid {
		t := completedAt.Time
		completedPtr = &t
	}

	c.JSON(http.StatusOK, gin.H{
		"execution_id": id,
		"job_id":       jobspecID,
		"provider_id":  providerID,
		"region":       region,
		"status":       status,
		"started_at":   startedPtr,
		"completed_at": completedPtr,
		"created_at":   createdAt,
		"output":       json.RawMessage(outputJSON),
		"receipt":      json.RawMessage(receiptJSON),
	})
}

func (s *APIServer) listExecutions(c *gin.Context) {
	if s.execsRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "execution store not available"})
		return
	}
	jobspecID := c.Query("jobspec_id")
	if jobspecID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing jobspec_id query param"})
		return
	}
	receipts, err := s.execsRepo.ListExecutionsByJobSpecID(c.Request.Context(), jobspecID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list executions", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": len(receipts), "receipts": receipts})
}

func (s *APIServer) cancelExecution(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "cancelExecution not implemented yet"})
}

func (s *APIServer) discoverProviders(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "discoverProviders not implemented yet"})
}

func (s *APIServer) getProvider(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "getProvider not implemented yet"})
}

func (s *APIServer) latestResult(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "latestResult not implemented yet"})
}

func (s *APIServer) getExecutionDiffs(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "getExecutionDiffs not implemented yet"})
}

func (s *APIServer) analyzeDiffs(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "analyzeDiffs not implemented yet"})
}
