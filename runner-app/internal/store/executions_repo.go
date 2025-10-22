package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// ExecutionsRepo provides persistence operations for executions
type ExecutionsRepo struct {
	DB *sql.DB
}

// ListExecutionsByJobSpecIDPaginated returns executions for a JobSpec with limit/offset
func (r *ExecutionsRepo) ListExecutionsByJobSpecIDPaginated(ctx context.Context, jobspecID string, limit, offset int) ([]*models.Receipt, error) {
    if r.DB == nil {
        return nil, errors.New("database connection is nil")
    }
    if limit <= 0 {
        limit = 20
    }
    if offset < 0 {
        offset = 0
    }

    rows, err := r.DB.QueryContext(ctx, `
        SELECT e.receipt_data
        FROM executions e
        JOIN jobs j ON e.job_id = j.id
        WHERE j.jobspec_id = $1 AND e.receipt_data IS NOT NULL
        ORDER BY e.created_at DESC
        LIMIT $2 OFFSET $3
    `, jobspecID, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to query executions: %w", err)
    }
    defer rows.Close()

    var receipts []*models.Receipt
    for rows.Next() {
        var receiptJSON []byte
        if err := rows.Scan(&receiptJSON); err != nil {
            return nil, fmt.Errorf("failed to scan execution row: %w", err)
        }
        var receipt models.Receipt
        if err := json.Unmarshal(receiptJSON, &receipt); err != nil {
            return nil, fmt.Errorf("failed to unmarshal receipt: %w", err)
        }
        receipts = append(receipts, &receipt)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating execution rows: %w", err)
    }
    return receipts, nil
}

// GetReceiptByJobSpecID returns the latest Receipt for a JobSpec
func (r *ExecutionsRepo) GetReceiptByJobSpecID(ctx context.Context, jobspecID string) (*models.Receipt, error) {
	if r.DB == nil {
		return nil, errors.New("database connection is nil")
	}

	var receiptJSON []byte
	row := r.DB.QueryRowContext(ctx, `
		SELECT e.receipt_data
		FROM executions e
		JOIN jobs j ON e.job_id = j.id
		WHERE j.jobspec_id = $1 AND e.receipt_data IS NOT NULL
		ORDER BY e.created_at DESC
		LIMIT 1
	`, jobspecID)

	err := row.Scan(&receiptJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no receipt found for job: %s", jobspecID)
		}
		return nil, fmt.Errorf("failed to get receipt: %w", err)
	}

	var receipt models.Receipt
	if err := json.Unmarshal(receiptJSON, &receipt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal receipt: %w", err)
	}

	return &receipt, nil
}

