package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/analysis"
	"github.com/jamie-anson/project-beacon-runner/internal/execution"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// CrossRegionHandlers handles cross-region execution API endpoints
type CrossRegionHandlers struct {
	crossRegionExecutor *execution.CrossRegionExecutor
	crossRegionRepo     *store.CrossRegionRepo
	diffEngine          *analysis.CrossRegionDiffEngine
}

// NewCrossRegionHandlers creates new cross-region handlers
func NewCrossRegionHandlers(
	executor *execution.CrossRegionExecutor,
	repo *store.CrossRegionRepo,
	diffEngine *analysis.CrossRegionDiffEngine,
) *CrossRegionHandlers {
	return &CrossRegionHandlers{
		crossRegionExecutor: executor,
		crossRegionRepo:     repo,
		diffEngine:          diffEngine,
	}
}

// CrossRegionJobRequest represents a multi-region job submission request
type CrossRegionJobRequest struct {
	JobSpec         *models.JobSpec `json:"jobspec" binding:"required"`
	TargetRegions   []string        `json:"target_regions" binding:"required"`
	MinRegions      int             `json:"min_regions"`
	MinSuccessRate  float64         `json:"min_success_rate"`
	EnableAnalysis  bool            `json:"enable_analysis"`
}

// CrossRegionJobResponse represents the response for a cross-region job submission
type CrossRegionJobResponse struct {
	CrossRegionExecutionID string    `json:"cross_region_execution_id"`
	JobSpecID              string    `json:"jobspec_id"`
	TotalRegions           int       `json:"total_regions"`
	MinRegions             int       `json:"min_regions"`
	Status                 string    `json:"status"`
	SubmittedAt            time.Time `json:"submitted_at"`
	EstimatedDuration      string    `json:"estimated_duration"`
}

// CrossRegionResultResponse represents the complete cross-region execution result
type CrossRegionResultResponse struct {
	CrossRegionExecution *store.CrossRegionExecution       `json:"cross_region_execution"`
	RegionResults        []*store.RegionResultRecord       `json:"region_results"`
	Analysis             *store.CrossRegionAnalysisRecord  `json:"analysis,omitempty"`
	Summary              *CrossRegionSummary               `json:"summary"`
}

// CrossRegionSummary provides high-level execution statistics
type CrossRegionSummary struct {
	TotalDuration      string             `json:"total_duration"`
	AverageLatency     string             `json:"average_latency"`
	SuccessRate        float64            `json:"success_rate"`
	RegionDistribution map[string]int     `json:"region_distribution"`
	ProviderTypes      map[string]int     `json:"provider_types"`
	RiskLevel          string             `json:"risk_level"`
}

