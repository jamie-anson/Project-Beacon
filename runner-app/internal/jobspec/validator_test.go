package jobspec

import (
	"encoding/json"
	"testing"

	"github.com/jamie-anson/project-beacon-runner/pkg/crypto"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func TestJobSpecValidation(t *testing.T) {
	validator := NewValidator()

	// Generate a key pair for testing
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create a sample JobSpec
	jobspec := validator.CreateSampleJobSpec()

	// Sign the JobSpec
	if err := jobspec.Sign(keyPair.PrivateKey); err != nil {
		t.Fatalf("Failed to sign JobSpec: %v", err)
	}

	// Convert to JSON
	jobspecJSON, err := json.Marshal(jobspec)
	if err != nil {
		t.Fatalf("Failed to marshal JobSpec: %v", err)
	}

	// Validate the signed JobSpec
	validatedJobSpec, err := validator.ValidateJobSpec(jobspecJSON)
	if err != nil {
		t.Fatalf("JobSpec validation failed: %v", err)
	}

	// Verify the validated JobSpec matches original
	if validatedJobSpec.ID != jobspec.ID {
		t.Errorf("Expected ID %s, got %s", jobspec.ID, validatedJobSpec.ID)
	}

	if validatedJobSpec.Signature != jobspec.Signature {
		t.Errorf("Signature mismatch after validation")
	}

	t.Logf("✅ JobSpec validation successful")
	t.Logf("JobSpec ID: %s", validatedJobSpec.ID)
	t.Logf("Signature: %s", validatedJobSpec.Signature[:32]+"...")
	t.Logf("Public Key: %s", validatedJobSpec.PublicKey[:32]+"...")
}

func TestJobSpecSignatureVerification(t *testing.T) {
	validator := NewValidator()

	// Generate a key pair for testing
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create and sign a JobSpec
	jobspec := validator.CreateSampleJobSpec()
	if err := jobspec.Sign(keyPair.PrivateKey); err != nil {
		t.Fatalf("Failed to sign JobSpec: %v", err)
	}

	// Test valid signature verification
	if err := jobspec.VerifySignature(); err != nil {
		t.Errorf("Valid signature verification failed: %v", err)
	}

	// Test invalid signature (tamper with data)
	originalBenchmarkName := jobspec.Benchmark.Name
	jobspec.Benchmark.Name = "Tampered Benchmark"
	
	if err := jobspec.VerifySignature(); err == nil {
		t.Errorf("Expected signature verification to fail for tampered data")
	}

	// Restore original data
	jobspec.Benchmark.Name = originalBenchmarkName

	// Test invalid signature (empty signature)
	originalSignature := jobspec.Signature
	jobspec.Signature = ""
	
	if err := jobspec.VerifySignature(); err == nil {
		t.Errorf("Expected signature verification to fail for empty signature")
	}

	// Restore signature
	jobspec.Signature = originalSignature

	t.Logf("✅ Signature verification tests passed")
}

func TestJobSpecValidationRules(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name        string
		modifySpec  func(*models.JobSpec)
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid JobSpec",
			modifySpec: func(js *models.JobSpec) {
				// No modifications - should be valid
			},
			expectError: false,
		},
		{
			name: "Missing ID",
			modifySpec: func(js *models.JobSpec) {
				js.ID = ""
			},
			expectError: true,
			errorMsg:    "jobspec ID is required",
		},
		{
			name: "Missing Version",
			modifySpec: func(js *models.JobSpec) {
				js.Version = ""
			},
			expectError: true,
			errorMsg:    "jobspec version is required",
		},
		{
			name: "Missing Benchmark Name",
			modifySpec: func(js *models.JobSpec) {
				js.Benchmark.Name = ""
			},
			expectError: true,
			errorMsg:    "benchmark name is required",
		},
		{
			name: "Missing Container Image",
			modifySpec: func(js *models.JobSpec) {
				js.Benchmark.Container.Image = ""
			},
			expectError: true,
			errorMsg:    "container image is required",
		},
		{
			name: "No Regions",
			modifySpec: func(js *models.JobSpec) {
				js.Constraints.Regions = []string{}
			},
			expectError: true,
			errorMsg:    "at least one region constraint is required",
		},
		{
			name: "Missing Input Hash",
			modifySpec: func(js *models.JobSpec) {
				js.Benchmark.Input.Hash = ""
			},
			expectError: true,
			errorMsg:    "input hash is required for integrity verification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh JobSpec for each test
			jobspec := validator.CreateSampleJobSpec()
			
			// Apply modifications
			tt.modifySpec(jobspec)

			// Test validation
			err := jobspec.Validate()
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected validation error but got none")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected validation error: %v", err)
				}
			}
		})
	}

	t.Logf("✅ JobSpec validation rules tests passed")
}

func TestRegionValidation(t *testing.T) {
	validator := NewValidator()

	validRegions := []string{"US", "EU", "APAC"}
	if err := validator.ValidateRegions(validRegions); err != nil {
		t.Errorf("Valid regions should pass validation: %v", err)
	}

	invalidRegions := []string{"US", "INVALID", "EU"}
	if err := validator.ValidateRegions(invalidRegions); err == nil {
		t.Errorf("Invalid regions should fail validation")
	}

	t.Logf("✅ Region validation tests passed")
}

func TestExecutionCostEstimation(t *testing.T) {
	validator := NewValidator()
	jobspec := validator.CreateSampleJobSpec()

	cost, err := validator.EstimateExecutionCost(jobspec)
	if err != nil {
		t.Errorf("Cost estimation failed: %v", err)
	}

	if cost <= 0 {
		t.Errorf("Expected positive cost, got %f", cost)
	}

	t.Logf("✅ Estimated execution cost: $%.4f", cost)
}

func TestKeyGeneration(t *testing.T) {
	// Test key pair generation
	keyPair1, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate first key pair: %v", err)
	}

	keyPair2, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate second key pair: %v", err)
	}

	// Ensure keys are different
	if string(keyPair1.PublicKey) == string(keyPair2.PublicKey) {
		t.Errorf("Generated key pairs should be different")
	}

	// Test key encoding/decoding
	publicKeyB64 := crypto.PublicKeyToBase64(keyPair1.PublicKey)
	decodedPublicKey, err := crypto.PublicKeyFromBase64(publicKeyB64)
	if err != nil {
		t.Errorf("Failed to decode public key: %v", err)
	}

	if string(keyPair1.PublicKey) != string(decodedPublicKey) {
		t.Errorf("Public key encoding/decoding mismatch")
	}

	t.Logf("✅ Key generation and encoding tests passed")
	t.Logf("Public Key 1: %s", publicKeyB64[:32]+"...")
}
