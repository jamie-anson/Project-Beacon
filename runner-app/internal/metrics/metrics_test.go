package metrics

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSummaryReturnsMap(t *testing.T) {
	// Increment a couple counters to ensure non-empty gather
	JobsEnqueuedTotal.Inc()
	JobsProcessedTotal.Inc()

	m, err := Summary()
	if err != nil {
		t.Fatalf("Summary error: %v", err)
	}
	if m == nil {
		t.Fatalf("expected map, got nil")
	}
}

func TestHandlerAndGinMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinMiddleware())
	r.GET("/ping", func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
