package api

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/jamie-anson/project-beacon-runner/internal/config"
    "github.com/jamie-anson/project-beacon-runner/internal/flags"
)

// Admin: forbidden when no Authorization bearer is provided and ADMIN_TOKENS is unset
func TestContract_Admin_Unauthorized_403(t *testing.T) {
    // Ensure ADMIN_TOKENS is unset
    old := os.Getenv("ADMIN_TOKENS")
    _ = os.Unsetenv("ADMIN_TOKENS")
    t.Cleanup(func(){ if old != "" { _ = os.Setenv("ADMIN_TOKENS", old) } })

    r := newTestRouter()
    req := httptest.NewRequest(http.MethodGet, "/admin/flags", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusForbidden { t.Fatalf("want 403, got %d; body=%s", w.Code, w.Body.String()) }
    if w.Header().Get("X-Request-ID") == "" { t.Fatalf("missing X-Request-ID header") }
}

// Admin: flags GET/PUT with token
func TestContract_Admin_Flags_GetPut_200(t *testing.T) {
    // Set admin token via ADMIN_TOKENS
    old := os.Getenv("ADMIN_TOKENS")
    _ = os.Setenv("ADMIN_TOKENS", "secret")
    t.Cleanup(func(){ if old == "" { _ = os.Unsetenv("ADMIN_TOKENS") } else { _ = os.Setenv("ADMIN_TOKENS", old) } })

    // Reset flags and restore after
    orig := flags.Get()
    t.Cleanup(func(){ flags.Set(orig) })
    flags.Set(flags.Flags{EnableCache:true, EnableWebSockets:true, ReadOnlyMode:false})

    r := newTestRouter()

    // GET /admin/flags
    req := httptest.NewRequest(http.MethodGet, "/admin/flags", nil)
    req.Header.Set("Authorization", "Bearer secret")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusOK { t.Fatalf("want 200, got %d; body=%s", w.Code, w.Body.String()) }
    if w.Header().Get("X-Request-ID") == "" { t.Fatalf("missing X-Request-ID header") }

    var got flags.Flags
    if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil { t.Fatalf("unmarshal flags: %v", err) }

    // PUT /admin/flags toggle read_only_mode
    body := bytes.NewBufferString(`{"read_only_mode":true}`)
    req2 := httptest.NewRequest(http.MethodPut, "/admin/flags", body)
    req2.Header.Set("Authorization", "Bearer secret")
    req2.Header.Set("Content-Type", "application/json")
    w2 := httptest.NewRecorder()
    r.ServeHTTP(w2, req2)
    if w2.Code != http.StatusOK { t.Fatalf("want 200, got %d; body=%s", w2.Code, w2.Body.String()) }
    if w2.Header().Get("X-Request-ID") == "" { t.Fatalf("missing X-Request-ID header on PUT") }
    var got2 flags.Flags
    if err := json.Unmarshal(w2.Body.Bytes(), &got2); err != nil { t.Fatalf("unmarshal flags put: %v", err) }
    if !got2.ReadOnlyMode { t.Fatalf("expected read_only_mode=true after update, got %v", got2) }
}

// Admin: config 403 without header and 200 with bearer token
func TestContract_Admin_Config_Auth(t *testing.T) {
    old := os.Getenv("ADMIN_TOKENS")
    _ = os.Setenv("ADMIN_TOKENS", "secret")
    t.Cleanup(func(){ if old == "" { _ = os.Unsetenv("ADMIN_TOKENS") } else { _ = os.Setenv("ADMIN_TOKENS", old) } })

    cfg := &config.Config{
        HTTPPort: "8090",
        DatabaseURL: "postgres://user:pass@localhost:5432/db",
        RedisURL: "redis://default:redpass@localhost:6379/0",
        PortStrategy: "fixed",
        ResolvedAddr: "127.0.0.1:8090",
        JobsQueueName: "jobs",
        IPFSURL: "http://127.0.0.1:5001",
        IPFSGateway: "https://ipfs.io/ipfs/",
        YagnaURL: "http://127.0.0.1:7465",
    }
    r := SetupRoutes(nil, cfg, nil)

    // 403 when header missing
    req := httptest.NewRequest(http.MethodGet, "/admin/config", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusForbidden { t.Fatalf("want 403, got %d", w.Code) }
    if w.Header().Get("X-Request-ID") == "" { t.Fatalf("missing X-Request-ID header on 401") }

    // 200 with bearer token
    req2 := httptest.NewRequest(http.MethodGet, "/admin/config", nil)
    req2.Header.Set("Authorization", "Bearer secret")
    w2 := httptest.NewRecorder()
    r.ServeHTTP(w2, req2)
    if w2.Code != http.StatusOK { t.Fatalf("want 200, got %d; body=%s", w2.Code, w2.Body.String()) }
    if w2.Header().Get("X-Request-ID") == "" { t.Fatalf("missing X-Request-ID header on 200") }
    var conf map[string]any
    if err := json.Unmarshal(w2.Body.Bytes(), &conf); err != nil { t.Fatalf("unmarshal config: %v", err) }
    // Verify redaction occurred
    if v, _ := conf["database_url"].(string); v == "" || v == cfg.DatabaseURL || v == "postgres://user:@localhost:5432/db" { t.Fatalf("expected redacted database_url, got %q", v) }
    if v, _ := conf["redis_url"].(string); v == "" || v == cfg.RedisURL || v == "redis://default:@localhost:6379/0" { t.Fatalf("expected redacted redis_url, got %q", v) }
}

// Admin: in debug mode, /admin/port and /admin/hints are public
func TestContract_Admin_PortHints_Debug_Public(t *testing.T) {
    // Force debug mode for this test
    oldMode := gin.Mode()
    gin.SetMode(gin.DebugMode)
    t.Cleanup(func(){ gin.SetMode(oldMode) })

    // ADMIN_TOKEN unset should not matter for public endpoints in debug
    old := os.Getenv("ADMIN_TOKEN")
    _ = os.Unsetenv("ADMIN_TOKEN")
    t.Cleanup(func(){ if old != "" { _ = os.Setenv("ADMIN_TOKEN", old) } })

    cfg := &config.Config{HTTPPort:"8090", PortStrategy:"fixed", ResolvedAddr:"0.0.0.0:8090"}
    r := SetupRoutes(nil, cfg, nil)

    // /admin/port
    req := httptest.NewRequest(http.MethodGet, "/admin/port", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusOK { t.Fatalf("want 200, got %d; body=%s", w.Code, w.Body.String()) }
    if w.Header().Get("X-Request-ID") == "" { t.Fatalf("missing X-Request-ID header") }

    // /admin/hints
    req2 := httptest.NewRequest(http.MethodGet, "/admin/hints", nil)
    w2 := httptest.NewRecorder()
    r.ServeHTTP(w2, req2)
    if w2.Code != http.StatusOK { t.Fatalf("want 200, got %d; body=%s", w2.Code, w2.Body.String()) }
    if w2.Header().Get("X-Request-ID") == "" { t.Fatalf("missing X-Request-ID header on hints") }
}
