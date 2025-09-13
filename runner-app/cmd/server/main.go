package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/handlers"
	"github.com/jamie-anson/project-beacon-runner/internal/middleware"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
	"github.com/jamie-anson/project-beacon-runner/internal/execution"
	"github.com/jamie-anson/project-beacon-runner/internal/analysis"
)

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,PUT,POST,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next()
	}
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), corsMiddleware(), middleware.AuthMiddleware())

	// Initialize cross-region components
	crossRegionRepo := store.NewCrossRegionRepo(nil) // TODO: Initialize with proper DB connection
	diffEngine := analysis.NewCrossRegionDiffEngine()
	crossRegionExecutor := execution.NewCrossRegionExecutor(nil, nil, nil) // TODO: Initialize with proper dependencies
	crossRegionHandlers := handlers.NewCrossRegionHandlers(crossRegionExecutor, crossRegionRepo, diffEngine)

	// Health and admin endpoints
	r.GET("/health", handlers.Health)
	r.GET("/auth/whoami", handlers.WhoAmI)
	r.GET("/admin/config", handlers.GetAdminConfig)
	r.PUT("/admin/config", handlers.PutAdminConfig)

	// Cross-region API endpoints
	api := r.Group("/api/v1")
	{
		api.POST("/jobs/cross-region", crossRegionHandlers.SubmitCrossRegionJob)
		api.GET("/executions/:id/cross-region", crossRegionHandlers.GetCrossRegionResult)
		api.GET("/executions/:id/diff-analysis", crossRegionHandlers.GetDiffAnalysis)
		api.GET("/executions/:id/regions/:region", crossRegionHandlers.GetRegionResult)
		api.GET("/executions/cross-region", crossRegionHandlers.ListCrossRegionExecutions)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8091"
	}
	addr := ":" + port
	if err := r.Run(addr); err != nil {
		log.Printf("server error: %v", err)
	}
}
