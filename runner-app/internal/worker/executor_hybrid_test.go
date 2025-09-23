package worker

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/jamie-anson/project-beacon-runner/internal/hybrid"
    models "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func newMinimalSpec() *models.JobSpec {
    return &models.JobSpec{Benchmark: models.BenchmarkSpec{Input: models.InputSpec{Type: "prompt", Data: map[string]any{"prompt": "Who are you?"}}}}
}

func TestHybridExecutor_UnknownModelError_Propagates(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost || r.URL.Path != "/inference" { http.NotFound(w, r); return }
        w.Header().Set("Content-Type", "application/json"); w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte(`{"success": false, "error": "Unknown model: llama-3.2-1b"}`))
    }))
    defer srv.Close()

    exec := NewHybridExecutor(hybrid.New(srv.URL))
    spec := newMinimalSpec()

    providerID, status, outJSON, receiptJSON, err := exec.Execute(context.Background(), spec, "EU")
    if err == nil || status != "failed" || providerID != "" || len(receiptJSON) != 0 { t.Fatalf("unexpected success or fields") }
    var out map[string]any; _ = json.Unmarshal(outJSON, &out); if _, ok := out["router_error"]; !ok { t.Fatalf("missing router_error in output") }
}

func TestHybridExecutor_Success_MapsFields(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost || r.URL.Path != "/inference" { http.NotFound(w, r); return }
        w.Header().Set("Content-Type", "application/json"); w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte(`{"success": true, "response": "hello", "provider_used": "modal-eu-west"}`))
    }))
    defer srv.Close()

    exec := NewHybridExecutor(hybrid.New(srv.URL))
    spec := newMinimalSpec()

    providerID, status, outJSON, receiptJSON, err := exec.Execute(context.Background(), spec, "EU")
    if err != nil || status != "completed" || providerID != "modal-eu-west" || len(receiptJSON) != 0 { t.Fatalf("unexpected fields or error") }
    var out map[string]any; _ = json.Unmarshal(outJSON, &out); if out["response"] != "hello" { t.Fatalf("bad response") }
}

func TestHybridExecutor_SendsRegionPreference(t *testing.T) {
    type Posted struct {
        RegionPreference string `json:"region_preference"`
    }
    var seen Posted

    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost || r.URL.Path != "/inference" { http.NotFound(w, r); return }
        defer r.Body.Close()
        _ = json.NewDecoder(r.Body).Decode(&seen)
        w.Header().Set("Content-Type", "application/json"); w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte(`{"success": true, "response": "ok", "provider_used": "modal-eu-west"}`))
    }))
    defer srv.Close()

    exec := NewHybridExecutor(hybrid.New(srv.URL))
    spec := newMinimalSpec()
    _, _, _, _, err := exec.Execute(context.Background(), spec, "EU")
    if err != nil { t.Fatalf("unexpected error: %v", err) }
    if seen.RegionPreference != "eu-west" {
        t.Fatalf("region_preference sent = %q, want %q", seen.RegionPreference, "eu-west")
    }
}
