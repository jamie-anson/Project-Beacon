package api

import (
	"log"
	"net/http"
	"net"
	"runtime"
	"time"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/config"
	"github.com/jamie-anson/project-beacon-runner/internal/flags"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
	"github.com/jamie-anson/project-beacon-runner/internal/queue"
)

// AdminHandler bundles simple admin operations
type AdminHandler struct {
	cfg         *config.Config
	jobsService *service.JobsService
	queueClient interface {
		GetCircuitBreakerStats() string
	}
}

// GetOutboxStats returns DB outbox unpublished stats
func (h *AdminHandler) GetOutboxStats(c *gin.Context) {
    if h.jobsService == nil || h.jobsService.Outbox == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "outbox not available"})
        return
    }
    count, oldest, err := h.jobsService.Outbox.GetUnpublishedStats(c.Request.Context())
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get outbox stats", "details": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"unpublished_count": count, "oldest_age_seconds": oldest})
}

// GetQueueRuntimeStats returns Redis queue lengths (main, retry, dead, processing)
func (h *AdminHandler) GetQueueRuntimeStats(c *gin.Context) {
    if h.queueClient == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "queue not available"})
        return
    }
    // Try to access underlying queue client for stats
    qc, ok := h.queueClient.(*queue.Client)
    if !ok || qc == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "queue client does not expose stats"})
        return
    }
    // Use RedisQueue stats if advanced queue is initialized; otherwise fall back to simple LLEN
    if rq, ok := any(qc).(*queue.Client); ok && rq != nil {
        // queue.Client has no direct stats, so attempt best-effort simple LLENs
        rc := rq.GetRedisClient()
        if rc == nil {
            c.JSON(http.StatusServiceUnavailable, gin.H{"error": "redis client not available"})
            return
        }
        // Determine queue name from config
        qname := h.cfg.JobsQueueName
        mainLen := rc.LLen(c.Request.Context(), qname).Val()
        deadLen := rc.LLen(c.Request.Context(), qname+":dead").Val()
        retryLen := rc.ZCard(c.Request.Context(), qname+":retry").Val()
        // processing is a set of keys pattern; approximate by key count
        processingKeys, _ := rc.Keys(c.Request.Context(), qname+":processing:*").Result()
        c.JSON(http.StatusOK, gin.H{
            "queue": qname,
            "main": mainLen,
            "retry": retryLen,
            "dead": deadLen,
            "processing": len(processingKeys),
        })
        return
    }
    c.JSON(http.StatusServiceUnavailable, gin.H{"error": "unable to access queue stats"})
}

// RepublishJobByID republishes a specific job to the outbox queue.
// Body: {"job_id": "<jobspec_id>"}
func (h *AdminHandler) RepublishJobByID(c *gin.Context) {
    if h.jobsService == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "jobs service not available"})
        return
    }
    var req struct{ JobID string `json:"job_id"` }
    body, err := c.GetRawData()
    if err != nil || len(body) == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_body"})
        return
    }
    if err := json.Unmarshal(body, &req); err != nil || req.JobID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "missing_job_id"})
        return
    }
    if err := h.jobsService.RepublishJob(c.Request.Context(), req.JobID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "republish_failed", "details": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "republished", "job_id": req.JobID})
}

func NewAdminHandler(cfg *config.Config) *AdminHandler {
	return &AdminHandler{cfg: cfg}
}

func NewAdminHandlerWithJobsService(cfg *config.Config, jobsService *service.JobsService) *AdminHandler {
	return &AdminHandler{cfg: cfg, jobsService: jobsService}
}

func NewAdminHandlerWithQueue(cfg *config.Config, jobsService *service.JobsService, queueClient interface{ GetCircuitBreakerStats() string }) *AdminHandler {
	return &AdminHandler{cfg: cfg, jobsService: jobsService, queueClient: queueClient}
}

// GetCircuitBreakerStats returns circuit breaker statistics for Redis operations
func (h *AdminHandler) GetCircuitBreakerStats(c *gin.Context) {
	if h.queueClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Queue service not available",
		})
		return
	}

	stats := h.queueClient.GetCircuitBreakerStats()
	c.JSON(http.StatusOK, gin.H{
		"circuit_breaker_stats": stats,
		"timestamp": "2025-09-15T14:59:00Z",
	})
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
	// l := logging.FromContext(ctx)
	
	// Find jobs stuck in "created" status
	stuckJobs, err := h.jobsService.JobsRepo.ListJobsByStatus(ctx, "created", 100)
	if err != nil {
		log.Printf("Failed to find stuck jobs: %v", err)
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
			log.Printf("Failed to republish stuck job %s: %v", job.ID, err)
			continue
		}
		republished++
		log.Printf("Republished stuck job: %s", job.ID)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "republished stuck jobs",
		"total_found": len(stuckJobs),
		"republished": republished,
	})
}

// RepairStuckJobsHandler handles job repair requests
func (h *AdminHandler) RepairStuckJobsHandler(c *gin.Context) {
	if h.jobsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Jobs service not available",
		})
		return
	}

	// Parse max age parameter (default: 30 minutes)
	maxAgeStr := c.DefaultQuery("max_age", "30m")
	maxAge, err := time.ParseDuration(maxAgeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid max_age parameter",
			"details": err.Error(),
		})
		return
	}

	// Create repair service and run repair
	repairService := service.NewJobRepairService(h.jobsService)
	summary, err := repairService.RepairStuckJobs(c.Request.Context(), maxAge)
	if err != nil {
		log.Printf("Failed to repair stuck jobs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to repair stuck jobs",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Job repair completed",
		"summary": summary,
	})
}

// GetStuckJobsStats returns statistics about potentially stuck jobs
func (h *AdminHandler) GetStuckJobsStats(c *gin.Context) {
	if h.jobsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Jobs service not available",
		})
		return
	}

	repairService := service.NewJobRepairService(h.jobsService)
	stats, err := repairService.GetStuckJobsStats(c.Request.Context())
	if err != nil {
		log.Printf("Failed to get stuck jobs stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get stuck jobs stats",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
		"timestamp": time.Now().UTC(),
	})
}

// GetResourceStats returns current system resource usage statistics
func (h *AdminHandler) GetResourceStats(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	stats := map[string]interface{}{
		"memory": map[string]interface{}{
			"heap_alloc_mb":    float64(m.HeapAlloc) / 1024 / 1024,
			"heap_sys_mb":      float64(m.HeapSys) / 1024 / 1024,
			"stack_inuse_mb":   float64(m.StackInuse) / 1024 / 1024,
			"heap_objects":     m.HeapObjects,
			"mallocs":          m.Mallocs,
			"frees":            m.Frees,
		},
		"gc": map[string]interface{}{
			"cycles":           m.NumGC,
			"pause_ms":         float64(m.PauseNs[(m.NumGC+255)%256]) / 1000000,
		},
		"goroutines":       runtime.NumGoroutine(),
		"timestamp":        time.Now().UTC(),
	}

	c.JSON(http.StatusOK, stats)
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
