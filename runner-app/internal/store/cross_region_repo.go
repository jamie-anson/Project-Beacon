package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// CrossRegionRepo handles database operations for cross-region executions
type CrossRegionRepo struct {
	db *sql.DB
}

// NewCrossRegionRepo creates a new cross-region repository
func NewCrossRegionRepo(db *sql.DB) *CrossRegionRepo {
	return &CrossRegionRepo{db: db}
}

// CrossRegionExecution represents a cross-region execution record
type CrossRegionExecution struct {
	ID                 string    `json:"id"`
	JobSpecID          string    `json:"jobspec_id"`
	TotalRegions       int       `json:"total_regions"`
	SuccessCount       int       `json:"success_count"`
	FailureCount       int       `json:"failure_count"`
	MinRegionsRequired int       `json:"min_regions_required"`
	MinSuccessRate     float64   `json:"min_success_rate"`
	Status             string    `json:"status"`
	StartedAt          time.Time `json:"started_at"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	DurationMs         *int64    `json:"duration_ms,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// RegionResultRecord represents a region execution result record
type RegionResultRecord struct {
	ID                      string                 `json:"id"`
	CrossRegionExecutionID  string                 `json:"cross_region_execution_id"`
	Region                  string                 `json:"region"`
	ProviderID              *string                `json:"provider_id,omitempty"`
	ProviderInfo            map[string]interface{} `json:"provider_info,omitempty"`
	Status                  string                 `json:"status"`
	StartedAt               time.Time              `json:"started_at"`
	CompletedAt             *time.Time             `json:"completed_at,omitempty"`
	DurationMs              *int64                 `json:"duration_ms,omitempty"`
	ExecutionOutput         map[string]interface{} `json:"execution_output,omitempty"`
	ErrorMessage            *string                `json:"error_message,omitempty"`
	Scoring                 map[string]interface{} `json:"scoring,omitempty"`
	Metadata                map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt               time.Time              `json:"created_at"`
	UpdatedAt               time.Time              `json:"updated_at"`
}

// CrossRegionAnalysisRecord represents a cross-region analysis record
type CrossRegionAnalysisRecord struct {
	ID                      string                   `json:"id"`
	CrossRegionExecutionID  string                   `json:"cross_region_execution_id"`
	BiasVariance            *float64                 `json:"bias_variance,omitempty"`
	CensorshipRate          *float64                 `json:"censorship_rate,omitempty"`
	FactualConsistency      *float64                 `json:"factual_consistency,omitempty"`
	NarrativeDivergence     *float64                 `json:"narrative_divergence,omitempty"`
	KeyDifferences          []models.KeyDifference   `json:"key_differences,omitempty"`
	RiskAssessment          []models.RiskAssessment  `json:"risk_assessment,omitempty"`
	Summary                 *string                  `json:"summary,omitempty"`
	Recommendation          *string                  `json:"recommendation,omitempty"`
	CreatedAt               time.Time                `json:"created_at"`
	UpdatedAt               time.Time                `json:"updated_at"`
}

// CreateCrossRegionExecution creates a new cross-region execution record
func (r *CrossRegionRepo) CreateCrossRegionExecution(ctx context.Context, jobSpecID string, totalRegions, minRegions int, minSuccessRate float64) (*CrossRegionExecution, error) {
	id := uuid.New().String()
	
	query := `
		INSERT INTO cross_region_executions (
			id, jobspec_id, total_regions, min_regions_required, min_success_rate, status, started_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, jobspec_id, total_regions, success_count, failure_count, min_regions_required, 
				  min_success_rate, status, started_at, completed_at, duration_ms, created_at, updated_at
	`
	
	var exec CrossRegionExecution
	var completedAt sql.NullTime
	var durationMs sql.NullInt64
	
	err := r.db.QueryRowContext(ctx, query, id, jobSpecID, totalRegions, minRegions, minSuccessRate, "running", time.Now()).Scan(
		&exec.ID, &exec.JobSpecID, &exec.TotalRegions, &exec.SuccessCount, &exec.FailureCount,
		&exec.MinRegionsRequired, &exec.MinSuccessRate, &exec.Status, &exec.StartedAt,
		&completedAt, &durationMs, &exec.CreatedAt, &exec.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cross-region execution: %w", err)
	}
	
	if completedAt.Valid {
		exec.CompletedAt = &completedAt.Time
	}
	if durationMs.Valid {
		exec.DurationMs = &durationMs.Int64
	}
	
	return &exec, nil
}

// CreateRegionResult creates a new region result record
func (r *CrossRegionRepo) CreateRegionResult(ctx context.Context, crossRegionExecID, region string, startedAt time.Time) (*RegionResultRecord, error) {
	id := uuid.New().String()
	
	query := `
		INSERT INTO region_results (
			id, cross_region_execution_id, region, status, started_at
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id, cross_region_execution_id, region, provider_id, status, started_at, 
				  completed_at, duration_ms, error_message, created_at, updated_at
	`
	
	var result RegionResultRecord
	var providerID sql.NullString
	var completedAt sql.NullTime
	var durationMs sql.NullInt64
	var errorMessage sql.NullString
	
	err := r.db.QueryRowContext(ctx, query, id, crossRegionExecID, region, "running", startedAt).Scan(
		&result.ID, &result.CrossRegionExecutionID, &result.Region, &providerID, &result.Status,
		&result.StartedAt, &completedAt, &durationMs, &errorMessage, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create region result: %w", err)
	}
	
	if providerID.Valid {
		result.ProviderID = &providerID.String
	}
	if completedAt.Valid {
		result.CompletedAt = &completedAt.Time
	}
	if durationMs.Valid {
		result.DurationMs = &durationMs.Int64
	}
	if errorMessage.Valid {
		result.ErrorMessage = &errorMessage.String
	}
	
	return &result, nil
}

// UpdateRegionResult updates a region result with completion data
func (r *CrossRegionRepo) UpdateRegionResult(ctx context.Context, id string, status string, completedAt time.Time, durationMs int64, providerID *string, output map[string]interface{}, errorMsg *string, scoring map[string]interface{}, metadata map[string]interface{}) error {
	var outputJSON, scoringJSON, metadataJSON []byte
	var err error
	
	if output != nil {
		outputJSON, err = json.Marshal(output)
		if err != nil {
			return fmt.Errorf("failed to marshal output: %w", err)
		}
	}
	
	if scoring != nil {
		scoringJSON, err = json.Marshal(scoring)
		if err != nil {
			return fmt.Errorf("failed to marshal scoring: %w", err)
		}
	}
	
	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}
	
	query := `
		UPDATE region_results 
		SET status = $2, completed_at = $3, duration_ms = $4, provider_id = $5, 
			execution_output = $6, error_message = $7, scoring = $8, metadata = $9
		WHERE id = $1
	`
	
	_, err = r.db.ExecContext(ctx, query, id, status, completedAt, durationMs, providerID, outputJSON, errorMsg, scoringJSON, metadataJSON)
	if err != nil {
		return fmt.Errorf("failed to update region result: %w", err)
	}
	
	return nil
}

