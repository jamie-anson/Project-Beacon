package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
)

// KeyPair represents an Ed25519 key pair
type KeyPair struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// GenerateKeyPair creates a new Ed25519 key pair
func GenerateKeyPair() (*KeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	return &KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, nil
}

// PublicKeyFromBase64 decodes a base64-encoded public key
func PublicKeyFromBase64(encoded string) (ed25519.PublicKey, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	if len(keyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: expected %d, got %d", ed25519.PublicKeySize, len(keyBytes))
	}

	return ed25519.PublicKey(keyBytes), nil
}

// PrivateKeyFromBase64 decodes a base64-encoded private key
func PrivateKeyFromBase64(encoded string) (ed25519.PrivateKey, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(keyBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size: expected %d, got %d", ed25519.PrivateKeySize, len(keyBytes))
	}

	return ed25519.PrivateKey(keyBytes), nil
}

// PublicKeyToBase64 encodes a public key as base64
func PublicKeyToBase64(key ed25519.PublicKey) string {
	return base64.StdEncoding.EncodeToString(key)
}

// PrivateKeyToBase64 encodes a private key as base64
func PrivateKeyToBase64(key ed25519.PrivateKey) string {
	return base64.StdEncoding.EncodeToString(key)
}

// SignJSON signs a JSON-serializable object with Ed25519
func SignJSON(data interface{}, privateKey ed25519.PrivateKey) (string, error) {
	// Serialize data to canonical JSON
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	// Sign the JSON bytes
	signature := ed25519.Sign(privateKey, jsonBytes)

	// Return base64-encoded signature
	return base64.StdEncoding.EncodeToString(signature), nil
}

// VerifyJSONSignature verifies an Ed25519 signature for JSON data
func VerifyJSONSignature(data interface{}, signatureBase64 string, publicKey ed25519.PublicKey) error {
	// Serialize data to canonical JSON
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Decode signature
	signature, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Verify signature
	if !ed25519.Verify(publicKey, jsonBytes, signature) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// SignableData represents data that can be signed (excludes signature fields)
type SignableData struct {
	Data interface{} `json:"data"`
}

// CreateSignableJobSpec creates a signable version of JobSpec (without signature/public_key)
// It returns a STRUCT copy (not a map) so JSON field order is deterministic.
func CreateSignableJobSpec(jobspec interface{}) (interface{}, error) {
	// Use reflection to create a shallow copy and zero out Signature/PublicKey if present
	v := reflect.ValueOf(jobspec)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		// Fallback to previous behavior, but still deterministic by re-marshaling into a generic struct-like map
		// Note: callers should pass a struct pointer for deterministic signing.
		var m map[string]interface{}
		b, err := json.Marshal(jobspec)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal jobspec: %w", err)
		}
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("failed to unmarshal jobspec: %w", err)
		}
		delete(m, "signature")
		delete(m, "public_key")
		return m, nil
	}
	t := v.Type()
	copy := reflect.New(t).Elem()
	copy.Set(v)
	// Zero fields named "Signature" and "PublicKey" if they exist
	if f := copy.FieldByName("Signature"); f.IsValid() && f.CanSet() && f.Kind() == reflect.String {
		f.SetString("")
	}
	if f := copy.FieldByName("PublicKey"); f.IsValid() && f.CanSet() && f.Kind() == reflect.String {
		f.SetString("")
	}
	return copy.Interface(), nil
}

// CreateSignableReceipt creates a signable version of Receipt (without signature/public_key)
// It returns a STRUCT copy (not a map) so JSON field order is deterministic.
func CreateSignableReceipt(receipt interface{}) (interface{}, error) {
	v := reflect.ValueOf(receipt)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		var m map[string]interface{}
		b, err := json.Marshal(receipt)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal receipt: %w", err)
		}
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("failed to unmarshal receipt: %w", err)
		}
		delete(m, "signature")
		delete(m, "public_key")
		return m, nil
	}
	t := v.Type()
	copy := reflect.New(t).Elem()
	copy.Set(v)
	if f := copy.FieldByName("Signature"); f.IsValid() && f.CanSet() && f.Kind() == reflect.String {
		f.SetString("")
	}
	if f := copy.FieldByName("PublicKey"); f.IsValid() && f.CanSet() && f.Kind() == reflect.String {
		f.SetString("")
	}
	return copy.Interface(), nil
}