// ListExecutionsByJobSpecID returns all executions for a JobSpec
func (r *ExecutionsRepo) ListExecutionsByJobSpecID(ctx context.Context, jobspecID string) ([]*models.Receipt, error) {
	if r.DB == nil {
		return nil, errors.New("database connection is nil")
	}

	rows, err := r.DB.QueryContext(ctx, `
		SELECT e.receipt_data
		FROM executions e
		JOIN jobs j ON e.job_id = j.id
		WHERE j.jobspec_id = $1 AND e.receipt_data IS NOT NULL
		ORDER BY e.created_at DESC
	`, jobspecID)

	if err != nil {
		return nil, fmt.Errorf("failed to query executions: %w", err)
	}
	defer rows.Close()

	var receipts []*models.Receipt
	for rows.Next() {
		var receiptJSON []byte
		if err := rows.Scan(&receiptJSON); err != nil {
			return nil, fmt.Errorf("failed to scan execution row: %w", err)
		}

		var receipt models.Receipt
		if err := json.Unmarshal(receiptJSON, &receipt); err != nil {
			return nil, fmt.Errorf("failed to unmarshal receipt: %w", err)
		}

		receipts = append(receipts, &receipt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating execution rows: %w", err)
	}

	return receipts, nil
}

// UpdateExecutionStatus updates the status of an execution
func (r *ExecutionsRepo) UpdateExecutionStatus(ctx context.Context, executionID int64, status string) error {
	if r.DB == nil {
		return errors.New("database connection is nil")
	}

	_, err := r.DB.ExecContext(ctx, `
		UPDATE executions 
		SET status = $1 
		WHERE id = $2
	`, status, executionID)

	return err
}

// GetLatestByJobSpecID returns the most recent execution for a given JobSpec ID (legacy method)
func (r *ExecutionsRepo) GetLatestByJobSpecID(
	ctx context.Context,
	jobspecID string,
) (
	id int64,
	providerID string,
	region string,
	status string,
	startedAt sql.NullTime,
	completedAt sql.NullTime,
	outputJSON []byte,
	receiptJSON []byte,
	createdAt time.Time,
	err error,
) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT e.id, e.provider_id, e.region, e.status, e.started_at, e.completed_at, e.output_data, e.receipt_data, e.created_at
		FROM executions e
		JOIN jobs j ON e.job_id = j.id
		WHERE j.jobspec_id = $1
		ORDER BY e.created_at DESC
		LIMIT 1
	`, jobspecID)
	err = row.Scan(&id, &providerID, &region, &status, &startedAt, &completedAt, &outputJSON, &receiptJSON, &createdAt)
	return
}

func NewExecutionsRepo(db *sql.DB) *ExecutionsRepo {
    return &ExecutionsRepo{DB: db}
}

// UpdateRegionVerification upserts region verification fields for an execution.
// Pass sql.Null* with Valid=false to skip updating a particular field.
// Note: requires columns to exist in the DB schema; compile-safe prior to migration.
func (r *ExecutionsRepo) UpdateRegionVerification(
    ctx context.Context,
    executionID int64,
    regionClaimed sql.NullString,
    regionObserved sql.NullString,
    regionVerified sql.NullBool,
    verificationMethod sql.NullString,
    evidenceRef sql.NullString,
) error {
    if r.DB == nil {
        return errors.New("database connection is nil")
    }
    _, err := r.DB.ExecContext(ctx, `
        UPDATE executions SET
            region_claimed = COALESCE($2, region_claimed),
            region_observed = COALESCE($3, region_observed),
            region_verified = COALESCE($4, region_verified),
            verification_method = COALESCE($5, verification_method),
            preflight_evidence_ref = COALESCE($6, preflight_evidence_ref)
        WHERE id = $1
    `,
        executionID,
        nullStringOrNil(regionClaimed),
        nullStringOrNil(regionObserved),
        nullBoolOrNil(regionVerified),
        nullStringOrNil(verificationMethod),
        nullStringOrNil(evidenceRef),
    )
    return err
}

// Helpers to pass NULL when sql.Null* is invalid.
func nullStringOrNil(v sql.NullString) interface{} {
    if v.Valid {
        return v.String
    }
    return nil
}

func nullBoolOrNil(v sql.NullBool) interface{} {
    if v.Valid {
        return v.Bool
    }
    return nil
}

// CreateExecution creates a new execution record from a Receipt
func (r *ExecutionsRepo) CreateExecution(ctx context.Context, jobspecID string, receipt *models.Receipt) (int64, error) {
	if r.DB == nil {
		return 0, errors.New("database connection is nil")
	}

	// Serialize output and receipt data
	outputJSON, err := json.Marshal(receipt.Output)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal output: %w", err)
	}

	receiptJSON, err := json.Marshal(receipt)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal receipt: %w", err)
	}

	row := r.DB.QueryRowContext(ctx, `
		INSERT INTO executions (job_id, provider_id, region, status, started_at, completed_at, output_data, receipt_data)
		VALUES ((SELECT id FROM jobs WHERE jobspec_id = $1), $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, jobspecID, receipt.ExecutionDetails.ProviderID, receipt.ExecutionDetails.Region, 
		receipt.ExecutionDetails.Status, receipt.ExecutionDetails.StartedAt, 
		receipt.ExecutionDetails.CompletedAt, outputJSON, receiptJSON)

	var id int64
	if err := row.Scan(&id); err != nil {
		return 0, fmt.Errorf("failed to insert execution: %w", err)
	}
	return id, nil
}

