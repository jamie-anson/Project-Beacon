package models

import (
	"crypto/ed25519"
	"fmt"

	"github.com/jamie-anson/project-beacon-runner/pkg/crypto"
)

// VerifySignature verifies the JobSpec's cryptographic signature
func (js *JobSpec) VerifySignature() error {
	if js.Signature == "" {
		return fmt.Errorf("signature is required")
	}
	if js.PublicKey == "" {
		return fmt.Errorf("public key is required")
	}

	publicKey, err := crypto.PublicKeyFromBase64(js.PublicKey)
	if err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}

	// Create signable data by zeroing signature and public key fields
	signableData, err := crypto.CreateSignableJobSpec(js)
	if err != nil {
		return fmt.Errorf("failed to create signable data: %w", err)
	}

	// Debug: Log the canonical JSON being verified
	canonicalBytes, _ := crypto.CanonicalJSON(signableData)
	fmt.Printf("[SIGNATURE DEBUG] Server canonical JSON: %s\n", string(canonicalBytes))
	fmt.Printf("[SIGNATURE DEBUG] Server canonical length: %d\n", len(canonicalBytes))

	if err := crypto.VerifyJSONSignature(signableData, js.Signature, publicKey); err != nil {
		// Try fallback canonicalization for backward compatibility
		if fallbackData, fallbackErr := crypto.CanonicalizeJobSpecV1(js); fallbackErr == nil {
			if fallbackVerifyErr := crypto.VerifyJSONSignature(fallbackData, js.Signature, publicKey); fallbackVerifyErr == nil {
				return nil
			}
		}
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// Sign signs the JobSpec with the provided private key
func (js *JobSpec) Sign(privateKey ed25519.PrivateKey) error {
	// Set public key from private key
	publicKey := privateKey.Public().(ed25519.PublicKey)
	js.PublicKey = crypto.PublicKeyToBase64(publicKey)

	// Create signable data (without signature and public_key fields)
	signableData, err := crypto.CreateSignableJobSpec(js)
	if err != nil {
		return fmt.Errorf("failed to create signable data: %w", err)
	}

	// Sign the data
	signature, err := crypto.SignJSON(signableData, privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign data: %w", err)
	}

	js.Signature = signature
	return nil
}

// Sign signs the Receipt with the provided private key
func (r *Receipt) Sign(privateKey ed25519.PrivateKey) error {
	// Set public key from private key
	publicKey := privateKey.Public().(ed25519.PublicKey)
	r.PublicKey = crypto.PublicKeyToBase64(publicKey)

	// Create signable data (without signature and public_key fields)
	signableData, err := crypto.CreateSignableReceipt(r)
	if err != nil {
		return fmt.Errorf("failed to create signable data: %w", err)
	}

	// Sign the data
	signature, err := crypto.SignJSON(signableData, privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign data: %w", err)
	}

	r.Signature = signature
	return nil
}

// VerifySignature verifies the Receipt's cryptographic signature
func (r *Receipt) VerifySignature() error {
	if r.Signature == "" {
		return fmt.Errorf("signature is required")
	}
	if r.PublicKey == "" {
		return fmt.Errorf("public key is required")
	}

	publicKey, err := crypto.PublicKeyFromBase64(r.PublicKey)
	if err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}

	// Create signable data by zeroing signature and public key fields
	signableData, err := crypto.CreateSignableReceipt(r)
	if err != nil {
		return fmt.Errorf("failed to create signable data: %w", err)
	}

	if err := crypto.VerifyJSONSignature(signableData, r.Signature, publicKey); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}
