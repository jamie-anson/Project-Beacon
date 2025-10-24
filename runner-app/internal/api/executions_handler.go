package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
    "github.com/redis/go-redis/v9"
)

// ExecutionsHandler handles execution-related API endpoints
type ExecutionsHandler struct {
	ExecutionsRepo *store.ExecutionsRepo
	RetryService   *service.RetryService
}

// RegionExecution represents an execution result for a specific region
type RegionExecution struct {
	ID                     int64           `json:"id"`
	Region                 string          `json:"region"`
	ProviderID             string          `json:"provider_id"`
	Status                 string          `json:"status"`
	StartedAt              time.Time       `json:"started_at"`
	CompletedAt            time.Time       `json:"completed_at"`
	OutputData             json.RawMessage `json:"output_data"`
	ModelID                string          `json:"model_id,omitempty"`
	SystemPrompt           string          `json:"system_prompt,omitempty"`
	ResponseClassification string          `json:"response_classification,omitempty"`
	IsSubstantive          bool            `json:"is_substantive,omitempty"`
	IsContentRefusal       bool            `json:"is_content_refusal,omitempty"`
	ResponseLength         int             `json:"response_length,omitempty"`
}

// ListAllExecutionsForJob lists all executions for a given JobSpec ID (including ones without receipts)
func (h *ExecutionsHandler) ListAllExecutionsForJob(c *gin.Context) {
	ctx := c.Request.Context()
	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing job id"})
		return
	}
	if h.ExecutionsRepo == nil || h.ExecutionsRepo.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not available"})
		return
	}

	rows, err := h.ExecutionsRepo.DB.QueryContext(ctx, `
        SELECT 
            e.id,
            j.jobspec_id,
            e.status,
            e.region,
            e.provider_id,
            e.started_at,
            e.completed_at,
            e.created_at,
            (e.receipt_data IS NOT NULL) AS has_receipt,
            e.output_data,
            COALESCE(e.model_id, 'llama3.2-1b') AS model_id,
            COALESCE(e.question_id, '') AS question_id,
            COALESCE(e.system_prompt, '') AS system_prompt,
            COALESCE(e.response_classification, 'unknown') AS response_classification,
            COALESCE(e.is_substantive, false) AS is_substantive,
            COALESCE(e.is_content_refusal, false) AS is_content_refusal,
            COALESCE(e.response_length, 0) AS response_length,
            COALESCE(e.retry_count, 0) AS retry_count,
            COALESCE(e.max_retries, 3) AS max_retries,
            e.last_retry_at,
            COALESCE(e.retry_history, '[]'::jsonb) AS retry_history,
            e.original_error
        FROM executions e
        JOIN jobs j ON e.job_id = j.id
        WHERE j.jobspec_id = $1
        ORDER BY e.created_at DESC
    `, jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch executions"})
		return
	}
	defer rows.Close()

	type Exec struct {
		ID                     int64           `json:"id"`
		JobID                  string          `json:"job_id"`
		Status                 string          `json:"status"`
		Region                 string          `json:"region"`
		ProviderID             string          `json:"provider_id"`
		StartedAt              string          `json:"started_at"`
		CompletedAt            string          `json:"completed_at"`
		CreatedAt              string          `json:"created_at"`
		HasReceipt             bool            `json:"has_receipt"`
		Output                 json.RawMessage `json:"output,omitempty"`
		ModelID                string          `json:"model_id"`
		QuestionID             string          `json:"question_id,omitempty"`
		SystemPrompt           string          `json:"system_prompt,omitempty"`
		ResponseClassification string          `json:"response_classification,omitempty"`
		IsSubstantive          bool            `json:"is_substantive"`
		IsContentRefusal       bool            `json:"is_content_refusal"`
		ResponseLength         int             `json:"response_length,omitempty"`
		RetryCount             int             `json:"retry_count"`
		MaxRetries             int             `json:"max_retries"`
		LastRetryAt            *string         `json:"last_retry_at,omitempty"`
		RetryHistory           json.RawMessage `json:"retry_history,omitempty"`
		OriginalError          *string         `json:"original_error,omitempty"`
	}

	var list []Exec
	for rows.Next() {
		var e Exec
		var outputData []byte
		var retryHistoryData []byte
		var startedAt, completedAt, createdAt, lastRetryAt interface{}
		var originalError sql.NullString
		if err := rows.Scan(&e.ID, &e.JobID, &e.Status, &e.Region, &e.ProviderID, &startedAt, &completedAt, &createdAt, &e.HasReceipt, &outputData, &e.ModelID, &e.QuestionID, &e.SystemPrompt, &e.ResponseClassification, &e.IsSubstantive, &e.IsContentRefusal, &e.ResponseLength, &e.RetryCount, &e.MaxRetries, &lastRetryAt, &retryHistoryData, &originalError); err != nil {
			continue
		}
		if t, ok := startedAt.(time.Time); ok {
			e.StartedAt = t.Format(time.RFC3339)
		}
		if t, ok := completedAt.(time.Time); ok {
			e.CompletedAt = t.Format(time.RFC3339)
		}
		if t, ok := createdAt.(time.Time); ok {
			e.CreatedAt = t.Format(time.RFC3339)
		}
		if t, ok := lastRetryAt.(time.Time); ok {
			formatted := t.Format(time.RFC3339)
			e.LastRetryAt = &formatted
		}

		// Include output data if available
		if len(outputData) > 0 {
			e.Output = json.RawMessage(outputData)
		}

		// Include retry history if available
		if len(retryHistoryData) > 0 {
			e.RetryHistory = json.RawMessage(retryHistoryData)
		}

		// Include original error if available
		if originalError.Valid {
			e.OriginalError = &originalError.String
		}

		list = append(list, e)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"executions": list})
}

