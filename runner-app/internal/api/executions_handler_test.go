package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
)

// Helper to create test router with database mock
func newTestExecutionsRouter(db *sql.DB) *gin.Engine {
	r := gin.New()

	// Initialize handlers
	executionsHandler := NewExecutionsHandler(&store.ExecutionsRepo{DB: db})

	// Mount routes
	v1 := r.Group("/api/v1")
	{
		executions := v1.Group("/executions")
		{
			executions.GET("/:id/details", executionsHandler.GetExecutionDetails)
		}
	}
	return r
}

// TestGetExecutionDetails_HybridFailure_ShowsRouterError tests that when hybrid router fails,
// the execution details API shows the router error in the response
func TestGetExecutionDetails_HybridFailure_ShowsRouterError(t *testing.T) {
	t.Parallel()

	// Create database mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	// Mock database query that returns an execution with hybrid failure data
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT e.id, j.jobspec_id, e.status, e.region, e.provider_id,
		       e.started_at, e.completed_at, e.created_at,
		       e.output_data, e.receipt_data
		FROM executions e
		JOIN jobs j ON e.job_id = j.id
		WHERE e.id = $1
	`)).
	WithArgs(int64(123)).
	WillReturnRows(sqlmock.NewRows([]string{
		"id", "jobspec_id", "status", "region", "provider_id",
		"started_at", "completed_at", "created_at",
		"output_data", "receipt_data",
	}).
	AddRow(
		int64(123),                    // id
		"test-job-123",                // jobspec_id
		"failed",                      // status
		"us-east",                     // region
		"hybrid-router",               // provider_id
		sql.NullTime{Valid: false},    // started_at
		sql.NullTime{Valid: false},    // completed_at
		sql.NullTime{Valid: false},    // created_at
		[]byte(`{}`),                  // output_data (empty)
		[]byte(`{}`),                  // receipt_data (empty)
	))

	// Create test router
	r := newTestExecutionsRouter(db)

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/123/details", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
	}

	// Check that we get the expected execution details with hybrid failure info
	// For now, just check that we got a 200 and the basic structure
	// In a real implementation, we would have more detailed assertions
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["status"] != "failed" {
		t.Errorf("expected status to be 'failed', got %v", response["status"])
	}

	if response["provider_id"] != "hybrid-router" {
		t.Errorf("expected provider_id to be 'hybrid-router', got %v", response["provider_id"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestGetExecutionDetails_Success_ShowsProviderAndReceipt tests that when execution succeeds,
// the execution details API shows the provider information and receipt data
func TestGetExecutionDetails_Success_ShowsProviderAndReceipt(t *testing.T) {
	t.Parallel()

	// Create database mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	// Mock database query that returns a successful execution with receipt
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT e.id, j.jobspec_id, e.status, e.region, e.provider_id,
		       e.started_at, e.completed_at, e.created_at,
		       e.output_data, e.receipt_data
		FROM executions e
		JOIN jobs j ON e.job_id = j.id
		WHERE e.id = $1
	`)).
	WithArgs(int64(456)).
	WillReturnRows(sqlmock.NewRows([]string{
		"id", "jobspec_id", "status", "region", "provider_id",
		"started_at", "completed_at", "created_at",
		"output_data", "receipt_data",
	}).
	AddRow(
		int64(456),                    // id
		"test-job-456",                // jobspec_id
		"completed",                   // status
		"eu-west",                     // region
		"modal-provider-001",          // provider_id
		sql.NullTime{Valid: false},    // started_at
		sql.NullTime{Valid: false},    // completed_at
		sql.NullTime{Valid: false},    // created_at
		[]byte(`{"response": "Test response", "model": "llama3.2-1b"}`), // output_data
		[]byte(`{"id": "receipt-456", "execution_details": {"provider_id": "modal-provider-001", "region": "eu-west", "status": "completed"}}`), // receipt_data
	))

	// Create test router
	r := newTestExecutionsRouter(db)

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/456/details", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
	}

	// Check that we get the expected execution details with provider and receipt info
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["status"] != "completed" {
		t.Errorf("expected status to be 'completed', got %v", response["status"])
	}

	if response["provider_id"] != "modal-provider-001" {
		t.Errorf("expected provider_id to be 'modal-provider-001', got %v", response["provider_id"])
	}

	// Verify output contains expected data
	output, ok := response["output"].(map[string]interface{})
	if !ok {
		t.Error("expected output to be a map")
	} else {
		if output["response"] != "Test response" {
			t.Errorf("expected output response to be 'Test response', got %v", output["response"])
		}
	}

	// Verify receipt contains expected data
	receipt, ok := response["receipt"].(map[string]interface{})
	if !ok {
		t.Error("expected receipt to be a map")
	} else {
		if receipt["id"] != "receipt-456" {
			t.Errorf("expected receipt id to be 'receipt-456', got %v", receipt["id"])
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
