package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
)

// TestJobWithExecutions tests our Phase 3 fix for returning execution summaries
func TestJobWithExecutions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	// Mock job lookup
	jobRows := sqlmock.NewRows([]string{"jobspec_data", "status", "created_at", "updated_at"}).
		AddRow(`{"id":"test-job","version":"1.0"}`, "completed", time.Now(), time.Now())
	mock.ExpectQuery(`SELECT (.+) FROM jobs WHERE jobspec_id`).
		WithArgs("test-job").
		WillReturnRows(jobRows)

	// Mock execution summaries query (our Phase 3 fix)
	execRows := sqlmock.NewRows([]string{"id", "status", "region", "provider_id", "started_at", "completed_at", "created_at"}).
		AddRow(1, "completed", "us-east", "modal-us-east", time.Now(), time.Now(), time.Now()).
		AddRow(2, "completed", "eu-west", "modal-eu-west", time.Now(), time.Now(), time.Now())
	
	mock.ExpectQuery(`SELECT (.+) FROM executions e JOIN jobs j`).
		WithArgs("test-job", 20, 0).
		WillReturnRows(execRows)

	// Setup JobsService with database
	jobsService := &service.JobsService{
		DB:             db,
		ExecutionsRepo: &store.ExecutionsRepo{DB: db},
	}

	// Setup router
	router := gin.New()
	handler := &JobsHandler{
		jobsService: jobsService,
	}
	router.GET("/api/v1/jobs/:id", handler.GetJob)

	// Test request with include=executions
	req := httptest.NewRequest("GET", "/api/v1/jobs/test-job?include=executions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify executions are included
	executions, ok := response["executions"].([]interface{})
	if !ok {
		t.Fatalf("executions field missing or wrong type")
	}

	if len(executions) != 2 {
		t.Fatalf("expected 2 executions, got %d", len(executions))
	}

	// Verify execution structure
	exec1 := executions[0].(map[string]interface{})
	if exec1["status"] != "completed" {
		t.Errorf("expected status=completed, got %v", exec1["status"])
	}
	if exec1["region"] != "us-east" {
		t.Errorf("expected region=us-east, got %v", exec1["region"])
	}
	if exec1["provider_id"] != "modal-us-east" {
		t.Errorf("expected provider_id=modal-us-east, got %v", exec1["provider_id"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestExecutionsWithOutput tests our Phase 2 fix for returning execution output data
func TestExecutionsWithOutput(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	// Mock executions query with output data (our Phase 2 fix)
	outputData := `{"response":"Test LLM response","provider":"modal-us-east"}`
	execRows := sqlmock.NewRows([]string{"id", "job_id", "status", "region", "provider_id", "started_at", "completed_at", "created_at", "receipt_id", "output_data"}).
		AddRow(1, "test-job", "completed", "us-east", "modal-us-east", time.Now(), time.Now(), time.Now(), "", outputData)
	
	mock.ExpectQuery(`SELECT (.+) FROM executions e JOIN jobs j`).
		WithArgs("test-job", 10, 0).
		WillReturnRows(execRows)

	// Setup handler
	handler := &ExecutionsHandler{
		ExecutionsRepo: &store.ExecutionsRepo{DB: db},
	}

	// Setup router
	router := gin.New()
	router.GET("/api/v1/executions", handler.ListExecutions)

	// Test request
	req := httptest.NewRequest("GET", "/api/v1/executions?job_id=test-job", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify executions with output are included
	executions, ok := response["executions"].([]interface{})
	if !ok {
		t.Fatalf("executions field missing or wrong type")
	}

	if len(executions) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(executions))
	}

	// Verify execution has output field
	exec1 := executions[0].(map[string]interface{})
	output, ok := exec1["output"].(map[string]interface{})
	if !ok {
		t.Fatalf("output field missing or wrong type")
	}

	if output["response"] != "Test LLM response" {
		t.Errorf("expected response='Test LLM response', got %v", output["response"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
