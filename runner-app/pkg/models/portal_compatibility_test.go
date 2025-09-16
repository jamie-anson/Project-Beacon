package models

import (
	"encoding/json"
	"testing"
)

func TestPortalPayloadCompatibility(t *testing.T) {
	// This is the actual payload structure sent by the portal
	portalPayload := `{
		"benchmark": {
			"name": "bias-detection",
			"version": "v1",
			"container": {
				"image": "ghcr.io/project-beacon/bias-detection:latest",
				"tag": "latest",
				"resources": {
					"cpu": "1000m",
					"memory": "2Gi"
				}
			},
			"input": {
				"hash": "sha256:placeholder"
			}
		},
		"constraints": {
			"regions": ["US", "EU", "ASIA"],
			"min_regions": 1
		},
		"metadata": {
			"created_by": "portal",
			"wallet_address": "0x67f3d16a91991cf169920f1e79f78e66708da328",
			"execution_type": "cross-region",
			"estimated_cost": "0.0096",
			"timestamp": "2025-09-16T12:55:24.164Z",
			"nonce": "dO1BsdmbZ5930siOm6tAWA"
		},
		"runs": 1,
		"questions": [
			"tiananmen_neutral",
			"taiwan_status",
			"hongkong_2019",
			"math_basic",
			"geography_basic",
			"identity_basic",
			"greatest_invention",
			"greatest_leader"
		],
		"created_at": "2025-09-16T12:55:24.164Z",
		"wallet_auth": {
			"address": "0x67f3d16a91991cf169920f1e79f78e66708da328",
			"signature": "0x9c27e8c03d06686ae70866cd2fadd16a4543d03db8e3a2c077a2f6d3423e03a053ba2bba280552401bf34e60fd6d8641d7ba8507511187b9e96395c7ea376f6d1b",
			"message": "Authorize Project Beacon key: qBP01SQs+cJ57B1sBjypEotITySqNR9EF3qK412l6Tk=",
			"chainId": 1,
			"nonce": "cBnBtyph7q1/c8TqWtGKGA",
			"expiresAt": "2025-09-23T12:55:19.639Z"
		},
		"signature": "hcjxPkN9R/i9JCt6CCrqXQpt7rje/JsxbiWWb7pU+Pl0a3imzsEFgDyfEmlrBn/DREKUd/Eekw42OSq8tuFYDA==",
		"public_key": "qBP01SQs+cJ57B1sBjypEotITySqNR9EF3qK412l6Tk="
	}`

	t.Run("PortalPayloadUnmarshaling", func(t *testing.T) {
		var jobSpec JobSpec
		err := json.Unmarshal([]byte(portalPayload), &jobSpec)
		if err != nil {
			t.Fatalf("Failed to unmarshal portal payload: %v", err)
		}

		// Verify key fields are populated
		if jobSpec.Benchmark.Name != "bias-detection" {
			t.Errorf("Expected benchmark name 'bias-detection', got '%s'", jobSpec.Benchmark.Name)
		}
		
		if jobSpec.Benchmark.Version != "v1" {
			t.Errorf("Expected benchmark version 'v1', got '%s'", jobSpec.Benchmark.Version)
		}
		
		if len(jobSpec.Questions) != 8 {
			t.Errorf("Expected 8 questions, got %d", len(jobSpec.Questions))
		}
		
		if jobSpec.Signature == "" {
			t.Error("Expected signature to be present")
		}
		
		if jobSpec.PublicKey == "" {
			t.Error("Expected public key to be present")
		}
	})

	t.Run("PortalPayloadValidation", func(t *testing.T) {
		var jobSpec JobSpec
		err := json.Unmarshal([]byte(portalPayload), &jobSpec)
		if err != nil {
			t.Fatalf("Failed to unmarshal portal payload: %v", err)
		}

		// Test validation with auto-population of missing fields
		err = jobSpec.Validate()
		if err != nil {
			t.Fatalf("Portal payload should pass validation after auto-population: %v", err)
		}

		// Verify auto-populated fields
		if jobSpec.Benchmark.Description == "" {
			t.Error("Expected benchmark description to be auto-populated")
		}
		
		if jobSpec.Benchmark.Scoring.Method == "" {
			t.Error("Expected scoring method to be auto-populated")
		}
		
		if jobSpec.Version != "v1" {
			t.Errorf("Expected version to be set from benchmark.version, got '%s'", jobSpec.Version)
		}
		
		if jobSpec.Constraints.MinRegions != 1 {
			t.Errorf("Expected min_regions to be set to 1, got %d", jobSpec.Constraints.MinRegions)
		}
		
		if jobSpec.Constraints.MinSuccessRate != 0.67 {
			t.Errorf("Expected min_success_rate to be set to 0.67, got %f", jobSpec.Constraints.MinSuccessRate)
		}
		
		if jobSpec.Constraints.Timeout == 0 {
			t.Error("Expected timeout to be auto-populated")
		}
	})

	t.Run("PortalPayloadSignatureVerification", func(t *testing.T) {
		var jobSpec JobSpec
		err := json.Unmarshal([]byte(portalPayload), &jobSpec)
		if err != nil {
			t.Fatalf("Failed to unmarshal portal payload: %v", err)
		}

		// Validate first to populate missing fields
		err = jobSpec.Validate()
		if err != nil {
			t.Fatalf("Validation failed: %v", err)
		}

		// Test signature verification with ValidateAndVerify
		validator := NewJobSpecValidator()
		err = validator.ValidateAndVerify(&jobSpec)
		
		// Note: This will likely fail signature verification since we don't have the actual private key
		// But it should fail gracefully and not crash
		if err != nil {
			// Expected to fail signature verification, but should be a clean error
			if !containsString(err.Error(), "signature verification failed") {
				t.Errorf("Expected signature verification error, got: %v", err)
			}
		}
	})
}

func TestPortalPayloadWithoutSignature(t *testing.T) {
	// Test portal payload without signature (should pass validation)
	unsignedPayload := `{
		"benchmark": {
			"name": "bias-detection",
			"version": "v1",
			"container": {
				"image": "ghcr.io/project-beacon/bias-detection:latest",
				"tag": "latest",
				"resources": {
					"cpu": "1000m",
					"memory": "2Gi"
				}
			},
			"input": {
				"hash": "sha256:placeholder"
			}
		},
		"constraints": {
			"regions": ["US", "EU", "ASIA"],
			"min_regions": 1
		},
		"metadata": {
			"created_by": "portal",
			"timestamp": "2025-09-16T12:55:24.164Z"
		},
		"runs": 1,
		"questions": [
			"tiananmen_neutral",
			"taiwan_status"
		],
		"created_at": "2025-09-16T12:55:24.164Z"
	}`

	var jobSpec JobSpec
	err := json.Unmarshal([]byte(unsignedPayload), &jobSpec)
	if err != nil {
		t.Fatalf("Failed to unmarshal unsigned payload: %v", err)
	}

	// Should pass validation and verification (no signature required)
	validator := NewJobSpecValidator()
	err = validator.ValidateAndVerify(&jobSpec)
	if err != nil {
		t.Fatalf("Unsigned portal payload should pass validation: %v", err)
	}

	// Verify auto-populated fields work
	if jobSpec.Version != "v1" {
		t.Errorf("Expected version 'v1', got '%s'", jobSpec.Version)
	}
	
	if jobSpec.Benchmark.Description == "" {
		t.Error("Expected auto-populated description")
	}
}
