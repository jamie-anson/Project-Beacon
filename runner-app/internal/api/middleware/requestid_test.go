package middleware

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
)

func TestRequestID_GeneratesWhenMissing(t *testing.T) {
    t.Parallel()
    r := gin.New()
    r.Use(RequestID())
    r.GET("/ping", func(c *gin.Context) { c.String(200, "ok") })

    w := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/ping", nil)
    r.ServeHTTP(w, req)

    if got := w.Header().Get("X-Request-ID"); got == "" {
        t.Fatalf("expected X-Request-ID header to be set")
    }
}

func TestRequestID_EchoesProvided(t *testing.T) {
    t.Parallel()
    r := gin.New()
    r.Use(RequestID())
    r.GET("/ping", func(c *gin.Context) { c.String(200, "ok") })

    w := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/ping", nil)
    req.Header.Set("X-Request-ID", "abc-123")
    r.ServeHTTP(w, req)

    if got := w.Header().Get("X-Request-ID"); got != "abc-123" {
        t.Fatalf("expected X-Request-ID to echo 'abc-123', got %q", got)
    }
}
