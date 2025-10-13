//go:build manual
// +build manual

// This file is a standalone demo utility. It is excluded from normal builds/tests.
// Run explicitly with: go run -tags manual ./test_wallet_auth_signature.go
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/crypto"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func main() {
	// Generate keypair
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	publicKeyB64 := crypto.PublicKeyToBase64(publicKey)

	// Create JobSpec with wallet_auth (like portal does)
	js := &models.JobSpec{
		Version: "v1",
		Benchmark: models.BenchmarkSpec{
			Name:        "test",
			Description: "test",
			Container: models.ContainerSpec{
				Image: "test:latest",
				Resources: models.ResourceSpec{CPU: "100m", Memory: "128Mi"},
			},
			Input: models.InputSpec{Type: "prompt", Data: map[string]interface{}{"text": "test"}, Hash: "hash"},
			Scoring: models.ScoringSpec{Method: "similarity", Parameters: map[string]interface{}{}},
			Metadata: map[string]interface{}{},
		},
		Constraints: models.ExecutionConstraints{
			Regions: []string{"US"},
			MinRegions: 1,
			MinSuccessRate: 0.67,
			Timeout: 5 * time.Minute,
			ProviderTimeout: 1 * time.Minute,
		},
		Metadata: map[string]interface{}{"created_by": "test"},
		Questions: []string{"test"},
		WalletAuth: &models.WalletAuth{
			Address: "0x67f3d16a91991cf169920f1e79f78e66708da328",
			Signature: "0x9c27e8c03d06686ae70866cd2fadd16a4543d03db8e3a2c077a2f6d3423e03a053ba2bba280552401bf34e60fd6d8641d7ba8507511187b9e96395c7ea376f6d1b",
			Message: "Test message",
			ChainID: 1,
			Nonce: "testnonce",
			ExpiresAt: "2025-10-16T16:58:07.659Z",
		},
		PublicKey: publicKeyB64,
		CreatedAt: time.Now(),
	}

	// Sign it (adds signature)
	if err := js.Sign(privateKey); err != nil {
		fmt.Printf("SIGN ERROR: %v\n", err)
		return
	}

	fmt.Println("=== SIGNED JOBSPEC ===")
	jsonBytes, _ := json.MarshalIndent(js, "", "  ")
	fmt.Printf("%s\n\n", string(jsonBytes))

	// Now verify
	fmt.Println("=== VERIFICATION ===")
	if err := js.VerifySignature(); err != nil {
		fmt.Printf("VERIFY ERROR: %v\n", err)
	} else {
		fmt.Printf("SUCCESS: Signature verified!\n")
	}
}
