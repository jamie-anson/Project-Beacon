package worker

import (
	"os"
	"testing"
)

// TestMain provides a package-level guard to opt-in to heavy worker tests.
// Set RUN_WORKER_TESTS=1 to enable these tests locally.
func TestMain(m *testing.M) {
	if os.Getenv("RUN_WORKER_TESTS") == "" {
		// Exit 0 to mark the package as skipped without running tests.
		os.Exit(0)
	}
	os.Exit(m.Run())
}