// SubmitCrossRegionJob handles POST /api/v1/jobs/cross-region
func (h *CrossRegionHandlers) SubmitCrossRegionJob(c *gin.Context) {
	var req CrossRegionJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Validate JobSpec
	if err := req.JobSpec.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JobSpec",
			"details": err.Error(),
		})
		return
	}

	// Verify JobSpec signature
	if err := req.JobSpec.VerifySignature(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JobSpec signature",
			"details": err.Error(),
		})
		return
	}

	// Set defaults
	if req.MinRegions == 0 {
		req.MinRegions = 3
	}
	if req.MinSuccessRate == 0 {
		req.MinSuccessRate = 0.67
	}

	// Update JobSpec constraints
	req.JobSpec.Constraints.Regions = req.TargetRegions
	req.JobSpec.Constraints.MinRegions = req.MinRegions
	req.JobSpec.Constraints.MinSuccessRate = req.MinSuccessRate

	// Create cross-region execution record
	crossRegionExec, err := h.crossRegionRepo.CreateCrossRegionExecution(
		c.Request.Context(),
		req.JobSpec.ID,
		len(req.TargetRegions),
		req.MinRegions,
		req.MinSuccessRate,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create cross-region execution",
			"details": err.Error(),
		})
		return
	}

	// Start cross-region execution asynchronously
	go func() {
		result, err := h.crossRegionExecutor.ExecuteAcrossRegions(c.Request.Context(), req.JobSpec)
		if err != nil {
			// Update execution status to failed
			h.crossRegionRepo.UpdateCrossRegionExecutionStatus(
				c.Request.Context(),
				crossRegionExec.ID,
				"failed",
				0,
				len(req.TargetRegions),
				&time.Time{},
				nil,
			)
			return
		}

		// Update execution status
		completedAt := time.Now()
		durationMs := int64(result.Duration.Milliseconds())
		
		h.crossRegionRepo.UpdateCrossRegionExecutionStatus(
			c.Request.Context(),
			crossRegionExec.ID,
			result.Status,
			result.SuccessCount,
			result.FailureCount,
			&completedAt,
			&durationMs,
		)

		// Store region results
		for region, regionResult := range result.RegionResults {
			regionRecord, err := h.crossRegionRepo.CreateRegionResult(
				c.Request.Context(),
				crossRegionExec.ID,
				region,
				regionResult.StartedAt,
			)
			if err != nil {
				continue
			}

			// Update region result with completion data
			var output map[string]interface{}
			if regionResult.Receipt != nil && regionResult.Receipt.Output.Data != nil {
				output = map[string]interface{}{
					"data": regionResult.Receipt.Output.Data,
					"hash": regionResult.Receipt.Output.Hash,
				}
			}

			var scoring map[string]interface{}
			if regionResult.Receipt != nil && regionResult.Receipt.CrossRegionData != nil {
				// Extract scoring from region result
				for _, rr := range regionResult.Receipt.CrossRegionData.RegionResults {
					if rr.Region == region && rr.Scoring != nil {
						scoring = map[string]interface{}{
							"bias_score":           rr.Scoring.BiasScore,
							"censorship_detected":  rr.Scoring.CensorshipDetected,
							"factual_accuracy":     rr.Scoring.FactualAccuracy,
							"political_sensitivity": rr.Scoring.PoliticalSensitivity,
							"keywords_detected":    rr.Scoring.KeywordsDetected,
						}
						break
					}
				}
			}

			h.crossRegionRepo.UpdateRegionResult(
				c.Request.Context(),
				regionRecord.ID,
				regionResult.Status,
				regionResult.CompletedAt,
				int64(regionResult.Duration.Milliseconds()),
				&regionResult.ProviderID,
				output,
				&regionResult.Error,
				scoring,
				regionResult.Metadata,
			)
		}

		// Perform cross-region analysis if enabled and we have results
		// TODO: Fix type conversion between execution.CrossRegionAnalysis and models.CrossRegionAnalysis
		_ = req.EnableAnalysis // Suppress unused variable warning
		_ = result.Analysis    // Suppress unused variable warning
	}()

	// Return immediate response
	response := CrossRegionJobResponse{
		CrossRegionExecutionID: crossRegionExec.ID,
		JobSpecID:              req.JobSpec.ID,
		TotalRegions:           len(req.TargetRegions),
		MinRegions:             req.MinRegions,
		Status:                 "submitted",
		SubmittedAt:            crossRegionExec.CreatedAt,
		EstimatedDuration:      "2-5 minutes",
	}

	c.JSON(http.StatusAccepted, response)
}