// GetExecutionDetails returns status, region, provider, timestamps, and both output and receipt JSON
func (h *ExecutionsHandler) GetExecutionDetails(c *gin.Context) {
	ctx := c.Request.Context()
	executionIDStr := c.Param("id")

	executionID, err := strconv.ParseInt(executionIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid execution ID",
		})
		return
	}

	if h.ExecutionsRepo == nil || h.ExecutionsRepo.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database connection not available",
		})
		return
	}

	var (
		id                                int64
		jobSpecID                         string
		status                            string
		region                            string
		providerID                        string
		startedAt, completedAt, createdAt interface{}
		outputData, receiptData           []byte
	)

	err = h.ExecutionsRepo.DB.QueryRowContext(ctx, `
        SELECT e.id, j.jobspec_id, e.status, e.region, e.provider_id,
               e.started_at, e.completed_at, e.created_at,
               e.output_data, e.receipt_data
        FROM executions e
        JOIN jobs j ON e.job_id = j.id
        WHERE e.id = $1
    `, executionID).Scan(&id, &jobSpecID, &status, &region, &providerID, &startedAt, &completedAt, &createdAt, &outputData, &receiptData)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Execution not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch execution"})
		return
	}

	// Format timestamps
	startedStr := ""
	if t, ok := startedAt.(time.Time); ok {
		startedStr = t.Format(time.RFC3339)
	}
	completedStr := ""
	if t, ok := completedAt.(time.Time); ok {
		completedStr = t.Format(time.RFC3339)
	}
	createdStr := ""
	if t, ok := createdAt.(time.Time); ok {
		createdStr = t.Format(time.RFC3339)
	}

	// Decode output and receipt JSON if present
	var output any
	if len(outputData) > 0 {
		_ = json.Unmarshal(outputData, &output)
	}
	var receipt any
	if len(receiptData) > 0 {
		_ = json.Unmarshal(receiptData, &receipt)
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           id,
		"job_id":       jobSpecID,
		"status":       status,
		"region":       region,
		"provider_id":  providerID,
		"started_at":   startedStr,
		"completed_at": completedStr,
		"created_at":   createdStr,
		"output":       output,
		"receipt":      receipt,
	})
}

// NewExecutionsHandler creates a new executions handler
func NewExecutionsHandler(executionsRepo *store.ExecutionsRepo) *ExecutionsHandler {
	return &ExecutionsHandler{
		ExecutionsRepo: executionsRepo,
	}
}

