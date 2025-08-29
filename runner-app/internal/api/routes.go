package api

import (
    "github.com/gin-gonic/gin"
    "github.com/redis/go-redis/v9"
    "github.com/jamie-anson/project-beacon-runner/internal/api/middleware"
    "github.com/jamie-anson/project-beacon-runner/internal/config"
    "github.com/jamie-anson/project-beacon-runner/internal/service"
    rbac "github.com/jamie-anson/project-beacon-runner/internal/middleware"
)

func SetupRoutes(jobsService *service.JobsService, cfg *config.Config, redisClient *redis.Client) *gin.Engine {
    r := gin.Default()

    // Add middleware
    r.Use(middleware.RequestID())
    r.Use(middleware.ValidateJSON())
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
