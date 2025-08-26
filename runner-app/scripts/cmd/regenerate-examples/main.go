package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/crypto"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <examples-dir>\n", os.Args[0])
		os.Exit(1)
	}

	examplesDir := os.Args[1]
	
	// Generate a key pair for signing examples
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate key pair: %v\n", err)
		os.Exit(1)
	}

	// Process jobspec-who-are-you.json
	if err := regenerateExample(examplesDir, "jobspec-who-are-you.json", keyPair); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to regenerate example: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Examples regenerated successfully!")
	fmt.Printf("Public key: %s\n", crypto.PublicKeyToBase64(keyPair.PublicKey))
}

func regenerateExample(examplesDir, filename string, keyPair *crypto.KeyPair) error {
	inputPath := filepath.Join(examplesDir, filename)
	outputPath := filepath.Join(examplesDir, filename+".signed")

	// Read the unsigned example
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	// Parse JobSpec
	var jobSpec models.JobSpec
	if err := json.Unmarshal(data, &jobSpec); err != nil {
		return fmt.Errorf("unmarshal jobspec: %w", err)
	}

	// Update timestamps and nonce
	now := time.Now().UTC()
	jobSpec.CreatedAt = now
	
	// Ensure metadata map exists
	if jobSpec.Metadata == nil {
		jobSpec.Metadata = make(map[string]interface{})
	}
	
	// Add security metadata
	jobSpec.Metadata["timestamp"] = now.Format(time.RFC3339)
	jobSpec.Metadata["nonce"] = generateNonce(jobSpec.ID)

	// Sign the JobSpec
	if err := jobSpec.Sign(keyPair.PrivateKey); err != nil {
		return fmt.Errorf("sign jobspec: %w", err)
	}

	// Write signed version
	signedData, err := json.MarshalIndent(&jobSpec, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal signed jobspec: %w", err)
	}

	if err := os.WriteFile(outputPath, signedData, 0644); err != nil {
		return fmt.Errorf("write signed file: %w", err)
	}

	fmt.Printf("Generated: %s\n", outputPath)
	return nil
}

func generateNonce(jobID string) string {
	// Generate 8 random bytes
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to timestamp-based nonce
		return fmt.Sprintf("%s-nonce-%d", jobID, time.Now().UnixNano())
	}
	
	randomHex := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("%s-nonce-%s", jobID, randomHex)
}