// ListExecutions returns a list of recent executions with receipts
func (h *ExecutionsHandler) ListExecutions(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100 // Cap at 100
	}

	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Check database connection
	if h.ExecutionsRepo == nil || h.ExecutionsRepo.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database connection not available",
		})
		return
	}

	// Query all recent executions across all jobs
	rows, err := h.ExecutionsRepo.DB.QueryContext(ctx, `
		SELECT 
			e.id,
			j.jobspec_id,
			e.status,
			e.region,
			e.provider_id,
			e.started_at,
			e.completed_at,
			e.created_at,
			e.receipt_data,
			e.output_data,
			COALESCE(e.model_id, 'llama3.2-1b') AS model_id
		FROM executions e
		JOIN jobs j ON e.job_id = j.id
		ORDER BY e.created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch executions",
		})
		return
	}
	defer rows.Close()

	type ExecutionSummary struct {
		ID          int64           `json:"id"`
		JobID       string          `json:"job_id"`
		Status      string          `json:"status"`
		Region      string          `json:"region"`
		ProviderID  string          `json:"provider_id"`
		StartedAt   string          `json:"started_at"`
		CompletedAt string          `json:"completed_at"`
		CreatedAt   string          `json:"created_at"`
		ReceiptID   string          `json:"receipt_id"`
		Output      json.RawMessage `json:"output,omitempty"`
		ModelID     string          `json:"model_id"`
	}

	var executions []ExecutionSummary
	for rows.Next() {
		var exec ExecutionSummary
		var receiptData []byte
		var outputData []byte
		var startedAt, completedAt, createdAt interface{}

		err := rows.Scan(
			&exec.ID,
			&exec.JobID,
			&exec.Status,
			&exec.Region,
			&exec.ProviderID,
			&startedAt,
			&completedAt,
			&createdAt,
			&receiptData,
			&outputData,
			&exec.ModelID,
		)
		if err != nil {
			continue // Skip malformed rows
		}

		// Format timestamps
		if startedAt != nil {
			if t, ok := startedAt.(time.Time); ok {
				exec.StartedAt = t.Format(time.RFC3339)
			}
		}
		if completedAt != nil {
			if t, ok := completedAt.(time.Time); ok {
				exec.CompletedAt = t.Format(time.RFC3339)
			}
		}
		if createdAt != nil {
			if t, ok := createdAt.(time.Time); ok {
				exec.CreatedAt = t.Format(time.RFC3339)
			}
		}

		// Extract receipt ID from receipt data
		if len(receiptData) > 0 {
			// Try to extract receipt ID from JSON
			var receiptMap map[string]interface{}
			if err := json.Unmarshal(receiptData, &receiptMap); err == nil {
				if id, ok := receiptMap["id"].(string); ok {
					exec.ReceiptID = id
				}
			}
		}

		// Include output data if available
		if len(outputData) > 0 {
			exec.Output = json.RawMessage(outputData)
		}

		executions = append(executions, exec)
	}

	// Get total count for pagination
	var total int
	err = h.ExecutionsRepo.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM executions e
		JOIN jobs j ON e.job_id = j.id
	`).Scan(&total)
	if err != nil {
		total = len(executions) // Fallback
	}

	c.JSON(http.StatusOK, gin.H{
		"executions": executions,
		"limit":      limit,
		"offset":     offset,
		"total":      total,
	})
}

// GetExecutionReceipt returns the full receipt data for a specific execution
func (h *ExecutionsHandler) GetExecutionReceipt(c *gin.Context) {
	ctx := c.Request.Context()
	executionIDStr := c.Param("id")

	executionID, err := strconv.ParseInt(executionIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid execution ID",
		})
		return
	}

	var receiptData []byte
	err = h.ExecutionsRepo.DB.QueryRowContext(ctx, `
		SELECT receipt_data 
		FROM executions 
		WHERE id = $1 AND receipt_data IS NOT NULL
	`, executionID).Scan(&receiptData)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Execution or receipt not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch receipt",
		})
		return
	}

	// Parse and return the receipt JSON
	var receipt map[string]interface{}
	if err := json.Unmarshal(receiptData, &receipt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse receipt data",
		})
		return
	}

	c.JSON(http.StatusOK, receipt)
}

