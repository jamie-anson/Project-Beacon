package security

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"strings"

	"github.com/jamie-anson/project-beacon-runner/internal/config"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/internal/security"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
	beaconcrypto "github.com/jamie-anson/project-beacon-runner/pkg/crypto"
)

// SecurityPipeline orchestrates all security validations for JobSpecs
type SecurityPipeline struct {
	trustEvaluator   *TrustEvaluator
	replayProtection *security.ReplayProtection
	rateLimiter      *security.RateLimiter
	cfg              *config.Config
}

// NewSecurityPipeline creates a new security pipeline
func NewSecurityPipeline(cfg *config.Config, replayProtection *security.ReplayProtection, rateLimiter *security.RateLimiter) *SecurityPipeline {
	return &SecurityPipeline{
		trustEvaluator:   NewTrustEvaluator(cfg),
		replayProtection: replayProtection,
		rateLimiter:      rateLimiter,
		cfg:              cfg,
	}
}

// ValidationError represents a structured validation error
type ValidationError struct {
	Message   string
	ErrorCode string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// ValidateJobSpec performs comprehensive security validation
func (s *SecurityPipeline) ValidateJobSpec(ctx context.Context, spec *models.JobSpec, rawBody []byte, clientIP string) error {
	l := logging.FromContext(ctx)
	
	// 1. Trust evaluation
	if err := s.trustEvaluator.EvaluateJobSpecTrustWithError(ctx, spec); err != nil {
		if trustErr, ok := err.(*TrustViolationError); ok {
			return &ValidationError{
				Message:   trustErr.Error(),
				ErrorCode: trustErr.ErrorCode(),
			}
		}
		return err
	}
	
	// 2. Rate limiting check for signature failures
	if s.rateLimiter != nil {
		kid := spec.PublicKey
		if err := s.rateLimiter.CheckSignatureFailureRate(ctx, clientIP, kid); err != nil {
			l.Warn().Str("client_ip", clientIP).Str("kid", kid).Msg("rate limit exceeded for signature failures")
			return &ValidationError{
				Message:   "rate limit exceeded",
				ErrorCode: "rate_limit_exceeded",
			}
		}
	}
	
	// 3. Timestamp and nonce validation (if trust enforcement enabled)
	if s.cfg != nil && s.cfg.TrustEnforce {
		if err := s.validateTimestampAndNonce(ctx, spec, clientIP); err != nil {
			return err
		}
	}
	
	// 4. Signature verification
	if s.cfg != nil && s.cfg.SigBypass {
		l.Warn().Str("job_id", spec.ID).Msg("RUNNER_SIG_BYPASS enabled: skipping signature verification (dev only)")
	} else {
		if err := s.validateSignature(ctx, spec, rawBody, clientIP); err != nil {
			return err
		}
	}
	
	// 5. Debug shadow verification (non-fatal)
	s.performShadowVerification(ctx, spec)
	
	return nil
}

// validateTimestampAndNonce validates timestamp and nonce for replay protection
func (s *SecurityPipeline) validateTimestampAndNonce(ctx context.Context, spec *models.JobSpec, clientIP string) error {
	l := logging.FromContext(ctx)
	
	// Check if replay protection is available when required
	if s.cfg.ReplayProtectionEnabled && s.replayProtection == nil {
		return &ValidationError{
			Message:   "replay protection unavailable",
			ErrorCode: "protection_unavailable:replay",
		}
	}
	
	// Validate metadata presence
	if spec.Metadata == nil {
		return &ValidationError{
			Message:   "missing metadata",
			ErrorCode: "missing_field:metadata",
		}
	}
	
	// Validate timestamp
	tsVal, ok := spec.Metadata["timestamp"]
	if !ok || tsVal == nil {
		return &ValidationError{
			Message:   "missing timestamp in metadata",
			ErrorCode: "missing_field:timestamp",
		}
	}
	
	tsStr, isStr := tsVal.(string)
	if !isStr || strings.TrimSpace(tsStr) == "" {
		return &ValidationError{
			Message:   "invalid timestamp format",
			ErrorCode: "invalid_field:timestamp",
		}
	}
	
	// Validate nonce if replay protection enabled
	if s.cfg.ReplayProtectionEnabled && s.replayProtection != nil {
		nonceVal, exists := spec.Metadata["nonce"]
		if exists {
			if nonceStr, ok := nonceVal.(string); ok && nonceStr != "" {
				kid := spec.PublicKey
				if err := s.replayProtection.CheckAndRecordNonce(ctx, kid, nonceStr); err != nil {
					l.Error().Err(err).Str("kid", kid).Str("nonce", nonceStr).Msg("replay protection check failed")
					return &ValidationError{
						Message:   "replay protection failed: " + err.Error(),
						ErrorCode: "replay_detected",
					}
				}
			}
		}
	}
	
	return nil
}

// validateSignature performs signature verification with fallback
func (s *SecurityPipeline) validateSignature(ctx context.Context, spec *models.JobSpec, rawBody []byte, clientIP string) error {
	l := logging.FromContext(ctx)
	
	validator := models.NewJobSpecValidator()
	err := validator.ValidateAndVerify(spec)
	if err != nil {
		// Map errors to clearer taxonomy
		msg := err.Error()
		code := "validation_error"
		switch {
		case strings.Contains(msg, "signature is required"):
			code = "missing_field:signature"
		case strings.Contains(msg, "public key is required"):
			code = "missing_field:public_key"
		case strings.Contains(msg, "invalid public key"):
			code = "invalid_encoding:public_key"
		case strings.Contains(msg, "canonicalize") || strings.Contains(msg, "canonicalization"):
			code = "canonicalization_error"
		case strings.Contains(msg, "signature verification failed"):
			code = "signature_mismatch"
		}
		
		// Fallback: try raw-body canonicalization for portal compatibility
		if code == "signature_mismatch" && len(rawBody) > 0 && spec.PublicKey != "" && spec.Signature != "" {
			if ferr := beaconcrypto.VerifySignatureRaw(rawBody, spec.Signature, spec.PublicKey, []string{"signature", "public_key"}); ferr == nil {
				l.Info().Str("job_id", spec.ID).Msg("compat signature verify (raw) succeeded; accepting")
				return nil
			} else {
				// Record signature failure for rate limiting when fallback fails too
				s.recordSignatureFailure(ctx, clientIP, spec.PublicKey)
			}
		} else if code == "signature_mismatch" {
			// Record signature failure for rate limiting
			s.recordSignatureFailure(ctx, clientIP, spec.PublicKey)
		}
		
		if code != "signature_mismatch" {
			// Record signature failure for canonicalization mismatch too
			if code == "canonicalization_error" {
				s.recordSignatureFailure(ctx, clientIP, spec.PublicKey)
			}
		}
		
		l.Error().Str("error_code", code).Err(err).Str("job_id", spec.ID).Msg("jobspec validation failed")
		return &ValidationError{
			Message:   msg,
			ErrorCode: code,
		}
	}
	
	return nil
}

// recordSignatureFailure records signature failure for rate limiting
func (s *SecurityPipeline) recordSignatureFailure(ctx context.Context, clientIP, publicKey string) {
	if s.rateLimiter != nil {
		s.rateLimiter.RecordSignatureFailure(ctx, clientIP, publicKey)
	}
}

// performShadowVerification performs debug shadow verification (non-fatal)
func (s *SecurityPipeline) performShadowVerification(ctx context.Context, spec *models.JobSpec) {
	if !logging.DebugEnabled() {
		return
	}
	
	l := logging.FromContext(ctx)
	
	if spec.PublicKey != "" && spec.Signature != "" {
		if pk, pkErr := beaconcrypto.PublicKeyFromBase64(spec.PublicKey); pkErr == nil {
			if canonV1, cErr := beaconcrypto.CanonicalizeJobSpecV1(spec); cErr == nil {
				if sig, sErr := base64.StdEncoding.DecodeString(spec.Signature); sErr == nil {
					v := ed25519.Verify(pk, canonV1, sig)
					l.Debug().Bool("shadow_v1_verify_ok", v).Int("canon_v1_len", len(canonV1)).Msg("debug: shadow v1 signature verify")
				} else {
					l.Debug().Err(sErr).Msg("debug: shadow v1: failed to decode signature base64")
				}
			} else {
				l.Debug().Err(cErr).Msg("debug: shadow v1: canonicalization error")
			}
		} else {
			l.Debug().Err(pkErr).Msg("debug: shadow v1: invalid public key")
		}
	}
}

// performDebugCanonicalization performs debug canonicalization comparison (non-fatal)
func (s *SecurityPipeline) performDebugCanonicalization(ctx context.Context, spec *models.JobSpec) {
	if !logging.DebugEnabled() {
		return
	}
	
	l := logging.FromContext(ctx)
	
	if spec.Signature != "" && spec.PublicKey != "" {
		if signable, derr := beaconcrypto.CreateSignableJobSpec(spec); derr == nil {
			var lenCurrent, lenV1 int
			var eq bool
			if canonCurrent, cerr := beaconcrypto.CanonicalJSON(signable); cerr == nil {
				lenCurrent = len(canonCurrent)
				if canonV1, verr := beaconcrypto.CanonicalizeJobSpecV1(spec); verr == nil {
					lenV1 = len(canonV1)
					eq = string(canonCurrent) == string(canonV1)
				} else {
					l.Debug().Err(verr).Msg("debug: v1 canonicalization failed")
				}
				l.Debug().Int("canon_current_len", lenCurrent).Int("canon_v1_len", lenV1).Bool("canon_equal", eq).Msg("debug: jobspec canonicalization compare (current vs v1)")
			} else {
				l.Debug().Err(cerr).Msg("debug: failed to canonicalize signable jobspec (current)")
			}
		} else {
			l.Debug().Err(derr).Msg("debug: failed to build signable jobspec")
		}
	} else {
		l.Debug().Msg("debug: missing signature or public key prior to verify")
	}
}