// UpdateCrossRegionExecutionStatus updates the status and counts of a cross-region execution
func (r *CrossRegionRepo) UpdateCrossRegionExecutionStatus(ctx context.Context, id string, status string, successCount, failureCount int, completedAt *time.Time, durationMs *int64) error {
	query := `
		UPDATE cross_region_executions 
		SET status = $2, success_count = $3, failure_count = $4, completed_at = $5, duration_ms = $6
		WHERE id = $1
	`
	
	_, err := r.db.ExecContext(ctx, query, id, status, successCount, failureCount, completedAt, durationMs)
	if err != nil {
		return fmt.Errorf("failed to update cross-region execution status: %w", err)
	}
	
	return nil
}

// CreateCrossRegionAnalysis creates a new cross-region analysis record
func (r *CrossRegionRepo) CreateCrossRegionAnalysis(ctx context.Context, crossRegionExecID string, analysis *models.CrossRegionAnalysis) (*CrossRegionAnalysisRecord, error) {
	id := uuid.New().String()
	
	keyDifferencesJSON, err := json.Marshal(analysis.KeyDifferences)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal key differences: %w", err)
	}
	
	riskAssessmentJSON, err := json.Marshal(analysis.RiskAssessment)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal risk assessment: %w", err)
	}
	
	query := `
		INSERT INTO cross_region_analyses (
			id, cross_region_execution_id, bias_variance, censorship_rate, factual_consistency,
			narrative_divergence, key_differences, risk_assessment, summary, recommendation
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, cross_region_execution_id, bias_variance, censorship_rate, factual_consistency,
				  narrative_divergence, summary, recommendation, created_at, updated_at
	`
	
	var record CrossRegionAnalysisRecord
	var biasVariance, censorshipRate, factualConsistency, narrativeDivergence sql.NullFloat64
	var summary, recommendation sql.NullString
	
	err = r.db.QueryRowContext(ctx, query, id, crossRegionExecID, analysis.BiasVariance, analysis.CensorshipRate,
		analysis.FactualConsistency, analysis.NarrativeDivergence, keyDifferencesJSON, riskAssessmentJSON,
		analysis.Summary, analysis.Recommendation).Scan(
		&record.ID, &record.CrossRegionExecutionID, &biasVariance, &censorshipRate, &factualConsistency,
		&narrativeDivergence, &summary, &recommendation, &record.CreatedAt, &record.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cross-region analysis: %w", err)
	}
	
	if biasVariance.Valid {
		record.BiasVariance = &biasVariance.Float64
	}
	if censorshipRate.Valid {
		record.CensorshipRate = &censorshipRate.Float64
	}
	if factualConsistency.Valid {
		record.FactualConsistency = &factualConsistency.Float64
	}
	if narrativeDivergence.Valid {
		record.NarrativeDivergence = &narrativeDivergence.Float64
	}
	if summary.Valid {
		record.Summary = &summary.String
	}
	if recommendation.Valid {
		record.Recommendation = &recommendation.String
	}
	
	record.KeyDifferences = analysis.KeyDifferences
	record.RiskAssessment = analysis.RiskAssessment
	
	return &record, nil
}