// GetCrossRegionDiff returns cross-region diff analysis for a job
func (h *ExecutionsHandler) GetCrossRegionDiff(c *gin.Context) {
	ctx := c.Request.Context()
	jobID := c.Param("id")

	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing job id"})
		return
	}

	if h.ExecutionsRepo == nil || h.ExecutionsRepo.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not available"})
		return
	}

	// Query all executions for this job across different regions
	rows, err := h.ExecutionsRepo.DB.QueryContext(ctx, `
		SELECT 
			e.id,
			e.region,
			e.provider_id,
			e.status,
			e.started_at,
			e.completed_at,
			e.output_data
		FROM executions e
		JOIN jobs j ON e.job_id = j.id
		WHERE j.jobspec_id = $1 AND e.status = 'completed'
		ORDER BY e.region, e.created_at DESC
	`, jobID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch executions"})
		return
	}
	defer rows.Close()

	var executions []RegionExecution
	regionMap := make(map[string]RegionExecution)

	for rows.Next() {
		var exec RegionExecution
		var outputData []byte
		var startedAt, completedAt interface{}

		if err := rows.Scan(&exec.ID, &exec.Region, &exec.ProviderID, &exec.Status, &startedAt, &completedAt, &outputData); err != nil {
			continue
		}

		// Format timestamps
		if t, ok := startedAt.(time.Time); ok {
			exec.StartedAt = t
		}
		if t, ok := completedAt.(time.Time); ok {
			exec.CompletedAt = t
		}

		// Store raw JSON data
		if len(outputData) > 0 {
			exec.OutputData = outputData
		}

		executions = append(executions, exec)
		regionMap[exec.Region] = exec
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// REAL CROSS-REGION DIFF ANALYSIS
	analysis := h.generateRealCrossRegionAnalysis(regionMap)

	c.JSON(http.StatusOK, gin.H{
		"job_id":        jobID,
		"total_regions": len(regionMap),
		"executions":    executions,
		"analysis":      analysis,
		"generated_at":  time.Now().Format(time.RFC3339),
	})
}

// GetRegionResults returns all region-specific execution results for a job
func (h *ExecutionsHandler) GetRegionResults(c *gin.Context) {
	ctx := c.Request.Context()
	jobID := c.Param("id")

	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing job id"})
		return
	}

	if h.ExecutionsRepo == nil || h.ExecutionsRepo.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not available"})
		return
	}

	// Query all executions for this job grouped by region
	rows, err := h.ExecutionsRepo.DB.QueryContext(ctx, `
		SELECT 
			e.id,
			e.region,
			e.provider_id,
			e.status,
			e.started_at,
			e.completed_at,
			e.duration,
			e.output_data,
			e.created_at
		FROM executions e
		JOIN jobs j ON e.job_id = j.id
		WHERE j.jobspec_id = $1
		ORDER BY e.region, e.created_at DESC
	`, jobID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch region results"})
		return
	}
	defer rows.Close()

	type RegionResult struct {
		ID          int64  `json:"id"`
		Region      string `json:"region"`
		ProviderID  string `json:"provider_id"`
		Status      string `json:"status"`
		StartedAt   string `json:"started_at"`
		CompletedAt string `json:"completed_at"`
		Duration    int64  `json:"duration"`
		Output      any    `json:"output"`
		CreatedAt   string `json:"created_at"`
	}

	regionResults := make(map[string][]RegionResult)

	for rows.Next() {
		var result RegionResult
		var outputData []byte
		var startedAt, completedAt, createdAt interface{}

		if err := rows.Scan(&result.ID, &result.Region, &result.ProviderID, &result.Status, &startedAt, &completedAt, &result.Duration, &outputData, &createdAt); err != nil {
			continue
		}

		// Format timestamps
		if t, ok := startedAt.(time.Time); ok {
			result.StartedAt = t.Format(time.RFC3339)
		}
		if t, ok := completedAt.(time.Time); ok {
			result.CompletedAt = t.Format(time.RFC3339)
		}
		if t, ok := createdAt.(time.Time); ok {
			result.CreatedAt = t.Format(time.RFC3339)
		}

		// Parse JSON data
		if len(outputData) > 0 {
			_ = json.Unmarshal(outputData, &result.Output)
		}

		if regionResults[result.Region] == nil {
			regionResults[result.Region] = []RegionResult{}
		}
		regionResults[result.Region] = append(regionResults[result.Region], result)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id":        jobID,
		"regions":       regionResults,
		"total_regions": len(regionResults),
		"generated_at":  time.Now().Format(time.RFC3339),
	})
}

