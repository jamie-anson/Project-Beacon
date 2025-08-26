package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <examples-dir>\n", os.Args[0])
		os.Exit(1)
	}

	examplesDir := os.Args[1]
	
	// Find all JSON files
	jsonFiles, err := filepath.Glob(filepath.Join(examplesDir, "*.json"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to find JSON files: %v\n", err)
		os.Exit(1)
	}

	// Filter out .signed files (they're handled separately)
	var exampleFiles []string
	for _, file := range jsonFiles {
		if !strings.HasSuffix(file, ".signed") {
			exampleFiles = append(exampleFiles, file)
		}
	}

	if len(exampleFiles) == 0 {
		fmt.Println("No example files found")
		return
	}

	allValid := true
	for _, file := range exampleFiles {
		if !validateExampleSchema(file) {
			allValid = false
		}
	}

	if !allValid {
		os.Exit(1)
	}

	fmt.Printf("✅ All %d example schemas validated successfully!\n", len(exampleFiles))
}

func validateExampleSchema(filePath string) bool {
	fmt.Printf("Validating schema: %s\n", filepath.Base(filePath))

	// Read example
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
		fmt.Printf("  ❌ Schema validation failed: %v\n", err)
		return false
	}

	// Check required fields
	if jobSpec.ID == "" {
		fmt.Printf("  ❌ Missing required field: id\n")
		return false
	}
	if jobSpec.Version == "" {
		fmt.Printf("  ❌ Missing required field: version\n")
		return false
	}
	if jobSpec.Benchmark.Name == "" {
		fmt.Printf("  ❌ Missing required field: benchmark.name\n")
		return false
	}

	// Check security metadata for examples
	if jobSpec.Metadata != nil {
		if _, hasTimestamp := jobSpec.Metadata["timestamp"]; hasTimestamp {
			fmt.Printf("  ✅ Has security timestamp\n")
		}
		if _, hasNonce := jobSpec.Metadata["nonce"]; hasNonce {
			fmt.Printf("  ✅ Has security nonce\n")
		}
	}

	fmt.Printf("  ✅ Valid schema\n")
	return true
}
