package handlers

import (
 "net/http"
 "os"
 "strconv"
 "sync"
 "time"

 "github.com/gin-gonic/gin"
 "github.com/jamie-anson/project-beacon-runner/internal/config"
 "github.com/jamie-anson/project-beacon-runner/internal/db"
 "github.com/jamie-anson/project-beacon-runner/internal/middleware"
 "github.com/jamie-anson/project-beacon-runner/internal/service"
)

var (
mu  sync.RWMutex
cfg = config.DefaultAdminConfig()
)

func Health(c *gin.Context) {
c.JSON(http.StatusOK, gin.H{"ok": true})
}

func WhoAmI(c *gin.Context) {
 role := middleware.GetRole(c)
 c.JSON(http.StatusOK, gin.H{"role": role})
}

func GetAdminConfig(c *gin.Context) {
	role := middleware.GetRole(c)
	if role != middleware.RoleAdmin && role != middleware.RoleOperator {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden", "role": role})
		return
	}
	mu.RLock()
	defer mu.RUnlock()
	c.JSON(http.StatusOK, cfg)
}

func PutAdminConfig(c *gin.Context) {
	role := middleware.GetRole(c)
	if role != middleware.RoleAdmin {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden", "role": role})
		return
	}
	var upd config.AdminConfigUpdate
	if err := c.ShouldBindJSON(&upd); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}
	mu.Lock()
	cfg = config.SanitizeAndMerge(cfg, upd)
	mu.Unlock()
	c.JSON(http.StatusOK, gin.H{"ok": true, "config": cfg})
}

func TriggerMigration(c *gin.Context) {
	role := middleware.GetRole(c)
	if role != middleware.RoleAdmin {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden", "role": role})
		return
	}

	// Force migration execution with golang-migrate
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DATABASE_URL not configured"})
		return
	}

	// Initialize database with forced migration
	database, err := db.Initialize(dbURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Migration failed", "details": err.Error()})
		return
	}

	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "Migration triggered successfully"})
}

func CheckStuckJobs(c *gin.Context) {
	role := middleware.GetRole(c)
	if role != middleware.RoleAdmin && role != middleware.RoleOperator {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden", "role": role})
		return
	}

	// Get database connection from context or service
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DATABASE_URL not configured"})
		return
	}

	database, err := db.Initialize(dbURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed", "details": err.Error()})
		return
	}
	defer database.Close()

	timeoutService := service.NewJobTimeoutService(database.DB)
	timeoutThreshold := 15 * time.Minute // Default 15 minutes

	// Get threshold from query parameter if provided
	if thresholdStr := c.Query("threshold_minutes"); thresholdStr != "" {
		if minutes, err := strconv.Atoi(thresholdStr); err == nil && minutes > 0 {
			timeoutThreshold = time.Duration(minutes) * time.Minute
		}
	}

	stuckJobs, err := timeoutService.GetStuckJobsDetails(c.Request.Context(), timeoutThreshold)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stuck jobs", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stuck_jobs_count": len(stuckJobs),
		"timeout_threshold_minutes": int(timeoutThreshold.Minutes()),
		"stuck_jobs": stuckJobs,
	})
}

func TimeoutStuckJobs(c *gin.Context) {
	role := middleware.GetRole(c)
	if role != middleware.RoleAdmin {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden", "role": role})
		return
	}

	// Get database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DATABASE_URL not configured"})
		return
	}

	database, err := db.Initialize(dbURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed", "details": err.Error()})
		return
	}
	defer database.Close()

	timeoutService := service.NewJobTimeoutService(database.DB)
	timeoutThreshold := 15 * time.Minute // Default 15 minutes

	// Get threshold from query parameter if provided
	if thresholdStr := c.Query("threshold_minutes"); thresholdStr != "" {
		if minutes, err := strconv.Atoi(thresholdStr); err == nil && minutes > 0 {
			timeoutThreshold = time.Duration(minutes) * time.Minute
		}
	}

	// Get stuck jobs before cleanup for reporting
	stuckJobs, err := timeoutService.GetStuckJobsDetails(c.Request.Context(), timeoutThreshold)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stuck jobs", "details": err.Error()})
		return
	}

	// Perform timeout cleanup
	if err := timeoutService.TimeoutStuckJobs(c.Request.Context(), timeoutThreshold); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to timeout stuck jobs", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"message": "Stuck jobs timeout completed",
		"timed_out_count": len(stuckJobs),
		"timeout_threshold_minutes": int(timeoutThreshold.Minutes()),
		"timed_out_jobs": stuckJobs,
	})
}

// EmergencyStopJob immediately stops a running job by marking it as failed
// POST /admin/jobs/:id/emergency-stop
func EmergencyStopJob(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_id required"})
		return
	}

	// Get database connection
	database, err := db.GetDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection failed", "details": err.Error()})
		return
	}
	defer database.Close()

	// Get current job status
	var currentStatus string
	err = database.DB.QueryRowContext(c.Request.Context(), 
		"SELECT status FROM jobs WHERE id = $1", jobID).Scan(&currentStatus)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found", "job_id": jobID})
		return
	}

	// Check if job is already in a terminal state
	if currentStatus == "completed" || currentStatus == "failed" || currentStatus == "cancelled" {
		c.JSON(http.StatusOK, gin.H{
			"ok": true,
			"message": "Job already in terminal state",
			"job_id": jobID,
			"status": currentStatus,
			"action": "none",
		})
		return
	}

	// Update job status to failed with emergency stop reason
	_, err = database.DB.ExecContext(c.Request.Context(),
		`UPDATE jobs 
		SET status = 'failed', 
		    updated_at = NOW(),
		    metadata = COALESCE(metadata, '{}'::jsonb) || '{"emergency_stop": true, "stopped_at": "'||NOW()||'", "stopped_by": "admin"}'::jsonb
		WHERE id = $1`,
		jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to stop job", "details": err.Error()})
		return
	}

	// Mark all running executions as failed
	result, err := database.DB.ExecContext(c.Request.Context(),
		`UPDATE executions 
		SET status = 'failed',
		    completed_at = NOW(),
		    output = COALESCE(output, '{}'::jsonb) || '{"error": "Emergency stop by admin", "emergency_stop": true}'::jsonb
		WHERE job_id = $1 
		AND status IN ('pending', 'running', 'processing')`,
		jobID)
	
	executionsStopped := int64(0)
	if err == nil && result != nil {
		executionsStopped, _ = result.RowsAffected()
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"message": "Job emergency stopped",
		"job_id": jobID,
		"previous_status": currentStatus,
		"new_status": "failed",
		"executions_stopped": executionsStopped,
		"stopped_at": time.Now().UTC().Format(time.RFC3339),
	})
}