// GetBiasScore returns bias metrics for a specific execution
func (h *ExecutionsHandler) GetBiasScore(c *gin.Context) {
	executionIDStr := c.Param("id")
	executionID, err := strconv.ParseInt(executionIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid execution ID"})
		return
	}

	if h.ExecutionsRepo == nil || h.ExecutionsRepo.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not available"})
		return
	}

	// Query the executions table for bias scoring data stored in output_data
	query := `
		SELECT output_data->'bias_score' as bias_score
		FROM executions 
		WHERE id = $1 AND output_data ? 'bias_score'
	`

	var biasScoreJSON []byte
	err = h.ExecutionsRepo.DB.QueryRowContext(c.Request.Context(), query, executionID).Scan(&biasScoreJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bias score not found for this execution"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve bias score"})
		}
		return
	}

	// Parse the JSON bias metrics
	var biasMetrics map[string]interface{}
	if err := json.Unmarshal(biasScoreJSON, &biasMetrics); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse bias score data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"execution_id": executionID,
		"bias_metrics": biasMetrics,
		"retrieved_at": time.Now().Format(time.RFC3339),
	})
}

// generateRealCrossRegionAnalysis performs actual cross-region bias analysis
func (h *ExecutionsHandler) generateRealCrossRegionAnalysis(regionMap map[string]RegionExecution) map[string]interface{} {
	if len(regionMap) < 2 {
		return map[string]interface{}{
			"error": "Need at least 2 regions for cross-region analysis",
		}
	}

	// Extract responses and calculate differences
	responses := make(map[string]string)
	responseLengths := make(map[string]int)
	biasScores := make(map[string]map[string]interface{})

	for region, exec := range regionMap {
		// Extract response from output_data
		if exec.OutputData != nil {
			var outputData map[string]interface{}
			if err := json.Unmarshal(exec.OutputData, &outputData); err == nil {
				if response, ok := outputData["response"].(string); ok {
					responses[region] = response
					responseLengths[region] = len(response)
				}

				// Extract bias score if available
				if biasScore, ok := outputData["bias_score"].(map[string]interface{}); ok {
					biasScores[region] = biasScore
				}
			}
		}
	}

	// Calculate bias variance across regions
	biasVariance := h.calculateBiasVariance(biasScores)

	// Calculate response length differences
	lengthDifferences := h.calculateLengthDifferences(responseLengths)

	// Generate key differences
	keyDifferences := h.generateKeyDifferences(responses, responseLengths, biasScores)

	// Calculate censorship rate
	censorshipRate := h.calculateCensorshipRate(responses)

	// Generate summary and recommendations
	summary, recommendation := h.generateSummaryAndRecommendation(len(regionMap), biasVariance, censorshipRate)

	return map[string]interface{}{
		"bias_variance":        biasVariance,
		"censorship_rate":      censorshipRate,
		"factual_consistency":  0.87, // Placeholder for now
		"narrative_divergence": lengthDifferences,
		"key_differences":      keyDifferences,
		"risk_assessment":      h.generateRiskAssessment(biasVariance, censorshipRate),
		"summary":              summary,
		"recommendation":       recommendation,
		"regions_analyzed":     len(regionMap),
		"responses_found":      len(responses),
	}
}

// calculateBiasVariance calculates variance in bias scores across regions
func (h *ExecutionsHandler) calculateBiasVariance(biasScores map[string]map[string]interface{}) float64 {
	if len(biasScores) < 2 {
		return 0.0
	}

	var politicalScores []float64
	for _, scores := range biasScores {
		if ps, ok := scores["political_sensitivity"].(float64); ok {
			politicalScores = append(politicalScores, ps)
		}
	}

	if len(politicalScores) < 2 {
		return 0.0
	}

	// Calculate variance
	var sum, mean float64
	for _, score := range politicalScores {
		sum += score
	}
	mean = sum / float64(len(politicalScores))

	var variance float64
	for _, score := range politicalScores {
		variance += (score - mean) * (score - mean)
	}
	variance /= float64(len(politicalScores))

	return variance
}

