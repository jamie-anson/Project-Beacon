package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestCreateDemand_RetryBackoff(t *testing.T) {
	var hits int32
	// Server that returns 500 twice, then 201 with JSON demandId
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/market-api/v1/demands" && r.Method == http.MethodPost {
			c := atomic.AddInt32(&hits, 1)
			if c <= 2 {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"transient"}`))
				return
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"demandId": "d-123"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := &YagnaRESTClient{ BaseURL: srv.URL, Timeout: 2 * time.Second, HTTPClient: srv.Client(), MarketBase: "/market-api/v1", ActivityBase: "/activity-api/v1" }
	ctx := context.Background()
	id, err := c.CreateDemand(ctx, DemandSpec{Constraints: "", Properties: map[string]any{}, Metadata: map[string]any{}})
	if err != nil {
		t.Fatalf("CreateDemand returned error: %v", err)
	}
	if id != "d-123" {
		t.Fatalf("expected demandId d-123, got %s", id)
	}
	if atomic.LoadInt32(&hits) < 3 {
		t.Fatalf("expected at least 3 attempts, got %d", hits)
	}
}

func TestProbe_SucceedsOnMarketPath(t *testing.T) {
	// Server responds 200 on /market-api/v1/demands
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/market-api/v1/demands" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"version":"test"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := &YagnaRESTClient{ BaseURL: srv.URL, Timeout: 200 * time.Millisecond, HTTPClient: srv.Client(), MarketBase: "/market-api/v1", ActivityBase: "/activity-api/v1" }
	hit, ver, err := c.Probe(context.Background())
	if err != nil {
		t.Fatalf("Probe returned error: %v", err)
	}
	if hit == "" {
		t.Fatalf("expected non-empty hit path")
	}
	if ver == nil {
		t.Fatalf("expected non-nil version map")
	}
}
