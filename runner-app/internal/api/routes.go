package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/golem"
	"github.com/jamie-anson/project-beacon-runner/internal/jobspec"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// APIServer holds the dependencies for API handlers
type APIServer struct {
	golemService *golem.Service
	executor     *golem.ExecutionEngine
	validator    *jobspec.Validator
}

// NewAPIServer creates a new API server with dependencies
func NewAPIServer() *APIServer {
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
	
	return &APIServer{
		golemService: golemService,
		executor:     executor,
		validator:    validator,
	}
}

// SetupRoutes configures all API routes
func SetupRoutes(r *gin.Engine) {
	server := NewAPIServer()
	
	// API version group
	v1 := r.Group("/api/v1")
	{
		// Job management
		v1.POST("/jobs", server.createJob)
		v1.GET("/jobs/:id", server.getJob)
		v1.GET("/jobs", server.listJobs)
		v1.DELETE("/jobs/:id", server.deleteJob)

		// Execution management
		v1.POST("/jobs/:id/execute", server.executeJob)
		v1.GET("/executions/:id", server.getExecution)
		v1.GET("/executions", server.listExecutions)
		v1.POST("/executions/:id/cancel", server.cancelExecution)

		// Provider discovery
		v1.POST("/providers/discover", server.discoverProviders)
		v1.GET("/providers/:id", server.getProvider)

		// Cost estimation
		v1.POST("/jobs/estimate", server.estimateCost)

		// Cross-region diff analysis
		v1.GET("/executions/:id/diffs", server.getExecutionDiffs)
		v1.POST("/diffs/analyze", server.analyzeDiffs)

		// Health check
		v1.GET("/health", server.healthCheck)
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
	
	// In a real implementation, save to database here
	
	c.JSON(http.StatusCreated, gin.H{
		"message": "JobSpec created successfully",
		"job_id": jobspec.ID,
		"status": "created",
		"jobspec": jobspec,
	})
}

func (s *APIServer) getJob(c *gin.Context) {
	jobID := c.Param("id")
	
	// In a real implementation, fetch from database
	c.JSON(http.StatusOK, gin.H{
		"message": "Get job endpoint - database integration pending",
		"job_id":  jobID,
		"status":  "placeholder",
	})
}

func (s *APIServer) listJobs(c *gin.Context) {
	// In a real implementation, fetch from database with pagination
	c.JSON(http.StatusOK, gin.H{
		"message": "List jobs endpoint - database integration pending",
		"status":  "placeholder",
		"jobs":    []interface{}{},
	})
}

func (s *APIServer) deleteJob(c *gin.Context) {
	jobID := c.Param("id")
	
	// In a real implementation, delete from database
	c.JSON(http.StatusOK, gin.H{
		"message": "Delete job endpoint - database integration pending",
		"job_id":  jobID,
		"status":  "deleted",
	})
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

// Health check handler
func (s *APIServer) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"timestamp": time.Now().UTC(),
		"version": "1.0.0",
		"service": "project-beacon-runner",
		"components": gin.H{
			"golem_service": "ready",
			"execution_engine": "ready",
			"jobspec_validator": "ready",
		},
	})
}
