package ipfs

import (
    "context"
    "encoding/json"
    "io"
    "net/http"
    "net/http/httptest"
    "os"
    "strings"
    "testing"
)

func TestNewClient_DefaultsAndGatewayURL(t *testing.T) {
    c := NewClient(Config{})
    if c == nil {
        t.Fatal("NewClient returned nil")
    }
    // Default gateway should be used when empty
    url := c.GetGatewayURL("abc")
    if !strings.HasSuffix(url, "/ipfs/abc") {
        t.Fatalf("unexpected gateway url: %s", url)
    }
}

func TestPinWithStoracha_NoURL_NoOp(t *testing.T) {
    t.Setenv("STORACHA_PIN_URL", "")
    if err := pinWithStoracha(context.Background(), "cid123"); err != nil {
        t.Fatalf("expected nil when no pin url set, got %v", err)
    }
}

func TestPinWithStoracha_SendsRequest_OK(t *testing.T) {
    var seen struct {
        Method string
        Auth   string
        Body   map[string]string
    }
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        seen.Method = r.Method
        seen.Auth = r.Header.Get("Authorization")
        b, _ := io.ReadAll(r.Body)
        _ = json.Unmarshal(b, &seen.Body)
        w.WriteHeader(200)
    }))
    defer srv.Close()

    t.Setenv("STORACHA_PIN_URL", srv.URL)
    t.Setenv("STORACHA_TOKEN", "t0k")

    if err := pinWithStoracha(context.Background(), "cid123"); err != nil {
        t.Fatalf("pinWithStoracha error: %v", err)
    }
    if seen.Method != http.MethodPost {
        t.Fatalf("expected POST, got %s", seen.Method)
    }
    if !strings.HasPrefix(seen.Auth, "Bearer ") {
        t.Fatalf("expected Authorization header, got %q", seen.Auth)
    }
    if seen.Body["cid"] != "cid123" {
        t.Fatalf("unexpected body: %+v", seen.Body)
    }
}

func TestPinWithStoracha_ErrorStatus(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        http.Error(w, "nope", http.StatusInternalServerError)
    }))
    defer srv.Close()

    t.Setenv("STORACHA_PIN_URL", srv.URL)
    os.Unsetenv("STORACHA_TOKEN")

    if err := pinWithStoracha(context.Background(), "cid123"); err == nil {
        t.Fatalf("expected error on non-2xx response")
    }
}
