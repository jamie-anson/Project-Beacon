package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/crypto"
)

func TestJobSpecValidation(t *testing.T) {
	tests := []struct {
		name    string
		jobspec JobSpec
		wantErr bool
	}{
		{
			name: "valid jobspec",
			jobspec: JobSpec{
				ID:      "test-job-1",
				Version: "1.0",
				Benchmark: BenchmarkSpec{
					Name: "Who are you?",
					Container: ContainerSpec{
						Image: "alpine:latest",
						Resources: ResourceSpec{
							CPU:    "1000m",
							Memory: "512Mi",
						},
					},
					Input: InputSpec{
						Type: "prompt",
						Data: map[string]interface{}{"prompt": "Who are you?"},
						Hash: "abc123",
					},
				},
				Constraints: ExecutionConstraints{
					Regions:    []string{"US", "EU", "APAC"},
					MinRegions: 3,
					Timeout:    10 * time.Minute,
				},
				CreatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			jobspec: JobSpec{
				Version: "1.0",
				Benchmark: BenchmarkSpec{
					Name: "test",
					Container: ContainerSpec{Image: "alpine:latest"},
					Input: InputSpec{Hash: "abc123"},
				},
				Constraints: ExecutionConstraints{Regions: []string{"US"}},
			},
			wantErr: true,
		},
		{
			name: "missing regions",
			jobspec: JobSpec{
				ID:      "test-job-1",
				Version: "1.0",
				Benchmark: BenchmarkSpec{
					Name: "test",
					Container: ContainerSpec{Image: "alpine:latest"},
					Input: InputSpec{Hash: "abc123"},
				},
				Constraints: ExecutionConstraints{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.jobspec.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("JobSpec.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJobSpecSigningAndVerification(t *testing.T) {
	// Generate a test key pair
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create a test JobSpec
	jobspec := JobSpec{
		ID:      "test-job-1",
		Version: "1.0",
		Benchmark: BenchmarkSpec{
			Name:        "Who are you?",
			Description: "Simple identity benchmark",
			Container: ContainerSpec{
				Image: "alpine:latest",
				Tag:   "3.18",
				Command: []string{"echo", "I am an AI assistant"},
				Resources: ResourceSpec{
					CPU:    "1000m",
					Memory: "512Mi",
				},
			},
			Input: InputSpec{
				Type: "prompt",
				Data: map[string]interface{}{
					"prompt": "Who are you?",
				},
				Hash: "sha256:abc123def456",
			},
			Scoring: ScoringSpec{
				Method: "similarity",
				Parameters: map[string]interface{}{
					"threshold": 0.8,
				},
			},
		},
		Constraints: ExecutionConstraints{
			Regions:    []string{"US", "EU", "APAC"},
			MinRegions: 3,
			Timeout:    10 * time.Minute,
		},
		Metadata: map[string]interface{}{
			"requester": "test-user",
			"priority":  "normal",
		},
		CreatedAt: time.Now(),
	}

	// Test signing
	err = jobspec.Sign(keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("Failed to sign JobSpec: %v", err)
	}

	// Verify signature and public key are set
	if jobspec.Signature == "" {
		t.Error("Signature should be set after signing")
	}
	if jobspec.PublicKey == "" {
		t.Error("PublicKey should be set after signing")
	}

	// Test verification
	err = jobspec.VerifySignature()
	if err != nil {
		t.Fatalf("Failed to verify JobSpec signature: %v", err)
	}

	// Test that tampering breaks verification
	originalName := jobspec.Benchmark.Name
	jobspec.Benchmark.Name = "Modified benchmark"
	err = jobspec.VerifySignature()
	if err == nil {
		t.Error("Expected signature verification to fail after tampering")
	}

	// Restore original and verify it works again
	jobspec.Benchmark.Name = originalName
	err = jobspec.VerifySignature()
	if err != nil {
		t.Fatalf("Signature verification should work after restoring data: %v", err)
	}
}

func TestJobSpecJSONSerialization(t *testing.T) {
	// Create a test JobSpec
	original := JobSpec{
		ID:      "test-job-1",
		Version: "1.0",
		Benchmark: BenchmarkSpec{
			Name: "Who are you?",
			Container: ContainerSpec{
				Image: "alpine:latest",
				Resources: ResourceSpec{
					CPU:    "1000m",
					Memory: "512Mi",
				},
			},
			Input: InputSpec{
				Type: "prompt",
				Data: map[string]interface{}{"prompt": "Who are you?"},
				Hash: "abc123",
			},
		},
		Constraints: ExecutionConstraints{
			Regions:    []string{"US", "EU"},
			MinRegions: 2,
			Timeout:    5 * time.Minute,
		},
		CreatedAt: time.Now(),
	}

	// Test JSON marshaling
	jsonBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal JobSpec: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled JobSpec
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal JobSpec: %v", err)
	}

	// Verify key fields are preserved
	if unmarshaled.ID != original.ID {
		t.Errorf("ID mismatch: got %s, want %s", unmarshaled.ID, original.ID)
	}
	if unmarshaled.Benchmark.Name != original.Benchmark.Name {
		t.Errorf("Benchmark name mismatch: got %s, want %s", unmarshaled.Benchmark.Name, original.Benchmark.Name)
	}
	if len(unmarshaled.Constraints.Regions) != len(original.Constraints.Regions) {
		t.Errorf("Regions length mismatch: got %d, want %d", len(unmarshaled.Constraints.Regions), len(original.Constraints.Regions))
	}
}

func TestReceiptSigningAndVerification(t *testing.T) {
	// Generate a test key pair
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create a test Receipt
	receipt := Receipt{
		ID:        "receipt-1",
		JobSpecID: "job-1",
		ExecutionDetails: ExecutionDetails{
			TaskID:      "task-123",
			ProviderID:  "provider-456",
			Region:      "US",
			StartedAt:   time.Now().Add(-5 * time.Minute),
			CompletedAt: time.Now(),
			Duration:    5 * time.Minute,
			Status:      "completed",
		},
		Output: ExecutionOutput{
			Data: map[string]interface{}{
				"response": "I am an AI assistant",
			},
			Hash: "sha256:output123",
		},
		Provenance: ProvenanceInfo{
			BenchmarkHash: "sha256:benchmark456",
			ProviderInfo: map[string]interface{}{
				"node_id": "provider-456",
				"region":  "US",
			},
		},
		CreatedAt: time.Now(),
	}

	// Test signing
	err = receipt.Sign(keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("Failed to sign Receipt: %v", err)
	}

	// Test verification
	err = receipt.VerifySignature()
	if err != nil {
		t.Fatalf("Failed to verify Receipt signature: %v", err)
	}

	// Test that tampering breaks verification
	originalOutput := receipt.Output.Hash
	receipt.Output.Hash = "tampered-hash"
	err = receipt.VerifySignature()
	if err == nil {
		t.Error("Expected signature verification to fail after tampering")
	}

	// Restore and verify
	receipt.Output.Hash = originalOutput
	err = receipt.VerifySignature()
	if err != nil {
		t.Fatalf("Signature verification should work after restoring data: %v", err)
	}
}
