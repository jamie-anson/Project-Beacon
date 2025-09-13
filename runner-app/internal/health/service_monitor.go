package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ExternalServiceStatus represents the health status of an external service
type ExternalServiceStatus struct {
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	Status       string    `json:"status"` // "healthy", "degraded", "down"
	LastChecked  time.Time `json:"last_checked"`
	ResponseTime int64     `json:"response_time_ms"`
	Error        string    `json:"error,omitempty"`
}

// ServiceMonitor monitors external service health
type ServiceMonitor struct {
	services map[string]string // name -> URL
	client   *http.Client
}

// NewServiceMonitor creates a new service monitor
func NewServiceMonitor() *ServiceMonitor {
	return &ServiceMonitor{
		services: map[string]string{
			"hybrid_router": "https://beacon-hybrid-router.fly.dev/health",
			"runner_app":    "https://beacon-hybrid-router.fly.dev/status",
		},
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CheckService checks the health of a specific service
func (sm *ServiceMonitor) CheckService(ctx context.Context, name, url string) ExternalServiceStatus {
	start := time.Now()
	status := ExternalServiceStatus{
		Name:        name,
		URL:         url,
		LastChecked: start,
		Status:      "down",
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		status.Error = fmt.Sprintf("Failed to create request: %v", err)
		return status
	}

	resp, err := sm.client.Do(req)
	if err != nil {
		status.Error = fmt.Sprintf("Connection failed: %v", err)
		status.ResponseTime = time.Since(start).Milliseconds()
		return status
	}
	defer resp.Body.Close()

	status.ResponseTime = time.Since(start).Milliseconds()

	if resp.StatusCode == http.StatusOK {
		// Try to parse response for additional health info
		var healthResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&healthResp); err == nil {
			if healthStatus, ok := healthResp["status"].(string); ok {
				if healthStatus == "healthy" {
					status.Status = "healthy"
				} else {
					status.Status = "degraded"
					status.Error = fmt.Sprintf("Service reports status: %s", healthStatus)
				}
			} else {
				status.Status = "healthy" // Default to healthy if we got 200 OK
			}
		} else {
			status.Status = "healthy" // Default to healthy if we got 200 OK
		}
	} else {
		status.Status = "degraded"
		status.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return status
}

// CheckAllServices checks all monitored services
func (sm *ServiceMonitor) CheckAllServices(ctx context.Context) map[string]ExternalServiceStatus {
	results := make(map[string]ExternalServiceStatus)
	
	for name, url := range sm.services {
		results[name] = sm.CheckService(ctx, name, url)
	}
	
	return results
}

// GetInfrastructureStatus returns overall infrastructure health
func (sm *ServiceMonitor) GetInfrastructureStatus(ctx context.Context) InfrastructureStatus {
	services := sm.CheckAllServices(ctx)
	
	healthyCount := 0
	degradedCount := 0
	downCount := 0
	
	for _, service := range services {
		switch service.Status {
		case "healthy":
			healthyCount++
		case "degraded":
			degradedCount++
		case "down":
			downCount++
		}
	}
	
	totalServices := len(services)
	overallStatus := "healthy"
	
	if downCount > 0 {
		overallStatus = "degraded"
		if downCount == totalServices {
			overallStatus = "down"
		}
	} else if degradedCount > 0 {
		overallStatus = "degraded"
	}
	
	return InfrastructureStatus{
		OverallStatus:   overallStatus,
		TotalServices:   totalServices,
		HealthyServices: healthyCount,
		DegradedServices: degradedCount,
		DownServices:    downCount,
		Services:        services,
		LastChecked:     time.Now(),
	}
}

// InfrastructureStatus represents overall infrastructure health
type InfrastructureStatus struct {
	OverallStatus     string                           `json:"overall_status"`
	TotalServices     int                              `json:"total_services"`
	HealthyServices   int                              `json:"healthy_services"`
	DegradedServices  int                              `json:"degraded_services"`
	DownServices      int                              `json:"down_services"`
	Services          map[string]ExternalServiceStatus `json:"services"`
	LastChecked       time.Time                        `json:"last_checked"`
}