// InsertExecution inserts an execution row associated to a job via jobspec_id lookup (legacy method)
func (r *ExecutionsRepo) InsertExecution(
	ctx context.Context,
	jobspecID string,
	providerID string,
	region string,
	status string,
	startedAt time.Time,
	completedAt time.Time,
	outputJSON []byte,
	receiptJSON []byte,
) (int64, error) {
	return r.InsertExecutionWithModel(ctx, jobspecID, providerID, region, status, startedAt, completedAt, outputJSON, receiptJSON, "llama3.2-1b")
}

// InsertExecutionWithModel inserts an execution row with model_id support
func (r *ExecutionsRepo) InsertExecutionWithModel(
	ctx context.Context,
	jobspecID string,
	providerID string,
	region string,
	status string,
	startedAt time.Time,
	completedAt time.Time,
	outputJSON []byte,
	receiptJSON []byte,
	modelID string,
) (int64, error) {
	// Delegate to InsertExecutionWithModelAndQuestion with empty question_id
	return r.InsertExecutionWithModelAndQuestion(ctx, jobspecID, providerID, region, status, startedAt, completedAt, outputJSON, receiptJSON, modelID, "")
}

// InsertExecutionWithModelAndQuestion inserts an execution row with model_id and question_id support
func (r *ExecutionsRepo) InsertExecutionWithModelAndQuestion(
	ctx context.Context,
	jobspecID string,
	providerID string,
	region string,
	status string,
	startedAt time.Time,
	completedAt time.Time,
	outputJSON []byte,
	receiptJSON []byte,
	modelID string,
	questionID string,
) (int64, error) {
	// First, verify the job exists
	var jobID int64
	err := r.DB.QueryRowContext(ctx, `SELECT id FROM jobs WHERE jobspec_id = $1`, jobspecID).Scan(&jobID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("job not found in database for jobspec_id: %s", jobspecID)
		}
		return 0, fmt.Errorf("failed to lookup job: %w", err)
	}

	// Always insert with question_id column (NULL if empty for backward compatibility)
	var questionIDPtr *string
	if questionID != "" {
		questionIDPtr = &questionID
	}
	
	row := r.DB.QueryRowContext(ctx, `
		INSERT INTO executions (job_id, provider_id, region, status, started_at, completed_at, output_data, receipt_data, model_id, question_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`, jobID, providerID, region, status, startedAt, completedAt, outputJSON, receiptJSON, modelID, questionIDPtr, startedAt)
	
	var id int64
	if err := row.Scan(&id); err != nil {
		return 0, fmt.Errorf("failed to insert execution: %w", err)
	}
	return id, nil
}

