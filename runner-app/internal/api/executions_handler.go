package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
)

// ExecutionsHandler handles execution-related API endpoints
type ExecutionsHandler struct {
	ExecutionsRepo *store.ExecutionsRepo
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
		WHERE e.receipt_data IS NOT NULL
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
			exec.StartedAt = startedAt.(string)
		}
		if completedAt != nil {
			exec.CompletedAt = completedAt.(string)
		}
		if createdAt != nil {
			exec.CreatedAt = createdAt.(string)
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
		WHERE e.receipt_data IS NOT NULL
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
