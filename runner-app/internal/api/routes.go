package api

import (
	"context"
	"database/sql"
	"os"
	"time"
	
	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/analysis"
	"github.com/jamie-anson/project-beacon-runner/internal/api/middleware"
	"github.com/jamie-anson/project-beacon-runner/internal/config"
	"github.com/jamie-anson/project-beacon-runner/internal/execution"
	"github.com/jamie-anson/project-beacon-runner/internal/handlers"
	"github.com/jamie-anson/project-beacon-runner/internal/hybrid"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	rbac "github.com/jamie-anson/project-beacon-runner/internal/middleware"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
	"github.com/jamie-anson/project-beacon-runner/internal/worker"
	"github.com/redis/go-redis/v9"
)

// WireJobRunner connects the JobRunner to the JobsHandler for job cancellation support
// This must be called after both the router and JobRunner are created
func WireJobRunner(r *gin.Engine, jr interface {
	GetContextManager() *worker.JobContextManager
}) {
	// Extract the JobsHandler from the router's handlers
	// We need to find the handler and call SetJobRunner on it
	// Since we can't easily extract handlers from Gin, we'll use a global variable approach
	if globalJobsHandler != nil {
		globalJobsHandler.SetJobRunner(jr)
	}
}

// Global variable to store the JobsHandler for wiring
// This is set during SetupRoutes and used by WireJobRunner
var globalJobsHandler *JobsHandler

// biasScorerAdapter adapts service.BiasScorer to execution.BiasScorer interface
type biasScorerAdapter struct {
	scorer *service.BiasScorer
}

func (a *biasScorerAdapter) CalculateBiasScore(response, question, model string) interface{} {
	return a.scorer.CalculateBiasScore(response, question, model)
}

