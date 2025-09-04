package health

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
)

// HealthMiddleware provides health check endpoints
func HealthMiddleware(hm *HealthManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add health manager to context for other handlers
		c.Set("health_manager", hm)
		c.Next()
	}
}

// LivenessHandler handles Kubernetes liveness probes
func LivenessHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "alive",
			"timestamp": time.Now().UTC(),
		})
	}
}

// ReadinessHandler handles Kubernetes readiness probes
func ReadinessHandler(hm *HealthManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		
		overallStatus, services := hm.GetHealthChecker().GetOverallHealth(ctx)
		
		// Service is ready if overall status is healthy or degraded
		ready := overallStatus == StatusHealthy || overallStatus == StatusDegraded
		
		statusCode := http.StatusOK
		if !ready {
			statusCode = http.StatusServiceUnavailable
		}
		
		c.JSON(statusCode, gin.H{
			"ready":     ready,
			"status":    overallStatus,
			"services":  services,
			"timestamp": time.Now().UTC(),
		})
	}
}

// HealthHandler provides detailed health information
func HealthHandler(hm *HealthManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		summary := hm.GetHealthSummary(ctx)
		
		// Determine HTTP status based on overall health
		overallStatus := summary["overall_status"].(ServiceStatus)
		statusCode := http.StatusOK
		
		switch overallStatus {
		case StatusUnhealthy:
			statusCode = http.StatusServiceUnavailable
		case StatusDegraded:
			statusCode = http.StatusOK // Still OK, but with warnings
		}
		
		c.JSON(statusCode, summary)
	}
}

// CircuitBreakerHandler provides circuit breaker status
func CircuitBreakerHandler(hm *HealthManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats := hm.GetCircuitBreakerManager().AllStats()
		
		response := gin.H{
			"circuit_breakers": stats,
			"timestamp":        time.Now().UTC(),
		}
		
		c.JSON(http.StatusOK, response)
	}
}

// CircuitBreakerResetHandler resets circuit breakers (admin only)
func CircuitBreakerResetHandler(hm *HealthManager, logger *logging.StructuredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Log the reset action
		logger.Info("circuit breakers reset requested",
			"user_agent", c.Request.UserAgent(),
			"remote_addr", c.ClientIP())
		
		hm.ResetCircuitBreakers()
		
		c.JSON(http.StatusOK, gin.H{
			"message":   "circuit breakers reset",
			"timestamp": time.Now().UTC(),
		})
	}
}

// HealthCheckMiddleware performs health checks on critical dependencies
func HealthCheckMiddleware(hm *HealthManager, criticalServices []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		
		// Check critical services before processing request
		for _, serviceName := range criticalServices {
			health := hm.GetHealthChecker().CheckService(ctx, serviceName)
			
			if health.Status == StatusUnhealthy {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error":   "service unavailable",
					"service": serviceName,
					"status":  health.Status,
				})
				c.Abort()
				return
			}
		}
		
		c.Next()
	}
}

// MetricsHealthHandler provides health metrics for Prometheus
func MetricsHealthHandler(hm *HealthManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		services := hm.GetHealthChecker().CheckAllServices(ctx)
		
		// Generate Prometheus-style metrics
		metrics := ""

		// Add HELP/TYPE headers once
		metrics += "# HELP service_health Health status of service (0=unknown, 1=healthy, 2=degraded, 3=unhealthy)\n"
		metrics += "# TYPE service_health gauge\n"
		metrics += "# HELP service_response_time_seconds Response time of service health check\n"
		metrics += "# TYPE service_response_time_seconds gauge\n"

		for _, service := range services {
			statusValue := 0
			switch service.Status {
			case StatusHealthy:
				statusValue = 1
			case StatusDegraded:
				statusValue = 2
			case StatusUnhealthy:
				statusValue = 3
			}

			metrics += fmt.Sprintf("service_health{service=\"%s\"} %d\n", service.Name, statusValue)
			metrics += fmt.Sprintf("service_response_time_seconds{service=\"%s\"} %.6f\n", service.Name, service.ResponseTime.Seconds())
		}

		c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		c.String(http.StatusOK, metrics)
	}
}
