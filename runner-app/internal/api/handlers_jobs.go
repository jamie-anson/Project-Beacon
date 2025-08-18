package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// Job management handlers

func (s *APIServer) createJob(c *gin.Context) {
	var jobspecData map[string]interface{}
	if err := c.ShouldBindJSON(&jobspecData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON",
			"details": err.Error(),
		})
		return
	}
	
	// Convert to JobSpec struct
	var jobspec models.JobSpec
	jobspecBytes, err := json.Marshal(jobspecData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process JobSpec",
		})
		return
	}
	
	if err := json.Unmarshal(jobspecBytes, &jobspec); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JobSpec format",
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
	
	// Parse and validate JobSpec
	var jobspecForValidation models.JobSpec
	if err := json.Unmarshal(jobspecJSON, &jobspecForValidation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JobSpec format",
			"details": err.Error(),
		})
		return
	}
	
	// Validate JobSpec
	err = s.validator.ValidateAndVerify(&jobspecForValidation)
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
		if err := s.jobsSvc.CreateJob(c.Request.Context(), &jobspecForValidation, jobspecJSON); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to store job",
				"details": err.Error(),
			})
			return
		}
		persisted = true
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Job created successfully",
		"job_id":    jobspec.ID,
		"persisted": persisted,
		"jobspec":   jobspec,
	})
}

func (s *APIServer) estimateCost(c *gin.Context) {
	var jobspecData map[string]interface{}
	if err := c.ShouldBindJSON(&jobspecData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON",
			"details": err.Error(),
		})
		return
	}
	
	// Convert to JobSpec struct
	var jobspec models.JobSpec
	jobspecBytes, err := json.Marshal(jobspecData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process JobSpec",
		})
		return
	}
	
	if err := json.Unmarshal(jobspecBytes, &jobspec); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JobSpec format",
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
	
	// Parse and validate JobSpec
	var jobspecForValidation models.JobSpec
	if err := json.Unmarshal(jobspecJSON, &jobspecForValidation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JobSpec format",
			"details": err.Error(),
		})
		return
	}
	
	// Validate JobSpec
	err = s.validator.ValidateAndVerify(&jobspecForValidation)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "JobSpec validation failed",
			"details": err.Error(),
		})
		return
	}

	
	ctx := context.Background()
	cost, err := s.executor.EstimateExecutionCost(ctx, &jobspecForValidation)
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