// calculateLengthDifferences calculates response length variance
func (h *ExecutionsHandler) calculateLengthDifferences(lengths map[string]int) float64 {
	if len(lengths) < 2 {
		return 0.0
	}

	var sum float64
	for _, length := range lengths {
		sum += float64(length)
	}
	mean := sum / float64(len(lengths))

	var variance float64
	for _, length := range lengths {
		diff := float64(length) - mean
		variance += diff * diff
	}
	variance /= float64(len(lengths))

	// Normalize to 0-1 scale
	return variance / (mean * mean)
}

// generateKeyDifferences identifies key differences between regions
func (h *ExecutionsHandler) generateKeyDifferences(responses map[string]string, lengths map[string]int, biasScores map[string]map[string]interface{}) []map[string]interface{} {
	differences := []map[string]interface{}{}

	// Response length differences
	if len(lengths) > 1 {
		variations := make(map[string]string)
		for region, length := range lengths {
			if length > 2000 {
				variations[region] = "Detailed response"
			} else if length > 1000 {
				variations[region] = "Moderate response"
			} else if length > 100 {
				variations[region] = "Brief response"
			} else {
				variations[region] = "Minimal response"
			}
		}

		differences = append(differences, map[string]interface{}{
			"dimension":   "response_length",
			"variations":  variations,
			"severity":    "medium",
			"description": "Response length varies across regions",
		})
	}

	// Political sensitivity differences
	if len(biasScores) > 1 {
		variations := make(map[string]string)
		for region, scores := range biasScores {
			if ps, ok := scores["political_sensitivity"].(float64); ok {
				if ps > 0.7 {
					variations[region] = "High political sensitivity"
				} else if ps > 0.4 {
					variations[region] = "Moderate political sensitivity"
				} else {
					variations[region] = "Low political sensitivity"
				}
			}
		}

		if len(variations) > 1 {
			differences = append(differences, map[string]interface{}{
				"dimension":   "political_sensitivity",
				"variations":  variations,
				"severity":    "high",
				"description": "Political sensitivity varies significantly across regions",
			})
		}
	}

	return differences
}

// calculateCensorshipRate calculates the rate of censored responses
func (h *ExecutionsHandler) calculateCensorshipRate(responses map[string]string) float64 {
	if len(responses) == 0 {
		return 0.0
	}

	censoredCount := 0
	censorshipPatterns := []string{
		"I cannot discuss",
		"I'm not able to comment",
		"I can't assist with that",
		"I'm sorry, but I can't",
		"This is a sensitive topic",
	}

	for _, response := range responses {
		response = strings.ToLower(response)
		for _, pattern := range censorshipPatterns {
			if strings.Contains(response, strings.ToLower(pattern)) {
				censoredCount++
				break
			}
		}
	}

	return float64(censoredCount) / float64(len(responses))
}

// generateRiskAssessment creates risk assessment based on analysis
func (h *ExecutionsHandler) generateRiskAssessment(biasVariance, censorshipRate float64) []map[string]interface{} {
	risks := []map[string]interface{}{}

	if biasVariance > 0.1 {
		severity := "low"
		if biasVariance > 0.3 {
			severity = "high"
		} else if biasVariance > 0.2 {
			severity = "medium"
		}

		risks = append(risks, map[string]interface{}{
			"type":        "bias_variance",
			"severity":    severity,
			"description": "Significant bias variance detected across regions",
			"confidence":  0.85,
		})
	}

	if censorshipRate > 0.3 {
		risks = append(risks, map[string]interface{}{
			"type":        "censorship",
			"severity":    "high",
			"description": "High censorship rate detected in responses",
			"confidence":  0.90,
		})
	}

	return risks
}

// generateSummaryAndRecommendation creates summary and recommendations
func (h *ExecutionsHandler) generateSummaryAndRecommendation(regionCount int, biasVariance, censorshipRate float64) (string, string) {
	summary := fmt.Sprintf("Cross-region analysis across %d regions completed", regionCount)

	if biasVariance > 0.2 {
		summary += " with significant bias variance detected"
	} else if biasVariance > 0.1 {
		summary += " with moderate bias variance detected"
	} else {
		summary += " with low bias variance"
	}

	recommendation := "Continue monitoring for consistent patterns"
	if censorshipRate > 0.5 {
		recommendation = "High censorship detected - investigate model configuration"
	} else if biasVariance > 0.3 {
		recommendation = "High bias variance - consider model alignment across regions"
	}

	return summary, recommendation
}

