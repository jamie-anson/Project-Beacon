package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/db"
)

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
		v1.GET("/diffs", server.listDiffs)
		v1.GET("/diffs/:id", server.getDiff)
		
		// IPFS bundles
		v1.POST("/jobs/:id/bundle", server.createIPFSBundle)
		v1.GET("/bundles/:cid", server.getIPFSBundle)
		v1.GET("/bundles", server.listIPFSBundles)

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

	// System endpoints
	r.GET("/debug/yagna", server.debugYagna)

	// WebSocket endpoint used by the Portal
	r.GET("/ws", func(c *gin.Context) {
		server.wsHub.ServeWS(c.Writer, c.Request)
	})
}
