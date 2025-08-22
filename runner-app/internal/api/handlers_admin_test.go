package api

import (
    "bytes"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/jamie-anson/project-beacon-runner/internal/config"
    "github.com/jamie-anson/project-beacon-runner/internal/service"
)

// helper to build router with admin routes
func newAdminTestRouter() *gin.Engine {
    cfg := &config.Config{HTTPPort: ":8090"}
    // JobsService can be nil; admin routes don't use it
    return SetupRoutes(service.NewJobsService(nil), cfg)
}

func TestAdmin_Unauthorized_WhenNoTokenConfigured(t *testing.T) {
    t.Parallel()
    r := newAdminTestRouter()

    req := httptest.NewRequest(http.MethodGet, "/admin/flags", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusUnauthorized {
        t.Fatalf("expected 401 when ADMIN_TOKEN not set, got %d; body=%s", w.Code, w.Body.String())
    }
}

func TestAdmin_Unauthorized_WrongToken(t *testing.T) {
    t.Setenv("ADMIN_TOKEN", "super-secret")
    r := newAdminTestRouter()

    req := httptest.NewRequest(http.MethodGet, "/admin/config", nil)
    // Wrong token header
    req.Header.Set("X-Admin-Token", "wrong")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusUnauthorized {
        t.Fatalf("expected 401 for wrong token, got %d; body=%s", w.Code, w.Body.String())
    }
}

func TestAdmin_Authorized_CorrectToken_AllowsAccess(t *testing.T) {
    t.Setenv("ADMIN_TOKEN", "super-secret")
    r := newAdminTestRouter()

    req := httptest.NewRequest(http.MethodGet, "/admin/flags", nil)
    req.Header.Set("X-Admin-Token", "super-secret")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200 with correct token, got %d; body=%s", w.Code, w.Body.String())
    }
    if ct := w.Header().Get("Content-Type"); ct == "" {
        t.Fatalf("expected JSON response with content type, got none")
    }
}

func TestAdmin_UpdateFlags_InvalidJSON_Returns400(t *testing.T) {
    t.Setenv("ADMIN_TOKEN", "super-secret")
    r := newAdminTestRouter()

    req := httptest.NewRequest(http.MethodPut, "/admin/flags", bytes.NewBufferString("{ not-json }"))
    req.Header.Set("X-Admin-Token", "super-secret")
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusBadRequest {
        t.Fatalf("expected 400 on invalid JSON, got %d; body=%s", w.Code, w.Body.String())
    }
}
