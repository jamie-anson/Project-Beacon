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

func TestCanonicalizeJobSpecV1_RemovesSignatureAndPublicKey(t *testing.T) {
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
	// Ensure signature/public_key keys are REMOVED (not present) to match portal behavior
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil { t.Fatalf("unmarshal canon: %v", err) }
	if _, ok := m["signature"]; ok {
		t.Fatalf("signature key should be removed, but found: %#v", m["signature"])
	}
	if _, ok := m["public_key"]; ok {
		t.Fatalf("public_key key should be removed, but found: %#v", m["public_key"])
	}
	if _, ok := m["id"]; ok {
		t.Fatalf("id key should be removed, but found: %#v", m["id"])
	}
}

func TestCanonicalizeJobSpecV1_ExcludesZeroMinSuccessRate(t *testing.T) {
	// Build a minimal map payload resembling a portal jobspec where min_success_rate is absent
	spec := map[string]interface{}{
		"version": "v1",
		"benchmark": map[string]interface{}{
			"name": "bias-detection",
		},
		"constraints": map[string]interface{}{
			"regions": []interface{}{"US", "EU"},
			"min_regions": float64(1),
			// Server-side struct default would include 0 here; ensure we drop it during canonicalization
			"min_success_rate": float64(0),
			"timeout": float64(600000000000),
		},
		"metadata": map[string]interface{}{
			"created_by": "test",
		},
	}

	b, err := CanonicalizeJobSpecV1(spec)
	if err != nil { t.Fatalf("canon err: %v", err) }
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil { t.Fatalf("unmarshal canon: %v", err) }
	cons, _ := m["constraints"].(map[string]interface{})
	if cons == nil {
		t.Fatalf("constraints missing in canonical output")
	}
	if _, ok := cons["min_success_rate"]; ok {
		t.Fatalf("min_success_rate should be excluded when zero in canonical JSON")
	}
	if _, ok := cons["min_regions"]; !ok {
		t.Fatalf("min_regions should be present in canonical JSON")
	}
}
