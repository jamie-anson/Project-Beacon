package main

import (
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/jamie-anson/project-beacon-runner/pkg/crypto"
)

func main() {
	// Generate key pair
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate key pair: %v\n", err)
		os.Exit(1)
	}

	// Encode keys
	privateKeyB64 := crypto.PrivateKeyToBase64(keyPair.PrivateKey)
	publicKeyB64 := crypto.PublicKeyToBase64(keyPair.PublicKey)

	// Calculate Key ID (KID) - SHA256 hash of public key
	hash := sha256.Sum256([]byte(publicKeyB64))
	kid := fmt.Sprintf("%x", hash)

	fmt.Printf("🔐 New Ed25519 Key Pair Generated\n")
	fmt.Printf("=====================================\n")
	fmt.Printf("Private Key: %s\n", privateKeyB64)
	fmt.Printf("Public Key:  %s\n", publicKeyB64)
	fmt.Printf("Key ID (KID): %s\n", kid)
	fmt.Printf("\n🔒 Security Instructions:\n")
	fmt.Printf("- Store private key securely (environment variable or secure file)\n")
	fmt.Printf("- Share public key with runner operators for trusted keys allowlist\n")
	fmt.Printf("- Use Key ID for identification in logs and configuration\n")
	fmt.Printf("\n💾 Storage Commands:\n")
	fmt.Printf("export BEACON_PRIVATE_KEY=\"%s\"\n", privateKeyB64)
	fmt.Printf("export BEACON_PUBLIC_KEY=\"%s\"\n", publicKeyB64)
}
