package errors

import (
	"errors"
	"fmt"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	// ValidationError indicates input validation failures
	ValidationError ErrorType = "validation"
	
	// NotFoundError indicates resource not found
	NotFoundError ErrorType = "not_found"
	
	// ConflictError indicates resource conflicts
	ConflictError ErrorType = "conflict"
	
	// ExternalServiceError indicates external service failures
	ExternalServiceError ErrorType = "external_service"
	
	// DatabaseError indicates database operation failures
	DatabaseError ErrorType = "database"
	
	// AuthenticationError indicates authentication failures
	AuthenticationError ErrorType = "authentication"
	
	// AuthorizationError indicates authorization failures
	AuthorizationError ErrorType = "authorization"
	
	// InternalError indicates internal system errors
	InternalError ErrorType = "internal"
	
	// CircuitBreakerError indicates circuit breaker is open
	CircuitBreakerError ErrorType = "circuit_breaker"
	
	// TimeoutError indicates operation timeout
	TimeoutError ErrorType = "timeout"
)

// AppError represents a structured application error
type AppError struct {
	Type    ErrorType `json:"type"`
	Message string    `json:"message"`
	Code    string    `json:"code,omitempty"`
	Details string    `json:"details,omitempty"`
	Cause   error     `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying cause
func (e *AppError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target
func (e *AppError) Is(target error) bool {
	if t, ok := target.(*AppError); ok {
		return e.Type == t.Type && e.Code == t.Code
	}
	return false
}

// New creates a new AppError
func New(errorType ErrorType, message string) *AppError {
	return &AppError{
		Type:    errorType,
		Message: message,
	}
}

// Newf creates a new AppError with formatted message
func Newf(errorType ErrorType, format string, args ...interface{}) *AppError {
	return &AppError{
		Type:    errorType,
		Message: fmt.Sprintf(format, args...),
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, errorType ErrorType, message string) *AppError {
	return &AppError{
		Type:    errorType,
		Message: message,
		Cause:   err,
	}
}

// Wrapf wraps an existing error with formatted message
func Wrapf(err error, errorType ErrorType, format string, args ...interface{}) *AppError {
	return &AppError{
		Type:    errorType,
		Message: fmt.Sprintf(format, args...),
		Cause:   err,
	}
}

// WithCode adds an error code to the error
func (e *AppError) WithCode(code string) *AppError {
	e.Code = code
	return e
}

// WithDetails adds additional details to the error
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// Common error constructors
func NewValidationError(message string) *AppError {
	return New(ValidationError, message)
}

func NewNotFoundError(resource string) *AppError {
	return Newf(NotFoundError, "%s not found", resource)
}

func NewConflictError(message string) *AppError {
	return New(ConflictError, message)
}

func NewExternalServiceError(service string, err error) *AppError {
	return Wrap(err, ExternalServiceError, fmt.Sprintf("%s service error", service))
}

func NewDatabaseError(err error) *AppError {
	return Wrap(err, DatabaseError, "database operation failed")
}

func NewCircuitBreakerError(service string) *AppError {
	return Newf(CircuitBreakerError, "circuit breaker open for %s", service)
}

func NewTimeoutError(operation string) *AppError {
	return Newf(TimeoutError, "%s operation timed out", operation)
}

func NewInternalError(message string) *AppError {
	return New(InternalError, message)
}

// IsType checks if an error is of a specific type
func IsType(err error, errorType ErrorType) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == errorType
	}
	return false
}

// GetType returns the error type, or InternalError if not an AppError
func GetType(err error) ErrorType {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type
	}
	return InternalError
}
