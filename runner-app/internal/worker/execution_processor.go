package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/analysis"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
	models "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// ExecutionProcessor handles post-execution processing including validation and classification
type ExecutionProcessor struct {
	Repo            *store.ExecutionsRepo
	PromptFormatter *analysis.RegionalPromptFormatter
}

// NewExecutionProcessor creates a new execution processor with default components
func NewExecutionProcessor(repo *store.ExecutionsRepo) *ExecutionProcessor {
	return &ExecutionProcessor{
		Repo:               repo,
		PromptFormatter:    analysis.NewRegionalPromptFormatter(),
	}
}

// ClassifiedExecutionResult contains the complete execution result with classification
type ClassifiedExecutionResult struct {
	ProviderID             string
	Status                 string
	OutputJSON             []byte
	ReceiptJSON            []byte
	ModelID                string
	IsSubstantive          bool
	IsContentRefusal       bool
	IsTechnicalError       bool
	ResponseClassification string
	ResponseLength         int
	SystemPrompt           string
	ValidationErrors       []analysis.ValidationError
}

// ProcessAndStoreExecution validates, classifies, and stores an execution result
func (p *ExecutionProcessor) ProcessAndStoreExecution(
	ctx context.Context,
	spec *models.JobSpec,
	region string,
	providerID string,
	status string,
	startedAt time.Time,
	completedAt time.Time,
	outputJSON []byte,
	receiptJSON []byte,
	modelID string,
) (int64, error) {
	l := logging.FromContext(ctx)

	// Parse output to extract response
	var output map[string]interface{}
	response := ""
	success := status == "completed"
	
	if len(outputJSON) > 0 {
		if err := json.Unmarshal(outputJSON, &output); err == nil {
			if resp, ok := output["response"].(string); ok {
				response = resp
			}
		}
	}

	// Classify response
	classification := analysis.ClassifyResponse(response, success)
	
	l.Info().
		Str("job_id", spec.ID).
		Str("region", region).
		Str("model", modelID).
		Str("classification", classification.Classification).
		Int("response_length", classification.ResponseLength).
		Bool("is_substantive", classification.IsSubstantive).
		Bool("is_refusal", classification.IsContentRefusal).
		Msg("response classified")

	// Extract system prompt from receipt if available
	systemPrompt := ""
	if len(receiptJSON) > 0 {
		var modalOutput analysis.ModalOutput
		if err := json.Unmarshal(receiptJSON, &modalOutput); err == nil {
			systemPrompt = modalOutput.Receipt.Output.SystemPrompt
			
			// Validate output structure
			_, validationErrors := analysis.ValidateModalOutput(receiptJSON)
			if len(validationErrors) > 0 {
				l.Warn().
					Str("job_id", spec.ID).
					Str("region", region).
					Int("error_count", len(validationErrors)).
					Interface("errors", validationErrors).
					Msg("output validation warnings")
			}
		}
	}

	// If system prompt not in receipt, generate expected one for logging
	if systemPrompt == "" && p.PromptFormatter != nil {
		systemPrompt = p.PromptFormatter.GetSystemPrompt(region)
		l.Warn().
			Str("job_id", spec.ID).
			Str("region", region).
			Msg("system prompt not found in receipt, using expected prompt")
	}

	// Store execution with classification
	executionID, err := p.Repo.InsertExecutionWithClassification(
		ctx,
		spec.ID,
		providerID,
		region,
		status,
		startedAt,
		completedAt,
		outputJSON,
		receiptJSON,
		modelID,
		classification.IsSubstantive,
		classification.IsContentRefusal,
		classification.IsTechnicalError,
		classification.Classification,
		classification.ResponseLength,
		systemPrompt,
	)

	if err != nil {
		return 0, fmt.Errorf("failed to store execution: %w", err)
	}

	l.Info().
		Str("job_id", spec.ID).
		Int64("execution_id", executionID).
		Str("classification", classification.Classification).
		Msg("execution stored with classification")

	return executionID, nil
}

// ValidateAndClassify performs validation and classification without storing
// Useful for testing or pre-validation
func (p *ExecutionProcessor) ValidateAndClassify(
	receiptJSON []byte,
	response string,
	success bool,
) (*ClassifiedExecutionResult, error) {
	result := &ClassifiedExecutionResult{
		Status: "completed",
	}

	if !success {
		result.Status = "failed"
	}

	// Classify response
	classification := analysis.ClassifyResponse(response, success)
	result.IsSubstantive = classification.IsSubstantive
	result.IsContentRefusal = classification.IsContentRefusal
	result.IsTechnicalError = classification.IsTechnicalError
	result.ResponseClassification = classification.Classification
	result.ResponseLength = classification.ResponseLength

	// Validate output if receipt provided
	if len(receiptJSON) > 0 {
		modalOutput, validationErrors := analysis.ValidateModalOutput(receiptJSON)
		result.ValidationErrors = validationErrors
		
		if modalOutput != nil {
			result.SystemPrompt = modalOutput.Receipt.Output.SystemPrompt
		}
	}

	return result, nil
}
