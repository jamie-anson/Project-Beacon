package security

import (
	"context"
	"fmt"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/config"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// TrustEvaluator handles trust evaluation for JobSpecs
type TrustEvaluator struct {
	cfg *config.Config
}

// NewTrustEvaluator creates a new trust evaluator
func NewTrustEvaluator(cfg *config.Config) *TrustEvaluator {
	return &TrustEvaluator{
		cfg: cfg,
	}
}

// EvaluateJobSpecTrust evaluates trust status for a JobSpec
func (t *TrustEvaluator) EvaluateJobSpecTrust(ctx context.Context, spec *models.JobSpec) error {
	l := logging.FromContext(ctx)
	
	// Skip trust evaluation if no public key
	if spec.PublicKey == "" {
		return nil
	}
	
	// Load trusted keys registry
	reg, err := config.GetTrustedKeys()
	if err != nil {
		l.Warn().Err(err).Msg("trusted-keys: load error")
		return nil // Don't fail on registry load error
	}
	
	if reg == nil {
		return nil // No registry configured
	}
	
	// Evaluate key trust
	entry := reg.ByPublicKey(spec.PublicKey)
	status, reason := config.EvaluateKeyTrust(entry, time.Now().UTC())
	
	// Log trust evaluation result
	if entry != nil {
		l.Info().Str("trust_status", status).Str("reason", reason).Str("kid", entry.KID).Msg("trusted-keys: evaluation")
	} else {
		l.Info().Str("trust_status", status).Str("reason", reason).Msg("trusted-keys: evaluation")
	}
	
	// Enforce trust if enabled
	if t.cfg != nil && t.cfg.TrustEnforce {
		if status != "trusted" {
			return fmt.Errorf("untrusted signing key: %s", reason)
		}
	}
	
	return nil
}

// TrustViolationError represents a trust violation with specific error code
type TrustViolationError struct {
	Status string
	Reason string
}

func (e *TrustViolationError) Error() string {
	return fmt.Sprintf("untrusted signing key: %s", e.Reason)
}

func (e *TrustViolationError) ErrorCode() string {
	return "trust_violation:" + e.Status
}

// EvaluateJobSpecTrustWithError evaluates trust and returns structured error
func (t *TrustEvaluator) EvaluateJobSpecTrustWithError(ctx context.Context, spec *models.JobSpec) error {
	l := logging.FromContext(ctx)
	
	// Skip trust evaluation if no public key
	if spec.PublicKey == "" {
		return nil
	}
	
	// Load trusted keys registry
	reg, err := config.GetTrustedKeys()
	if err != nil {
		l.Warn().Err(err).Msg("trusted-keys: load error")
		return nil // Don't fail on registry load error
	}
	
	if reg == nil {
		return nil // No registry configured
	}
	
	// Evaluate key trust
	entry := reg.ByPublicKey(spec.PublicKey)
	status, reason := config.EvaluateKeyTrust(entry, time.Now().UTC())
	
	// Log trust evaluation result
	if entry != nil {
		l.Info().Str("trust_status", status).Str("reason", reason).Str("kid", entry.KID).Msg("trusted-keys: evaluation")
	} else {
		l.Info().Str("trust_status", status).Str("reason", reason).Msg("trusted-keys: evaluation")
	}
	
	// Enforce trust if enabled
	if t.cfg != nil && t.cfg.TrustEnforce {
		if status != "trusted" {
			return &TrustViolationError{
				Status: status,
				Reason: reason,
			}
		}
	}
	
	return nil
}
