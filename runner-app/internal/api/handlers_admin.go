package api

import (
	"net/http"
	"net"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/config"
	"github.com/jamie-anson/project-beacon-runner/internal/flags"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
)

// AdminHandler bundles simple admin operations
type AdminHandler struct {
	cfg         *config.Config
	jobsService *service.JobsService
}

func NewAdminHandler(cfg *config.Config) *AdminHandler {
	return &AdminHandler{cfg: cfg}
}

func NewAdminHandlerWithJobsService(cfg *config.Config, jobsService *service.JobsService) *AdminHandler {
	return &AdminHandler{cfg: cfg, jobsService: jobsService}
}

// GetFlags returns current feature flags
func (h *AdminHandler) GetFlags(c *gin.Context) {
	c.JSON(http.StatusOK, flags.Get())
}

// UpdateFlags updates feature flags from JSON body
func (h *AdminHandler) UpdateFlags(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unable to read body"})
		return
	}
	if err := flags.UpdateFromJSON(body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, flags.Get())
}

// GetConfig returns a redacted view of current configuration
func (h *AdminHandler) GetConfig(c *gin.Context) {
	type out struct {
		HTTPPort      string `json:"http_port"`
		PortStrategy  string `json:"port_strategy"`
		ResolvedAddr  string `json:"resolved_addr"`
		DatabaseURL   string `json:"database_url"`
		RedisURL      string `json:"redis_url"`
		DBTimeoutMS   int    `json:"db_timeout_ms"`
		RedisTimeoutMS int   `json:"redis_timeout_ms"`
		JobsQueueName string `json:"jobs_queue_name"`
		IPFSURL       string `json:"ipfs_url"`
		IPFSGateway   string `json:"ipfs_gateway"`
		YagnaURL      string `json:"yagna_url"`
	}
	cfg := h.cfg
	resp := out{
		HTTPPort:       cfg.HTTPPort,
		PortStrategy:   cfg.PortStrategy,
		ResolvedAddr:   cfg.ResolvedAddr,
		DatabaseURL:    redactDSN(cfg.DatabaseURL),
		RedisURL:       redactDSN(cfg.RedisURL),
		DBTimeoutMS:    int(cfg.DBTimeout / 1e6),
		RedisTimeoutMS: int(cfg.RedisTimeout / 1e6),
		JobsQueueName:  cfg.JobsQueueName,
		IPFSURL:        cfg.IPFSURL,
		IPFSGateway:    cfg.IPFSGateway,
		YagnaURL:       cfg.YagnaURL,
	}
	c.JSON(http.StatusOK, resp)
}

// GetPortInfo returns the port binding details
func (h *AdminHandler) GetPortInfo(c *gin.Context) {
	type out struct {
		Strategy   string `json:"strategy"`
		HTTPPort   string `json:"http_port"`
		Resolved   string `json:"resolved_addr"`
		RangeStart int    `json:"range_start"`
		RangeEnd   int    `json:"range_end"`
		AddrFile   string `json:"addr_file"`
	}
	cfg := h.cfg
	c.JSON(http.StatusOK, out{
		Strategy:   cfg.PortStrategy,
		HTTPPort:   cfg.HTTPPort,
		Resolved:   cfg.ResolvedAddr,
		RangeStart: cfg.PortRangeStart,
		RangeEnd:   cfg.PortRangeEnd,
		AddrFile:   cfg.AddrFile,
	})
}

// GetHints returns convenience hints like base URL for clients and tests
func (h *AdminHandler) GetHints(c *gin.Context) {
    cfg := h.cfg
    baseURL := ""
    if host, port, err := net.SplitHostPort(cfg.ResolvedAddr); err == nil {
        if host == "" || host == "0.0.0.0" || host == "::" {
            baseURL = "http://localhost:" + port
        } else {
            baseURL = "http://" + host + ":" + port
        }
    }
    c.JSON(http.StatusOK, gin.H{
        "strategy":      cfg.PortStrategy,
        "resolved_addr": cfg.ResolvedAddr,
        "base_url":      baseURL,
    })
}

// RepublishStuckJobs finds jobs in "created" status and republishes them to the outbox queue
func (h *AdminHandler) RepublishStuckJobs(c *gin.Context) {
	if h.jobsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "jobs service not available"})
		return
	}

	ctx := c.Request.Context()
	l := logging.FromContext(ctx)
	
	// Find jobs stuck in "created" status
	stuckJobs, err := h.jobsService.JobsRepo.ListJobsByStatus(ctx, "created", 100)
	if err != nil {
		l.Error().Err(err).Msg("failed to find stuck jobs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find stuck jobs"})
		return
	}

	if len(stuckJobs) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "no stuck jobs found",
			"republished": 0,
		})
		return
	}

	republished := 0
	for _, job := range stuckJobs {
		// Republish job to outbox
		err := h.jobsService.RepublishJob(ctx, job.ID)
		if err != nil {
			l.Error().Err(err).Str("job_id", job.ID).Msg("failed to republish job")
			continue
		}
		republished++
		l.Info().Str("job_id", job.ID).Msg("republished stuck job")
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "republished stuck jobs",
		"total_found": len(stuckJobs),
		"republished": republished,
	})
}

func redactDSN(s string) string {
	// naive redaction: remove password between : and @
	// e.g., postgres://user:pass@host/db -> postgres://user:****@host/db
	out := s
	// find ":" after scheme
	// keep it simple; best-effort redaction
	for i := 0; i < len(out); i++ {
		if out[i] == ':' {
			// find '@'
			for j := i + 1; j < len(out); j++ {
				if out[j] == '@' {
					return out[:i+1] + "****" + out[j:]
				}
			}
			break
		}
	}
	return out
}
