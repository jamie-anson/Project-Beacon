package api

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/config"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/internal/metrics"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
	beaconcrypto "github.com/jamie-anson/project-beacon-runner/pkg/crypto"
)

// JobsHandler handles job-related requests
type JobsHandler struct {
	jobsService *service.JobsService
	cfg         *config.Config
}

// NewJobsHandler creates a new jobs handler
func NewJobsHandler(jobsService *service.JobsService, cfg *config.Config) *JobsHandler {
	return &JobsHandler{
		jobsService: jobsService,
		cfg:         cfg,
	}
}

// CreateJob handles job creation requests
func (h *JobsHandler) CreateJob(c *gin.Context) {
	l := logging.FromContext(c.Request.Context())
	l.Info().Msg("api: CreateJob request")
	// Parse incoming JobSpec
	var spec models.JobSpec
	if err := c.ShouldBindJSON(&spec); err != nil {
		l.Error().Err(err).Msg("invalid JSON")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}

	// Validate spec
	validator := models.NewJobSpecValidator()
	// DEBUG: pre-verify canonical signable JSON using current and v1 encoders
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
					l.Warn().Err(verr).Msg("debug: v1 canonicalization failed")
				}
				l.Info().Int("canon_current_len", lenCurrent).Int("canon_v1_len", lenV1).Bool("canon_equal", eq).Msg("debug: jobspec canonicalization compare (current vs v1)")
			} else {
				l.Warn().Err(cerr).Msg("debug: failed to canonicalize signable jobspec (current)")
			}
		} else {
			l.Warn().Err(derr).Msg("debug: failed to build signable jobspec")
		}
	} else {
		l.Info().Msg("debug: missing signature or public key prior to verify")
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
        l.Error().Str("error_code", code).Err(err).Str("job_id", spec.ID).Msg("jobspec validation failed")
        c.JSON(http.StatusBadRequest, gin.H{"error": msg, "error_code": code})
        return
    }

    // Shadow verification using v1 canonicalization (non-fatal)
    if spec.PublicKey != "" && spec.Signature != "" {
        if pk, pkErr := beaconcrypto.PublicKeyFromBase64(spec.PublicKey); pkErr == nil {
            if canonV1, cErr := beaconcrypto.CanonicalizeJobSpecV1(&spec); cErr == nil {
                if sig, sErr := base64.StdEncoding.DecodeString(spec.Signature); sErr == nil {
                    v := ed25519.Verify(pk, canonV1, sig)
                    l.Info().Bool("shadow_v1_verify_ok", v).Int("canon_v1_len", len(canonV1)).Msg("debug: shadow v1 signature verify")
                } else {
                    l.Warn().Err(sErr).Msg("debug: shadow v1: failed to decode signature base64")
                }
            } else {
                l.Warn().Err(cErr).Msg("debug: shadow v1: canonicalization error")
            }
        } else {
            l.Warn().Err(pkErr).Msg("debug: shadow v1: invalid public key")
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
