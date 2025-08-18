package golem

import "errors"

// Typed errors used across engine/client for consistent handling
var (
    ErrNotFound    = errors.New("golem: not found")
    ErrUnavailable = errors.New("golem: service unavailable")
    ErrTimeout     = errors.New("golem: timeout")
    ErrCanceled    = errors.New("golem: canceled")
)
