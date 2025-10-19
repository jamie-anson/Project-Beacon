package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetJobBiasAnalysis(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns bias analysis for valid job ID", func(t *testing.T) {
		// Setup
		mockRepo := &MockCrossRegionRepo{
			GetByJobSpecIDFunc: func(ctx context.Context, jobSpecID string) (*store.CrossRegionExecution, error) {
				return &store.CrossRegionExecution{
					ID:           "exec-123",
					JobSpecID:    jobSpecID,
					TotalRegions: 3,
					Status:       "completed",
				}, nil
			},
			GetCrossRegionAnalysisByExecutionIDFunc: func(ctx context.Context, execID string) (*store.CrossRegionAnalysisRecord, error) {
				biasVariance := 0.68
				censorshipRate := 0.67
				summary := "Cross-region analysis completed with significant findings..."

				return &store.CrossRegionAnalysisRecord{
					ID:                     "analysis-456",
					CrossRegionExecutionID: execID,
					BiasVariance:           &biasVariance,
					CensorshipRate:         &censorshipRate,
					Summary:                &summary,
					KeyDifferences:         []models.KeyDifference{},
					RiskAssessment:         []models.RiskAssessment{},
				}, nil
			},
			GetRegionResultsFunc: func(ctx context.Context, execID string) ([]*store.RegionResultRecord, error) {
				return []*store.RegionResultRecord{
					{
						ID:     "region-1",
						Region: "us_east",
						Scoring: map[string]interface{}{
							"bias_score":            0.15,
							"censorship_detected":   false,
							"political_sensitivity": 0.3,
							"factual_accuracy":      0.85,
						},
					},
					{
						ID:     "region-2",
						Region: "asia_pacific",
						Scoring: map[string]interface{}{
							"bias_score":            0.78,
							"censorship_detected":   true,
							"political_sensitivity": 0.92,
							"factual_accuracy":      0.12,
						},
					},
				}, nil
			},
		}

		handler := &CrossRegionHandlers{
			crossRegionRepo: mockRepo,
		}

		router := gin.New()
		router.GET("/api/v2/jobs/:jobId/bias-analysis", handler.GetJobBiasAnalysis)

		// Execute
		req := httptest.NewRequest("GET", "/api/v2/jobs/test-job-123/bias-analysis", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "test-job-123", response["job_id"])
		assert.Equal(t, "exec-123", response["cross_region_execution_id"])

		// Verify analysis
		analysis, ok := response["analysis"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, 0.68, analysis["bias_variance"])
		assert.Contains(t, analysis["summary"], "Cross-region analysis")

		// Verify region scores
		regionScores, ok := response["region_scores"].(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, regionScores, "us_east")
		assert.Contains(t, regionScores, "asia_pacific")

		usEast := regionScores["us_east"].(map[string]interface{})
		assert.Equal(t, 0.15, usEast["bias_score"])
		assert.Equal(t, false, usEast["censorship_detected"])
	})

	t.Run("returns 404 when job not found", func(t *testing.T) {
		mockRepo := &MockCrossRegionRepo{
			GetByJobSpecIDFunc: func(ctx context.Context, jobSpecID string) (*store.CrossRegionExecution, error) {
				return nil, fmt.Errorf("cross-region execution not found for jobspec_id: %s", jobSpecID)
			},
		}

		handler := &CrossRegionHandlers{
			crossRegionRepo: mockRepo,
		}

		router := gin.New()
		router.GET("/api/v2/jobs/:jobId/bias-analysis", handler.GetJobBiasAnalysis)

		req := httptest.NewRequest("GET", "/api/v2/jobs/nonexistent-job/bias-analysis", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Contains(t, response["error"], "not found")
	})

	t.Run("returns 404 when analysis not available", func(t *testing.T) {
		mockRepo := &MockCrossRegionRepo{
			GetByJobSpecIDFunc: func(ctx context.Context, jobSpecID string) (*store.CrossRegionExecution, error) {
				return &store.CrossRegionExecution{
					ID:        "exec-789",
					JobSpecID: jobSpecID,
				}, nil
			},
			GetCrossRegionAnalysisByExecutionIDFunc: func(ctx context.Context, execID string) (*store.CrossRegionAnalysisRecord, error) {
				return nil, fmt.Errorf("analysis not found for execution_id: %s", execID)
			},
		}

		handler := &CrossRegionHandlers{
			crossRegionRepo: mockRepo,
		}

		router := gin.New()
		router.GET("/api/v2/jobs/:jobId/bias-analysis", handler.GetJobBiasAnalysis)

		req := httptest.NewRequest("GET", "/api/v2/jobs/job-without-analysis/bias-analysis", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Contains(t, response["error"], "Bias analysis not found")
	})

	t.Run("returns 500 when region results fetch fails", func(t *testing.T) {
		mockRepo := &MockCrossRegionRepo{
			GetByJobSpecIDFunc: func(ctx context.Context, jobSpecID string) (*store.CrossRegionExecution, error) {
				return &store.CrossRegionExecution{
					ID:        "exec-999",
					JobSpecID: jobSpecID,
				}, nil
			},
			GetCrossRegionAnalysisByExecutionIDFunc: func(ctx context.Context, execID string) (*store.CrossRegionAnalysisRecord, error) {
				summary := "test"
				return &store.CrossRegionAnalysisRecord{
					ID:      "analysis-999",
					Summary: &summary,
				}, nil
			},
			GetRegionResultsFunc: func(ctx context.Context, execID string) ([]*store.RegionResultRecord, error) {
				return nil, fmt.Errorf("database connection failed")
			},
		}

		handler := &CrossRegionHandlers{
			crossRegionRepo: mockRepo,
		}

		router := gin.New()
		router.GET("/api/v2/jobs/:jobId/bias-analysis", handler.GetJobBiasAnalysis)

		req := httptest.NewRequest("GET", "/api/v2/jobs/test-job/bias-analysis", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("handles regions without scoring data", func(t *testing.T) {
		biasVariance := 0.5
		summary := "test"

		mockRepo := &MockCrossRegionRepo{
			GetByJobSpecIDFunc: func(ctx context.Context, jobSpecID string) (*store.CrossRegionExecution, error) {
				return &store.CrossRegionExecution{
					ID:        "exec-123",
					JobSpecID: jobSpecID,
				}, nil
			},
			GetCrossRegionAnalysisByExecutionIDFunc: func(ctx context.Context, execID string) (*store.CrossRegionAnalysisRecord, error) {
				return &store.CrossRegionAnalysisRecord{
					ID:             "analysis-123",
					BiasVariance:   &biasVariance,
					Summary:        &summary,
					KeyDifferences: []models.KeyDifference{},
					RiskAssessment: []models.RiskAssessment{},
				}, nil
			},
			GetRegionResultsFunc: func(ctx context.Context, execID string) ([]*store.RegionResultRecord, error) {
				return []*store.RegionResultRecord{
					{
						ID:      "region-1",
						Region:  "us_east",
						Scoring: nil, // No scoring data
					},
					{
						ID:     "region-2",
						Region: "eu_west",
						Scoring: map[string]interface{}{
							"bias_score": 0.2,
						},
					},
				}, nil
			},
		}

		handler := &CrossRegionHandlers{
			crossRegionRepo: mockRepo,
		}

		router := gin.New()
		router.GET("/api/v2/jobs/:jobId/bias-analysis", handler.GetJobBiasAnalysis)

		req := httptest.NewRequest("GET", "/api/v2/jobs/test-job/bias-analysis", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		regionScores := response["region_scores"].(map[string]interface{})

		// Should only include region with scoring
		assert.Contains(t, regionScores, "eu_west")
		assert.NotContains(t, regionScores, "us_east")
	})

	t.Run("returns 400 for empty job ID", func(t *testing.T) {
		handler := &CrossRegionHandlers{}

		router := gin.New()
		router.GET("/api/v2/jobs/:jobId/bias-analysis", handler.GetJobBiasAnalysis)

		// Gin won't match this route without a jobId, so test the handler directly
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{}

		handler.GetJobBiasAnalysis(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGetDiffAnalysis_SavesPersistence(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("saves analysis to database after generation", func(t *testing.T) {
		analysisSaved := false

		mockRepo := &MockCrossRegionRepo{
			GetRegionResultsFunc: func(ctx context.Context, execID string) ([]*store.RegionResultRecord, error) {
				return []*store.RegionResultRecord{
					{
						ID:     "region-1",
						Region: "us_east",
						Status: "success",
						ExecutionOutput: map[string]interface{}{
							"response": "test response",
						},
					},
					{
						ID:     "region-2",
						Region: "eu_west",
						Status: "success",
						ExecutionOutput: map[string]interface{}{
							"response": "test response 2",
						},
					},
				}, nil
			},
			CreateCrossRegionAnalysisFunc: func(ctx context.Context, execID string, analysis *models.CrossRegionAnalysis) (*store.CrossRegionAnalysisRecord, error) {
				analysisSaved = true
				return &store.CrossRegionAnalysisRecord{
					ID: "saved-analysis",
				}, nil
			},
		}

		mockDiffEngine := &MockDiffEngine{
			AnalyzeCrossRegionDifferencesFunc: func(ctx context.Context, regionResults map[string]*models.RegionResult) (*models.CrossRegionAnalysis, error) {
				return &models.CrossRegionAnalysis{
					BiasVariance: 0.5,
					Summary:      "Test summary",
				}, nil
			},
		}

		handler := &CrossRegionHandlers{
			crossRegionRepo: mockRepo,
			diffEngine:      mockDiffEngine,
		}

		router := gin.New()
		router.GET("/api/v1/executions/:id/diff-analysis", handler.GetDiffAnalysis)

		req := httptest.NewRequest("GET", "/api/v1/executions/exec-123/diff-analysis", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, analysisSaved, "Analysis should have been saved to database")
	})

	t.Run("continues even if persistence fails", func(t *testing.T) {
		mockRepo := &MockCrossRegionRepo{
			GetRegionResultsFunc: func(ctx context.Context, execID string) ([]*store.RegionResultRecord, error) {
				return []*store.RegionResultRecord{
					{Region: "us_east", Status: "success", ExecutionOutput: map[string]interface{}{"response": "test"}},
					{Region: "eu_west", Status: "success", ExecutionOutput: map[string]interface{}{"response": "test"}},
				}, nil
			},
			CreateCrossRegionAnalysisFunc: func(ctx context.Context, execID string, analysis *models.CrossRegionAnalysis) (*store.CrossRegionAnalysisRecord, error) {
				return nil, fmt.Errorf("database write failed")
			},
		}

		mockDiffEngine := &MockDiffEngine{
			AnalyzeCrossRegionDifferencesFunc: func(ctx context.Context, regionResults map[string]*models.RegionResult) (*models.CrossRegionAnalysis, error) {
				return &models.CrossRegionAnalysis{
					Summary: "Test summary",
				}, nil
			},
		}

		handler := &CrossRegionHandlers{
			crossRegionRepo: mockRepo,
			diffEngine:      mockDiffEngine,
		}

		router := gin.New()
		router.GET("/api/v1/executions/:id/diff-analysis", handler.GetDiffAnalysis)

		req := httptest.NewRequest("GET", "/api/v1/executions/exec-123/diff-analysis", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should still return 200 with analysis
		assert.Equal(t, http.StatusOK, w.Code)

		// Should have warning header
		assert.Contains(t, w.Header().Get("X-Warning"), "Failed to persist analysis")
	})
}

// Mock types for testing

type MockCrossRegionRepo struct {
	CreateCrossRegionExecutionFunc          func(ctx context.Context, jobSpecID string, totalRegions, minRegions int, minSuccessRate float64) (*store.CrossRegionExecution, error)
	UpdateCrossRegionExecutionStatusFunc    func(ctx context.Context, executionID string, status string, successCount, failureCount int, completedAt *time.Time, durationMs *int64) error
	CreateRegionResultFunc                  func(ctx context.Context, executionID string, region string, startedAt time.Time) (*store.RegionResultRecord, error)
	UpdateRegionResultFunc                  func(ctx context.Context, regionResultID string, status string, completedAt time.Time, durationMs int64, providerID *string, output map[string]interface{}, errorMsg *string, scoring map[string]interface{}, metadata map[string]interface{}) error
	CreateCrossRegionAnalysisFunc           func(ctx context.Context, execID string, analysis *models.CrossRegionAnalysis) (*store.CrossRegionAnalysisRecord, error)
	GetCrossRegionExecutionFunc             func(ctx context.Context, executionID string) (*store.CrossRegionExecution, error)
	GetRegionResultsFunc                    func(ctx context.Context, execID string) ([]*store.RegionResultRecord, error)
	GetByJobSpecIDFunc                      func(ctx context.Context, jobSpecID string) (*store.CrossRegionExecution, error)
	GetCrossRegionAnalysisByExecutionIDFunc func(ctx context.Context, execID string) (*store.CrossRegionAnalysisRecord, error)
}

func (m *MockCrossRegionRepo) CreateCrossRegionExecution(ctx context.Context, jobSpecID string, totalRegions, minRegions int, minSuccessRate float64) (*store.CrossRegionExecution, error) {
	if m.CreateCrossRegionExecutionFunc != nil {
		return m.CreateCrossRegionExecutionFunc(ctx, jobSpecID, totalRegions, minRegions, minSuccessRate)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *MockCrossRegionRepo) UpdateCrossRegionExecutionStatus(ctx context.Context, executionID string, status string, successCount, failureCount int, completedAt *time.Time, durationMs *int64) error {
	if m.UpdateCrossRegionExecutionStatusFunc != nil {
		return m.UpdateCrossRegionExecutionStatusFunc(ctx, executionID, status, successCount, failureCount, completedAt, durationMs)
	}
	return nil
}

func (m *MockCrossRegionRepo) CreateRegionResult(ctx context.Context, executionID string, region string, startedAt time.Time) (*store.RegionResultRecord, error) {
	if m.CreateRegionResultFunc != nil {
		return m.CreateRegionResultFunc(ctx, executionID, region, startedAt)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *MockCrossRegionRepo) UpdateRegionResult(ctx context.Context, regionResultID string, status string, completedAt time.Time, durationMs int64, providerID *string, output map[string]interface{}, errorMsg *string, scoring map[string]interface{}, metadata map[string]interface{}) error {
	if m.UpdateRegionResultFunc != nil {
		return m.UpdateRegionResultFunc(ctx, regionResultID, status, completedAt, durationMs, providerID, output, errorMsg, scoring, metadata)
	}
	return nil
}

func (m *MockCrossRegionRepo) GetByJobSpecID(ctx context.Context, jobSpecID string) (*store.CrossRegionExecution, error) {
	if m.GetByJobSpecIDFunc != nil {
		return m.GetByJobSpecIDFunc(ctx, jobSpecID)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *MockCrossRegionRepo) GetCrossRegionExecution(ctx context.Context, executionID string) (*store.CrossRegionExecution, error) {
	if m.GetCrossRegionExecutionFunc != nil {
		return m.GetCrossRegionExecutionFunc(ctx, executionID)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *MockCrossRegionRepo) GetCrossRegionAnalysisByExecutionID(ctx context.Context, execID string) (*store.CrossRegionAnalysisRecord, error) {
	if m.GetCrossRegionAnalysisByExecutionIDFunc != nil {
		return m.GetCrossRegionAnalysisByExecutionIDFunc(ctx, execID)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *MockCrossRegionRepo) GetRegionResults(ctx context.Context, execID string) ([]*store.RegionResultRecord, error) {
	if m.GetRegionResultsFunc != nil {
		return m.GetRegionResultsFunc(ctx, execID)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *MockCrossRegionRepo) CreateCrossRegionAnalysis(ctx context.Context, execID string, analysis *models.CrossRegionAnalysis) (*store.CrossRegionAnalysisRecord, error) {
	if m.CreateCrossRegionAnalysisFunc != nil {
		return m.CreateCrossRegionAnalysisFunc(ctx, execID, analysis)
	}
	return nil, fmt.Errorf("not implemented")
}

type MockDiffEngine struct {
	AnalyzeCrossRegionDifferencesFunc func(ctx context.Context, regionResults map[string]*models.RegionResult) (*models.CrossRegionAnalysis, error)
}

func (m *MockDiffEngine) AnalyzeCrossRegionDifferences(ctx context.Context, regionResults map[string]*models.RegionResult) (*models.CrossRegionAnalysis, error) {
	if m.AnalyzeCrossRegionDifferencesFunc != nil {
		return m.AnalyzeCrossRegionDifferencesFunc(ctx, regionResults)
	}
	return nil, nil
}
