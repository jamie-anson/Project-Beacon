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
			name: "missing ID (auto-generated)",
			jobspec: JobSpec{
				Version: "1.0",
				Benchmark: BenchmarkSpec{
					Name: "test",
					Container: ContainerSpec{Image: "alpine:latest"},
					Input: InputSpec{Hash: "abc123"},
				},
				Constraints: ExecutionConstraints{Regions: []string{"US"}},
			},
			wantErr: false, // ID is now auto-generated, so validation should pass
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

func TestNewReceipt_DefaultsAndFields(t *testing.T) {
    exec := ExecutionDetails{
        TaskID:     "task-1",
        ProviderID: "prov-1",
        Region:     "US",
        StartedAt:  time.Now().Add(-time.Minute),
        CompletedAt: time.Now(),
        Duration:   time.Minute,
        Status:     "completed",
    }
    out := ExecutionOutput{Data: map[string]any{"ok": true}, Hash: "sha256:deadbeef", Metadata: map[string]any{"k":"v"}}
    prov := ProvenanceInfo{BenchmarkHash: "sha256:beadfeed", ProviderInfo: map[string]any{"node":"n1"}, ExecutionEnv: map[string]any{"os":"linux"}}

    r := NewReceipt("job-xyz", exec, out, prov)
    if r.SchemaVersion != "v0.1.0" { t.Fatalf("schema version: %s", r.SchemaVersion) }
    if r.ID == "" || r.JobSpecID != "job-xyz" { t.Fatalf("id/jobspec mismatch: %s/%s", r.ID, r.JobSpecID) }
    if r.CreatedAt.IsZero() { t.Fatalf("createdAt should be set") }
    if len(r.Attestations) != 0 { t.Fatalf("expected no attestations") }
    if r.ExecutionDetails.TaskID != exec.TaskID || r.Output.Hash != out.Hash || r.Provenance.BenchmarkHash != prov.BenchmarkHash {
        t.Fatalf("field propagation mismatch")
    }
}

func TestReceiptAddAttestation_Appends(t *testing.T) {
    r := &Receipt{Attestations: []ExecutionAttestation{}}
    a1 := ExecutionAttestation{Type: "provider", Source: "p1", Timestamp: time.Now(), Statement: "ok", Evidence: "h1"}
    a2 := ExecutionAttestation{Type: "network", Source: "n1", Timestamp: time.Now(), Statement: "ok2", Evidence: "h2"}
    r.AddAttestation(a1)
    r.AddAttestation(a2)
    if len(r.Attestations) != 2 { t.Fatalf("want 2 attestations, got %d", len(r.Attestations)) }
    if r.Attestations[0].Source != "p1" || r.Attestations[1].Source != "n1" { t.Fatalf("attestation order/fields mismatch") }
}

func TestJobSpecVerifySignature_ErrorBranches(t *testing.T) {
    js := JobSpec{}
    if err := js.VerifySignature(); err == nil { t.Fatalf("expected error when signature/public key missing") }

    js = JobSpec{Signature: "abc"}
    if err := js.VerifySignature(); err == nil { t.Fatalf("expected error when public key missing") }

    js = JobSpec{Signature: "abc", PublicKey: "!!!notb64"}
    if err := js.VerifySignature(); err == nil { t.Fatalf("expected invalid public key error") }

    // Wrong key: sign with one key but set different public key
    kp1, err := crypto.GenerateKeyPair(); if err != nil { t.Fatal(err) }
    kp2, err := crypto.GenerateKeyPair(); if err != nil { t.Fatal(err) }
    js = JobSpec{
        ID: "id1", Version: "v1",
        Benchmark: BenchmarkSpec{
            Name: "bench",
            Container: ContainerSpec{Image: "alpine", Resources: ResourceSpec{CPU:"1000m", Memory:"512Mi"}},
            Input: InputSpec{Type:"prompt", Data: map[string]any{"q":"hi"}, Hash: "sha256:abc"},
        },
        Constraints: ExecutionConstraints{Regions: []string{"US"}, MinRegions:1, Timeout: time.Minute},
        CreatedAt: time.Now(),
    }
    if err := js.Sign(kp1.PrivateKey); err != nil { t.Fatalf("sign error: %v", err) }
    js.PublicKey = crypto.PublicKeyToBase64(kp2.PublicKey) // override with mismatched key
    if err := js.VerifySignature(); err == nil { t.Fatalf("expected signature verification failure with mismatched key") }
}

func TestReceiptVerifySignature_ErrorBranches(t *testing.T) {
    r := Receipt{}
    if err := r.VerifySignature(); err == nil { t.Fatalf("expected error when signature/public key missing") }
    r = Receipt{Signature: "abc"}
    if err := r.VerifySignature(); err == nil { t.Fatalf("expected error when public key missing") }
    r = Receipt{Signature: "abc", PublicKey: "notb64!!"}
    if err := r.VerifySignature(); err == nil { t.Fatalf("expected invalid public key error") }
}

func TestJobSpecValidate_ErrorBranchesAndDefaults(t *testing.T) {
    // missing version
    js := JobSpec{ID: "id", Benchmark: BenchmarkSpec{Name: "n", Container: ContainerSpec{Image:"img", Resources: ResourceSpec{CPU:"1", Memory:"1Mi"}}, Input: InputSpec{Hash:"h"}}, Constraints: ExecutionConstraints{Regions: []string{"US"}}}
    if err := js.Validate(); err == nil { t.Fatalf("expected error for missing version") }

    // missing benchmark name
    js = JobSpec{ID: "id", Version:"v", Benchmark: BenchmarkSpec{Container: ContainerSpec{Image:"img", Resources: ResourceSpec{CPU:"1", Memory:"1Mi"}}, Input: InputSpec{Hash:"h"}}, Constraints: ExecutionConstraints{Regions: []string{"US"}}}
    if err := js.Validate(); err == nil { t.Fatalf("expected error for missing benchmark name") }

    // missing container image
    js = JobSpec{ID: "id", Version:"v", Benchmark: BenchmarkSpec{Name:"n", Container: ContainerSpec{}, Input: InputSpec{Hash:"h"}}, Constraints: ExecutionConstraints{Regions: []string{"US"}}}
    if err := js.Validate(); err == nil { t.Fatalf("expected error for missing container image") }

    // missing input hash
    js = JobSpec{ID: "id", Version:"v", Benchmark: BenchmarkSpec{Name:"n", Container: ContainerSpec{Image:"img", Resources: ResourceSpec{CPU:"1", Memory:"1Mi"}}, Input: InputSpec{}}, Constraints: ExecutionConstraints{Regions: []string{"US"}}}
    if err := js.Validate(); err == nil { t.Fatalf("expected error for missing input hash") }

    // defaults applied when zero values
    js = JobSpec{
        ID: "id", Version:"v",
        Benchmark: BenchmarkSpec{Name:"n", Container: ContainerSpec{Image:"img", Resources: ResourceSpec{CPU:"1", Memory:"1Mi"}}, Input: InputSpec{Hash:"h"}},
        Constraints: ExecutionConstraints{Regions: []string{"US"}},
    }
    if err := js.Validate(); err != nil { t.Fatalf("unexpected validate error: %v", err) }
    if js.Constraints.MinRegions == 0 || js.Constraints.MinSuccessRate == 0 || js.Constraints.Timeout == 0 || js.Constraints.ProviderTimeout == 0 {
        t.Fatalf("expected defaults to be applied: %+v", js.Constraints)
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
