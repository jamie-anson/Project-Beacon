package metrics

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

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

func TestStartPeriodicUpdates_CancelledContext_Returns(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    cancel() // cancel immediately
    c := &Collector{}
    done := make(chan struct{})
    go func() { c.StartPeriodicUpdates(ctx); close(done) }()
    select {
    case <-done:
        // ok
    case <-time.After(200 * time.Millisecond):
        t.Fatal("StartPeriodicUpdates did not return promptly on canceled context")
    }
}

func TestHandler_Serves(t *testing.T) {
    h := Handler()
    req := httptest.NewRequest("GET", "/metrics", nil)
    w := httptest.NewRecorder()
    h.ServeHTTP(w, req)
    if w.Code != 200 {
        t.Fatalf("expected 200 from metrics handler, got %d", w.Code)
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
