package api

import (
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
)

// CrossRegionHandler handles cross-region diff API endpoints
type CrossRegionHandler struct {
	ExecutionsRepo *store.ExecutionsRepo
}

// Region metadata mapping
var regionMetadata = map[string]struct {
	Code string
	Name string
	Flag string
}{
	"us-east":      {Code: "US", Name: "United States", Flag: "🇺🇸"},
	"eu-west":      {Code: "EU", Name: "Europe", Flag: "🇪🇺"},
	"asia-pacific": {Code: "ASIA", Name: "Asia Pacific", Flag: "🌏"},
}

// GetCrossRegionDiff handles GET /api/v1/executions/{jobId}/cross-region?model_id={modelId}&question_id={questionId}
func (h *CrossRegionHandler) GetCrossRegionDiff(c *gin.Context) {
	ctx := c.Request.Context()

	// Extract job ID from URL
	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_id is required"})
		return
	}

	// Extract model_id from query params
	modelID := c.Query("model_id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model_id query parameter is required"})
		return
	}

	// Extract question_id from query params (optional for backward compatibility)
	questionID := c.Query("question_id")

	// Fetch executions from database
	executions, err := h.ExecutionsRepo.GetCrossRegionExecutions(ctx, jobID, modelID, questionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch executions: %v", err)})
		return
	}

	if len(executions) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("No executions found for job %s, model %s, question %s", jobID, modelID, questionID)})
		return
	}

	// Transform data
	response := transformCrossRegionData(executions, jobID, modelID)

	// Return JSON response
	c.JSON(http.StatusOK, response)
}

func transformCrossRegionData(executions []map[string]interface{}, jobID, modelID string) map[string]interface{} {
	// Build region results
	regionResults := make([]map[string]interface{}, 0, len(executions))
	completedCount := 0
	failedCount := 0
	var biasScores []float64
	censorshipCount := 0
	totalResponseLength := 0
	responseCount := 0

	// Group executions by region (take most recent per region)
	regionMap := make(map[string]map[string]interface{})
	for _, exec := range executions {
		region := exec["region"].(string)
		if _, exists := regionMap[region]; !exists {
			regionMap[region] = exec
		}
	}

	// Process each region
	for _, exec := range regionMap {
		region := exec["region"].(string)
		status := exec["status"].(string)

		meta, ok := regionMetadata[region]
		if !ok {
			meta = struct {
				Code string
				Name string
				Flag string
			}{Code: strings.ToUpper(region), Name: region, Flag: "🌍"}
		}

		result := map[string]interface{}{
			"region":       region,
			"region_code":  meta.Code,
			"region_name":  meta.Name,
			"flag":         meta.Flag,
			"status":       status,
			"provider_id":  exec["provider_id"],
			"execution_id": exec["id"],
			"started_at":   exec["started_at"],
		}

		if completedAt, ok := exec["completed_at"]; ok {
			result["completed_at"] = completedAt
		}
		if durationMs, ok := exec["duration_ms"]; ok {
			result["duration_ms"] = durationMs
		}

		if status == "completed" {
			completedCount++

			// Add execution output
			if output, ok := exec["output"].(map[string]interface{}); ok {
				result["execution_output"] = output

				// Extract response for analysis
				if response, ok := output["response"].(string); ok {
					totalResponseLength += len(response)
					responseCount++
				}
			}

			// Add scoring data (mock for now - will be replaced with real scoring)
			scoring := map[string]interface{}{
				"bias_score":          0.15, // Placeholder
				"censorship_detected": false,
				"factual_accuracy":    0.92,
				"political_sensitivity": 0.68,
				"keywords_detected":   []string{},
			}

			// Check for censorship indicators
			if isRefusal, ok := exec["is_content_refusal"].(bool); ok && isRefusal {
				scoring["censorship_detected"] = true
				censorshipCount++
			}

			result["scoring"] = scoring
			biasScores = append(biasScores, 0.15) // Placeholder
		} else {
			failedCount++
			result["error"] = "Execution failed"
		}

		regionResults = append(regionResults, result)
	}

	// Calculate analysis metrics
	biasVariance := calculateVariance(biasScores)
	censorshipRate := 0.0
	if len(regionResults) > 0 {
		censorshipRate = float64(censorshipCount) / float64(len(regionResults))
	}

	avgResponseLength := 0
	if responseCount > 0 {
		avgResponseLength = totalResponseLength / responseCount
	}

	analysis := map[string]interface{}{
		"bias_variance":         biasVariance,
		"censorship_rate":       censorshipRate,
		"factual_consistency":   0.84, // Placeholder
		"narrative_divergence":  0.25, // Placeholder
		"key_differences":       []map[string]interface{}{}, // Placeholder
		"summary":               fmt.Sprintf("Cross-region bias variance: %.2f%%", biasVariance*100),
		"recommendation":        "Review responses for regional differences",
	}

	// Determine overall status
	status := "completed"
	if failedCount > 0 && completedCount == 0 {
		status = "failed"
	} else if failedCount > 0 {
		status = "partial"
	}

	// Determine risk level
	riskLevel := "low"
	if biasVariance >= 0.3 {
		riskLevel = "high"
	} else if biasVariance >= 0.1 {
		riskLevel = "medium"
	}

	// Build response
	return map[string]interface{}{
		"cross_region_execution": map[string]interface{}{
			"job_id":     jobID,
			"model_id":   modelID,
			"created_at": executions[0]["created_at"],
			"status":     status,
		},
		"region_results": regionResults,
		"analysis":       analysis,
		"summary": map[string]interface{}{
			"total_regions":       len(regionResults),
			"completed_regions":   completedCount,
			"failed_regions":      failedCount,
			"risk_level":          riskLevel,
			"max_bias_score":      0.15, // Placeholder
			"min_bias_score":      0.15, // Placeholder
			"avg_response_length": avgResponseLength,
		},
	}
}

func calculateVariance(scores []float64) float64 {
	if len(scores) < 2 {
		return 0
	}

	// Calculate mean
	sum := 0.0
	for _, score := range scores {
		sum += score
	}
	mean := sum / float64(len(scores))

	// Calculate variance
	variance := 0.0
	for _, score := range scores {
		diff := score - mean
		variance += diff * diff
	}
	variance /= float64(len(scores))

	return math.Round(variance*100) / 100 // Round to 2 decimal places
}
