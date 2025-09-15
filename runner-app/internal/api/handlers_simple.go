package api

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/jamie-anson/project-beacon-runner/internal/config"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/internal/metrics"
	"github.com/jamie-anson/project-beacon-runner/internal/security"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
	beaconcrypto "github.com/jamie-anson/project-beacon-runner/pkg/crypto"
)

// JobsHandler handles job-related requests
type JobsHandler struct {
	jobsService      *service.JobsService
	cfg              *config.Config
	replayProtection *security.ReplayProtection
	rateLimiter      *security.RateLimiter
}

// NewJobsHandler creates a new jobs handler
func NewJobsHandler(jobsService *service.JobsService, cfg *config.Config, redisClient *redis.Client) *JobsHandler {
	var replayProtection *security.ReplayProtection
	var rateLimiter *security.RateLimiter
	
	if redisClient != nil {
		replayProtection = security.NewReplayProtection(redisClient, cfg.TimestampMaxAge)
		rateLimiter = security.NewRateLimiter(redisClient)
	}
	
	return &JobsHandler{
		jobsService:      jobsService,
		cfg:              cfg,
		replayProtection: replayProtection,
		rateLimiter:      rateLimiter,
	}
}

// CreateJob handles job creation requests
func (h *JobsHandler) CreateJob(c *gin.Context) {
	l := logging.FromContext(c.Request.Context())
	l.Info().Msg("api: CreateJob request")
	// Parse incoming JobSpec with raw body capture for signature fallback
	var spec models.JobSpec
	raw, rErr := io.ReadAll(c.Request.Body)
	if rErr != nil {
		l.Error().Err(rErr).Msg("failed to read request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(raw))
	if err := c.ShouldBindJSON(&spec); err != nil {
		l.Error().Err(err).Msg("invalid JSON")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}

	// Enforce questions for bias-detection v1 using raw JSON (fail fast if missing)
	// Only apply to v1 bias-detection
	if strings.EqualFold(spec.Version, "v1") && strings.Contains(strings.ToLower(spec.Benchmark.Name), "bias") {
		var tmp map[string]interface{}
		if err := json.Unmarshal(raw, &tmp); err == nil {
			qv, ok := tmp["questions"]
			if !ok {
				l.Warn().Str("job_id", spec.ID).Msg("rejecting: missing questions for bias-detection v1")
				c.JSON(http.StatusBadRequest, gin.H{"error": "questions are required for bias-detection v1 jobspec", "error_code": "missing_field:questions"})
				return
			}
			arr, isArr := qv.([]interface{})
			if !isArr || len(arr) == 0 {
				l.Warn().Str("job_id", spec.ID).Msg("rejecting: empty questions for bias-detection v1")
				c.JSON(http.StatusBadRequest, gin.H{"error": "questions must be a non-empty array for bias-detection v1 jobspec", "error_code": "invalid_field:questions"})
				return
			}
			l.Info().Str("job_id", spec.ID).Int("questions_count", len(arr)).Msg("questions validation passed for bias-detection v1")
		}
	}

	// Auto-generate ID if missing (for job creation endpoint)
	if spec.ID == "" && spec.JobSpecID == "" {
		// Generate ID from benchmark name and timestamp
		timestamp := time.Now().Unix()
		if spec.Benchmark.Name != "" {
			spec.ID = fmt.Sprintf("%s-%d", spec.Benchmark.Name, timestamp)
		} else {
			spec.ID = fmt.Sprintf("job-%d", timestamp)
		}
		l.Info().Str("generated_id", spec.ID).Msg("auto-generated job ID")
	}

	// Log questions presence after struct binding
	if len(spec.Questions) > 0 {
		l.Info().Str("job_id", spec.ID).Int("questions_present", len(spec.Questions)).Strs("questions", spec.Questions).Msg("JobSpec questions parsed successfully")
	} else {
		l.Info().Str("job_id", spec.ID).Msg("JobSpec has no questions field")
	}

	// Validate spec
	validator := models.NewJobSpecValidator()
    // DEBUG: pre-verify canonical signable JSON using current and v1 encoders (gated)
    if logging.DebugEnabled() {
        if spec.Signature != "" && spec.PublicKey != "" {
            if signable, derr := beaconcrypto.CreateSignableJobSpec(&spec); derr == nil {
                var lenCurrent, lenV1 int
                var eq bool
                if canonCurrent, cerr := beaconcrypto.CanonicalJSON(signable); cerr == nil {
                    lenCurrent = len(canonCurrent)
                    if canonV1, verr := beaconcrypto.CanonicalizeJobSpecV1(&spec); verr == nil {
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
    // Trust evaluation (allowlist). Enforce if enabled via cfg.TrustEnforce.
    if spec.PublicKey != "" {
        if reg, terr := config.GetTrustedKeys(); terr != nil {
            l.Warn().Err(terr).Msg("trusted-keys: load error")
        } else if reg != nil {
            entry := reg.ByPublicKey(spec.PublicKey)
            status, reason := config.EvaluateKeyTrust(entry, time.Now().UTC())
            if entry != nil {
                l.Info().Str("trust_status", status).Str("reason", reason).Str("kid", entry.KID).Msg("trusted-keys: evaluation")
            } else {
                l.Info().Str("trust_status", status).Str("reason", reason).Msg("trusted-keys: evaluation")
            }
            if h.cfg != nil && h.cfg.TrustEnforce {
                if status != "trusted" {
                    code := "trust_violation:" + status
                    c.JSON(http.StatusBadRequest, gin.H{"error": "untrusted signing key: " + reason, "error_code": code})
                    return
                }
            }
        }
    }

    // Rate limiting check for signature failures
    if h.rateLimiter != nil {
        clientIP := c.ClientIP()
        kid := spec.PublicKey
        if err := h.rateLimiter.CheckSignatureFailureRate(c.Request.Context(), clientIP, kid); err != nil {
            l.Warn().Str("client_ip", clientIP).Str("kid", kid).Msg("rate limit exceeded for signature failures")
            c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded", "error_code": "rate_limit_exceeded"})
            return
        }
    }

    // Enforce presence of timestamp and nonce when trust enforcement is enabled
    if h.cfg != nil && h.cfg.TrustEnforce {
        // If replay protection is enabled but Redis is unavailable, fail fast
        if h.cfg.ReplayProtectionEnabled && h.replayProtection == nil {
            c.JSON(http.StatusServiceUnavailable, gin.H{"error": "replay protection unavailable", "error_code": "protection_unavailable:replay"})
            return
        }
        if spec.Metadata == nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "missing metadata", "error_code": "missing_field:metadata"})
            return
        }
        if tsVal, ok := spec.Metadata["timestamp"]; !ok || tsVal == nil || (func(v interface{}) bool { s, sok := v.(string); return !sok || strings.TrimSpace(s) == "" })(tsVal) {
            c.JSON(http.StatusBadRequest, gin.H{"error": "timestamp is required", "error_code": "missing_field:timestamp"})
            return
        }
        if nonceVal, ok := spec.Metadata["nonce"]; !ok || nonceVal == nil || (func(v interface{}) bool { s, sok := v.(string); return !sok || strings.TrimSpace(s) == "" })(nonceVal) {
            c.JSON(http.StatusBadRequest, gin.H{"error": "nonce is required", "error_code": "missing_field:nonce"})
            return
        }
    }

    // Timestamp validation
    if spec.Metadata != nil {
        if tsVal, exists := spec.Metadata["timestamp"]; exists && h.cfg != nil && h.cfg.TimestampMaxSkew > 0 && h.cfg.TimestampMaxAge > 0 {
            if tsStr, ok := tsVal.(string); ok {
                if ts, err := time.Parse(time.RFC3339, tsStr); err == nil {
                    if reason, terr := security.ValidateTimestampWithReason(ts, h.cfg.TimestampMaxSkew, h.cfg.TimestampMaxAge); terr != nil {
                        l.Error().Err(terr).Str("reason", reason).Time("spec_timestamp", ts).Msg("timestamp validation failed")
                        c.JSON(http.StatusBadRequest, gin.H{"error": "timestamp validation failed", "error_code": "timestamp_invalid", "details": gin.H{"reason": reason}})
                        return
                    }
                } else {
                    l.Error().Err(err).Str("timestamp", tsStr).Msg("invalid timestamp format")
                    c.JSON(http.StatusBadRequest, gin.H{"error": "timestamp validation failed", "error_code": "timestamp_invalid", "details": gin.H{"reason": "format_invalid"}})
                    return
                }
            }
        }
    }

    // Replay protection (can be disabled via cfg.ReplayProtectionEnabled)
    if h.replayProtection != nil && spec.Metadata != nil && (h.cfg == nil || h.cfg.ReplayProtectionEnabled) {
        if nonceVal, exists := spec.Metadata["nonce"]; exists {
            if nonceStr, ok := nonceVal.(string); ok && nonceStr != "" {
                kid := spec.PublicKey
                if err := h.replayProtection.CheckAndRecordNonce(c.Request.Context(), kid, nonceStr); err != nil {
                    l.Error().Err(err).Str("kid", kid).Str("nonce", nonceStr).Msg("replay protection check failed")
                    c.JSON(http.StatusBadRequest, gin.H{"error": "replay protection failed: " + err.Error(), "error_code": "replay_detected"})
                    return
                }
            }
        }
    }
    if h.cfg != nil && h.cfg.SigBypass {
        l.Warn().Str("job_id", spec.ID).Msg("RUNNER_SIG_BYPASS enabled: skipping signature verification (dev only)")
    } else {
        if err := validator.ValidateAndVerify(&spec); err != nil {
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
            if code == "signature_mismatch" && len(raw) > 0 && spec.PublicKey != "" && spec.Signature != "" {
                if ferr := beaconcrypto.VerifySignatureRaw(raw, spec.Signature, spec.PublicKey, []string{"signature", "public_key"}); ferr == nil {
                    l.Info().Str("job_id", spec.ID).Msg("compat signature verify (raw) succeeded; accepting")
                } else {
                    // Record signature failure for rate limiting when fallback fails too
                    if h.rateLimiter != nil {
                        clientIP := c.ClientIP()
                        kid := spec.PublicKey
                        h.rateLimiter.RecordSignatureFailure(c.Request.Context(), clientIP, kid)
                    }
                    l.Error().Str("error_code", code).Err(err).Str("job_id", spec.ID).Msg("jobspec validation failed")
                    c.JSON(http.StatusBadRequest, gin.H{"error": msg, "error_code": code})
                    return
                }
            } else if code == "signature_mismatch" {
                // Record signature failure for rate limiting
                if h.rateLimiter != nil {
                    clientIP := c.ClientIP()
                    kid := spec.PublicKey // Use public key as identifier
                    h.rateLimiter.RecordSignatureFailure(c.Request.Context(), clientIP, kid)
                }
                l.Error().Str("error_code", code).Err(err).Str("job_id", spec.ID).Msg("jobspec validation failed")
                c.JSON(http.StatusBadRequest, gin.H{"error": msg, "error_code": code})
                return
            }

            if code != "signature_mismatch" {
                // For other validation errors, return immediately
                // Record signature failure for canonicalization mismatch too
                if h.rateLimiter != nil && code == "canonicalization_error" {
                    clientIP := c.ClientIP()
                    kid := spec.PublicKey // Use public key as identifier
                    h.rateLimiter.RecordSignatureFailure(c.Request.Context(), clientIP, kid)
                }
                l.Error().Str("error_code", code).Err(err).Str("job_id", spec.ID).Msg("jobspec validation failed")
                c.JSON(http.StatusBadRequest, gin.H{"error": msg, "error_code": code})
                return
            }
        }
    }

    // Shadow verification using v1 canonicalization (non-fatal, gated)
    if logging.DebugEnabled() {
        if spec.PublicKey != "" && spec.Signature != "" {
            if pk, pkErr := beaconcrypto.PublicKeyFromBase64(spec.PublicKey); pkErr == nil {
                if canonV1, cErr := beaconcrypto.CanonicalizeJobSpecV1(&spec); cErr == nil {
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

	// Marshal canonical JSON for persistence/outbox
	jobspecJSON, err := json.Marshal(&spec)
	if err != nil {
		l.Error().Err(err).Str("job_id", spec.ID).Msg("failed to marshal jobspec")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal jobspec"})
		return
	}

	if h.jobsService == nil || h.jobsService.DB == nil {
		l.Error().Msg("persistence unavailable")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "persistence unavailable"})
		return
	}
    // Idempotency support
    idemKey, hasKey := GetIdempotencyKey(c)
    if hasKey && h.jobsService != nil {
        jobID, reused, ierr := h.jobsService.IdempotentCreateJob(c.Request.Context(), idemKey, &spec, jobspecJSON)
        if ierr != nil {
            l.Error().Err(ierr).Str("job_id", spec.ID).Msg("IdempotentCreateJob error")
            c.JSON(http.StatusInternalServerError, gin.H{"error": ierr.Error()})
            return
        }
        if reused {
            l.Info().Str("job_id", jobID).Bool("idempotent", true).Msg("idempotent key reused; returning existing job")
            c.JSON(http.StatusOK, gin.H{"id": jobID, "idempotent": true})
            return
        }
        l.Info().Str("job_id", jobID).Msg("job enqueued (idempotent create)")
        metrics.JobsEnqueuedTotal.Inc()
        c.JSON(http.StatusAccepted, gin.H{"id": jobID, "status": "enqueued"})
        return
    }

    // Non-idempotent path
    if err := h.jobsService.CreateJob(c.Request.Context(), &spec, jobspecJSON); err != nil {
        l.Error().Err(err).Str("job_id", spec.ID).Msg("CreateJob service error")
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    l.Info().Str("job_id", spec.ID).Msg("job enqueued")
    metrics.JobsEnqueuedTotal.Inc()
    c.JSON(http.StatusAccepted, gin.H{"id": spec.ID, "status": "enqueued"})
}

// GetJob handles job retrieval requests
func (h *JobsHandler) GetJob(c *gin.Context) {
	l := logging.FromContext(c.Request.Context())
	jobID := c.Param("id")
	if jobID == "" {
		l.Error().Msg("missing job id")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing job id"})
		return
	}
	if h.jobsService == nil || h.jobsService.DB == nil {
		l.Error().Msg("persistence unavailable")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "persistence unavailable"})
		return
	}
	spec, status, err := h.jobsService.GetJob(c.Request.Context(), jobID)
	if err != nil {
		// Map not-found to 404; keep other errors as 500
		if strings.Contains(err.Error(), "job not found") {
			l.Info().Str("job_id", jobID).Msg("job not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		l.Error().Err(err).Str("job_id", jobID).Msg("GetJob service error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if spec == nil {
		l.Info().Str("job_id", jobID).Msg("job not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	// Optional includes
	include := strings.ToLower(c.Query("include"))
	if include == "executions" || include == "all" || include == "latest" {
		if h.jobsService.ExecutionsRepo != nil {
			if include == "latest" {
                rec, lerr := h.jobsService.GetLatestReceiptCached(c.Request.Context(), jobID)
                if lerr != nil {
                    // If no receipt yet, still return job with empty executions
                    l.Info().Str("job_id", jobID).Msg("no latest receipt yet")
                    c.JSON(http.StatusOK, gin.H{"job": spec, "status": status, "executions": []interface{}{}})
                    return
                }
                l.Info().Str("job_id", jobID).Msg("returning latest receipt")
                c.JSON(http.StatusOK, gin.H{"job": spec, "status": status, "executions": []interface{}{rec}})
                return
            }

			// Pagination params for executions list
			execLimit := 20
			if v := c.Query("exec_limit"); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
					execLimit = n
				}
			}
			execOffset := 0
			if v := c.Query("exec_offset"); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n >= 0 {
					execOffset = n
				}
			}

			// Prefer paginated method if available
			recs, lerr := h.jobsService.ExecutionsRepo.ListExecutionsByJobSpecIDPaginated(c.Request.Context(), jobID, execLimit, execOffset)
			if lerr != nil {
				// Fallback to non-paginated list
				recs, lerr2 := h.jobsService.ExecutionsRepo.ListExecutionsByJobSpecID(c.Request.Context(), jobID)
				if lerr2 != nil {
					l.Error().Err(lerr2).Str("job_id", jobID).Msg("list executions error")
					c.JSON(http.StatusInternalServerError, gin.H{"error": lerr2.Error()})
					return
				}
				l.Info().Str("job_id", jobID).Int("count", len(recs)).Msg("returning executions (fallback)")
				c.JSON(http.StatusOK, gin.H{"job": spec, "status": status, "executions": recs})
				return
			}
			l.Info().Str("job_id", jobID).Int("count", len(recs)).Msg("returning executions (paginated)")
			c.JSON(http.StatusOK, gin.H{"job": spec, "status": status, "executions": recs})
			return
		}
	}
	l.Info().Str("job_id", jobID).Msg("returning job without executions")
	c.JSON(http.StatusOK, gin.H{"job": spec, "status": status})
}

// ListJobs handles job listing requests
func (h *JobsHandler) ListJobs(c *gin.Context) {
	l := logging.FromContext(c.Request.Context())
	if h.jobsService == nil || h.jobsService.DB == nil || h.jobsService.JobsRepo == nil {
		l.Error().Msg("persistence unavailable")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "persistence unavailable"})
		return
	}

	// Parse limit (default 50)
	limitStr := c.Query("limit")
	limit := 50
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}

	rows, err := h.jobsService.JobsRepo.ListRecentJobs(c.Request.Context(), limit)
	if err != nil {
		l.Error().Err(err).Int("limit", limit).Msg("ListRecentJobs error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type item struct {
		ID        string    `json:"id"`
		Status    string    `json:"status"`
		CreatedAt time.Time `json:"created_at"`
	}
	var out []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.ID, &it.Status, &it.CreatedAt); err != nil {
			l.Error().Err(err).Msg("rows scan error")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		l.Error().Err(err).Msg("rows iteration error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	l.Info().Int("count", len(out)).Msg("returning recent jobs")
	c.JSON(http.StatusOK, gin.H{"jobs": out})
}
