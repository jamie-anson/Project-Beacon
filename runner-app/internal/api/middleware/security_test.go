package middleware

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
)

func routerWithSecurity() *gin.Engine {
    gin.SetMode(gin.ReleaseMode)
    r := gin.New()
    r.Use(gin.Recovery())
    r.Use(RequestID())
    r.Use(SecurityHeaders())
    r.Use(RateLimiting())
    // simple endpoint
    r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
    return r
}

func TestSecurityHeaders_PresentOnResponse(t *testing.T) {
    r := routerWithSecurity()
    req := httptest.NewRequest(http.MethodGet, "/ping", nil)
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
    }

    // Headers expected (HSTS only on TLS, so skip that one)
    checks := []string{
        "X-Content-Type-Options",
        "X-XSS-Protection",
        "X-Frame-Options",
        "Content-Security-Policy",
        "Referrer-Policy",
        "Permissions-Policy",
    }
    for _, h := range checks {
        if got := w.Header().Get(h); got == "" {
            t.Fatalf("expected header %s to be set", h)
        }
    }
}

func TestRateLimiting_Returns429_AfterLimit(t *testing.T) {
    r := routerWithSecurity()

    // Make 65 requests; limit is 60. Expect final ones to be 429.
    var lastCode int
    for i := 0; i < 65; i++ {
        req := httptest.NewRequest(http.MethodGet, "/ping", nil)
        // Fix client IP so the bucket is shared
        req.RemoteAddr = "203.0.113.1:12345"
        w := httptest.NewRecorder()
        r.ServeHTTP(w, req)
        lastCode = w.Code
        // On first request, should include rate limit headers as well
        if i == 0 {
            if w.Header().Get("X-RateLimit-Limit") == "" || w.Header().Get("X-RateLimit-Remaining") == "" || w.Header().Get("X-RateLimit-Reset") == "" {
                t.Fatalf("expected rate limit headers on success response")
            }
        }
        if i >= 60 {
            if w.Code != http.StatusTooManyRequests {
                t.Fatalf("expected 429 after exceeding limit, got %d at i=%d", w.Code, i)
            }
            if w.Header().Get("X-RateLimit-Limit") != "60" {
                t.Fatalf("expected X-RateLimit-Limit=60, got %s", w.Header().Get("X-RateLimit-Limit"))
            }
            if w.Header().Get("X-RateLimit-Remaining") != "0" {
                t.Fatalf("expected X-RateLimit-Remaining=0 at 429, got %s", w.Header().Get("X-RateLimit-Remaining"))
            }
            if w.Header().Get("X-RateLimit-Reset") == "" {
                t.Fatalf("expected X-RateLimit-Reset to be set")
            }
        }
    }

    if lastCode != http.StatusTooManyRequests {
        t.Fatalf("expected final response to be 429, got %d", lastCode)
    }
}
