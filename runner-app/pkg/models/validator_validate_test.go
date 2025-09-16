package models

import (
    "testing"
    "time"

    "github.com/jamie-anson/project-beacon-runner/pkg/crypto"
)

func makeValidJobSpec() *JobSpec {
    return &JobSpec{
        ID:      "job-validate-1",
        Version: "v1",
        Benchmark: BenchmarkSpec{
            Name:        "who-are-you",
            Description: "test benchmark",
            Container: ContainerSpec{
                Image: "alpine",
                Tag:   "3.19",
                Resources: ResourceSpec{
                    CPU:    "100m",
                    Memory: "64Mi",
                },
            },
            Input: InputSpec{
                Type: "prompt",
                Data: map[string]interface{}{"text": "Hello"},
                Hash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", // 64 hex chars
            },
            Scoring: ScoringSpec{Method: "custom", Parameters: map[string]interface{}{}},
            Metadata: map[string]interface{}{},
        },
        Constraints: ExecutionConstraints{
            Regions:         []string{"US"},
            MinRegions:      1,
            MinSuccessRate:  0.67,
            Timeout:         1 * time.Minute,
            ProviderTimeout: 10 * time.Second,
        },
        Metadata:  map[string]interface{}{},
        CreatedAt: time.Now(),
    }
}

func TestValidateAndVerify_Success_WithSignedJobSpec(t *testing.T) {
    js := makeValidJobSpec()
    kp, err := crypto.GenerateKeyPair()
    if err != nil { t.Fatalf("keygen: %v", err) }
    if err := js.Sign(kp.PrivateKey); err != nil {
        t.Fatalf("sign: %v", err)
    }

    v := NewJobSpecValidator()
    if err := v.ValidateAndVerify(js); err != nil {
        t.Fatalf("ValidateAndVerify expected success, got err: %v", err)
    }
}

func TestValidateAndVerify_Succeeds_MissingSignature(t *testing.T) {
    js := makeValidJobSpec()
    // Intentionally do not sign - signature verification is now optional
    v := NewJobSpecValidator()
    if err := v.ValidateAndVerify(js); err != nil {
        t.Fatalf("expected success without signature, got err: %v", err)
    }
}
