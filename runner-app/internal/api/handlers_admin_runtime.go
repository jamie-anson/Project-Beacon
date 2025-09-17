package api

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/config"
	"github.com/jamie-anson/project-beacon-runner/internal/queue"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
)

// RuntimeHandler handles runtime and system monitoring operations
type RuntimeHandler struct {
	cfg         *config.Config
	jobsService *service.JobsService
	queueClient interface {
		GetCircuitBreakerStats() string
	}
}

// NewRuntimeHandler creates a new RuntimeHandler
func NewRuntimeHandler(cfg *config.Config, jobsService *service.JobsService, queueClient interface{ GetCircuitBreakerStats() string }) *RuntimeHandler {
	return &RuntimeHandler{
		cfg:         cfg,
		jobsService: jobsService,
		queueClient: queueClient,
	}
}

// GetOutboxStats returns DB outbox unpublished stats
func (h *RuntimeHandler) GetOutboxStats(c *gin.Context) {
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
func (h *RuntimeHandler) GetQueueRuntimeStats(c *gin.Context) {
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

// GetCircuitBreakerStats returns circuit breaker statistics for Redis operations
func (h *RuntimeHandler) GetCircuitBreakerStats(c *gin.Context) {
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

// GetResourceStats returns current system resource usage statistics
func (h *RuntimeHandler) GetResourceStats(c *gin.Context) {
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
