package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
)

// ExecutionsHandler handles execution-related API endpoints
type ExecutionsHandler struct {
	ExecutionsRepo *store.ExecutionsRepo
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
            (e.receipt_data IS NOT NULL) AS has_receipt
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
        ID          int64  `json:"id"`
        JobID       string `json:"job_id"`
        Status      string `json:"status"`
        Region      string `json:"region"`
        ProviderID  string `json:"provider_id"`
        StartedAt   string `json:"started_at"`
        CompletedAt string `json:"completed_at"`
        CreatedAt   string `json:"created_at"`
        HasReceipt  bool   `json:"has_receipt"`
    }

    var list []Exec
    for rows.Next() {
        var e Exec
        var startedAt, completedAt, createdAt interface{}
        if err := rows.Scan(&e.ID, &e.JobID, &e.Status, &e.Region, &e.ProviderID, &startedAt, &completedAt, &createdAt, &e.HasReceipt); err != nil {
            continue
        }
        if t, ok := startedAt.(time.Time); ok { e.StartedAt = t.Format(time.RFC3339) }
        if t, ok := completedAt.(time.Time); ok { e.CompletedAt = t.Format(time.RFC3339) }
        if t, ok := createdAt.(time.Time); ok { e.CreatedAt = t.Format(time.RFC3339) }
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
        id int64
        jobSpecID string
        status string
        region string
        providerID string
        startedAt, completedAt, createdAt interface{}
        outputData, receiptData []byte
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
        "id": id,
        "job_id": jobSpecID,
        "status": status,
        "region": region,
        "provider_id": providerID,
        "started_at": startedStr,
        "completed_at": completedStr,
        "created_at": createdStr,
        "output": output,
        "receipt": receipt,
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
			e.receipt_data
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
		ID          int64  `json:"id"`
		JobID       string `json:"job_id"`
		Status      string `json:"status"`
		Region      string `json:"region"`
		ProviderID  string `json:"provider_id"`
		StartedAt   string `json:"started_at"`
		CompletedAt string `json:"completed_at"`
		CreatedAt   string `json:"created_at"`
		ReceiptID   string `json:"receipt_id"`
	}

	var executions []ExecutionSummary
	for rows.Next() {
		var exec ExecutionSummary
		var receiptData []byte
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

	type RegionExecution struct {
		ID          int64  `json:"id"`
		Region      string `json:"region"`
		ProviderID  string `json:"provider_id"`
		Status      string `json:"status"`
		StartedAt   string `json:"started_at"`
		CompletedAt string `json:"completed_at"`
		Output      any    `json:"output"`
	}

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
			exec.StartedAt = t.Format(time.RFC3339)
		}
		if t, ok := completedAt.(time.Time); ok {
			exec.CompletedAt = t.Format(time.RFC3339)
		}
		
		// Parse JSON data
		if len(outputData) > 0 {
			_ = json.Unmarshal(outputData, &exec.Output)
		}
		
		executions = append(executions, exec)
		regionMap[exec.Region] = exec
	}
	
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Generate mock analysis for now - this would be replaced with real analysis
	analysis := map[string]interface{}{
		"bias_variance": 0.23,
		"censorship_rate": 0.15,
		"factual_consistency": 0.87,
		"narrative_divergence": 0.31,
		"key_differences": []map[string]interface{}{
			{
				"dimension": "response_length",
				"variations": map[string]string{
					"US": "Detailed response",
					"EU": "Moderate response", 
					"ASIA": "Brief response",
				},
				"severity": "medium",
				"description": "Response length varies significantly across regions",
			},
		},
		"risk_assessment": []map[string]interface{}{
			{
				"type": "bias",
				"severity": "medium",
				"description": "Moderate regional bias detected in response patterns",
				"regions": []string{"US", "ASIA"},
				"confidence": 0.75,
			},
		},
		"summary": "Cross-region analysis shows moderate differences in AI responses",
		"recommendation": "Monitor for consistent patterns across multiple executions",
	}
	
	c.JSON(http.StatusOK, gin.H{
		"job_id": jobID,
		"total_regions": len(regionMap),
		"executions": executions,
		"analysis": analysis,
		"generated_at": time.Now().Format(time.RFC3339),
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
		"job_id": jobID,
		"regions": regionResults,
		"total_regions": len(regionResults),
		"generated_at": time.Now().Format(time.RFC3339),
	})
}
