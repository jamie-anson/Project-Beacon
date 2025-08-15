package api

import (
	"context"
	"encoding/json"
	"database/sql"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/golem"
	"github.com/jamie-anson/project-beacon-runner/internal/jobspec"
	"github.com/jamie-anson/project-beacon-runner/internal/db"
	"github.com/jamie-anson/project-beacon-runner/internal/metrics"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
	"github.com/jamie-anson/project-beacon-runner/internal/queue"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// APIServer holds the dependencies for API handlers
type APIServer struct {
	golemService *golem.Service
	executor     *golem.ExecutionEngine
	validator    *jobspec.Validator
	db           *db.DB
	jobsSvc      *service.JobsService
	jobsRepo     *store.JobsRepo
	execsRepo    *store.ExecutionsRepo
	q            *queue.Client // for health checks
}

// listProviders supports discovery via query params (e.g., GET /providers?region=US)
func (s *APIServer) listProviders(c *gin.Context) {
    region := c.Query("region")

    // Build minimal constraints
    var regions []string
    if region != "" {
        regions = []string{region}
    } else {
        // Default to common regions to return something meaningful with mock backend
        regions = []string{"US", "EU", "APAC"}
    }

    constraints := models.ExecutionConstraints{
        Regions:    regions,
        MinRegions: 1,
    }

    ctx := context.Background()
    providers, err := s.golemService.DiscoverProviders(ctx, constraints)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "Provider discovery failed",
            "details": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "providers":   providers,
        "count":       len(providers),
        "region":      region,
        "constraints": constraints,
    })
}

// Metrics summary handler
func (s *APIServer) metricsSummary(c *gin.Context) {
	sum, err := metrics.Summary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "metrics summary failed",
			"details": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"summary":   sum,
		"timestamp": time.Now().UTC(),
	})
}

// NewAPIServer creates a new API server with dependencies
func NewAPIServer(database *db.DB) *APIServer {
	// Initialize Golem service
	apiKey := os.Getenv("GOLEM_API_KEY")
	if apiKey == "" {
		apiKey = "test-key" // Default for testing
	}
	
	network := os.Getenv("GOLEM_NETWORK")
	if network == "" {
		network = "testnet" // Default for testing
	}
	
	golemService := golem.NewService(apiKey, network)
	executor := golem.NewExecutionEngine(golemService)
	validator := jobspec.NewValidator()
	// Initialize jobs service/repo if DB is available
	var jobsSvc *service.JobsService
	var jobsRepo *store.JobsRepo
	var execsRepo *store.ExecutionsRepo
	if database != nil && database.DB != nil {
		jobsSvc = service.NewJobsService(database.DB)
		jobsRepo = store.NewJobsRepo(database.DB)
		execsRepo = store.NewExecutionsRepo(database.DB)
	}
	// Queue client for health ping (not used for enqueue here)
	q := queue.MustNewFromEnv()
	
	return &APIServer{
		golemService: golemService,
		executor:     executor,
		validator:    validator,
		db:           database,
		jobsSvc:      jobsSvc,
		jobsRepo:     jobsRepo,
		execsRepo:    execsRepo,
		q:            q,
	}
}

// SetupRoutes configures all API routes
func SetupRoutes(r *gin.Engine, database *db.DB) {
	server := NewAPIServer(database)
	
	// API version group
	v1 := r.Group("/api/v1")
	{
		// Job management
		v1.POST("/jobs", server.createJob)
		v1.GET("/jobs/:id", server.getJob)
		v1.GET("/jobs", server.listJobs)
		v1.DELETE("/jobs/:id", server.deleteJob)

		// Results API
		v1.GET("/jobs/:id/results/latest", server.latestResult)

		// Execution management
		v1.POST("/jobs/:id/execute", server.executeJob)
		v1.GET("/executions/:id", server.getExecution)
		v1.GET("/executions", server.listExecutions)
		v1.POST("/executions/:id/cancel", server.cancelExecution)

		// Provider discovery
		v1.POST("/providers/discover", server.discoverProviders)
		v1.GET("/providers", server.listProviders)
		v1.GET("/providers/:id", server.getProvider)

		// Cost estimation
		v1.POST("/jobs/estimate", server.estimateCost)

		// Cross-region diff analysis
		v1.GET("/executions/:id/diffs", server.getExecutionDiffs)
		v1.POST("/diffs/analyze", server.analyzeDiffs)

		// Health check
		v1.GET("/health", server.healthCheck)

		// Metrics summary (JSON)
		v1.GET("/metrics/summary", server.metricsSummary)
	}
}

