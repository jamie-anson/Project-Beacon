package api

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/jamie-anson/project-beacon-runner/internal/ipfs"
)

// Transparency: root returns 200 with X-Request-ID header and a root field
func TestContract_Transparency_Root_200(t *testing.T) {
    t.Parallel()
    r := newTestRouter()
    req := httptest.NewRequest(http.MethodGet, "/api/v1/transparency/root", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusOK { t.Fatalf("want 200, got %d; body=%s", w.Code, w.Body.String()) }
    if w.Header().Get("X-Request-ID") == "" { t.Fatalf("missing X-Request-ID header") }
    var body map[string]any
    if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil { t.Fatalf("unmarshal: %v", err) }
    if _, ok := body["root"].(string); !ok { t.Fatalf("expected root string, got %v", body) }
}

// Transparency: proof validations and not found
func TestContract_Transparency_Proof_400_MissingIndex(t *testing.T) {
    t.Parallel()
    r := newTestRouter()
    req := httptest.NewRequest(http.MethodGet, "/api/v1/transparency/proof", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusBadRequest { t.Fatalf("want 400, got %d", w.Code) }
    if w.Header().Get("X-Request-ID") == "" { t.Fatalf("missing X-Request-ID header") }
    jsonEqual(t, w.Body.Bytes(), mustReadGolden(t, "transparency_proof_missing_index_400.json"))
}

func TestContract_Transparency_Proof_400_BadIndex(t *testing.T) {
    t.Parallel()
    r := newTestRouter()
    req := httptest.NewRequest(http.MethodGet, "/api/v1/transparency/proof?index=abc", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusBadRequest { t.Fatalf("want 400, got %d", w.Code) }
    if w.Header().Get("X-Request-ID") == "" { t.Fatalf("missing X-Request-ID header") }
    jsonEqual(t, w.Body.Bytes(), mustReadGolden(t, "transparency_proof_bad_index_400.json"))
}

func TestContract_Transparency_Proof_404_NotFound(t *testing.T) {
    t.Parallel()
    r := newTestRouter()
    req := httptest.NewRequest(http.MethodGet, "/api/v1/transparency/proof?index=999999", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusNotFound { t.Fatalf("want 404, got %d; body=%s", w.Code, w.Body.String()) }
    if w.Header().Get("X-Request-ID") == "" { t.Fatalf("missing X-Request-ID header") }
    jsonEqual(t, w.Body.Bytes(), mustReadGolden(t, "transparency_proof_404.json"))
}

// Transparency: bundle fetch error -> 502 with error body and X-Request-ID
func TestContract_Transparency_Bundle_502(t *testing.T) {
    t.Parallel()
    // Fake storage that always errors
    oldFactory := ipfsStorageFactory
    ipfsStorageFactory = func(_ *ipfs.Client) bundleFetcher {
        return &fakeFetcher{}
    }
    defer func() { ipfsStorageFactory = oldFactory }()

    r := newTestRouter()
    req := httptest.NewRequest(http.MethodGet, "/api/v1/transparency/bundles/bafybadcid", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusBadGateway { t.Fatalf("want 502, got %d; body=%s", w.Code, w.Body.String()) }
    if w.Header().Get("X-Request-ID") == "" { t.Fatalf("missing X-Request-ID header") }
    jsonEqual(t, w.Body.Bytes(), mustReadGolden(t, "transparency_bundle_502.json"))
}

// fakeFetcher implements bundleFetcher for tests
type fakeFetcher struct{}

func (f *fakeFetcher) FetchBundle(ctx context.Context, cid string) (*ipfs.Bundle, error) {
    return nil, fmt.Errorf("ipfs fetch failed")
}
