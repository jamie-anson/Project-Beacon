package main

import (
	"context"
	"os"
	"testing"
)

func TestInitOpenTelemetry_NoEndpoint(t *testing.T) {
	// Ensure env var is unset
	old := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	_ = os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	t.Cleanup(func() {
		if old != "" {
			_ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", old)
		}
	})

	tp, closeFn := initOpenTelemetry(context.Background(), "runner-test")
	if tp != nil || closeFn != nil {
		t.Fatalf("expected nil tracer provider and nil close func when endpoint unset")
	}
}
