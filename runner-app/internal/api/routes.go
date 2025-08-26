package api

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/jamie-anson/project-beacon-runner/internal/api/middleware"
	"github.com/jamie-anson/project-beacon-runner/internal/config"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
)

func SetupRoutes(jobsService *service.JobsService, cfg *config.Config, redisClient *redis.Client) *gin.Engine {
	r := gin.Default()

	// Add middleware
	r.Use(middleware.RequestID())
	r.Use(middleware.ValidateJSON())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.RateLimiting())

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
			jobs.POST("", middleware.ValidateJobSpec(), jobsHandler.CreateJob)
			jobs.GET("/:id", jobsHandler.GetJob)
			jobs.GET("", jobsHandler.ListJobs)
		}

		transp := v1.Group("/transparency")
		{
			transp.GET("/root", transparencyHandler.GetRoot)
			transp.GET("/proof", transparencyHandler.GetProof)
			transp.GET("/bundles/:cid", transparencyHandler.GetBundle)
		}
	}

	// Admin routes (protected)
	admin := r.Group("/admin", adminAuthMiddleware())
	{
		admin.GET("/flags", adminHandler.GetFlags)
		admin.PUT("/flags", adminHandler.UpdateFlags)
		admin.GET("/config", adminHandler.GetConfig)
	}
	// In debug mode, expose /admin/port without auth to aid discovery; otherwise keep it protected
	if gin.Mode() == gin.DebugMode {
		r.GET("/admin/port", adminHandler.GetPortInfo)
		r.GET("/admin/hints", adminHandler.GetHints)
	} else {
		admin.GET("/port", adminHandler.GetPortInfo)
		admin.GET("/hints", adminHandler.GetHints)
	}

	return r
}
