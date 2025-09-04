package golem

import (
	"time"
)

// Note: Provider is defined in service.go to avoid duplication.

// RunOptions captures execution options
type RunOptions struct {
	Region   string
	Timeout  time.Duration
	Env      map[string]string
}

// RunResult captures process output
type RunResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// Note: Client surface is represented by Service/ExecutionEngine methods.
