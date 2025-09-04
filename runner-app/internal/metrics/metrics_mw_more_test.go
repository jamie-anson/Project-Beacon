package metrics

import (
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/prometheus/client_golang/prometheus"
)

func TestGinMiddleware_PostAnd500(t *testing.T) {
    gin.SetMode(gin.TestMode)
    resetProm()
    r := gin.New()
    r.Use(GinMiddleware())

    r.POST("/create", func(c *gin.Context) { c.Status(204) })
    r.GET("/err", func(c *gin.Context) { c.AbortWithStatus(500) })

    // POST 204
    w := httptest.NewRecorder()
    r.ServeHTTP(w, httptest.NewRequest("POST", "/create", nil))
    if w.Code != 204 {
        t.Fatalf("expected 204, got %d", w.Code)
    }

    // GET 500
    w2 := httptest.NewRecorder()
    r.ServeHTTP(w2, httptest.NewRequest("GET", "/err", nil))
    if w2.Code != 500 {
        t.Fatalf("expected 500, got %d", w2.Code)
    }

    // Ensure metrics have been recorded (non-brittle check)
    fams, err := prometheus.DefaultGatherer.Gather()
    if err != nil {
        t.Fatalf("gather error: %v", err)
    }
    found := false
    for _, mf := range fams {
        if mf.GetName() == "http_requests_total" && len(mf.Metric) > 0 {
            found = true
            break
        }
    }
    if !found {
        t.Fatalf("expected http_requests_total samples to be present")
    }
}
