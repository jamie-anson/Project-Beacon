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

// newAdminTestRouterV2 builds a router with admin auth enabled for integration-style tests.
func newAdminTestRouterV2() *gin.Engine {
	cfg := &config.Config{
		HTTPPort:   ":8090",
		AdminToken: "super-secret",
	}
	return SetupRoutes(service.NewJobsService(nil), cfg, nil)
}

func TestAdminV2_Unauthorized_NoToken(t *testing.T) {
	t.Parallel()
	r := newAdminTestRouterV2()

	req := httptest.NewRequest(http.MethodGet, "/admin/flags", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when no auth provided, got %d; body=%s", w.Code, w.Body.String())
	}
}

func TestAdminV2_Unauthorized_WrongToken(t *testing.T) {
	t.Parallel()
	r := newAdminTestRouterV2()

	req := httptest.NewRequest(http.MethodGet, "/admin/config", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for wrong token, got %d; body=%s", w.Code, w.Body.String())
	}
}

func TestAdminV2_Authorized_CorrectToken(t *testing.T) {
	t.Parallel()
	r := newAdminTestRouterV2()

	req := httptest.NewRequest(http.MethodGet, "/admin/flags", nil)
	req.Header.Set("Authorization", "Bearer super-secret")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with correct token, got %d; body=%s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct == "" {
		t.Fatalf("expected JSON response with content type, got none")
	}
}

func TestAdminV2_UpdateFlags_InvalidJSON(t *testing.T) {
	t.Parallel()
	r := newAdminTestRouterV2()

	req := httptest.NewRequest(http.MethodPut, "/admin/flags", bytes.NewBufferString("{ not-json }"))
	req.Header.Set("Authorization", "Bearer super-secret")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on invalid JSON, got %d; body=%s", w.Code, w.Body.String())
	}
}
