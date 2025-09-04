package middleware

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
)

func routerWithCORS() *gin.Engine {
    gin.SetMode(gin.ReleaseMode)
    r := gin.New()
    r.Use(gin.Recovery())
    r.Use(RequestID())
    r.Use(CORS())
    // simple endpoint
    r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
    return r
}

func TestCORS_Preflight_AllowedOrigin(t *testing.T) {
    r := routerWithCORS()

    origin := "http://localhost:3000"
    req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
    req.Header.Set("Origin", origin)
    req.Header.Set("Access-Control-Request-Method", "GET")

    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusNoContent {
        t.Fatalf("expected 204 for preflight, got %d; body=%s", w.Code, w.Body.String())
    }

    if got := w.Header().Get("Access-Control-Allow-Origin"); got != origin {
        t.Fatalf("expected ACAO=%s, got %q", origin, got)
    }

    // Common CORS headers should be present
    mustHave := []string{
        "Access-Control-Allow-Methods",
        "Access-Control-Allow-Headers",
        "Access-Control-Expose-Headers",
        "Access-Control-Allow-Credentials",
        "Access-Control-Max-Age",
    }
    for _, h := range mustHave {
        if w.Header().Get(h) == "" {
            t.Fatalf("expected header %s to be set", h)
        }
    }
}

func TestCORS_Preflight_DisallowedOrigin(t *testing.T) {
    r := routerWithCORS()

    origin := "https://example.com"
    req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
    req.Header.Set("Origin", origin)
    req.Header.Set("Access-Control-Request-Method", "GET")

    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusNoContent {
        t.Fatalf("expected 204 for preflight, got %d; body=%s", w.Code, w.Body.String())
    }

    if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
        t.Fatalf("expected no ACAO for disallowed origin, got %q", got)
    }

    // Other CORS headers should still be present
    mustHave := []string{
        "Access-Control-Allow-Methods",
        "Access-Control-Allow-Headers",
        "Access-Control-Expose-Headers",
        "Access-Control-Allow-Credentials",
        "Access-Control-Max-Age",
    }
    for _, h := range mustHave {
        if w.Header().Get(h) == "" {
            t.Fatalf("expected header %s to be set", h)
        }
    }
}

func TestCORS_SimpleRequest_AllowedOrigin(t *testing.T) {
    r := routerWithCORS()

    origin := "http://127.0.0.1:8080"
    req := httptest.NewRequest(http.MethodGet, "/ping", nil)
    req.Header.Set("Origin", origin)

    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
    }

    if got := w.Header().Get("Access-Control-Allow-Origin"); got != origin {
        t.Fatalf("expected ACAO=%s, got %q", origin, got)
    }
}

func TestCORS_SimpleRequest_NoOrigin(t *testing.T) {
    r := routerWithCORS()

    req := httptest.NewRequest(http.MethodGet, "/ping", nil)

    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
    }

    if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
        t.Fatalf("expected no ACAO without Origin header, got %q", got)
    }
}
