package hybrid

import (
    "os"
    "testing"
    "time"
)

func TestNew_TimeoutFromHYBRID_ROUTER_TIMEOUT(t *testing.T) {
    old := os.Getenv("HYBRID_ROUTER_TIMEOUT")
    defer os.Setenv("HYBRID_ROUTER_TIMEOUT", old)
    os.Setenv("HYBRID_ROUTER_TIMEOUT", "5")

    c := New("http://example.com")
    if got, want := c.httpClient.Timeout, 5*time.Second; got != want {
        t.Fatalf("Timeout = %v, want %v", got, want)
    }
}

func TestNew_TimeoutFromHYBRID_TIMEOUT_Fallback(t *testing.T) {
    old1 := os.Getenv("HYBRID_ROUTER_TIMEOUT")
    old2 := os.Getenv("HYBRID_TIMEOUT")
    defer os.Setenv("HYBRID_ROUTER_TIMEOUT", old1)
    defer os.Setenv("HYBRID_TIMEOUT", old2)
    os.Unsetenv("HYBRID_ROUTER_TIMEOUT")
    os.Setenv("HYBRID_TIMEOUT", "7")

    c := New("http://example.com")
    if got, want := c.httpClient.Timeout, 7*time.Second; got != want {
        t.Fatalf("Timeout = %v, want %v", got, want)
    }
}
