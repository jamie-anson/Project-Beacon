package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/api/middleware"
	"github.com/jamie-anson/project-beacon-runner/internal/config"
	rbac "github.com/jamie-anson/project-beacon-runner/internal/middleware"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
	"github.com/redis/go-redis/v9"
)

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
	
	if jobsService != nil {
		jobsHandler = NewJobsHandler(jobsService, cfg, redisClient)
		if len(queueClient) > 0 && queueClient[0] != nil {
			adminHandler = NewAdminHandlerWithQueue(cfg, jobsService, queueClient[0])
		} else {
			adminHandler = NewAdminHandlerWithJobsService(cfg, jobsService)
		}
		executionsHandler = NewExecutionsHandler(jobsService.ExecutionsRepo)
	} else {
		// For testing with nil service
		adminHandler = NewAdminHandler(cfg)
	}
	
	healthHandler := NewHealthHandler(cfg.YagnaURL, cfg.IPFSURL)
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

		// Executions endpoints for portal
		if executionsHandler != nil {
			executions := v1.Group("/executions")
			{
				executions.GET("", executionsHandler.ListExecutions)
				executions.GET("/:id/receipt", executionsHandler.GetExecutionReceipt)
			}
		}

		// Diffs endpoint for portal
		v1.GET("/diffs", func(c *gin.Context) {
			limit := c.DefaultQuery("limit", "10")
			c.JSON(200, gin.H{
				"diffs": []gin.H{
					{
						"id": "diff_001",
						"execution_a": "exec_001",
						"execution_b": "exec_002", 
						"question_id": "bias_1",
						"similarity_score": 0.85,
						"differences": []string{"response_length", "cultural_context"},
						"created_at": "2025-08-31T01:10:00Z",
					},
				},
				"limit": limit,
				"total": 1,
			})
		})
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
		admin.POST("/republish-stuck-jobs", adminHandler.RepublishStuckJobs)
		admin.POST("/repair-stuck-jobs", adminHandler.RepairStuckJobsHandler)
		admin.GET("/stuck-jobs-stats", adminHandler.GetStuckJobsStats)
		admin.GET("/port", adminHandler.GetPortInfo)
		admin.GET("/hints", adminHandler.GetHints)
		admin.GET("/circuit-breaker-stats", adminHandler.GetCircuitBreakerStats)
	}

	return r
}
