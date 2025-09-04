package jobspec

import (
	"testing"
)

func TestValidateJSONSchema_Success(t *testing.T) {
	json := []byte(`{
		"id":"sample-benchmark-001",
		"version":"1.0",
		"benchmark":{
			"name":"Who Are You?",
			"container":{"image":"beacon/text-gen","tag":"latest","resources":{"cpu":"1000m","memory":"512Mi"}},
			"input":{"type":"prompt","data":{"prompt":"hi"},"hash":"sha256:abc"}
		},
		"constraints":{"regions":["US","EU"],"min_regions":2,"timeout":"10m"},
		"metadata":{},
		"created_at":"2025-01-01T00:00:00Z",
		"signature":"sig",
		"public_key":"pk"
	}`)
	if err := ValidateJSONSchema(json); err != nil {
		t.Fatalf("expected schema to be valid, got error: %v", err)
	}
}

func TestValidateJSONSchema_MissingRequired(t *testing.T) {
	if !schemaValidationEnabled {
		t.Skip("JSON Schema validation disabled; skipping negative test")
	}
	json := []byte(`{
		"version":"1.0",
		"benchmark":{ "name":"n","container":{"image":"i","resources":{"cpu":"1","memory":"1"}}, "input":{"type":"prompt","data":{},"hash":"h"}},
		"constraints":{"regions":["US"]},
		"created_at":"2025-01-01T00:00:00Z",
		"signature":"sig",
		"public_key":"pk"
	}`)
	if err := ValidateJSONSchema(json); err == nil {
		t.Fatalf("expected schema validation to fail for missing id")
	}
}
