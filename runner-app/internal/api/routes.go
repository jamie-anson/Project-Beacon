package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/api/middleware"
	"github.com/jamie-anson/project-beacon-runner/internal/config"
	rbac "github.com/jamie-anson/project-beacon-runner/internal/middleware"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
	"github.com/redis/go-redis/v9"
)

func SetupRoutes(jobsService *service.JobsService, cfg *config.Config, redisClient *redis.Client) *gin.Engine {
	r := gin.Default()

	// Add middleware
	r.Use(middleware.RequestID())
	r.Use(middleware.ValidateJSON())
	r.Use(middleware.CORS())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.RateLimiting())
	// RBAC role extraction (Bearer tokens)
	r.Use(rbac.AuthMiddleware())

	// Initialize handlers
	jobsHandler := NewJobsHandler(jobsService, cfg, redisClient)
	healthHandler := NewHealthHandler(cfg.YagnaURL, cfg.IPFSURL)
	transparencyHandler := NewTransparencyHandler()
	adminHandler := NewAdminHandler(cfg)

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
		jobs := v1.Group("/jobs")
		{
			jobs.POST("", middleware.ValidateJobSpec(), IdempotencyKeyMiddleware(), jobsHandler.CreateJob)
			jobs.GET("/:id", jobsHandler.GetJob)
			jobs.GET("", jobsHandler.ListJobs)
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

		// Executions endpoint for portal
		v1.GET("/executions", func(c *gin.Context) {
			limit := c.DefaultQuery("limit", "10")
			c.JSON(200, gin.H{
				"executions": []gin.H{
					{
						"id": "exec_001",
						"job_id": "job_123",
						"status": "completed",
						"started_at": "2025-08-31T01:00:00Z",
						"completed_at": "2025-08-31T01:01:30Z",
						"duration": 90,
						"provider_id": "0x536ec34be8b1395d54f69b8895f902f9b65b235b",
						"model": "llama3.2:1b",
					},
					{
						"id": "exec_002", 
						"job_id": "job_124",
						"status": "running",
						"started_at": "2025-08-31T01:05:00Z",
						"provider_id": "0x536ec34be8b1395d54f69b8895f902f9b65b235b",
						"model": "mistral:7b",
					},
				},
				"limit": limit,
				"total": 2,
			})
		})

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

	// Admin routes (RBAC; some public in debug mode)
	admin := r.Group("/admin")
	{
		admin.GET("/flags", rbac.RequireAnyRole(rbac.RoleAdmin), adminHandler.GetFlags)
		admin.PUT("/flags", rbac.RequireAnyRole(rbac.RoleAdmin), adminHandler.UpdateFlags)
		admin.GET("/config", rbac.RequireAnyRole(rbac.RoleAdmin, rbac.RoleOperator), adminHandler.GetConfig)

		if gin.Mode() == gin.DebugMode {
			// Public in debug for DX
			admin.GET("/port", adminHandler.GetPortInfo)
			admin.GET("/hints", adminHandler.GetHints)
		} else {
			admin.GET("/port", rbac.RequireAnyRole(rbac.RoleAdmin, rbac.RoleOperator), adminHandler.GetPortInfo)
			admin.GET("/hints", rbac.RequireAnyRole(rbac.RoleAdmin, rbac.RoleOperator), adminHandler.GetHints)
		}
	}

	return r
}
