package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
)

// BiasHandler handles bias scoring API endpoints
type BiasHandler struct {
	ExecutionService *service.ExecutionService
}

// NewBiasHandler creates a new bias handler
func NewBiasHandler(executionService *service.ExecutionService) *BiasHandler {
	return &BiasHandler{
		ExecutionService: executionService,
	}
}

// GetBiasScore returns bias metrics for a specific execution
// GET /api/v1/executions/{id}/bias-score
func (h *BiasHandler) GetBiasScore(c *gin.Context) {
	executionIDStr := c.Param("id")
	executionID, err := strconv.ParseInt(executionIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid execution ID"})
		return
	}

	biasScore, err := h.ExecutionService.BiasScorer.GetBiasScore(c.Request.Context(), executionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bias score not found"})
		return
	}

	c.JSON(http.StatusOK, biasScore)
}

// GetMultipleBiasScores returns bias metrics for multiple executions
// GET /api/v1/jobs/{id}/bias-scores
func (h *BiasHandler) GetMultipleBiasScores(c *gin.Context) {
	_ = c.Param("id") // jobID for future implementation
	
	// This would need to be implemented to get all executions for a job
	// and return their bias scores
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Multiple bias scores endpoint not yet implemented"})
}
