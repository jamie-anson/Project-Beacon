package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// System handlers: health, metrics, debug

// Debug Yagna configuration and a single probe attempt
func (s *APIServer) debugYagna(c *gin.Context) {
    ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
    defer cancel()

    backend := s.golemService.Backend()
    yagnaURL := s.golemService.YagnaURL()
    hasKey := s.golemService.AppKeyPresent()
    marketBase, activityBase := s.golemService.ClientBases()

    hit, ver, err := s.golemService.ProbeOnce(ctx)
    var yagnaStatus string
    if err != nil {
        yagnaStatus = "unreachable"
    } else if hit == "success" {
        yagnaStatus = "reachable"
    } else {
        yagnaStatus = "timeout"
    }

    c.JSON(http.StatusOK, gin.H{
        "backend":       backend,
        "yagna_url":     yagnaURL,
        "app_key":       hasKey,
        "market_base":   marketBase,
        "activity_base": activityBase,
        "probe_result": gin.H{
            "hit":     hit,
            "version": ver,
            "error":   err,
            "status":  yagnaStatus,
        },
    })
}

// Provider discovery handler
func (s *APIServer) listProviders(c *gin.Context) {
    c.JSON(http.StatusNotImplemented, gin.H{
        "error": "Provider discovery not implemented yet",
    })
}

// Metrics summary handler
func (s *APIServer) metricsSummary(c *gin.Context) {
	// TODO: Implement metrics collection
	c.JSON(http.StatusOK, gin.H{
		"summary":   "metrics not implemented yet",
		"timestamp": time.Now().UTC(),
	})
}

// Prometheus metrics handler
func (s *APIServer) prometheusMetrics(c *gin.Context) {
	promhttp.Handler().ServeHTTP(c.Writer, c.Request)
}

// Health check handler
func (s *APIServer) healthCheck(c *gin.Context) {
    // Quick health snapshot including DB and Redis
    ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Second)
    defer cancel()

    dbStatus := "disabled"
    if s.db != nil && s.db.DB != nil {
        if err := s.db.DB.PingContext(ctx); err != nil {
            dbStatus = "error: " + err.Error()
        } else {
            dbStatus = "healthy"
        }
    }

    redisStatus := "disabled"
    if s.q != nil {
        if err := s.q.Ping(ctx); err != nil {
            redisStatus = "error: " + err.Error()
        } else {
            redisStatus = "healthy"
        }
    }

    // Probe Yagna once
    yagnaHit, _, yagnaErr := s.golemService.ProbeOnce(ctx)
    yagnaStatus := "unreachable"
    if yagnaErr == nil && yagnaHit == "success" {
        yagnaStatus = "healthy"
    } else if yagnaErr != nil {
        yagnaStatus = "error: " + yagnaErr.Error()
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "healthy",
        "timestamp": time.Now().UTC(),
        "components": gin.H{
            "golem_service":     "ready",
            "execution_engine":  "ready",
            "jobspec_validator": "ready",
            "postgres":          dbStatus,
            "redis":             redisStatus,
            "yagna":             gin.H{"status": yagnaStatus, "hit_path": yagnaHit},
        },
    })
}
