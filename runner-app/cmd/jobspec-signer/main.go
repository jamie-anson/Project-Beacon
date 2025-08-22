package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jamie-anson/project-beacon-runner/pkg/crypto"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func main() {
	var (
		inputFile    = flag.String("input", "", "Input JobSpec JSON file")
		outputFile   = flag.String("output", "", "Output signed JobSpec file (default: input + .signed)")
		privateKey   = flag.String("key", "", "Base64-encoded private key (or set PRIVATE_KEY env var)")
		generateKey  = flag.Bool("generate-key", false, "Generate a new key pair and exit")
		validateOnly = flag.Bool("validate", false, "Only validate the JobSpec without signing")
		verifyOnly   = flag.Bool("verify", false, "Verify an already signed JobSpec and exit")
	)
	flag.Parse()

	if *generateKey {
		generateKeyPair()
		return
	}

	if *inputFile == "" {
		log.Fatal("Input file is required. Use -input flag.")
	}

	// Read the JobSpec file
	data, err := os.ReadFile(*inputFile)
	if err != nil {
		log.Fatalf("Failed to read input file: %v", err)
	}

	// Parse JobSpec
	var jobspec models.JobSpec
	if err := json.Unmarshal(data, &jobspec); err != nil {
		log.Fatalf("Failed to parse JobSpec: %v", err)
	}

	// Validate JobSpec structure
	validator := models.NewJobSpecValidator()
	validator.SanitizeJobSpec(&jobspec)
	
	if err := jobspec.Validate(); err != nil {
		log.Fatalf("JobSpec validation failed: %v", err)
	}

	fmt.Printf("✅ JobSpec validation passed\n")
	
	// Print summary
	summary := validator.ExtractJobSpecSummary(&jobspec)
	summaryJSON, _ := json.MarshalIndent(summary, "", "  ")
	fmt.Printf("JobSpec Summary:\n%s\n\n", summaryJSON)

	if *verifyOnly {
		if err := jobspec.VerifySignature(); err != nil {
			log.Fatalf("Signature verification failed: %v", err)
		}
		fmt.Println("✅ Signature verification passed")
		return
	}

	if *validateOnly {
		fmt.Println("✅ Validation complete (no signing requested)")
		return
	}

	// Get private key
	keyStr := *privateKey
	if keyStr == "" {
		keyStr = os.Getenv("PRIVATE_KEY")
	}
	if keyStr == "" {
		log.Fatal("Private key is required. Use -key flag or set PRIVATE_KEY env var.")
	}

	privateKeyBytes, err := crypto.PrivateKeyFromBase64(keyStr)
	if err != nil {
		log.Fatalf("Invalid private key: %v", err)
	}

	// Sign the JobSpec
	if err := jobspec.Sign(privateKeyBytes); err != nil {
		log.Fatalf("Failed to sign JobSpec: %v", err)
	}

	fmt.Printf("✅ JobSpec signed successfully\n")
	fmt.Printf("Public Key: %s\n", jobspec.PublicKey)
	fmt.Printf("Signature: %s\n\n", jobspec.Signature)

	// Verify the signature
	if err := jobspec.VerifySignature(); err != nil {
		log.Fatalf("Signature verification failed: %v", err)
	}

	fmt.Printf("✅ Signature verification passed\n")

	// Determine output file
	outFile := *outputFile
	if outFile == "" {
		outFile = *inputFile + ".signed"
	}

	// Write signed JobSpec
	signedData, err := json.MarshalIndent(jobspec, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal signed JobSpec: %v", err)
	}

	if err := os.WriteFile(outFile, signedData, 0644); err != nil {
		log.Fatalf("Failed to write output file: %v", err)
	}

	fmt.Printf("✅ Signed JobSpec written to: %s\n", outFile)
}

func generateKeyPair() {
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate key pair: %v", err)
	}

	fmt.Println("Generated Ed25519 Key Pair:")
	fmt.Printf("Private Key: %s\n", crypto.PrivateKeyToBase64(keyPair.PrivateKey))
	fmt.Printf("Public Key:  %s\n", crypto.PublicKeyToBase64(keyPair.PublicKey))
	fmt.Println()
	fmt.Println("⚠️  Keep the private key secure and never share it!")
	fmt.Println("💡 Use the private key with: export PRIVATE_KEY=\"<private_key>\"")
}