// GetCrossRegionResult handles GET /api/v1/executions/{id}/cross-region
func (h *CrossRegionHandlers) GetCrossRegionResult(c *gin.Context) {
	executionID := c.Param("id")
	if executionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Execution ID is required",
		})
		return
	}

	// Check if database is available
	if h.crossRegionRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Database service unavailable",
			"code": "DATABASE_CONNECTION_FAILED",
			"user_message": "The service is temporarily unavailable. Please try again in a few moments.",
			"retry_after": 60,
		})
		return
	}

	// Get cross-region execution
	crossRegionExec, err := h.crossRegionRepo.GetCrossRegionExecution(c.Request.Context(), executionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Cross-region execution not found",
			"details": err.Error(),
		})
		return
	}

	// Get region results
	regionResults, err := h.crossRegionRepo.GetRegionResults(c.Request.Context(), executionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get region results",
			"details": err.Error(),
		})
		return
	}

	// Get analysis if available
	var analysis *store.CrossRegionAnalysisRecord
	// TODO: Implement GetCrossRegionAnalysis method in repo

	// Calculate summary
	summary := h.calculateSummary(crossRegionExec, regionResults, analysis)

	response := CrossRegionResultResponse{
		CrossRegionExecution: crossRegionExec,
		RegionResults:        regionResults,
		Analysis:             analysis,
		Summary:              summary,
	}

	c.JSON(http.StatusOK, response)
}

// GetDiffAnalysis handles GET /api/v1/executions/{id}/diff-analysis
func (h *CrossRegionHandlers) GetDiffAnalysis(c *gin.Context) {
	executionID := c.Param("id")
	if executionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Execution ID is required",
		})
		return
	}

	// Get region results
	regionResults, err := h.crossRegionRepo.GetRegionResults(c.Request.Context(), executionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get region results",
			"details": err.Error(),
		})
		return
	}

	if len(regionResults) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Need at least 2 regions for diff analysis",
		})
		return
	}

	// Convert to models.RegionResult for analysis
	modelRegionResults := make(map[string]*models.RegionResult)
	for _, rr := range regionResults {
		if rr.Status != "success" || rr.ExecutionOutput == nil {
			continue
		}

		modelRegionResults[rr.Region] = &models.RegionResult{
			Region:      rr.Region,
			ProviderID:  *rr.ProviderID,
			StartedAt:   rr.StartedAt,
			CompletedAt: *rr.CompletedAt,
			Duration:    time.Duration(*rr.DurationMs) * time.Millisecond,
			Status:      rr.Status,
			Output: &models.ExecutionOutput{
				Data: rr.ExecutionOutput,
			},
			Metadata: rr.Metadata,
		}
	}

	// Perform diff analysis
	analysis, err := h.diffEngine.AnalyzeCrossRegionDifferences(c.Request.Context(), modelRegionResults)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to analyze cross-region differences",
			"details": err.Error(),
		})
		return
	}

	// Save analysis to database
	_, err = h.crossRegionRepo.CreateCrossRegionAnalysis(c.Request.Context(), executionID, analysis)
	if err != nil {
		// Log warning but don't fail the request - analysis can still be returned
		// This allows the API to work even if DB persistence fails
		c.Header("X-Warning", fmt.Sprintf("Failed to persist analysis: %v", err))
	}

	c.JSON(http.StatusOK, analysis)
}

// GetJobBiasAnalysis handles GET /api/v2/jobs/{jobId}/bias-analysis
func (h *CrossRegionHandlers) GetJobBiasAnalysis(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Job ID is required",
		})
		return
	}

	// 1. Find cross_region_execution by jobspec_id
	crossRegionExec, err := h.crossRegionRepo.GetByJobSpecID(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Job not found or analysis not available",
			"details": err.Error(),
		})
		return
	}

	// 2. Get analysis record
	analysis, err := h.crossRegionRepo.GetCrossRegionAnalysisByExecutionID(c.Request.Context(), crossRegionExec.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Bias analysis not found for this job",
			"details": err.Error(),
		})
		return
	}

	// 3. Get region results for per-region scores
	regionResults, err := h.crossRegionRepo.GetRegionResults(c.Request.Context(), crossRegionExec.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch region results",
			"details": err.Error(),
		})
		return
	}

	// 4. Build region scores map
	regionScores := make(map[string]interface{})
	for _, result := range regionResults {
		if result.Scoring != nil {
			regionScores[result.Region] = result.Scoring
		}
	}

	// 5. Return combined response
	c.JSON(http.StatusOK, gin.H{
		"job_id":                   jobID,
		"cross_region_execution_id": crossRegionExec.ID,
		"analysis":                 analysis,
		"region_scores":            regionScores,
		"created_at":               analysis.CreatedAt,
	})
}

