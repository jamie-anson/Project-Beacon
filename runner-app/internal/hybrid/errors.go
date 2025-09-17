package hybrid

import (
	"fmt"
	"net/http"
)

// HybridError represents different types of errors that can occur with the hybrid router
type HybridError struct {
	Type       ErrorType
	StatusCode int
	Message    string
	URL        string
	Cause      error
}

// ErrorType represents the category of hybrid router error
type ErrorType string

const (
	ErrorTypeHTTP       ErrorType = "http_error"
	ErrorTypeNotFound   ErrorType = "not_found"
	ErrorTypeTimeout    ErrorType = "timeout"
	ErrorTypeNetwork    ErrorType = "network"
	ErrorTypeRouter     ErrorType = "router_error"
	ErrorTypeJSON       ErrorType = "json_error"
	ErrorTypeUnknown    ErrorType = "unknown"
)

// Error implements the error interface
func (e *HybridError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("hybrid %s: HTTP %d on %s: %s", e.Type, e.StatusCode, e.URL, e.Message)
	}
	return fmt.Sprintf("hybrid %s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying error for error wrapping
func (e *HybridError) Unwrap() error {
	return e.Cause
}

// IsHTTPStatus checks if the error is an HTTP error with the specified status code
func (e *HybridError) IsHTTPStatus(code int) bool {
	return (e.Type == ErrorTypeHTTP || e.Type == ErrorTypeNotFound) && e.StatusCode == code
}

// IsNotFound checks if the error is a 404 Not Found error
func (e *HybridError) IsNotFound() bool {
	return e.Type == ErrorTypeNotFound || e.IsHTTPStatus(http.StatusNotFound)
}

// IsTimeout checks if the error is a timeout error
func (e *HybridError) IsTimeout() bool {
	return e.Type == ErrorTypeTimeout
}

// IsRouterError checks if the error is a router-level error (non-HTTP)
func (e *HybridError) IsRouterError() bool {
	return e.Type == ErrorTypeRouter
}

// NewHTTPError creates a new HTTP error
func NewHTTPError(statusCode int, message, url string) *HybridError {
	errorType := ErrorTypeHTTP
	if statusCode == http.StatusNotFound {
		errorType = ErrorTypeNotFound
	}
	
	return &HybridError{
		Type:       errorType,
		StatusCode: statusCode,
		Message:    message,
		URL:        url,
	}
}

// NewRouterError creates a new router error
func NewRouterError(message string) *HybridError {
	return &HybridError{
		Type:    ErrorTypeRouter,
		Message: message,
	}
}

// NewNetworkError creates a new network error
func NewNetworkError(message string, cause error) *HybridError {
	return &HybridError{
		Type:    ErrorTypeNetwork,
		Message: message,
		Cause:   cause,
	}
}

// NewJSONError creates a new JSON parsing error
func NewJSONError(message string, cause error) *HybridError {
	return &HybridError{
		Type:    ErrorTypeJSON,
		Message: message,
		Cause:   cause,
	}
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(message string, cause error) *HybridError {
	return &HybridError{
		Type:    ErrorTypeTimeout,
		Message: message,
		Cause:   cause,
	}
}

// IsHybridError checks if an error is a HybridError
func IsHybridError(err error) (*HybridError, bool) {
	if hybridErr, ok := err.(*HybridError); ok {
		return hybridErr, true
	}
	return nil, false
}

// IsHTTPStatus checks if an error is an HTTP error with the specified status code
func IsHTTPStatus(err error, code int) bool {
	if hybridErr, ok := IsHybridError(err); ok {
		return hybridErr.IsHTTPStatus(code)
	}
	return false
}

// IsNotFound checks if an error is a 404 Not Found error
func IsNotFound(err error) bool {
	if hybridErr, ok := IsHybridError(err); ok {
		return hybridErr.IsNotFound()
	}
	return false
}

// IsTimeout checks if an error is a timeout error
func IsTimeout(err error) bool {
	if hybridErr, ok := IsHybridError(err); ok {
		return hybridErr.IsTimeout()
	}
	return false
}

// IsRouterError checks if an error is a router-level error
func IsRouterError(err error) bool {
	if hybridErr, ok := IsHybridError(err); ok {
		return hybridErr.IsRouterError()
	}
	return false
}
