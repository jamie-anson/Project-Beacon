package middleware

import (
	"testing"
)

// TestValidationErrorError covers the Error() method formatting
func TestValidationErrorError(t *testing.T) {
	err := &ValidationError{Field: "id", Message: "is required"}
	if got, want := err.Error(), "id: is required"; got != want {
		t.Fatalf("unexpected error string: got %q want %q", got, want)
	}
}

// TestValidateJobSpecFields_NoOp covers the current no-op implementation for coverage
func TestValidateJobSpecFields_NoOp(t *testing.T) {
	if err := validateJobSpecFields(map[string]interface{}{"id": "x"}); err != nil {
		t.Fatalf("expected nil from validateJobSpecFields, got %v", err)
	}
}
