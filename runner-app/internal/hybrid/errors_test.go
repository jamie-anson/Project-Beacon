package hybrid

import (
	"errors"
	"net/http"
	"testing"
)

func TestHybridError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *HybridError
		expected string
	}{
		{
			name: "HTTP error with status code",
			err: &HybridError{
				Type:       ErrorTypeHTTP,
				StatusCode: 500,
				Message:    "Internal Server Error",
				URL:        "/api/inference",
			},
			expected: "hybrid http_error: HTTP 500 on /api/inference: Internal Server Error",
		},
		{
			name: "Router error without status code",
			err: &HybridError{
				Type:    ErrorTypeRouter,
				Message: "Model not available",
			},
			expected: "hybrid router_error: Model not available",
		},
		{
			name: "Network error",
			err: &HybridError{
				Type:    ErrorTypeNetwork,
				Message: "Connection refused",
			},
			expected: "hybrid network: Connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("HybridError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHybridError_IsHTTPStatus(t *testing.T) {
	err := &HybridError{
		Type:       ErrorTypeHTTP,
		StatusCode: 404,
		Message:    "Not Found",
	}

	if !err.IsHTTPStatus(404) {
		t.Error("Expected IsHTTPStatus(404) to be true")
	}

	if err.IsHTTPStatus(500) {
		t.Error("Expected IsHTTPStatus(500) to be false")
	}
}

func TestHybridError_IsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      *HybridError
		expected bool
	}{
		{
			name: "404 HTTP error",
			err: &HybridError{
				Type:       ErrorTypeHTTP,
				StatusCode: 404,
			},
			expected: true,
		},
		{
			name: "Not found error type",
			err: &HybridError{
				Type: ErrorTypeNotFound,
			},
			expected: true,
		},
		{
			name: "500 HTTP error",
			err: &HybridError{
				Type:       ErrorTypeHTTP,
				StatusCode: 500,
			},
			expected: false,
		},
		{
			name: "Router error",
			err: &HybridError{
				Type: ErrorTypeRouter,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.IsNotFound(); got != tt.expected {
				t.Errorf("HybridError.IsNotFound() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHybridError_IsTimeout(t *testing.T) {
	timeoutErr := &HybridError{
		Type:    ErrorTypeTimeout,
		Message: "Request timeout",
	}

	if !timeoutErr.IsTimeout() {
		t.Error("Expected IsTimeout() to be true for timeout error")
	}

	httpErr := &HybridError{
		Type:       ErrorTypeHTTP,
		StatusCode: 500,
	}

	if httpErr.IsTimeout() {
		t.Error("Expected IsTimeout() to be false for HTTP error")
	}
}

func TestHybridError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &HybridError{
		Type:  ErrorTypeNetwork,
		Cause: cause,
	}

	if unwrapped := err.Unwrap(); unwrapped != cause {
		t.Errorf("Expected Unwrap() to return %v, got %v", cause, unwrapped)
	}
}

func TestNewHTTPError(t *testing.T) {
	err := NewHTTPError(404, "Not Found", "/api/test")

	if err.Type != ErrorTypeNotFound {
		t.Errorf("Expected Type to be %v, got %v", ErrorTypeNotFound, err.Type)
	}

	if err.StatusCode != 404 {
		t.Errorf("Expected StatusCode to be 404, got %v", err.StatusCode)
	}

	if err.Message != "Not Found" {
		t.Errorf("Expected Message to be 'Not Found', got %v", err.Message)
	}

	if err.URL != "/api/test" {
		t.Errorf("Expected URL to be '/api/test', got %v", err.URL)
	}
}

func TestNewHTTPError_NonNotFound(t *testing.T) {
	err := NewHTTPError(500, "Internal Server Error", "/api/test")

	if err.Type != ErrorTypeHTTP {
		t.Errorf("Expected Type to be %v, got %v", ErrorTypeHTTP, err.Type)
	}

	if err.StatusCode != 500 {
		t.Errorf("Expected StatusCode to be 500, got %v", err.StatusCode)
	}
}

func TestIsHybridError(t *testing.T) {
	hybridErr := &HybridError{
		Type:    ErrorTypeRouter,
		Message: "test error",
	}

	// Test with HybridError
	if err, ok := IsHybridError(hybridErr); !ok || err != hybridErr {
		t.Error("Expected IsHybridError to return true and the error for HybridError")
	}

	// Test with regular error
	regularErr := errors.New("regular error")
	if _, ok := IsHybridError(regularErr); ok {
		t.Error("Expected IsHybridError to return false for regular error")
	}
}

func TestIsHTTPStatus(t *testing.T) {
	hybridErr := NewHTTPError(404, "Not Found", "/test")

	if !IsHTTPStatus(hybridErr, 404) {
		t.Errorf("Expected IsHTTPStatus to return true for matching status code, error type: %v", hybridErr.Type)
	}

	if IsHTTPStatus(hybridErr, 500) {
		t.Error("Expected IsHTTPStatus to return false for non-matching status code")
	}

	regularErr := errors.New("regular error")
	if IsHTTPStatus(regularErr, 404) {
		t.Error("Expected IsHTTPStatus to return false for regular error")
	}
}

func TestIsNotFound(t *testing.T) {
	notFoundErr := NewHTTPError(http.StatusNotFound, "Not Found", "/test")

	if !IsNotFound(notFoundErr) {
		t.Error("Expected IsNotFound to return true for 404 error")
	}

	serverErr := NewHTTPError(http.StatusInternalServerError, "Server Error", "/test")
	if IsNotFound(serverErr) {
		t.Error("Expected IsNotFound to return false for 500 error")
	}

	regularErr := errors.New("regular error")
	if IsNotFound(regularErr) {
		t.Error("Expected IsNotFound to return false for regular error")
	}
}

func TestIsTimeout(t *testing.T) {
	timeoutErr := NewTimeoutError("Request timeout", nil)

	if !IsTimeout(timeoutErr) {
		t.Error("Expected IsTimeout to return true for timeout error")
	}

	httpErr := NewHTTPError(500, "Server Error", "/test")
	if IsTimeout(httpErr) {
		t.Error("Expected IsTimeout to return false for HTTP error")
	}

	regularErr := errors.New("regular error")
	if IsTimeout(regularErr) {
		t.Error("Expected IsTimeout to return false for regular error")
	}
}

func TestIsRouterError(t *testing.T) {
	routerErr := NewRouterError("Model unavailable")

	if !IsRouterError(routerErr) {
		t.Error("Expected IsRouterError to return true for router error")
	}

	httpErr := NewHTTPError(500, "Server Error", "/test")
	if IsRouterError(httpErr) {
		t.Error("Expected IsRouterError to return false for HTTP error")
	}

	regularErr := errors.New("regular error")
	if IsRouterError(regularErr) {
		t.Error("Expected IsRouterError to return false for regular error")
	}
}
