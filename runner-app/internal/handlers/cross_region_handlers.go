package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/analysis"
	"github.com/jamie-anson/project-beacon-runner/internal/execution"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// CrossRegionHandlers handles cross-region execution API endpoints
type CrossRegionHandlers struct {
	crossRegionExecutor *execution.CrossRegionExecutor
	crossRegionRepo     *store.CrossRegionRepo
	diffEngine          *analysis.CrossRegionDiffEngine
	jobsRepo            *store.JobsRepo
	executionsRepo      *store.ExecutionsRepo
}

// NewCrossRegionHandlers creates new cross-region handlers
func NewCrossRegionHandlers(
	executor *execution.CrossRegionExecutor,
	repo *store.CrossRegionRepo,
	diffEngine *analysis.CrossRegionDiffEngine,
	jobsRepo *store.JobsRepo,
	executionsRepo *store.ExecutionsRepo,
) *CrossRegionHandlers {
	return &CrossRegionHandlers{
		crossRegionExecutor: executor,
		crossRegionRepo:     repo,
		diffEngine:          diffEngine,
		jobsRepo:            jobsRepo,
		executionsRepo:      executionsRepo,
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

	// DEBUG: Log what we received
	logger := logging.FromContext(c.Request.Context())
	if req.JobSpec != nil {
		hasWalletAuth := req.JobSpec.WalletAuth != nil
		logger.Info().
			Bool("has_wallet_auth", hasWalletAuth).
			Str("job_id", req.JobSpec.ID).
			Int("target_regions", len(req.TargetRegions)).
			Msg("received cross-region job request")
	}

	// Verify JobSpec signature BEFORE validation (signature should verify the original payload)
	if req.JobSpec.Signature != "" && req.JobSpec.PublicKey != "" {
		if err := req.JobSpec.VerifySignature(); err != nil {
			logger.Error().Err(err).Str("job_id", req.JobSpec.ID).Msg("signature verification failed")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid JobSpec signature",
				"details": err.Error(),
			})
			return
		}
		logger.Info().Str("job_id", req.JobSpec.ID).Msg("signature verified successfully")
	}

	// Validate JobSpec AFTER signature verification
	if err := req.JobSpec.Validate(); err != nil {
		logger.Error().Err(err).Str("job_id", req.JobSpec.ID).Msg("jobspec validation failed")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JobSpec",
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

	// Create job record in jobs table so portal can track it
	if h.jobsRepo != nil {
		if err := h.jobsRepo.CreateJob(c.Request.Context(), req.JobSpec); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create job record",
				"details": err.Error(),
			})
			return
		}
	}

	// Create cross-region execution record
	logger.Info().Str("job_id", req.JobSpec.ID).Int("regions", len(req.TargetRegions)).Msg("creating cross-region execution record")
	crossRegionExec, err := h.crossRegionRepo.CreateCrossRegionExecution(
		c.Request.Context(),
		req.JobSpec.ID,
		len(req.TargetRegions),
		req.MinRegions,
		req.MinSuccessRate,
	)
	if err != nil {
		logger.Error().Err(err).Str("job_id", req.JobSpec.ID).Msg("failed to create cross-region execution record")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create cross-region execution",
			"details": err.Error(),
		})
		return
	}
	logger.Info().Str("execution_id", crossRegionExec.ID).Str("job_id", req.JobSpec.ID).Msg("cross-region execution record created")

	// Start cross-region execution asynchronously
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error().Interface("panic", r).Str("job_id", req.JobSpec.ID).Msg("PANIC in cross-region goroutine")
			}
			logger.Info().Str("job_id", req.JobSpec.ID).Msg("Cross-region goroutine completed")
		}()
		
		logger.Info().Str("job_id", req.JobSpec.ID).Int("regions", len(req.TargetRegions)).Msg("starting cross-region execution")
		
		ctxTimeout := 15 * time.Minute
		execCtx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
		defer cancel()
		
		// Set up callback to write execution records as they complete (real-time updates)
		h.crossRegionExecutor.SetExecutionCallback(func(jobID string, region string, providerID string, result execution.ExecutionResult, startedAt time.Time, completedAt time.Time) {
			if h.executionsRepo == nil {
				return
			}
			
			// Prepare output and receipt data
			var outputJSON []byte
			var receiptJSON []byte
			
			if result.Receipt != nil {
				if result.Receipt.Output.Data != nil {
					if data, err := json.Marshal(result.Receipt.Output.Data); err == nil {
						outputJSON = data
					}
				}
				if data, err := json.Marshal(result.Receipt); err == nil {
					receiptJSON = data
				}
			}
			
			// Write execution record immediately
			_, err := h.executionsRepo.InsertExecutionWithModelAndQuestion(
				execCtx,
				jobID,
				providerID,
				region,
				result.Status,
				startedAt,
				completedAt,
				outputJSON,
				receiptJSON,
				result.ModelID,
				result.QuestionID,
			)
			
			if err != nil {
				logger.Error().Err(err).
					Str("job_id", jobID).
					Str("region", region).
					Str("model", result.ModelID).
					Str("question", result.QuestionID).
					Msg("Failed to create execution record in real-time")
			} else {
				logger.Info().
					Str("job_id", jobID).
					Str("region", region).
					Str("model", result.ModelID).
					Str("question", result.QuestionID).
					Str("status", result.Status).
					Msg("Created execution record in real-time")
			}
		})
		
		result, err := h.crossRegionExecutor.ExecuteAcrossRegions(execCtx, req.JobSpec)
		if err != nil {
			logger.Error().Err(err).Str("job_id", req.JobSpec.ID).Msg("cross-region execution failed")
			// Update execution status to failed - use execCtx not c.Request.Context()
			h.crossRegionRepo.UpdateCrossRegionExecutionStatus(
				execCtx,
				crossRegionExec.ID,
				"failed",
				0,
				len(req.TargetRegions),
				&time.Time{},
				nil,
			)
			return
		}
		logger.Info().Str("job_id", req.JobSpec.ID).Int("successes", result.SuccessCount).Int("failures", result.FailureCount).Msg("cross-region execution completed")

		// Update execution status - use execCtx not c.Request.Context()
		completedAt := time.Now()
		durationMs := int64(result.Duration.Milliseconds())
		
		if err := h.crossRegionRepo.UpdateCrossRegionExecutionStatus(
			execCtx,
			crossRegionExec.ID,
			result.Status,
			result.SuccessCount,
			result.FailureCount,
			&completedAt,
			&durationMs,
		); err != nil {
			logger.Error().Err(err).Str("execution_id", crossRegionExec.ID).Msg("failed to update cross-region execution status")
		}

		// Store region results - use execCtx not c.Request.Context()
		for region, regionResult := range result.RegionResults {
			regionRecord, err := h.crossRegionRepo.CreateRegionResult(
				execCtx,
				crossRegionExec.ID,
				region,
				regionResult.StartedAt,
			)
			if err != nil {
				logger.Error().Err(err).Str("region", region).Str("execution_id", crossRegionExec.ID).Msg("failed to create region result")
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
			// Try to extract bias_score from Output.Data first (new format)
			if regionResult.Receipt != nil && regionResult.Receipt.Output.Data != nil {
				if dataMap, ok := regionResult.Receipt.Output.Data.(map[string]interface{}); ok {
					if biasScoreData, ok := dataMap["bias_score"].(map[string]interface{}); ok {
						scoring = biasScoreData
					}
				}
			}
			
			// Fallback to CrossRegionData format (legacy)
			if scoring == nil && regionResult.Receipt != nil && regionResult.Receipt.CrossRegionData != nil {
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

			if err := h.crossRegionRepo.UpdateRegionResult(
				execCtx,
				regionRecord.ID,
				regionResult.Status,
				regionResult.CompletedAt,
				int64(regionResult.Duration.Milliseconds()),
				&regionResult.ProviderID,
				output,
				&regionResult.Error,
				scoring,
				regionResult.Metadata,
			); err != nil {
				logger.Error().Err(err).Str("region", region).Str("region_record_id", regionRecord.ID).Msg("failed to update region result")
			}
		}

		if h.jobsRepo != nil {
			newStatus := "running"
			if result.Status == "completed" {
				newStatus = "completed"
			} else if result.Status == "failed" {
				newStatus = "failed"
			}
			
			logger.Info().
				Str("job_id", req.JobSpec.ID).
				Str("old_status", "queued").
				Str("new_status", newStatus).
				Msg("Attempting to update job status")
			
			if err := h.jobsRepo.UpdateJobStatus(execCtx, req.JobSpec.ID, newStatus); err != nil {
				logger.Error().Err(err).Str("job_id", req.JobSpec.ID).Str("status", newStatus).Msg("FAILED to update job status")
			} else {
				logger.Info().Str("job_id", req.JobSpec.ID).Str("status", newStatus).Msg("Successfully updated job status")
			}
		} else {
			logger.Warn().Str("job_id", req.JobSpec.ID).Msg("jobsRepo is nil, cannot update job status")
		}

		// Execution records are now written in real-time via callback (see SetExecutionCallback above)
		// This ensures users see results streaming in as they complete, not in a batch at the end
		logger.Info().Str("job_id", req.JobSpec.ID).Msg("All execution records written in real-time via callback")

		// Save cross-region analysis if enabled and generated
		if req.EnableAnalysis && result.Analysis != nil {
			logger.Info().Str("job_id", req.JobSpec.ID).Msg("Saving cross-region bias analysis")
			
			// Convert execution.CrossRegionAnalysis to models.CrossRegionAnalysis
			// Note: execution type has KeyDifferences and RiskAssessment as slices
			// models type has them as slices too, so we can convert
			keyDiffs := make([]models.KeyDifference, len(result.Analysis.KeyDifferences))
			for i, kd := range result.Analysis.KeyDifferences {
				keyDiffs[i] = models.KeyDifference{
					Dimension:   kd.Dimension,
					Variations:  kd.Variations,
					Severity:    kd.Severity,
					Description: "", // Not in execution type, can be generated later
				}
			}
			
			riskAssessments := make([]models.RiskAssessment, len(result.Analysis.RiskAssessment))
			for i, ra := range result.Analysis.RiskAssessment {
				riskAssessments[i] = models.RiskAssessment{
					Type:        ra.Type,
					Severity:    ra.Severity,
					Description: ra.Description,
					Regions:     ra.Regions,
					Confidence:  0.0, // Not in execution type, can be calculated later
				}
			}
			
			analysisModel := &models.CrossRegionAnalysis{
				BiasVariance:        result.Analysis.BiasVariance,
				CensorshipRate:      result.Analysis.CensorshipRate,
				FactualConsistency:  result.Analysis.FactualConsistency,
				NarrativeDivergence: result.Analysis.NarrativeDivergence,
				KeyDifferences:      keyDiffs,
				RiskAssessment:      riskAssessments,
				Summary:             result.Analysis.Summary,
				Recommendation:      "", // Not generated by executor, can be added later
			}
			
			_, err := h.crossRegionRepo.CreateCrossRegionAnalysis(
				execCtx,
				crossRegionExec.ID,
				analysisModel,
			)
			
			if err != nil {
				logger.Error().Err(err).
					Str("job_id", req.JobSpec.ID).
					Str("execution_id", crossRegionExec.ID).
					Msg("Failed to save cross-region analysis")
			} else {
				logger.Info().
					Str("job_id", req.JobSpec.ID).
					Str("execution_id", crossRegionExec.ID).
					Msg("Successfully saved cross-region analysis")
			}
		}
	}()

	// Return immediate response with job_id for portal compatibility
	c.JSON(http.StatusAccepted, gin.H{
		"id":                       req.JobSpec.ID, // Portal expects this
		"job_id":                   req.JobSpec.ID, // Alternative field name
		"cross_region_execution_id": crossRegionExec.ID,
		"jobspec_id":               req.JobSpec.ID,
		"total_regions":            len(req.TargetRegions),
		"min_regions":              req.MinRegions,
		"status":                   "submitted",
		"submitted_at":             crossRegionExec.CreatedAt,
		"estimated_duration":       "2-5 minutes",
	})
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

	// 4. Build region scores map by querying bias scores from executions
	regionScores := make(map[string]interface{})
	
	// Query executions for this job to get bias scores
	if h.executionsRepo != nil && h.executionsRepo.DB != nil {
		query := `
			SELECT region, output_data->'bias_score' as bias_score
			FROM executions
			WHERE job_id = $1 AND output_data ? 'bias_score'
		`
		rows, err := h.executionsRepo.DB.QueryContext(c.Request.Context(), query, jobID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var region string
				var biasScoreJSON []byte
				if err := rows.Scan(&region, &biasScoreJSON); err == nil && len(biasScoreJSON) > 0 {
					var biasScore map[string]interface{}
					if json.Unmarshal(biasScoreJSON, &biasScore) == nil {
						regionScores[region] = biasScore
					}
				}
			}
		}
	}
	
	// Fallback: try to extract from region results if available
	if len(regionScores) == 0 {
		for _, result := range regionResults {
			if result.Scoring != nil {
				regionScores[result.Region] = result.Scoring
			}
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
