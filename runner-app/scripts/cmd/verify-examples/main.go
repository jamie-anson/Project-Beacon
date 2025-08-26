package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <examples-dir>\n", os.Args[0])
		os.Exit(1)
	}

	examplesDir := os.Args[1]
	
	// Find all .signed files
	signedFiles, err := filepath.Glob(filepath.Join(examplesDir, "*.signed"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to find signed files: %v\n", err)
		os.Exit(1)
	}

	if len(signedFiles) == 0 {
		fmt.Println("No signed examples found")
		return
	}

	allValid := true
	for _, file := range signedFiles {
		if !verifySignedExample(file) {
			allValid = false
		}
	}

	if !allValid {
		os.Exit(1)
	}

	fmt.Printf("✅ All %d signed examples verified successfully!\n", len(signedFiles))
}

func verifySignedExample(filePath string) bool {
	fmt.Printf("Verifying: %s\n", filepath.Base(filePath))

	// Read signed example
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("  ❌ Failed to read file: %v\n", err)
		return false
	}

	// Parse JobSpec
	var jobSpec models.JobSpec
	if err := json.Unmarshal(data, &jobSpec); err != nil {
		fmt.Printf("  ❌ Failed to parse JSON: %v\n", err)
		return false
	}

	// Validate structure
	if err := jobSpec.Validate(); err != nil {
		fmt.Printf("  ❌ Validation failed: %v\n", err)
		return false
	}

	// Verify signature
	if err := jobSpec.VerifySignature(); err != nil {
		fmt.Printf("  ❌ Signature verification failed: %v\n", err)
		return false
	}

	// Check security metadata
	if jobSpec.Metadata != nil {
		if _, hasTimestamp := jobSpec.Metadata["timestamp"]; !hasTimestamp {
			fmt.Printf("  ⚠️  Missing timestamp in metadata\n")
		}
		if _, hasNonce := jobSpec.Metadata["nonce"]; !hasNonce {
			fmt.Printf("  ⚠️  Missing nonce in metadata\n")
		}
	}

	fmt.Printf("  ✅ Valid signature and structure\n")
	return true
}
