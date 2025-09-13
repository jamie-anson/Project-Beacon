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
	"github.com/jamie-anson/project-beacon-runner/internal/db"
	"github.com/jamie-anson/project-beacon-runner/internal/websocket"
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

	// Initialize database connection
	dbURL := os.Getenv("DATABASE_URL")
	database, err := db.Initialize(dbURL)
	if err != nil {
		log.Printf("Failed to initialize database: %v", err)
		log.Println("Continuing with limited functionality...")
	}

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Initialize cross-region components with proper database connection
	crossRegionRepo := store.NewCrossRegionRepo(database.DB)
	diffEngine := analysis.NewCrossRegionDiffEngine()
	
	// TODO: Initialize CrossRegionExecutor with proper hybrid router and single region executor
	crossRegionExecutor := execution.NewCrossRegionExecutor(nil, nil, nil)
	crossRegionHandlers := handlers.NewCrossRegionHandlers(crossRegionExecutor, crossRegionRepo, diffEngine)

	// Health and admin endpoints
	r.GET("/health", handlers.Health)
	r.GET("/auth/whoami", handlers.WhoAmI)
	r.GET("/admin/config", handlers.GetAdminConfig)
	r.POST("/admin/migrate", handlers.TriggerMigration)
	r.PUT("/admin/config", handlers.PutAdminConfig)

	// Provider discovery endpoint for portal compatibility
	r.GET("/providers", func(c *gin.Context) {
		// Return empty providers list for now - TODO: integrate with hybrid router
		c.JSON(200, gin.H{
			"providers": []gin.H{},
			"status": "ok",
		})
	})

	// WebSocket endpoint
	r.GET("/ws", func(c *gin.Context) {
		hub.ServeWS(c.Writer, c.Request)
	})

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
