package crypto

import (
	"encoding/json"
	"testing"
)

// helper struct simulating minimal JobSpec fields used by CreateSignableJobSpec via reflection
// We keep field names "Signature" and "PublicKey" to ensure they are zeroed (not stripped).
type miniJobSpec struct {
	ID        string                 `json:"id"`
	Version   string                 `json:"version"`
	Metadata  map[string]interface{} `json:"metadata"`
	Signature string                 `json:"signature"`
	PublicKey string                 `json:"public_key"`
}

func TestCanonicalizeGenericV1_SortsKeysAndIsDeterministic(t *testing.T) {
	in1 := map[string]interface{}{
		"b": 2,
		"a": []interface{}{3, 1, 2},
		"z": map[string]interface{}{"y": 1, "x": 2},
	}
	in2 := map[string]interface{}{
		"z": map[string]interface{}{"x": 2, "y": 1},
		"a": []interface{}{3, 1, 2},
		"b": 2,
	}
	b1, err := CanonicalizeGenericV1(in1)
	if err != nil { t.Fatalf("canon1 err: %v", err) }
	b2, err := CanonicalizeGenericV1(in2)
	if err != nil { t.Fatalf("canon2 err: %v", err) }
	if string(b1) != string(b2) {
		t.Fatalf("canonicalization not deterministic.\n1=%s\n2=%s", string(b1), string(b2))
	}
}

func TestCanonicalizeJobSpecV1_RetainsZeroedSignatureAndPublicKey(t *testing.T) {
	js := miniJobSpec{
		ID:      "job-1",
		Version: "1",
		Metadata: map[string]interface{}{
			"k": "v",
		},
		Signature: "abc",
		PublicKey: "def",
	}
	b, err := CanonicalizeJobSpecV1(&js)
	if err != nil { t.Fatalf("canon err: %v", err) }
	// Ensure signature/public_key keys are present but zeroed in canonical bytes
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil { t.Fatalf("unmarshal canon: %v", err) }
	if v, ok := m["signature"]; !ok {
		t.Fatalf("signature key should be retained (zeroed)")
	} else if s, ok2 := v.(string); !ok2 || s != "" {
		t.Fatalf("signature should be empty string, got: %#v", v)
	}
	if v, ok := m["public_key"]; !ok {
		t.Fatalf("public_key key should be retained (zeroed)")
	} else if s, ok2 := v.(string); !ok2 || s != "" {
		t.Fatalf("public_key should be empty string, got: %#v", v)
	}
}