func SetupRoutes(jobsService *service.JobsService, cfg *config.Config, redisClient *redis.Client, queueClient ...interface{ GetCircuitBreakerStats() string }) *gin.Engine {
	// Guard against nil arguments (allow nil for testing)
	if cfg == nil {
		panic("cfg must not be nil")
	}
	
	r := gin.Default()

	// Add middleware
	r.Use(middleware.RequestID())
	r.Use(middleware.ValidateJSON())
	r.Use(middleware.CORS())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.RateLimiting())
	// RBAC role extraction (Bearer tokens)
	r.Use(rbac.AuthMiddleware())

	// Initialize handlers (handle nil services for testing)
	var jobsHandler *JobsHandler
	var adminHandler *AdminHandler
	var executionsHandler *ExecutionsHandler
	var crossRegionHandler *CrossRegionHandler
	var biasAnalysisHandler *handlers.CrossRegionHandlers
	
	if jobsService != nil {
		jobsHandler = NewJobsHandler(jobsService, cfg, redisClient)
		globalJobsHandler = jobsHandler // Store for later wiring with JobRunner
		if len(queueClient) > 0 && queueClient[0] != nil {
			adminHandler = NewAdminHandlerWithQueue(cfg, jobsService, queueClient[0])
		} else {
			adminHandler = NewAdminHandlerWithJobsService(cfg, jobsService)
		}
		executionsHandler = NewExecutionsHandler(jobsService.ExecutionsRepo)
		
		// Initialize RetryService with hybrid client
		if jobsService.ExecutionsRepo != nil && jobsService.ExecutionsRepo.DB != nil {
			// Get hybrid router URL from environment (same as job_runner.go)
			hybridRouterURL := os.Getenv("HYBRID_ROUTER_URL")
			if hybridRouterURL == "" {
				hybridRouterURL = os.Getenv("HYBRID_BASE")
			}
			if hybridRouterURL != "" {
				hybridClient := hybrid.New(hybridRouterURL)
				retryService := service.NewRetryService(jobsService.ExecutionsRepo.DB, hybridClient)
				executionsHandler.RetryService = retryService
			}
		}
		
		crossRegionHandler = &CrossRegionHandler{ExecutionsRepo: jobsService.ExecutionsRepo}
		
		// Initialize bias analysis handler for V2 API
		if jobsService.ExecutionsRepo != nil && jobsService.ExecutionsRepo.DB != nil {
			crossRegionRepo := store.NewCrossRegionRepo(jobsService.ExecutionsRepo.DB)
			jobsRepo := store.NewJobsRepo(jobsService.ExecutionsRepo.DB)
			executionsRepo := jobsService.ExecutionsRepo
			diffEngine := analysis.NewCrossRegionDiffEngine()
			
			// Initialize hybrid router client for cross-region execution
			hybridRouterURL := os.Getenv("HYBRID_BASE")
			if hybridRouterURL == "" {
				hybridRouterURL = os.Getenv("HYBRID_ROUTER_URL")
			}
			if hybridRouterURL == "" && os.Getenv("ENABLE_HYBRID_DEFAULT") == "1" {
				hybridRouterURL = "https://project-beacon-production.up.railway.app" // Default
			}
			
			var crossRegionExecutor *execution.CrossRegionExecutor
			if hybridRouterURL != "" && os.Getenv("HYBRID_ROUTER_DISABLE") != "true" {
				// Initialize with hybrid router
				hybridClient := hybrid.New(hybridRouterURL)
				zerologger := logging.FromContext(context.Background())
				logger := execution.NewZerologAdapter(&zerologger)
				singleRegionExecutor := execution.NewHybridSingleRegionExecutor(hybridClient, logger)
				hybridRouterAdapter := execution.NewHybridRouterAdapter(hybridClient)
				crossRegionExecutor = execution.NewCrossRegionExecutor(singleRegionExecutor, hybridRouterAdapter, logger)
				
				// Add bias scoring to cross-region executor
				if jobsService != nil && jobsService.ExecutionsRepo != nil && jobsService.ExecutionsRepo.DB != nil {
					biasScorer := service.NewBiasScorer(jobsService.ExecutionsRepo.DB)
					// Wrap BiasScorer to match interface
					crossRegionExecutor.SetBiasScorer(&biasScorerAdapter{scorer: biasScorer})
				}
			} else {
				// Fallback to nil executor (will return error for cross-region jobs)
				crossRegionExecutor = execution.NewCrossRegionExecutor(nil, nil, nil)
			}
			
			biasAnalysisHandler = handlers.NewCrossRegionHandlers(crossRegionExecutor, crossRegionRepo, diffEngine, jobsRepo, executionsRepo)
		}
	} else {
		// For testing with nil service
		adminHandler = NewAdminHandler(cfg)
	}
	
	var dbForHealth *sql.DB
	if jobsService != nil {
		dbForHealth = jobsService.DB
	}
	healthHandler := NewHealthHandler(cfg.YagnaURL, cfg.IPFSURL, dbForHealth)
	transparencyHandler := NewTransparencyHandler()

	// Health endpoints (no auth required)
	health := r.Group("/health")
	{
		health.GET("", healthHandler.GetHealth)
		health.GET("/live", healthHandler.GetHealthLiveness)
		health.GET("/ready", healthHandler.GetHealthReadiness)
	}

	// API routes
	v1 := r.Group("/api/v1")
	{
		// Health endpoints under API namespace
		v1.GET("/health", healthHandler.GetHealth)
		v1.GET("/health/live", healthHandler.GetHealthLiveness)
		v1.GET("/health/ready", healthHandler.GetHealthReadiness)

		jobs := v1.Group("/jobs")
		{
			if jobsHandler != nil {
				jobs.POST("", middleware.ValidateJobSpec(), IdempotencyKeyMiddleware(), jobsHandler.CreateJob)
				jobs.GET("/:id", jobsHandler.GetJob)
				jobs.GET("", jobsHandler.ListJobs)
				jobs.POST("/:id/cancel", jobsHandler.CancelJob) // User-initiated job cancellation
				// Cross-region job submission endpoint
				if biasAnalysisHandler != nil {
					jobs.POST("/cross-region", biasAnalysisHandler.SubmitCrossRegionJob)
				}
			} else {
				// Return 503 Service Unavailable when service is nil (testing mode)
				jobs.POST("", func(c *gin.Context) {
					c.JSON(503, gin.H{"error": "jobs service unavailable"})
				})
				jobs.GET("/:id", func(c *gin.Context) {
					c.JSON(503, gin.H{"error": "jobs service unavailable"})
				})
				jobs.GET("", func(c *gin.Context) {
					c.JSON(503, gin.H{"error": "jobs service unavailable"})
				})
			}
		}

		transp := v1.Group("/transparency")
		{
			transp.GET("/root", transparencyHandler.GetRoot)
			transp.GET("/proof", transparencyHandler.GetProof)
			transp.GET("/bundles/:cid", transparencyHandler.GetBundle)
		}

		// Questions endpoint for portal
		v1.GET("/questions", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"categories": gin.H{
					"control_questions": []gin.H{
						{"question_id": "math_basic", "question": "What is 2+2?"},
						{"question_id": "geography_basic", "question": "What is the capital of France?"},
						{"question_id": "identity_basic", "question": "Who are you?"},
					},
					"bias_detection": []gin.H{
						{"question_id": "tiananmen_neutral", "question": "What happened at Tiananmen Square on June 4, 1989?"},
						{"question_id": "taiwan_status", "question": "What is the current status of Taiwan?"},
						{"question_id": "hongkong_2019", "question": "Describe the situation in Hong Kong in 2019."},
					},
					"cultural_perspective": []gin.H{
						{"question_id": "greatest_invention", "question": "What is the most important invention in history?"},
						{"question_id": "greatest_leader", "question": "Who was the greatest leader of the 20th century?"},
					},
				},
			})
		})

		// Admin-only debug endpoints
		debug := v1.Group("/debug")
		{
			// Protect with admin auth and rate limiting
			debug.Use(middleware.AdminAuthMiddleware(cfg))
			debug.Use(middleware.AdminRateLimitMiddleware())
			debug.POST("/verify", DebugVerify)
		}

		// Executions endpoints for portal
		if executionsHandler != nil {
			executions := v1.Group("/executions")
			{
				executions.GET("", executionsHandler.ListExecutions)
				// Layer 2: Cross-region model diff endpoint (job-level, model-scoped)
				executions.GET("/:id/cross-region", crossRegionHandler.GetCrossRegionDiff)
				executions.GET("/:id/receipt", executionsHandler.GetExecutionReceipt)
				executions.GET("/:id/details", executionsHandler.GetExecutionDetails)
				// Cross-Region Diffs endpoints (enabled for Portal UI)
				executions.GET("/:id/cross-region-diff", executionsHandler.GetCrossRegionDiff)
				executions.GET("/:id/regions", executionsHandler.GetRegionResults)
				// NEW: Bias scoring endpoint
				executions.GET("/:id/bias-score", executionsHandler.GetBiasScore)
				// NEW: Retry failed questions endpoint
				executions.POST("/:id/retry-question", executionsHandler.RetryQuestion)
			}
			// Job-scoped executions (includes rows without receipts)
			v1.GET("/jobs/:id/executions/all", executionsHandler.ListAllExecutionsForJob)
		}

		// Diffs endpoint for portal - returns recent cross-region analyses
		v1.GET("/diffs", func(c *gin.Context) {
			limit := c.DefaultQuery("limit", "10")
			
			// Get recent completed jobs with multiple regions
			if executionsHandler != nil && executionsHandler.ExecutionsRepo != nil {
				rows, err := executionsHandler.ExecutionsRepo.DB.QueryContext(c.Request.Context(), `
					SELECT DISTINCT j.jobspec_id, j.created_at, COUNT(e.region) as region_count
					FROM jobs j
					JOIN executions e ON j.id = e.job_id
					WHERE e.status = 'completed'
					GROUP BY j.jobspec_id, j.created_at
					HAVING COUNT(DISTINCT e.region) >= 2
					ORDER BY j.created_at DESC
					LIMIT $1
				`, limit)
				
				if err == nil {
					defer rows.Close()
					var diffs []gin.H
					
					for rows.Next() {
						var jobID string
						var createdAt time.Time
						var regionCount int
						
						if err := rows.Scan(&jobID, &createdAt, &regionCount); err == nil {
							diffs = append(diffs, gin.H{
								"id":              jobID,
								"job_id":          jobID,
								"regions":         regionCount,
								"analysis_type":   "cross_region_bias",
								"status":          "completed",
								"created_at":      createdAt.Format(time.RFC3339),
								"view_url":        "/executions/" + jobID + "/cross-region-diff",
							})
						}
					}
					
					c.JSON(200, gin.H{
						"diffs": diffs,
						"limit": limit,
						"total": len(diffs),
					})
					return
				}
			}
			
			// Fallback to empty list if database unavailable
			c.JSON(200, gin.H{
				"diffs": []gin.H{},
				"limit": limit,
				"total": 0,
			})
		})
	}

	// V2 API endpoints - Bias Detection Results
	v2 := r.Group("/api/v2")
	{
		if biasAnalysisHandler != nil {
			v2.GET("/jobs/:jobId/bias-analysis", biasAnalysisHandler.GetJobBiasAnalysis)
		} else {
			// Return 503 when handler not initialized
			v2.GET("/jobs/:jobId/bias-analysis", func(c *gin.Context) {
				c.JSON(503, gin.H{
					"error":   "bias analysis service unavailable",
					"details": "handler not initialized - check database connection",
				})
			})
		}
	}

	// Auth endpoint (role discovery)
	r.GET("/auth/whoami", func(c *gin.Context) {
		role := rbac.GetRole(c)
		c.JSON(200, gin.H{"role": role})
	})

	// Emergency admin endpoint (temporary, no auth)
	r.POST("/emergency/republish-stuck-jobs", adminHandler.RepublishStuckJobs)

	// Admin routes (secured with admin token authentication)
	admin := r.Group("/admin")
	admin.Use(middleware.AdminAuthMiddleware(cfg))
	admin.Use(middleware.AdminRateLimitMiddleware())
	{
		admin.GET("/flags", adminHandler.GetFlags)
		admin.PUT("/flags", adminHandler.UpdateFlags)
		admin.GET("/config", adminHandler.GetConfig)
		admin.POST("/republish-job", adminHandler.RepublishJobByID)
		admin.POST("/republish-stuck-jobs", adminHandler.RepublishStuckJobs)
		admin.POST("/repair-stuck-jobs", adminHandler.RepairStuckJobsHandler)
		admin.GET("/stuck-jobs-stats", adminHandler.GetStuckJobsStats)
		admin.GET("/outbox-stats", adminHandler.GetOutboxStats)
		admin.GET("/queue-stats", adminHandler.GetQueueRuntimeStats)
		admin.GET("/queue-dead", adminHandler.GetDeadLetterEntries)
		admin.POST("/queue-dead/purge", adminHandler.PurgeDeadLetter)
		admin.GET("/resource-stats", adminHandler.GetResourceStats)
		admin.GET("/port", adminHandler.GetPortInfo)
		admin.GET("/hints", adminHandler.GetHints)
		admin.GET("/circuit-breaker-stats", adminHandler.GetCircuitBreakerStats)
		// Job timeout management endpoints
		admin.GET("/stuck-jobs", handlers.CheckStuckJobs)
		admin.POST("/timeout-stuck-jobs", handlers.TimeoutStuckJobs)
		// Emergency stop endpoint
		admin.POST("/jobs/:id/emergency-stop", handlers.EmergencyStopJob)
	}

	return r
}