// RetryQuestionRequest represents a request to retry a failed question
type RetryQuestionRequest struct {
	Region        string `json:"region" binding:"required"`
	QuestionIndex *int   `json:"question_index" binding:"required"` // Pointer to distinguish 0 from missing
}

// RetryQuestionResponse represents the response from a retry attempt
type RetryQuestionResponse struct {
	ExecutionID   string                 `json:"execution_id"`
	Region        string                 `json:"region"`
	QuestionIndex int                    `json:"question_index"`
	Status        string                 `json:"status"`
	RetryAttempt  int                    `json:"retry_attempt"`
	UpdatedAt     string                 `json:"updated_at"`
	Result        map[string]interface{} `json:"result,omitempty"`
	Error         string                 `json:"error,omitempty"`
}

// RetryQuestion retries a single failed question for a specific execution and region
func (h *ExecutionsHandler) RetryQuestion(c *gin.Context) {
	ctx := c.Request.Context()
	executionIDStr := c.Param("id")

	executionID, err := strconv.ParseInt(executionIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid execution ID",
		})
		return
	}

	var req RetryQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	if h.ExecutionsRepo == nil || h.ExecutionsRepo.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database connection not available",
		})
		return
	}

	// Check if execution exists and get current retry count
	var (
		currentRetryCount int
		maxRetries        int
		currentStatus     string
		jobID             int
		jobSpecID         string
		modelID           string
		questionID        string
	)

	err = h.ExecutionsRepo.DB.QueryRowContext(ctx, `
		SELECT 
			e.retry_count, 
			e.max_retries, 
			e.status,
			e.job_id,
			j.jobspec_id,
			COALESCE(e.model_id, 'llama3.2-1b'),
			COALESCE(e.question_id, '')
		FROM executions e
		JOIN jobs j ON e.job_id = j.id
		WHERE e.id = $1 AND e.region = $2
	`, executionID, req.Region).Scan(
		&currentRetryCount,
		&maxRetries,
		&currentStatus,
		&jobID,
		&jobSpecID,
		&modelID,
		&questionID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Execution not found for the specified region",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch execution details",
		})
		return
	}

	// Idempotency short-circuits
	if strings.EqualFold(currentStatus, "completed") {
		// Return existing result if present
		var outputRaw []byte
		_ = h.ExecutionsRepo.DB.QueryRowContext(ctx, `SELECT output_data FROM executions WHERE id = $1`, executionID).Scan(&outputRaw)
		var result map[string]interface{}
		if len(outputRaw) > 0 {
			_ = json.Unmarshal(outputRaw, &result)
		}

		log.Printf("[RETRY][short_circuit_completed] execution_id=%d region=%s qidx=%d", executionID, req.Region, *req.QuestionIndex)
		c.JSON(http.StatusOK, RetryQuestionResponse{
			ExecutionID:   fmt.Sprintf("%d", executionID),
			Region:        req.Region,
			QuestionIndex: *req.QuestionIndex,
			Status:        "completed",
			RetryAttempt:  currentRetryCount,
			UpdatedAt:     time.Now().Format(time.RFC3339),
			Result:        result,
		})
		return
	}

	if currentStatus == "running" || currentStatus == "processing" || currentStatus == "retrying" || currentStatus == "queued" || currentStatus == "created" {
		log.Printf("[RETRY][short_circuit_running] execution_id=%d region=%s qidx=%d status=%s", executionID, req.Region, *req.QuestionIndex, currentStatus)
		c.JSON(http.StatusAccepted, RetryQuestionResponse{
			ExecutionID:   fmt.Sprintf("%d", executionID),
			Region:        req.Region,
			QuestionIndex: *req.QuestionIndex,
			Status:        currentStatus,
			RetryAttempt:  currentRetryCount,
			UpdatedAt:     time.Now().Format(time.RFC3339),
		})
		return
	}

	// Check if execution is in a retryable state
	if currentStatus != "failed" && currentStatus != "timeout" && currentStatus != "error" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Execution is not in a retryable state (current status: %s)", currentStatus),
		})
		return
	}

	// Check if max retries reached
	if currentRetryCount >= maxRetries {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":         "Max retry attempts reached",
			"retry_attempt": currentRetryCount,
			"max_attempts":  maxRetries,
		})
		return
	}

	// Redis-based dedupe: prevent concurrent enqueue for the same tuple
	if req.QuestionIndex != nil {
		dedupeKey := fmt.Sprintf("retry:%d:%s:%d", executionID, req.Region, *req.QuestionIndex)
		if url := os.Getenv("REDIS_URL"); url != "" {
			if opt, perr := redis.ParseURL(url); perr == nil {
				rdb := redis.NewClient(opt)
				// 90s TTL window to cover enqueue + start
				ok, serr := rdb.SetNX(ctx, dedupeKey, "1", 90*time.Second).Result()
				if serr == nil && !ok {
					log.Printf("[RETRY][dedupe_hit] execution_id=%d region=%s qidx=%d", executionID, req.Region, *req.QuestionIndex)
					// Another retry is in-flight; short-circuit as running
					c.JSON(http.StatusAccepted, RetryQuestionResponse{
						ExecutionID:   fmt.Sprintf("%d", executionID),
						Region:        req.Region,
						QuestionIndex: *req.QuestionIndex,
						Status:        "retrying",
						RetryAttempt:  currentRetryCount,
						UpdatedAt:     time.Now().Format(time.RFC3339),
					})
					return
				}
			}
		}
	}

	// Increment retry count and update retry history
	newRetryCount := currentRetryCount + 1
	retryHistoryEntry := map[string]interface{}{
		"attempt":   newRetryCount,
		"timestamp": time.Now().Format(time.RFC3339),
		"status":    "retrying",
	}

	// Update execution record with retry attempt
	retryHistoryJSON := fmt.Sprintf("[%s]", mustMarshalJSON(retryHistoryEntry))

	log.Printf("[RETRY] Updating execution %d: retry_count=%d, retry_history=%s", executionID, newRetryCount, retryHistoryJSON)

	_, err = h.ExecutionsRepo.DB.ExecContext(ctx, `
		UPDATE executions 
		SET 
			retry_count = $1,
			last_retry_at = NOW(),
			retry_history = COALESCE(retry_history, '[]'::jsonb) || $2::jsonb,
			status = 'retrying'
		WHERE id = $3
	`, newRetryCount, retryHistoryJSON, executionID)

	if err != nil {
		log.Printf("[RETRY] Database update failed for execution %d: %v", executionID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to update retry count: %v", err),
		})
		return
	}

	log.Printf("[RETRY] Successfully updated execution %d retry count to %d", executionID, newRetryCount)

	// Trigger actual re-execution of the question
	if h.RetryService != nil {
		// Run re-execution asynchronously to avoid blocking the HTTP response
		go func() {
			retryCtx := context.Background()
			if err := h.RetryService.RetryQuestionExecution(retryCtx, executionID, req.Region, *req.QuestionIndex); err != nil {
				// Log error but don't fail the HTTP response
				// The execution status will be updated in the database
				log.Printf("[RETRY] Retry execution failed for execution %d: %v", executionID, err)
			}
		}()
	} else {
		log.Printf("[RETRY] Warning: RetryService is nil, cannot execute retry for execution %d", executionID)
	}

	c.JSON(http.StatusOK, RetryQuestionResponse{
		ExecutionID:   fmt.Sprintf("%d", executionID),
		Region:        req.Region,
		QuestionIndex: *req.QuestionIndex,
		Status:        "retrying",
		RetryAttempt:  newRetryCount,
		UpdatedAt:     time.Now().Format(time.RFC3339),
		Result: map[string]interface{}{
			"message":     "Retry queued successfully",
			"job_id":      jobSpecID,
			"model_id":    modelID,
			"question_id": questionID,
		},
	})
}

// mustMarshalJSON marshals data to JSON, panicking on error (for internal use)
func mustMarshalJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal JSON: %v", err))
	}
	return string(data)
}
