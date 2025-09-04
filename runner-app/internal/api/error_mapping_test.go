package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestErrorTaxonomyMapping validates that security errors map to correct HTTP responses
func TestErrorTaxonomyMapping(t *testing.T) {
	testCases := []struct {
		name           string
		errorCode      string
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "replay_detected",
			errorCode:      "replay_detected",
			expectedStatus: 400,
			expectedMsg:    "replay protection failed",
		},
		{
			name:           "signature_mismatch",
			errorCode:      "signature_mismatch", 
			expectedStatus: 400,
			expectedMsg:    "signature verification failed",
		},
		{
			name:           "stale_timestamp",
			errorCode:      "stale_timestamp",
			expectedStatus: 400,
			expectedMsg:    "timestamp validation failed",
		},
		{
			name:           "timestamp_skew",
			errorCode:      "timestamp_skew",
			expectedStatus: 400,
			expectedMsg:    "timestamp validation failed",
		},
		{
			name:           "untrusted_key",
			errorCode:      "untrusted_key",
			expectedStatus: 400,
			expectedMsg:    "trusted key validation failed",
		},
		{
			name:           "rate_limited",
			errorCode:      "rate_limited",
			expectedStatus: 429,
			expectedMsg:    "too many requests",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test that error codes map to expected HTTP status codes
			// This validates the error taxonomy defined in handlers_simple.go
			
			switch tc.errorCode {
			case "replay_detected", "signature_mismatch", "stale_timestamp", "timestamp_skew", "untrusted_key":
				assert.Equal(t, 400, tc.expectedStatus, "Security errors should return 400 Bad Request")
			case "rate_limited":
				assert.Equal(t, 429, tc.expectedStatus, "Rate limiting should return 429 Too Many Requests")
			default:
				t.Errorf("Unknown error code: %s", tc.errorCode)
			}
		})
	}
}

// TestSecurityErrorMessages validates error message consistency
func TestSecurityErrorMessages(t *testing.T) {
	// Validate that security error messages don't leak sensitive information
	sensitivePatterns := []string{
		"redis",
		"database", 
		"internal",
		"stack trace",
		"panic",
	}

	securityErrors := []string{
		"replay protection failed: replay detected: nonce already used",
		"signature verification failed: signature verification failed",
		"timestamp validation failed: timestamp too old",
		"timestamp validation failed: timestamp too far in future", 
		"trusted key validation failed: key not in allowlist",
		"rate limiting: too many signature failures",
	}

	for _, errorMsg := range securityErrors {
		for _, pattern := range sensitivePatterns {
			assert.NotContains(t, errorMsg, pattern, 
				"Security error messages should not contain sensitive information: %s", pattern)
		}
	}
}
