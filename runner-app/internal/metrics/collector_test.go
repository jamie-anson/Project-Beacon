package metrics

import (
    "context"
    "testing"
    "time"
    "github.com/prometheus/client_golang/prometheus"
)

func TestCollector_Recorders_NoPanic(t *testing.T) {
    // Reset registry to avoid cross-test duplicates
    reg := prometheus.NewRegistry()
    prometheus.DefaultRegisterer = reg
    prometheus.DefaultGatherer = reg
    c := NewCollector(nil, nil)

    // IPFS recorders
    c.RecordIPFSBundleCreated(2048)
    c.RecordIPFSPinOperation()
    c.RecordIPFSGatewayLatency(150 * time.Millisecond)

    // Transparency recorders
    c.RecordTransparencyLogEntry()
    c.RecordTransparencyVerificationFailure()
    c.RecordAnchorAttempt("ethereum", 1*time.Second, true)
    c.RecordAnchorAttempt("timestamp", 500*time.Millisecond, false)

    // Storage analytics
    c.RecordCompressionRatio(0.42)
}

func TestCollector_Update_NoRepo_NoError(t *testing.T) {
    // Reset registry to avoid cross-test duplicates and ensure metrics are initialized
    reg := prometheus.NewRegistry()
    prometheus.DefaultRegisterer = reg
    prometheus.DefaultGatherer = reg
    c := NewCollector(nil, nil)
    ctx := context.Background()

    if err := c.UpdateStorageMetrics(ctx); err != nil {
        t.Fatalf("UpdateStorageMetrics with nil repo returned err: %v", err)
    }
    if err := c.UpdateTransparencyMetrics(ctx); err != nil {
        t.Fatalf("UpdateTransparencyMetrics with nil repo returned err: %v", err)
    }
    if err := c.UpdateRetentionMetrics(ctx); err != nil {
        t.Fatalf("UpdateRetentionMetrics returned err: %v", err)
    }
}