// GetCrossRegionExecutions fetches all executions for a job, model, and optionally question across regions
// Returns executions ordered by region for cross-region comparison
// If questionID is empty, it will return all executions for the job+model (for backward compatibility)
func (r *ExecutionsRepo) GetCrossRegionExecutions(ctx context.Context, jobID, modelID, questionID string) ([]map[string]interface{}, error) {
	if r.DB == nil {
		return nil, errors.New("database connection is nil")
	}

	var query string
	var args []interface{}

	if questionID != "" {
		// Filter by question_id if provided
		query = `
			SELECT 
				e.id,
				e.job_id,
				e.region,
				e.status,
				e.provider_id,
				e.model_id,
				e.question_id,
				e.output_data,
				e.started_at,
				e.completed_at,
				e.created_at,
				e.response_length,
				e.response_classification,
				e.is_substantive,
				e.is_content_refusal,
				e.is_technical_error,
				j.jobspec_id
			FROM executions e
			JOIN jobs j ON e.job_id = j.id
			WHERE j.jobspec_id = $1 AND e.model_id = $2 AND e.question_id = $3
			ORDER BY e.region ASC, e.created_at DESC
		`
		args = []interface{}{jobID, modelID, questionID}
	} else {
		// Backward compatibility: return all executions for job+model if no question_id
		query = `
			SELECT 
				e.id,
				e.job_id,
				e.region,
				e.status,
				e.provider_id,
				e.model_id,
				e.question_id,
				e.output_data,
				e.started_at,
				e.completed_at,
				e.created_at,
				e.response_length,
				e.response_classification,
				e.is_substantive,
				e.is_content_refusal,
				e.is_technical_error,
				j.jobspec_id
			FROM executions e
			JOIN jobs j ON e.job_id = j.id
			WHERE j.jobspec_id = $1 AND e.model_id = $2
			ORDER BY e.region ASC, e.created_at DESC
		`
		args = []interface{}{jobID, modelID}
	}

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query cross-region executions: %w", err)
	}
	defer rows.Close()

	var executions []map[string]interface{}
	for rows.Next() {
		var (
			id                     int64
			jobIDVal               int64
			region                 string
			status                 string
			providerID             string
			modelIDVal             string
			questionID             sql.NullString
			outputData             []byte
			startedAt              time.Time
			completedAt            sql.NullTime
			createdAt              time.Time
			responseLength         sql.NullInt64
			responseClassification sql.NullString
			isSubstantive          sql.NullBool
			isContentRefusal       sql.NullBool
			isTechnicalError       sql.NullBool
			jobspecID              string
		)

		err := rows.Scan(
			&id, &jobIDVal, &region, &status, &providerID, &modelIDVal, &questionID,
			&outputData, &startedAt, &completedAt, &createdAt,
			&responseLength, &responseClassification,
			&isSubstantive, &isContentRefusal, &isTechnicalError,
			&jobspecID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution row: %w", err)
		}

		// Parse output_data JSON
		var output map[string]interface{}
		if len(outputData) > 0 {
			if err := json.Unmarshal(outputData, &output); err != nil {
				return nil, fmt.Errorf("failed to unmarshal output_data: %w", err)
			}
		}

		execution := map[string]interface{}{
			"id":          id,
			"job_id":      jobspecID,
			"region":      region,
			"status":      status,
			"provider_id": providerID,
			"model_id":    modelIDVal,
			"output":      output,
			"started_at":  startedAt,
			"created_at":  createdAt,
		}

		if questionID.Valid {
			execution["question_id"] = questionID.String
		}
		if completedAt.Valid {
			execution["completed_at"] = completedAt.Time
			// Calculate duration in milliseconds
			duration := completedAt.Time.Sub(startedAt).Milliseconds()
			execution["duration_ms"] = duration
		}
		if responseLength.Valid {
			execution["response_length"] = responseLength.Int64
		}
		if responseClassification.Valid {
			execution["response_classification"] = responseClassification.String
		}
		if isSubstantive.Valid {
			execution["is_substantive"] = isSubstantive.Bool
		}
		if isContentRefusal.Valid {
			execution["is_content_refusal"] = isContentRefusal.Bool
		}
		if isTechnicalError.Valid {
			execution["is_technical_error"] = isTechnicalError.Bool
		}

		executions = append(executions, execution)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating execution rows: %w", err)
	}

	return executions, nil
}

// InsertExecutionWithClassification inserts an execution row with response classification support
func (r *ExecutionsRepo) InsertExecutionWithClassification(
	ctx context.Context,
	jobspecID string,
	providerID string,
	region string,
	status string,
	startedAt time.Time,
	completedAt time.Time,
	outputJSON []byte,
	receiptJSON []byte,
	modelID string,
	isSubstantive bool,
	isContentRefusal bool,
	isTechnicalError bool,
	responseClassification string,
	responseLength int,
	systemPrompt string,
) (int64, error) {
	// First, verify the job exists
	var jobID int64
	err := r.DB.QueryRowContext(ctx, `SELECT id FROM jobs WHERE jobspec_id = $1`, jobspecID).Scan(&jobID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("job not found in database for jobspec_id: %s", jobspecID)
		}
		return 0, fmt.Errorf("failed to lookup job: %w", err)
	}

	// Insert execution with classification fields
	row := r.DB.QueryRowContext(ctx, `
		INSERT INTO executions (
			job_id, provider_id, region, status, started_at, completed_at, 
			output_data, receipt_data, model_id,
			is_substantive, is_content_refusal, is_technical_error,
			response_classification, response_length, system_prompt
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id
	`, jobID, providerID, region, status, startedAt, completedAt, 
		outputJSON, receiptJSON, modelID,
		isSubstantive, isContentRefusal, isTechnicalError,
		responseClassification, responseLength, systemPrompt)
	
	var id int64
	if err := row.Scan(&id); err != nil {
		return 0, fmt.Errorf("failed to insert execution with classification: %w", err)
	}
	return id, nil
}
