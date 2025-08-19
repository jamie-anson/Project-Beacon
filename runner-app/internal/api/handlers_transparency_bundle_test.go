package api

import (
    "context"
    "errors"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/jamie-anson/project-beacon-runner/internal/ipfs"
)

type fakeBundleStorage struct{
    bundle *ipfs.Bundle
    err error
}

func (f fakeBundleStorage) FetchBundle(ctx context.Context, cid string) (*ipfs.Bundle, error) {
    if f.err != nil { return nil, f.err }
    return f.bundle, nil
}

func routerWithBundle() *gin.Engine {
    gin.SetMode(gin.ReleaseMode)
    r := gin.New()
    r.Use(gin.Recovery())
    h := NewTransparencyHandler()
    r.GET("/bundles/:cid", h.GetBundle)
    return r
}

func TestTransparency_Bundle_FetchFailure_Returns502(t *testing.T) {
    // Override factory
    prev := ipfsStorageFactory
    ipfsStorageFactory = func(_ *ipfs.Client) bundleFetcher {
        return fakeBundleStorage{err: errors.New("ipfs fetch failed")}
    }
    defer func(){ ipfsStorageFactory = prev }()

    r := routerWithBundle()
    req := httptest.NewRequest(http.MethodGet, "/bundles/QmBadCid", nil)
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    if w.Code != http.StatusBadGateway {
        t.Fatalf("expected 502, got %d; body=%s", w.Code, w.Body.String())
    }
}

func TestTransparency_Bundle_Success_Returns200_WithGatewayURL(t *testing.T) {
    // Prepare a minimal bundle
    b := &ipfs.Bundle{ JobID: "job-1", ExecutionID: "exe-1" }

    // Override factory to return success
    prev := ipfsStorageFactory
    ipfsStorageFactory = func(_ *ipfs.Client) bundleFetcher {
        return fakeBundleStorage{bundle: b}
    }
    defer func(){ ipfsStorageFactory = prev }()

    r := routerWithBundle()
    cid := "QmGoodCid"
    req := httptest.NewRequest(http.MethodGet, "/bundles/"+cid, nil)
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
    }
    // Basic shape checks
    if w.Body.Len() == 0 {
        t.Fatalf("expected JSON body")
    }
    // Gateway URL comes from ipfs.NewFromEnv default: http://localhost:8080/ipfs/<cid>
    expectedGatewayPrefix := "\"gateway_url\":\"http://localhost:8080/ipfs/" + cid
    if !contains(w.Body.String(), expectedGatewayPrefix) {
        t.Fatalf("expected gateway_url with cid %s, body=%s", cid, w.Body.String())
    }
}

// contains is a tiny helper to avoid importing strings and to keep code minimal.
func contains(s, sub string) bool {
    return len(s) >= len(sub) && (func() bool { for i:=0; i+len(sub) <= len(s); i++ { if s[i:i+len(sub)] == sub { return true } }; return false })()
}
