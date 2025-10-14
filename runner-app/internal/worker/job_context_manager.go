package worker

import (
	"context"
	"sync"
)

// JobContextManager tracks cancellable contexts for running jobs
// This enables user-initiated job cancellation by storing cancel functions
// that can be triggered via the API
type JobContextManager struct {
	mu       sync.RWMutex
	contexts map[string]context.CancelFunc
}

// NewJobContextManager creates a new context manager
func NewJobContextManager() *JobContextManager {
	return &JobContextManager{
		contexts: make(map[string]context.CancelFunc),
	}
}

// Register stores a cancel function for a job
// Should be called when job execution starts
func (m *JobContextManager) Register(jobID string, cancel context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.contexts[jobID] = cancel
}

// Cancel triggers cancellation for a specific job
// Returns true if the job was found and cancelled, false otherwise
func (m *JobContextManager) Cancel(jobID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cancel, exists := m.contexts[jobID]; exists {
		cancel()
		delete(m.contexts, jobID)
		return true
	}
	return false
}

// Unregister removes a job's context (called on completion)
// Should be called when job finishes (success or failure)
func (m *JobContextManager) Unregister(jobID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.contexts, jobID)
}

// IsRunning checks if a job has an active context
func (m *JobContextManager) IsRunning(jobID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.contexts[jobID]
	return exists
}

// Count returns the number of jobs currently being tracked
func (m *JobContextManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.contexts)
}
