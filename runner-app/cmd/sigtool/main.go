package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/crypto"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sigtool",
	Short: "Project Beacon signature tool for JobSpec signing and verification",
	Long:  "A CLI tool for managing Ed25519 signatures on Project Beacon JobSpecs",
}

var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "Sign a JobSpec with Ed25519 private key",
	RunE:  runSign,
}

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify a signed JobSpec",
	RunE:  runVerify,
}

var keygenCmd = &cobra.Command{
	Use:   "keygen",
	Short: "Generate a new Ed25519 key pair",
	RunE:  runKeygen,
}

var extractCmd = &cobra.Command{
	Use:   "extract-pubkey",
	Short: "Extract public key from a signed JobSpec",
	RunE:  runExtract,
}

// Command flags
var (
	privateKey string
	publicKey  string
	inputFile  string
	outputFile string
	outputDir  string
)

func init() {
	// Sign command flags
	signCmd.Flags().StringVar(&privateKey, "private-key", "", "Base64-encoded Ed25519 private key")
	signCmd.Flags().StringVar(&inputFile, "input", "", "Input JobSpec JSON file")
	signCmd.Flags().StringVar(&outputFile, "output", "", "Output signed JobSpec file")
	signCmd.MarkFlagRequired("private-key")
	signCmd.MarkFlagRequired("input")
	signCmd.MarkFlagRequired("output")

	// Verify command flags
	verifyCmd.Flags().StringVar(&publicKey, "public-key", "", "Base64-encoded Ed25519 public key (optional, uses key from JobSpec if not provided)")
	verifyCmd.Flags().StringVar(&inputFile, "input", "", "Input signed JobSpec JSON file")
	verifyCmd.MarkFlagRequired("input")

	// Keygen command flags
	keygenCmd.Flags().StringVar(&outputDir, "output-dir", ".", "Directory to save key files")

	// Extract command flags
	extractCmd.Flags().StringVar(&inputFile, "input", "", "Input signed JobSpec JSON file")
	extractCmd.MarkFlagRequired("input")

	rootCmd.AddCommand(signCmd, verifyCmd, keygenCmd, extractCmd)
}

func runSign(cmd *cobra.Command, args []string) error {
	// Read private key
	privKey, err := crypto.PrivateKeyFromBase64(privateKey)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	// Read input JobSpec
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	var jobSpec models.JobSpec
	if err := json.Unmarshal(data, &jobSpec); err != nil {
		return fmt.Errorf("failed to parse JobSpec: %w", err)
	}

	// Add security metadata if missing
	if jobSpec.Metadata == nil {
		jobSpec.Metadata = make(map[string]interface{})
	}
	
	if _, hasTimestamp := jobSpec.Metadata["timestamp"]; !hasTimestamp {
		jobSpec.Metadata["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	}
	
	if _, hasNonce := jobSpec.Metadata["nonce"]; !hasNonce {
		jobSpec.Metadata["nonce"] = fmt.Sprintf("%s-nonce-%d", jobSpec.ID, time.Now().UnixNano())
	}

	// Sign the JobSpec
	if err := jobSpec.Sign(privKey); err != nil {
		return fmt.Errorf("failed to sign JobSpec: %w", err)
	}

	// Write signed JobSpec
	signedData, err := json.MarshalIndent(&jobSpec, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal signed JobSpec: %w", err)
	}

	if err := os.WriteFile(outputFile, signedData, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("✅ JobSpec signed successfully\n")
	fmt.Printf("Input:  %s\n", inputFile)
	fmt.Printf("Output: %s\n", outputFile)
	fmt.Printf("Public Key: %s\n", jobSpec.PublicKey)
	
	return nil
}

func runVerify(cmd *cobra.Command, args []string) error {
	// Read signed JobSpec
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	var jobSpec models.JobSpec
	if err := json.Unmarshal(data, &jobSpec); err != nil {
		return fmt.Errorf("failed to parse JobSpec: %w", err)
	}

	// Use provided public key or extract from JobSpec
	var pubKeyToUse string
	if publicKey != "" {
		pubKeyToUse = publicKey
	} else {
		pubKeyToUse = jobSpec.PublicKey
		if pubKeyToUse == "" {
			return fmt.Errorf("no public key provided and none found in JobSpec")
		}
	}

	// Verify signature
	if err := jobSpec.VerifySignature(); err != nil {
		fmt.Printf("❌ Signature verification failed: %v\n", err)
		return fmt.Errorf("verification failed")
	}

	fmt.Printf("✅ Signature verification successful\n")
	fmt.Printf("File: %s\n", inputFile)
	fmt.Printf("JobSpec ID: %s\n", jobSpec.ID)
	fmt.Printf("Public Key: %s\n", pubKeyToUse)
	
	// Check security metadata
	if jobSpec.Metadata != nil {
		if timestamp, ok := jobSpec.Metadata["timestamp"]; ok {
			fmt.Printf("Timestamp: %v\n", timestamp)
		}
		if nonce, ok := jobSpec.Metadata["nonce"]; ok {
			fmt.Printf("Nonce: %v\n", nonce)
		}
	}

	return nil
}

func runKeygen(cmd *cobra.Command, args []string) error {
	// Generate key pair
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Encode keys
	privateKeyB64 := crypto.PrivateKeyToBase64(keyPair.PrivateKey)
	publicKeyB64 := crypto.PublicKeyToBase64(keyPair.PublicKey)

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write private key
	privateKeyFile := filepath.Join(outputDir, "private.key")
	if err := os.WriteFile(privateKeyFile, []byte(privateKeyB64), 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Write public key
	publicKeyFile := filepath.Join(outputDir, "public.key")
	if err := os.WriteFile(publicKeyFile, []byte(publicKeyB64), 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	fmt.Printf("✅ Key pair generated successfully\n")
	fmt.Printf("Private Key: %s (permissions: 600)\n", privateKeyFile)
	fmt.Printf("Public Key:  %s (permissions: 644)\n", publicKeyFile)
	fmt.Printf("\nPublic Key: %s\n", publicKeyB64)
	fmt.Printf("Private Key: %s\n", privateKeyB64)
	
	fmt.Printf("\n🔐 Security Reminder:\n")
	fmt.Printf("- Keep your private key secure and never share it\n")
	fmt.Printf("- The public key can be shared with runner operators\n")
	fmt.Printf("- Store the private key in environment variables for production use\n")

	return nil
}

func runExtract(cmd *cobra.Command, args []string) error {
	// Read signed JobSpec
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	var jobSpec models.JobSpec
	if err := json.Unmarshal(data, &jobSpec); err != nil {
		return fmt.Errorf("failed to parse JobSpec: %w", err)
	}

	if jobSpec.PublicKey == "" {
		return fmt.Errorf("no public key found in JobSpec")
	}

	fmt.Printf("Public Key: %s\n", jobSpec.PublicKey)
	
	// Calculate Key ID (KID)
	pubKey, err := crypto.PublicKeyFromBase64(jobSpec.PublicKey)
	if err != nil {
		return fmt.Errorf("invalid public key in JobSpec: %w", err)
	}
	
	// For demonstration - in real implementation, calculate proper KID
	fmt.Printf("Key ID (KID): %x\n", pubKey[:8]) // First 8 bytes as hex
	
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
