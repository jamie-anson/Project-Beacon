package processors

import (
	"context"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// JobSpecProcessor handles JobSpec parsing and validation
type JobSpecProcessor struct {
	validator *models.JobSpecValidator
}

// NewJobSpecProcessor creates a new JobSpec processor
func NewJobSpecProcessor() *JobSpecProcessor {
	return &JobSpecProcessor{
		validator: models.NewJobSpecValidator(),
	}
}

// ParseRequest extracts and parses JobSpec from HTTP request
func (p *JobSpecProcessor) ParseRequest(c *gin.Context) (*models.JobSpec, []byte, error) {
	l := logging.FromContext(c.Request.Context())
	
	// Read raw body for signature verification fallback
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		l.Error().Err(err).Msg("failed to read request body")
		return nil, nil, fmt.Errorf("invalid request body: %w", err)
	}
	
	// Reset body for JSON binding
	c.Request.Body = io.NopCloser(bytes.NewReader(raw))
	
	var spec models.JobSpec
	if err := c.ShouldBindJSON(&spec); err != nil {
		l.Error().Err(err).Msg("invalid JSON")
		return nil, nil, fmt.Errorf("invalid JSON: %w", err)
	}
	
	l.Info().Str("job_id", spec.ID).Msg("JobSpec parsed successfully")
	return &spec, raw, nil
}

// ValidateBiasDetectionQuestions enforces questions requirement for bias-detection v1
func (p *JobSpecProcessor) ValidateBiasDetectionQuestions(spec *models.JobSpec, rawBody []byte) error {
	l := logging.FromContext(context.TODO()) // TODO: Pass context through
	
	// Only apply to v1 bias-detection
	if !strings.EqualFold(spec.Version, "v1") || !strings.Contains(strings.ToLower(spec.Benchmark.Name), "bias") {
		return nil
	}
	
	// Parse raw JSON to check questions field
	var tmp map[string]interface{}
	if err := json.Unmarshal(rawBody, &tmp); err != nil {
		return fmt.Errorf("failed to parse raw JSON: %w", err)
	}
	
	qv, ok := tmp["questions"]
	if !ok {
		l.Warn().Str("job_id", spec.ID).Msg("rejecting: missing questions for bias-detection v1")
		return fmt.Errorf("questions are required for bias-detection v1 jobspec")
	}
	
	arr, isArr := qv.([]interface{})
	if !isArr || len(arr) == 0 {
		l.Warn().Str("job_id", spec.ID).Msg("rejecting: empty questions for bias-detection v1")
		return fmt.Errorf("questions must be a non-empty array for bias-detection v1 jobspec")
	}
	
	l.Info().Str("job_id", spec.ID).Int("questions_count", len(arr)).Msg("questions validation passed for bias-detection v1")
	return nil
}

// LogQuestions logs the presence of questions for debugging
func (p *JobSpecProcessor) LogQuestions(spec *models.JobSpec) {
	l := logging.FromContext(context.TODO()) // TODO: Pass context through
	
	if len(spec.Questions) > 0 {
		l.Info().Str("job_id", spec.ID).Int("questions_present", len(spec.Questions)).Strs("questions", spec.Questions).Msg("JobSpec questions parsed successfully")
	} else {
		l.Info().Str("job_id", spec.ID).Msg("JobSpec has no questions field")
	}
}

// ValidateJobSpec performs core JobSpec validation
func (p *JobSpecProcessor) ValidateJobSpec(spec *models.JobSpec) error {
	l := logging.FromContext(context.TODO()) // TODO: Pass context through
	l.Info().Str("job_id_before_validation", spec.ID).Msg("JobSpec ID before validation")
	// Perform structural validation only here; signature verification is handled in SecurityPipeline
	if err := spec.Validate(); err != nil {
		l.Info().Str("job_id_after_validation_error", spec.ID).Msg("JobSpec ID after validation error")
		return err
	}
	l.Info().Str("job_id_after_validation_success", spec.ID).Msg("JobSpec ID after successful validation")
	return nil
}

// NormalizeModelsFromMetadata normalizes models from metadata after signature verification
func (p *JobSpecProcessor) NormalizeModelsFromMetadata(spec *models.JobSpec) {
	l := logging.FromContext(context.TODO()) // TODO: Pass context through
	
	// Skip if Models already populated
	if len(spec.Models) > 0 {
		l.Info().Str("job_id", spec.ID).Int("existing_models", len(spec.Models)).Msg("models already populated, skipping normalization")
		return
	}
	
	// Check for models in metadata
	raw, ok := spec.Metadata["models"]
	if !ok {
		l.Info().Str("job_id", spec.ID).Msg("no models in metadata, will use single-model execution")
		return
	}
	
	// Handle both array of strings and array of objects with id fields
	switch vv := raw.(type) {
	case []interface{}:
		for _, v := range vv {
			switch t := v.(type) {
			case string:
				// Simple string model ID
				modelSpec := models.ModelSpec{
					ID:       t,
					Name:     t,
					Provider: "hybrid",
					Regions:  spec.Constraints.Regions,
				}
				spec.Models = append(spec.Models, modelSpec)
				l.Info().Str("job_id", spec.ID).Str("model_id", t).Msg("normalized string model")
				
			case map[string]interface{}:
				// Object with id and optional name
				if id, ok := t["id"].(string); ok && id != "" {
					name, _ := t["name"].(string)
					if name == "" {
						name = id // fallback to id if name not provided
					}
					
					modelSpec := models.ModelSpec{
						ID:       id,
						Name:     name,
						Provider: "hybrid",
						Regions:  spec.Constraints.Regions,
					}
					spec.Models = append(spec.Models, modelSpec)
					l.Info().Str("job_id", spec.ID).Str("model_id", id).Str("model_name", name).Msg("normalized object model")
				}
			}
		}
	default:
		l.Warn().Str("job_id", spec.ID).Interface("models_type", vv).Msg("unsupported models format in metadata")
	}
	
	l.Info().Str("job_id", spec.ID).Int("normalized_models", len(spec.Models)).Msg("model normalization completed")
}

// ProcessJobSpec performs complete JobSpec processing pipeline
func (p *JobSpecProcessor) ProcessJobSpec(c *gin.Context) (*models.JobSpec, []byte, error) {
	// Parse request
	spec, rawBody, err := p.ParseRequest(c)
	if err != nil {
		return nil, nil, err
	}
	
	// Validate bias detection questions
	if err := p.ValidateBiasDetectionQuestions(spec, rawBody); err != nil {
		return nil, nil, err
	}
	
	// Log questions for debugging
	p.LogQuestions(spec)
	
	// Validate JobSpec (this includes ID generation if missing)
	if err := p.ValidateJobSpec(spec); err != nil {
		return nil, nil, err
	}
	
	// IMPORTANT: Normalize models AFTER signature verification and validation
	// This ensures signature verification uses the original payload without modifications
	p.NormalizeModelsFromMetadata(spec)
	
	return spec, rawBody, nil
}