// Job handlers
func (s *APIServer) createJob(c *gin.Context) {
	var jobspecData map[string]interface{}
	if err := c.ShouldBindJSON(&jobspecData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON format",
			"details": err.Error(),
		})
		return
	}
	
	// Convert to JSON for validation
	jobspecJSON, err := json.Marshal(jobspecData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process JobSpec",
		})
		return
	}
	
	// Validate JobSpec
	jobspec, err := s.validator.ValidateJobSpec(jobspecJSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "JobSpec validation failed",
			"details": err.Error(),
		})
		return
	}

	// Persist + outbox write transactionally via service (if available)
	var persisted bool
	if s.jobsSvc != nil {
		if err := s.jobsSvc.CreateJob(c.Request.Context(), jobspec, jobspecJSON); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to store job",
				"details": err.Error(),
			})
			return
		}
		persisted = true
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "JobSpec created successfully",
		"job_id": jobspec.ID,
		"status": "created",
		"persisted": persisted,
		"jobspec": jobspec,
	})
}

func (s *APIServer) getJob(c *gin.Context) {
    jobID := c.Param("id")

    if s.jobsRepo != nil {
        ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
        defer cancel()
        id, status, data, createdAt, updatedAt, err := s.jobsRepo.GetJob(ctx, jobID)
        if err == nil {
            var jobspecObj map[string]interface{}
            _ = json.Unmarshal(data, &jobspecObj)
            var created interface{} = createdAt
            var updated interface{} = updatedAt
            if createdAt.Valid {
                created = createdAt.Time
            }
            if updatedAt.Valid {
                updated = updatedAt.Time
            }
            c.JSON(http.StatusOK, gin.H{
                "job_id":     id,
                "status":     status,
                "jobspec":    jobspecObj,
                "created_at": created,
                "updated_at": updated,
            })
            return
        }
    }

	c.JSON(http.StatusNotFound, gin.H{
		"error":  "job not found",
		"job_id": jobID,
	})
}

func (s *APIServer) listJobs(c *gin.Context) {
    if s.jobsRepo != nil {
        ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
        defer cancel()
        rows, err := s.jobsRepo.ListRecentJobs(ctx, 100)
        if err == nil {
            defer rows.Close()
            var jobs []gin.H
            for rows.Next() {
                var id, status string
                var createdAt time.Time
                _ = rows.Scan(&id, &status, &createdAt)
                jobs = append(jobs, gin.H{"job_id": id, "status": status, "created_at": createdAt})
            }
            c.JSON(http.StatusOK, gin.H{"jobs": jobs})
            return
        }
    }
    c.JSON(http.StatusOK, gin.H{"jobs": []interface{}{} })
}

func (s *APIServer) deleteJob(c *gin.Context) {
    jobID := c.Param("id")

    deleted := false
    if s.jobsRepo != nil {
        ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
        defer cancel()
        if err := s.jobsRepo.DeleteJob(ctx, jobID); err == nil {
            // We can't easily know affected rows via repo; assume success indicates deletion
            deleted = true
        }
    }

	if deleted {
		c.JSON(http.StatusOK, gin.H{"job_id": jobID, "status": "deleted"})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "job not found", "job_id": jobID})
}

// Execution handlers
func (s *APIServer) executeJob(c *gin.Context) {
	jobID := c.Param("id")
	
	var jobspecData map[string]interface{}
	if err := c.ShouldBindJSON(&jobspecData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON format",
			"details": err.Error(),
		})
		return
	}
	
	// Convert to JSON for validation
	jobspecJSON, err := json.Marshal(jobspecData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process JobSpec",
		})
		return
	}
	
	// Validate JobSpec
	jobspec, err := s.validator.ValidateJobSpec(jobspecJSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "JobSpec validation failed",
			"details": err.Error(),
		})
		return
	}
	
	// Ensure JobSpec ID matches URL parameter
	if jobspec.ID != jobID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "JobSpec ID mismatch",
			"expected": jobID,
			"received": jobspec.ID,
		})
		return
	}
	
	// Execute across multiple regions
	ctx := context.Background()
	summary, err := s.executor.ExecuteMultiRegion(ctx, jobspec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Execution failed",
			"details": err.Error(),
		})
		return
	}
	
	// Validate execution meets requirements
	if err := s.executor.ValidateExecution(summary, jobspec); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": "Execution validation failed",
			"details": err.Error(),
			"summary": summary,
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Multi-region execution completed",
		"job_id": jobID,
		"summary": summary,
	})
}

func (s *APIServer) getExecution(c *gin.Context) {
	executionID := c.Param("id")
	
	// In a real implementation, fetch from database
	summary, err := s.executor.GetExecutionStatus(executionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Execution not found",
			"execution_id": executionID,
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"execution_id": executionID,
		"summary": summary,
	})
}

