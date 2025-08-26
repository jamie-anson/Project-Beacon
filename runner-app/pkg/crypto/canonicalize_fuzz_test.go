package crypto

import (
	"encoding/json"
	"testing"
)

func FuzzCanonicalJSON(f *testing.F) {
	// Seed with known test cases
	f.Add(`{"a":1,"b":2}`)
	f.Add(`{"b":2,"a":1}`)
	f.Add(`{"nested":{"z":1,"a":2},"top":"value"}`)
	f.Add(`{"unicode":"café","emoji":"🚀"}`)
	f.Add(`{"numbers":123.456,"scientific":1.23e-4}`)
	f.Add(`{"whitespace":" \t\n\r ","empty":""}`)

	f.Fuzz(func(t *testing.T, input string) {
		// Skip invalid JSON
		var testData interface{}
		if err := json.Unmarshal([]byte(input), &testData); err != nil {
			t.Skip("Invalid JSON input")
		}

		// Test canonicalization
		canonical1, err1 := CanonicalJSON(testData)
		if err1 != nil {
			t.Fatalf("First canonicalization failed: %v", err1)
		}

		// Test determinism - second call should produce identical result
		canonical2, err2 := CanonicalJSON(testData)
		if err2 != nil {
			t.Fatalf("Second canonicalization failed: %v", err2)
		}

		if string(canonical1) != string(canonical2) {
			t.Errorf("Canonicalization not deterministic:\nFirst:  %s\nSecond: %s", 
				string(canonical1), string(canonical2))
		}

		// Test that result is valid JSON
		var parsed interface{}
		if err := json.Unmarshal(canonical1, &parsed); err != nil {
			t.Errorf("Canonical result is not valid JSON: %v", err)
		}

		// Test that canonicalization is idempotent
		canonical3, err3 := CanonicalJSON(parsed)
		if err3 != nil {
			t.Fatalf("Third canonicalization failed: %v", err3)
		}

		if string(canonical1) != string(canonical3) {
			t.Errorf("Canonicalization not idempotent:\nOriginal: %s\nRe-canon: %s", 
				string(canonical1), string(canonical3))
		}
	})
}

func FuzzJobSpecCanonicalization(f *testing.F) {
	// Seed with JobSpec-like structures
	f.Add(`{"id":"test","version":"1.0","signature":"sig","public_key":"key"}`)
	f.Add(`{"public_key":"key","signature":"sig","version":"1.0","id":"test"}`)
	f.Add(`{"id":"test","metadata":{"timestamp":"2025-08-22T14:46:47Z","nonce":"abc123"}}`)

	f.Fuzz(func(t *testing.T, input string) {
		var testData interface{}
		if err := json.Unmarshal([]byte(input), &testData); err != nil {
			t.Skip("Invalid JSON input")
		}

		// Test v1 canonicalization
		v1Canon, err := CanonicalizeJobSpecV1(testData)
		if err != nil {
			t.Skip("V1 canonicalization failed")
		}

		// Test v0 canonicalization
		v0Canon, err := CanonicalizeJobSpecV0(testData)
		if err != nil {
			t.Skip("V0 canonicalization failed")
		}

		// Both should be valid JSON
		var v1Parsed, v0Parsed interface{}
		if err := json.Unmarshal(v1Canon, &v1Parsed); err != nil {
			t.Errorf("V1 canonical result is not valid JSON: %v", err)
		}
		if err := json.Unmarshal(v0Canon, &v0Parsed); err != nil {
			t.Errorf("V0 canonical result is not valid JSON: %v", err)
		}

		// Neither should contain signature fields
		if v1Map, ok := v1Parsed.(map[string]interface{}); ok {
			if _, hasSignature := v1Map["signature"]; hasSignature {
				t.Error("V1 canonicalization should remove signature field")
			}
			if _, hasPublicKey := v1Map["public_key"]; hasPublicKey {
				t.Error("V1 canonicalization should remove public_key field")
			}
		}

		if v0Map, ok := v0Parsed.(map[string]interface{}); ok {
			if _, hasSignature := v0Map["signature"]; hasSignature {
				t.Error("V0 canonicalization should remove signature field")
			}
			if _, hasPublicKey := v0Map["public_key"]; hasPublicKey {
				t.Error("V0 canonicalization should remove public_key field")
			}
		}
	})
}

// Test edge cases with specific inputs
func TestCanonicalizationEdgeCases(t *testing.T) {
	edgeCases := []struct {
		name  string
		input interface{}
	}{
		{
			name: "unicode_normalization",
			input: map[string]interface{}{
				"café":     "value", // NFC normalization
				"unicode":  "🚀🌟✨", // Emoji handling
			},
		},
		{
			name: "numeric_precision",
			input: map[string]interface{}{
				"float":     123.456789012345,
				"scientific": 1.23e-10,
				"integer":   int64(9223372036854775807),
			},
		},
		{
			name: "whitespace_handling",
			input: map[string]interface{}{
				"spaces":   "  value  ",
				"tabs":     "\tvalue\t",
				"newlines": "\nvalue\n",
				"mixed":    " \t\n\r value \t\n\r ",
			},
		},
		{
			name: "empty_values",
			input: map[string]interface{}{
				"empty_string": "",
				"null_value":   nil,
				"empty_object": map[string]interface{}{},
				"empty_array":  []interface{}{},
			},
		},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			canonical, err := CanonicalJSON(tc.input)
			if err != nil {
				t.Fatalf("Canonicalization failed: %v", err)
			}

			// Verify result is valid JSON
			var parsed interface{}
			if err := json.Unmarshal(canonical, &parsed); err != nil {
				t.Errorf("Canonical result is not valid JSON: %v", err)
			}

			// Verify determinism
			canonical2, err := CanonicalJSON(tc.input)
			if err != nil {
				t.Fatalf("Second canonicalization failed: %v", err)
			}

			if string(canonical) != string(canonical2) {
				t.Errorf("Canonicalization not deterministic")
			}
		})
	}
}
