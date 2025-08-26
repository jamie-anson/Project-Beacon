package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jamie-anson/project-beacon-runner/internal/config"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
)

func TestHealthEndpoints_OK(t *testing.T) {
	// Minimal config; values are echoed in handler responses/logs
	cfg := &config.Config{
		HTTPPort: "8090",
		YagnaURL: "http://localhost:7465",
		IPFSURL:  "http://localhost:5001",
	}
	r := SetupRoutes(service.NewJobsService(nil), cfg, nil)

	cases := []string{"/health", "/health/live", "/health/ready"}
	for _, path := range cases {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s expected 200, got %d; body=%s", path, w.Code, w.Body.String())
		}
	}
}
