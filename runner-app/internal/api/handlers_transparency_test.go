package api

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/jamie-anson/project-beacon-runner/internal/transparency"
)

func routerWithTransparency() *gin.Engine {
    gin.SetMode(gin.ReleaseMode)
    r := gin.New()
    r.Use(gin.Recovery())
    h := NewTransparencyHandler()
    r.GET("/root", h.GetRoot)
    r.GET("/proof", h.GetProof)
    // Intentionally do not register bundle route to avoid network paths in these tests
    return r
}

func TestTransparency_GetRoot_OK(t *testing.T) {
    r := routerWithTransparency()

    req := httptest.NewRequest(http.MethodGet, "/root", nil)
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
    }
    if w.Body.Len() == 0 {
        t.Fatalf("expected non-empty body")
    }
}

func TestTransparency_GetProof_MissingIndex(t *testing.T) {
    r := routerWithTransparency()

    req := httptest.NewRequest(http.MethodGet, "/proof", nil)
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    if w.Code != http.StatusBadRequest {
        t.Fatalf("expected 400, got %d; body=%s", w.Code, w.Body.String())
    }
}

func TestTransparency_GetProof_BadIndex_NonInteger(t *testing.T) {
    r := routerWithTransparency()

    req := httptest.NewRequest(http.MethodGet, "/proof?index=abc", nil)
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    if w.Code != http.StatusBadRequest {
        t.Fatalf("expected 400, got %d; body=%s", w.Code, w.Body.String())
    }
}

func TestTransparency_GetProof_BadIndex_Negative(t *testing.T) {
    r := routerWithTransparency()

    req := httptest.NewRequest(http.MethodGet, "/proof?index=-1", nil)
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    if w.Code != http.StatusBadRequest {
        t.Fatalf("expected 400, got %d; body=%s", w.Code, w.Body.String())
    }
}

func TestTransparency_GetProof_NotFound(t *testing.T) {
    // Ensure DefaultWriter has no entries so any non-negative index will be not found
    // Reset state by creating a new writer (package variable), but since DefaultWriter
    // is not easily resettable here, target a high index value that's unlikely to exist.
    // We keep this hermetic without network or DB.
    r := routerWithTransparency()

    req := httptest.NewRequest(http.MethodGet, "/proof?index=0", nil)
    w := httptest.NewRecorder()

    // To be robust, ensure the writer is empty at test start
    // If not empty due to other tests, choose a large index
    if len(transparency.DefaultWriter.Entries()) > 0 {
        req = httptest.NewRequest(http.MethodGet, "/proof?index=999999", nil)
    }

    r.ServeHTTP(w, req)

    if w.Code != http.StatusNotFound {
        t.Fatalf("expected 404 for missing proof, got %d; body=%s", w.Code, w.Body.String())
    }
}
