package models

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"
)

func TestSignatureVerification_ComprehensiveScenarios(t *testing.T) {
	// Generate a test key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	publicKeyB64 := base64.StdEncoding.EncodeToString(publicKey)

	tests := []struct {
		name           string
		setupJobSpec   func() *JobSpec
		expectError    bool
		errorContains  string
		description    string
	}{
		{
			name: "ValidSignedJobSpec",
			setupJobSpec: func() *JobSpec {
				js := createValidJobSpec()
				js.PublicKey = publicKeyB64
				if err := js.Sign(privateKey); err != nil {
					t.Fatalf("failed to sign job spec: %v", err)
				}
				return js
			},
			expectError: false,
			description: "Properly signed job spec should pass validation",
		},
		{
			name: "UnsignedJobSpec",
			setupJobSpec: func() *JobSpec {
				js := createValidJobSpec()
				// Leave signature and public key empty
				return js
			},
			expectError: false,
			description: "Unsigned job spec should pass validation (optional signatures)",
		},
		{
			name: "InvalidSignature",
			setupJobSpec: func() *JobSpec {
				js := createValidJobSpec()
				js.PublicKey = publicKeyB64
				js.Signature = "invalid_signature_data"
				return js
			},
			expectError:   true,
			errorContains: "signature verification failed",
			description:   "Job spec with invalid signature should fail",
		},
		{
			name: "MismatchedSignature",
			setupJobSpec: func() *JobSpec {
				js := createValidJobSpec()
				js.PublicKey = publicKeyB64
				
				// Sign the job spec
				if err := js.Sign(privateKey); err != nil {
					t.Fatalf("failed to sign job spec: %v", err)
				}
				
				// Then modify the content to invalidate the signature
				js.Benchmark.Name = "modified_after_signing"
				return js
			},
			expectError:   true,
			errorContains: "signature verification failed",
			description:   "Job spec modified after signing should fail verification",
		},
		{
			name: "InvalidPublicKey",
			setupJobSpec: func() *JobSpec {
				js := createValidJobSpec()
				js.PublicKey = "invalid_base64_key!!!"
				js.Signature = "some_signature"
				return js
			},
			expectError:   true,
			errorContains: "signature verification failed",
			description:   "Job spec with invalid public key should fail",
		},
		{
			name: "WrongPublicKey",
			setupJobSpec: func() *JobSpec {
				// Generate a different key pair
				wrongPublicKey, _, err := ed25519.GenerateKey(rand.Reader)
				if err != nil {
					t.Fatalf("failed to generate wrong key pair: %v", err)
				}
				
				js := createValidJobSpec()
				js.PublicKey = publicKeyB64
				
				// Sign with correct private key
				if err := js.Sign(privateKey); err != nil {
					t.Fatalf("failed to sign job spec: %v", err)
				}
				
				// But set wrong public key
				js.PublicKey = base64.StdEncoding.EncodeToString(wrongPublicKey)
				return js
			},
			expectError:   true,
			errorContains: "signature verification failed",
			description:   "Job spec signed with different key than public key should fail",
		},
		{
			name: "SignatureWithoutPublicKey",
			setupJobSpec: func() *JobSpec {
				js := createValidJobSpec()
				js.Signature = "some_signature_data"
				// PublicKey left empty
				return js
			},
			expectError: false,
			description: "Job spec with signature but no public key should pass (treated as unsigned)",
		},
		{
			name: "PublicKeyWithoutSignature",
			setupJobSpec: func() *JobSpec {
				js := createValidJobSpec()
				js.PublicKey = publicKeyB64
				// Signature left empty
				return js
			},
			expectError: false,
			description: "Job spec with public key but no signature should pass (treated as unsigned)",
		},
	}

	validator := NewJobSpecValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			js := tt.setupJobSpec()
			
			err := validator.ValidateAndVerify(js)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for %s, but got none. %s", tt.name, tt.description)
					return
				}
				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error for %s, but got: %v. %s", tt.name, err, tt.description)
				}
			}
		})
	}
}

func TestSignatureVerification_IDFieldHandling(t *testing.T) {
	// Generate a test key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	publicKeyB64 := base64.StdEncoding.EncodeToString(publicKey)

	t.Run("SignatureVerificationWithIDField", func(t *testing.T) {
		js := createValidJobSpec()
		js.PublicKey = publicKeyB64
		
		// Sign the job spec (this should work even if ID is present)
		if err := js.Sign(privateKey); err != nil {
			t.Fatalf("failed to sign job spec: %v", err)
		}
		
		// Set an ID after signing (simulating server-generated ID)
		js.ID = "server-generated-id-12345"
		
		validator := NewJobSpecValidator()
		err := validator.ValidateAndVerify(js)
		
		if err != nil {
			t.Errorf("signature verification should work even with ID field present, got error: %v", err)
		}
	})

	t.Run("PortalStyleSigning", func(t *testing.T) {
		// Simulate how the portal signs: without ID field
		js := createValidJobSpec()
		js.ID = "" // Portal doesn't set ID
		js.PublicKey = publicKeyB64
		
		// Sign without ID
		if err := js.Sign(privateKey); err != nil {
			t.Fatalf("failed to sign job spec: %v", err)
		}
		
		// Server adds ID during processing
		js.ID = "server-generated-id-67890"
		
		validator := NewJobSpecValidator()
		err := validator.ValidateAndVerify(js)
		
		if err != nil {
			t.Errorf("portal-style signing should work, got error: %v", err)
		}
	})
}

func TestSignatureVerification_EdgeCases(t *testing.T) {
	validator := NewJobSpecValidator()

	t.Run("EmptyJobSpec", func(t *testing.T) {
		js := &JobSpec{}
		
		err := validator.ValidateAndVerify(js)
		
		// Should fail validation (not signature verification)
		if err == nil {
			t.Error("empty job spec should fail validation")
		}
		// Should not be a signature verification error
		if containsString(err.Error(), "signature verification failed") {
			t.Error("empty job spec should fail validation, not signature verification")
		}
	})

	t.Run("NilJobSpec", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for nil job spec")
			}
		}()
		
		validator.ValidateAndVerify(nil)
	})
}

// Helper function to create a valid job spec for testing
func createValidJobSpec() *JobSpec {
	return &JobSpec{
		Version: "1.0",
		Benchmark: BenchmarkSpec{
			Name:        "test-benchmark",
			Description: "Test benchmark for signature verification",
			Container: ContainerSpec{
				Image: "test:latest",
				Resources: ResourceSpec{
					CPU:    "100m",
					Memory: "128Mi",
				},
			},
			Input: InputSpec{
				Type: "prompt",
				Data: map[string]interface{}{"text": "test"},
				Hash: "test-hash-12345",
			},
			Scoring: ScoringSpec{
				Method:     "similarity",
				Parameters: map[string]interface{}{"threshold": 0.8},
			},
			Metadata: map[string]interface{}{"test": true},
		},
		Constraints: ExecutionConstraints{
			Regions:         []string{"US"},
			MinRegions:      1,
			MinSuccessRate:  0.67,
			Timeout:         5 * time.Minute,
			ProviderTimeout: 1 * time.Minute,
		},
		Metadata:  map[string]interface{}{"created_by": "test"},
		CreatedAt: time.Now(),
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
			(s[:len(substr)] == substr || 
			 s[len(s)-len(substr):] == substr || 
			 containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