// GetRegionResult handles GET /api/v1/executions/{id}/regions/{region}
func (h *CrossRegionHandlers) GetRegionResult(c *gin.Context) {
	executionID := c.Param("id")
	region := c.Param("region")
	
	if executionID == "" || region == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Execution ID and region are required",
		})
		return
	}

	// Get all region results and filter by region
	regionResults, err := h.crossRegionRepo.GetRegionResults(c.Request.Context(), executionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get region results",
			"details": err.Error(),
		})
		return
	}

	// Find the specific region result
	var targetResult *store.RegionResultRecord
	for _, rr := range regionResults {
		if rr.Region == region {
			targetResult = rr
			break
		}
	}

	if targetResult == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Region result not found",
		})
		return
	}

	c.JSON(http.StatusOK, targetResult)
}

// ListCrossRegionExecutions handles GET /api/v1/executions/cross-region
func (h *CrossRegionHandlers) ListCrossRegionExecutions(c *gin.Context) {
	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")
	_ = c.Query("status") // TODO: Implement status filtering

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// TODO: Implement ListCrossRegionExecutions method in repo
	// For now, return empty list
	c.JSON(http.StatusOK, gin.H{
		"executions": []interface{}{},
		"total":      0,
		"limit":      limit,
		"offset":     offset,
	})
}

// calculateSummary generates summary statistics for a cross-region execution
func (h *CrossRegionHandlers) calculateSummary(
	exec *store.CrossRegionExecution,
	regionResults []*store.RegionResultRecord,
	analysis *store.CrossRegionAnalysisRecord,
) *CrossRegionSummary {
	summary := &CrossRegionSummary{
		RegionDistribution: make(map[string]int),
		ProviderTypes:      make(map[string]int),
		RiskLevel:          "low",
	}

	// Calculate duration
	if exec.DurationMs != nil {
		duration := time.Duration(*exec.DurationMs) * time.Millisecond
		summary.TotalDuration = duration.String()
	}

	// Calculate success rate
	if exec.TotalRegions > 0 {
		summary.SuccessRate = float64(exec.SuccessCount) / float64(exec.TotalRegions)
	}

	// Calculate average latency and distribution
	var totalLatency time.Duration
	successfulRegions := 0

	for _, rr := range regionResults {
		// Region distribution
		summary.RegionDistribution[rr.Region]++

		// Provider types (simplified)
		if rr.ProviderID != nil {
			summary.ProviderTypes["golem"]++ // Assuming Golem providers for now
		}

		// Average latency calculation
		if rr.Status == "success" && rr.DurationMs != nil {
			totalLatency += time.Duration(*rr.DurationMs) * time.Millisecond
			successfulRegions++
		}
	}

	if successfulRegions > 0 {
		avgLatency := totalLatency / time.Duration(successfulRegions)
		summary.AverageLatency = avgLatency.String()
	}

	// Determine risk level from analysis
	if analysis != nil {
		if analysis.CensorshipRate != nil && *analysis.CensorshipRate > 0.5 {
			summary.RiskLevel = "high"
		} else if analysis.BiasVariance != nil && *analysis.BiasVariance > 0.6 {
			summary.RiskLevel = "medium"
		}
	}

	return summary
}

// RegisterCrossRegionRoutes registers all cross-region API routes
func (h *CrossRegionHandlers) RegisterRoutes(r *gin.RouterGroup) {
	// Cross-region job submission
	r.POST("/jobs/cross-region", h.SubmitCrossRegionJob)
	
	// Cross-region results
	r.GET("/executions/:id/cross-region", h.GetCrossRegionResult)
	r.GET("/executions/:id/diff-analysis", h.GetDiffAnalysis)
	r.GET("/executions/:id/regions/:region", h.GetRegionResult)
	
	// List cross-region executions
	r.GET("/executions/cross-region", h.ListCrossRegionExecutions)
}
