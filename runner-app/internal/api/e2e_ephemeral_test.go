package api

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/config"
	"github.com/jamie-anson/project-beacon-runner/internal/serverbind"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
)

func TestE2E_EphemeralMode_WritesAddrFile_AndEndpointsRespond(t *testing.T) {
	// Use release mode to mimic non-debug; admin requires token
	gin.SetMode(gin.ReleaseMode)

	// Prepare config
	cfg := &config.Config{
		HTTPPort:          ":8090",
		PortStrategy:      "ephemeral",
		PortRangeStart:    8090,
		PortRangeEnd:      8099,
		AddrFile:          filepath.Join(t.TempDir(), ".runner-http.addr"),
		DatabaseURL:       "postgres://postgres:password@localhost:5433/beacon_runner?sslmode=disable",
		RedisURL:          "redis://localhost:6379",
		DBTimeout:         4 * time.Second,
		RedisTimeout:      2 * time.Second,
		WorkerFetchTimeout: 5 * time.Second,
		OutboxTick:        2 * time.Second,
		JobsQueueName:     "jobs",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config validate: %v", err)
	}

	// Router and server
	jobsService := &service.JobsService{}
	r := SetupRoutes(jobsService, cfg, nil)
	srv := &http.Server{Handler: r}

	// Bind ephemeral and write addr file
	ln, resolved, err := serverbind.ResolveAndListen(cfg.PortStrategy, cfg.HTTPPort, cfg.PortRangeStart, cfg.PortRangeEnd)
	if err != nil {
		t.Fatalf("ResolveAndListen failed: %v", err)
	}
	defer ln.Close()
	cfg.ResolvedAddr = resolved
	if err := serverbind.WriteAddrFile(cfg.AddrFile, resolved); err != nil {
		t.Fatalf("WriteAddrFile failed: %v", err)
	}

	// Start server
	done := make(chan struct{})
	go func() {
		_ = srv.Serve(ln)
		close(done)
	}()
	defer func() { _ = srv.Close(); <-done }()

	// Read addr file and form base URL
	b, err := os.ReadFile(cfg.AddrFile)
	if err != nil {
		t.Fatalf("read addr file: %v", err)
	}
	addrFromFile := string(b)
	_, port, err := net.SplitHostPort(addrFromFile)
	if err != nil || port == "" || port == "0" {
		t.Fatalf("invalid addr in file: %q err=%v", addrFromFile, err)
	}
	base := "http://localhost:" + port

	// Hit /health/live (no auth required)
	resp, err := http.Get(base + "/health/live")
	if err != nil {
		t.Fatalf("GET /health/live: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("/health/live status=%d body=%s", resp.StatusCode, string(body))
	}

	// Hit /admin/port with RBAC Bearer token
	_ = os.Setenv("ADMIN_TOKENS", "test-token")
	req, _ := http.NewRequest("GET", base+"/admin/port", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /admin/port: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp2.Body)
		t.Fatalf("/admin/port status=%d body=%s", resp2.StatusCode, string(body))
	}

	// Hit /admin/hints with token and validate contents
	req3, _ := http.NewRequest("GET", base+"/admin/hints", nil)
	req3.Header.Set("Authorization", "Bearer test-token")
	resp3, err := http.DefaultClient.Do(req3)
	if err != nil {
		t.Fatalf("GET /admin/hints: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp3.Body)
		t.Fatalf("/admin/hints status=%d body=%s", resp3.StatusCode, string(body))
	}
	var hints struct {
		BaseURL      string `json:"base_url"`
		ResolvedAddr string `json:"resolved_addr"`
		Strategy     string `json:"strategy"`
	}
	if err := json.NewDecoder(resp3.Body).Decode(&hints); err != nil {
		t.Fatalf("decode /admin/hints: %v", err)
	}
	if hints.BaseURL != base {
		t.Fatalf("/admin/hints base_url mismatch: got %q want %q", hints.BaseURL, base)
	}
	if hints.ResolvedAddr != addrFromFile {
		t.Fatalf("/admin/hints resolved_addr mismatch: got %q want %q", hints.ResolvedAddr, addrFromFile)
	}
	if hints.Strategy != "ephemeral" {
		t.Fatalf("/admin/hints strategy mismatch: got %q want %q", hints.Strategy, "ephemeral")
	}
}
