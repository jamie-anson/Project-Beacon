package metrics

import (
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/prometheus/client_golang/prometheus"
)

func TestGinMiddleware_DynamicPathAnd404(t *testing.T) {
    gin.SetMode(gin.TestMode)
    resetProm()
    r := gin.New()
    r.Use(GinMiddleware())

    r.GET("/ok", func(c *gin.Context) { c.Status(200) })
    r.GET("/item/:id", func(c *gin.Context) { c.Status(201) })

    // 200 on static path
    w := httptest.NewRecorder()
    r.ServeHTTP(w, httptest.NewRequest("GET", "/ok", nil))
    if w.Code != 200 {
        t.Fatalf("expected 200, got %d", w.Code)
    }

    // 201 on dynamic path; labels should use FullPath
    w2 := httptest.NewRecorder()
    r.ServeHTTP(w2, httptest.NewRequest("GET", "/item/42", nil))
    if w2.Code != 201 {
        t.Fatalf("expected 201, got %d", w2.Code)
    }

    // 404 on missing route; middleware still records
    w3 := httptest.NewRecorder()
    r.ServeHTTP(w3, httptest.NewRequest("GET", "/missing", nil))
    if w3.Code != 404 {
        t.Fatalf("expected 404, got %d", w3.Code)
    }

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
