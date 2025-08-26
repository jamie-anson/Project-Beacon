package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/metrics"
)

func TestMetricsAliasServesSameContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mount metrics on both canonical and alias routes
	router.Use(metrics.GinMiddleware())
	router.GET("/metrics", gin.WrapH(metrics.Handler()))
	router.GET("/api/v1/metrics", gin.WrapH(metrics.Handler()))

	// Helper to GET and capture body + content-type
	get := func(path string) (int, string, string) {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		res := w.Result()
		b, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		return res.StatusCode, res.Header.Get("Content-Type"), string(b)
	}

	status1, ct1, body1 := get("/metrics")
	status2, ct2, body2 := get("/api/v1/metrics")

	if status1 != http.StatusOK || status2 != http.StatusOK {
		t.Fatalf("expected 200 from both routes, got %d and %d", status1, status2)
	}

	// Prometheus default content-type is text/plain; version may vary; allow prefix match
	if !strings.HasPrefix(ct1, "text/plain") || !strings.HasPrefix(ct2, "text/plain") {
		t.Fatalf("unexpected content types: %q and %q", ct1, ct2)
	}

	// Bodies can differ due to route labels; ensure Prometheus text shape and common metrics exist
	if !strings.Contains(body1, "# HELP") || !strings.Contains(body2, "# HELP") {
		t.Fatalf("expected Prometheus HELP headers in both responses")
	}
	const commonMetric = "go_gc_duration_seconds"
	if !strings.Contains(body1, commonMetric) || !strings.Contains(body2, commonMetric) {
		t.Fatalf("expected to find common metric %q in both responses", commonMetric)
	}
}
