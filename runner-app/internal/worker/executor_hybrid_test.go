package worker

import (
	"errors"
	"testing"

	"github.com/jamie-anson/project-beacon-runner/internal/hybrid"
)

func TestBuildHybridFailure_HTTP404(t *testing.T) {
	err := hybrid.NewHTTPError(404, "Not Found", "https://router/inference")
	failure, code := buildHybridFailure("us-east", "llama3", nil, err)

	if code != "ROUTER_HTTP_404" {
		t.Fatalf("expected code ROUTER_HTTP_404, got %s", code)
	}
	if failure["stage"] != "router_http_request" {
		t.Fatalf("expected stage router_http_request, got %v", failure["stage"])
	}
	if failure["http_status"] != 404 {
		t.Fatalf("expected http_status 404, got %v", failure["http_status"])
	}
	if failure["url"] != "https://router/inference" {
		t.Fatalf("unexpected url %v", failure["url"])
	}
	if failure["region"] != "us-east" {
		t.Fatalf("unexpected region %v", failure["region"])
	}
	if failure["model"] != "llama3" {
		t.Fatalf("unexpected model %v", failure["model"])
	}
}

func TestBuildHybridFailure_Timeout(t *testing.T) {
	baseErr := errors.New("context deadline exceeded")
	terr := hybrid.NewTimeoutError("timeout", baseErr)
	failure, code := buildHybridFailure("eu-west", "mistral", nil, terr)

	if code != "ROUTER_TIMEOUT" {
		t.Fatalf("expected code ROUTER_TIMEOUT, got %s", code)
	}
	if failure["transient"] != true {
		t.Fatalf("expected transient true, got %v", failure["transient"])
	}
	if failure["stage"] != "router_http_timeout" {
		t.Fatalf("expected stage router_http_timeout, got %v", failure["stage"])
	}
}

func TestBuildHybridFailure_WithRouterMetadata(t *testing.T) {
	resp := &hybrid.InferenceResponse{
		Success:      false,
		Error:        "provider error",
		ProviderUsed: "modal-us-east",
		Metadata: map[string]any{
			"provider_type": "modal",
			"region":        "us-east",
			"failure": map[string]any{
				"code":    "MODAL_IDLE",
				"message": "app stopped",
			},
		},
	}

	failure, code := buildHybridFailure("us-east", "qwen", resp, hybrid.NewRouterError("router failure"))

	if code != "ROUTER_ERROR" {
		t.Fatalf("expected code ROUTER_ERROR, got %s", code)
	}
	if failure["provider"] != "modal-us-east" {
		t.Fatalf("expected provider modal-us-east, got %v", failure["provider"])
	}
	if failure["provider_type"] != "modal" {
		t.Fatalf("expected provider_type modal, got %v", failure["provider_type"])
	}
	if failure["router_failure"] == nil {
		t.Fatalf("expected nested router_failure metadata")
	}
}