// GetCrossRegionExecution retrieves a cross-region execution by ID
func (r *CrossRegionRepo) GetCrossRegionExecution(ctx context.Context, id string) (*CrossRegionExecution, error) {
	query := `
		SELECT id, jobspec_id, total_regions, success_count, failure_count, min_regions_required,
			   min_success_rate, status, started_at, completed_at, duration_ms, created_at, updated_at
		FROM cross_region_executions 
		WHERE id = $1
	`
	
	var exec CrossRegionExecution
	var completedAt sql.NullTime
	var durationMs sql.NullInt64
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&exec.ID, &exec.JobSpecID, &exec.TotalRegions, &exec.SuccessCount, &exec.FailureCount,
		&exec.MinRegionsRequired, &exec.MinSuccessRate, &exec.Status, &exec.StartedAt,
		&completedAt, &durationMs, &exec.CreatedAt, &exec.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("cross-region execution not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get cross-region execution: %w", err)
	}
	
	if completedAt.Valid {
		exec.CompletedAt = &completedAt.Time
	}
	if durationMs.Valid {
		exec.DurationMs = &durationMs.Int64
	}
	
	return &exec, nil
}

// GetRegionResults retrieves all region results for a cross-region execution
func (r *CrossRegionRepo) GetRegionResults(ctx context.Context, crossRegionExecID string) ([]*RegionResultRecord, error) {
	query := `
		SELECT id, cross_region_execution_id, region, provider_id, provider_info, status,
			   started_at, completed_at, duration_ms, execution_output, error_message,
			   scoring, metadata, created_at, updated_at
		FROM region_results 
		WHERE cross_region_execution_id = $1
		ORDER BY started_at
	`
	
	rows, err := r.db.QueryContext(ctx, query, crossRegionExecID)
	if err != nil {
		return nil, fmt.Errorf("failed to query region results: %w", err)
	}
	defer rows.Close()
	
	var results []*RegionResultRecord
	for rows.Next() {
		var result RegionResultRecord
		var providerID sql.NullString
		var providerInfoJSON, outputJSON, scoringJSON, metadataJSON sql.NullString
		var completedAt sql.NullTime
		var durationMs sql.NullInt64
		var errorMessage sql.NullString
		
		err := rows.Scan(
			&result.ID, &result.CrossRegionExecutionID, &result.Region, &providerID, &providerInfoJSON,
			&result.Status, &result.StartedAt, &completedAt, &durationMs, &outputJSON, &errorMessage,
			&scoringJSON, &metadataJSON, &result.CreatedAt, &result.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan region result: %w", err)
		}
		
		if providerID.Valid {
			result.ProviderID = &providerID.String
		}
		if completedAt.Valid {
			result.CompletedAt = &completedAt.Time
		}
		if durationMs.Valid {
			result.DurationMs = &durationMs.Int64
		}
		if errorMessage.Valid {
			result.ErrorMessage = &errorMessage.String
		}
		
		// Unmarshal JSON fields
		if providerInfoJSON.Valid {
			json.Unmarshal([]byte(providerInfoJSON.String), &result.ProviderInfo)
		}
		if outputJSON.Valid {
			json.Unmarshal([]byte(outputJSON.String), &result.ExecutionOutput)
		}
		if scoringJSON.Valid {
			json.Unmarshal([]byte(scoringJSON.String), &result.Scoring)
		}
		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &result.Metadata)
		}
		
		results = append(results, &result)
	}
	
	return results, nil
}
