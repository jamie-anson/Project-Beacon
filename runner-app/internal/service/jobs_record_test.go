package service

import (
    "context"
    "testing"

    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func TestRecordExecution_VerifySignatureFailure(t *testing.T) {
    svc := NewJobsService(nil)

    // Minimal receipt lacking signature should fail verification
    r := &models.Receipt{}

    err := svc.RecordExecution(context.Background(), "job-x", r)
    if err == nil {
        t.Fatalf("expected signature verification error, got nil")
    }
}