func (s *APIServer) listExecutions(c *gin.Context) {
	// In a real implementation, fetch from database with pagination
	c.JSON(http.StatusOK, gin.H{
		"message": "List executions endpoint - database integration pending",
		"status":  "placeholder",
		"executions": []interface{}{},
	})
}

func (s *APIServer) cancelExecution(c *gin.Context) {
	executionID := c.Param("id")
	
	ctx := context.Background()
	if err := s.executor.CancelExecution(ctx, executionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to cancel execution",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Execution cancelled",
		"execution_id": executionID,
		"status": "cancelled",
	})
}

// Provider discovery handlers
func (s *APIServer) discoverProviders(c *gin.Context) {
	var constraints models.ExecutionConstraints
	if err := c.ShouldBindJSON(&constraints); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid constraints format",
			"details": err.Error(),
		})
		return
	}
	
	ctx := context.Background()
	providers, err := s.golemService.DiscoverProviders(ctx, constraints)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Provider discovery failed",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
		"count": len(providers),
		"constraints": constraints,
	})
}

func (s *APIServer) getProvider(c *gin.Context) {
	providerID := c.Param("id")
	
	provider, err := s.golemService.GetProvider(providerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Provider not found",
			"provider_id": providerID,
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"provider": provider,
	})
}

// Cost estimation handler
func (s *APIServer) estimateCost(c *gin.Context) {
	var jobspecData map[string]interface{}
	if err := c.ShouldBindJSON(&jobspecData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON format",
			"details": err.Error(),
		})
		return
	}
	
	// Convert to JSON for validation
	jobspecJSON, err := json.Marshal(jobspecData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process JobSpec",
		})
		return
	}
	
	// Validate JobSpec
	jobspec, err := s.validator.ValidateJobSpec(jobspecJSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "JobSpec validation failed",
			"details": err.Error(),
		})
		return
	}
	
	ctx := context.Background()
	cost, err := s.executor.EstimateExecutionCost(ctx, jobspec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Cost estimation failed",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"job_id": jobspec.ID,
		"estimated_cost": cost,
		"currency": "USD",
		"regions": jobspec.Constraints.Regions,
		"timeout": jobspec.Constraints.Timeout.String(),
	})
}

// Diff analysis handlers
func (s *APIServer) getExecutionDiffs(c *gin.Context) {
	executionID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"message": "Get execution diffs endpoint - diff engine pending",
		"execution_id": executionID,
		"status": "placeholder",
	})
}

func (s *APIServer) analyzeDiffs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Analyze diffs endpoint - diff engine pending",
		"status": "placeholder",
	})
}

// Results: latest execution by JobSpec ID
func (s *APIServer) latestResult(c *gin.Context) {
    jobID := c.Param("id")
    if s.execsRepo == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{
            "error": "database not initialized",
        })
        return
    }

    ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
    defer cancel()

    id, providerID, region, status, startedAt, completedAt, outJSON, recJSON, createdAt, err := s.execsRepo.GetLatestByJobSpecID(ctx, jobID)
    if err != nil {
        if err == sql.ErrNoRows {
            c.JSON(http.StatusNotFound, gin.H{
                "error": "no executions found for job",
                "job_id": jobID,
            })
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "failed to fetch latest execution",
            "details": err.Error(),
        })
        return
    }

    // Decode stored JSONB into objects for response
    var output any
    var receipt any
    _ = json.Unmarshal(outJSON, &output)
    _ = json.Unmarshal(recJSON, &receipt)

    c.JSON(http.StatusOK, gin.H{
        "job_id": jobID,
        "execution": gin.H{
            "id": id,
            "provider_id": providerID,
            "region": region,
            "status": status,
            "started_at": startedAt.Time,
            "completed_at": completedAt.Time,
            "created_at": createdAt,
            "output": output,
            "receipt": receipt,
        },
    })
}

// Health check handler
func (s *APIServer) healthCheck(c *gin.Context) {
    // Quick health snapshot including DB and Redis
    ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Second)
    defer cancel()

    dbStatus := "disabled"
    if s.db != nil && s.db.DB != nil {
        if err := s.db.DB.PingContext(ctx); err != nil {
            dbStatus = "error"
        } else {
            dbStatus = "ready"
        }
    }
    redisStatus := "disabled"
    if s.q != nil {
        if err := s.q.Ping(ctx); err != nil {
            redisStatus = "error"
        } else {
            redisStatus = "ready"
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "healthy",
        "timestamp": time.Now().UTC(),
        "version": "1.0.0",
        "service": "project-beacon-runner",
        "components": gin.H{
            "golem_service":     "ready",
            "execution_engine":  "ready",
            "jobspec_validator": "ready",
            "postgres":          dbStatus,
            "redis":             redisStatus,
        },
    })
}
