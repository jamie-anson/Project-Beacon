package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/external"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	healthChecker *external.HealthChecker
	db            *sql.DB
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(yagnaURL, ipfsURL string, db *sql.DB) *HealthHandler {
	return &HealthHandler{
		healthChecker: external.NewHealthChecker(yagnaURL, ipfsURL),
		db:            db,
	}
}

// GetHealth returns the health status of all services
func (h *HealthHandler) GetHealth(c *gin.Context) {
	ctx := c.Request.Context()
	
	services := h.healthChecker.CheckAllServices(ctx)
	
	// Determine overall health
	overallStatus := "healthy"
	for _, service := range services {
		if service.Status == "unhealthy" {
			overallStatus = "unhealthy"
			break
		} else if service.Status == "degraded" && overallStatus == "healthy" {
			overallStatus = "degraded"
		}
	}
	
	response := gin.H{
		"status":   overallStatus,
		"services": services,
	}
	
	// Set appropriate HTTP status code
	var statusCode int
	switch overallStatus {
	case "degraded":
		statusCode = http.StatusOK // Still OK, but with warnings
	case "unhealthy":
		statusCode = http.StatusServiceUnavailable
	default:
		statusCode = http.StatusOK
	}
	
	c.JSON(statusCode, response)
}

// GetHealthLiveness returns a simple liveness check
func (h *HealthHandler) GetHealthLiveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}

// GetHealthReadiness returns readiness status based on circuit breakers
func (h *HealthHandler) GetHealthReadiness(c *gin.Context) {
	ctx := c.Request.Context()
	
	services := h.healthChecker.CheckAllServices(ctx)
	
	// Service is ready if no critical services have open circuit breakers
	ready := true
	criticalServices := []string{"database", "redis"} // Define critical services
	
	for _, service := range services {
		for _, critical := range criticalServices {
			if service.Name == critical && service.Status == "unhealthy" {
				ready = false
				break
			}
		}
		if !ready {
			break
		}
	}
	// DB ping for concrete readiness
	if h.db != nil && ready {
		if err := h.db.PingContext(ctx); err != nil {
			ready = false
		}
	}
	
	statusCode := http.StatusOK
	if !ready {
		statusCode = http.StatusServiceUnavailable
	}
	
	c.JSON(statusCode, gin.H{
		"ready":    ready,
		"services": services,
	})
}
