package worker

import (
    "testing"
    models "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func TestExtractModel_Default(t *testing.T) {
    got := extractModel(&models.JobSpec{})
    want := "llama3.2-1b"
    if got != want {
        t.Fatalf("extractModel default = %q, want %q", got, want)
    }
}

func TestExtractPrompt_Default(t *testing.T) {
    got := extractPrompt(&models.JobSpec{})
    if got == "" {
        t.Fatalf("extractPrompt returned empty string; want non-empty default prompt")
    }
}

func TestExtractPrompt_FromSpec(t *testing.T) {
    spec := &models.JobSpec{
        Benchmark: models.BenchmarkSpec{
            Input: models.InputSpec{
                Type: "prompt",
                Data: map[string]interface{}{"prompt": "Hi from test"},
            },
        },
    }
    got := extractPrompt(spec)
    if got != "Hi from test" {
        t.Fatalf("extractPrompt = %q, want %q", got, "Hi from test")
    }
}

func TestMapRegionToRouter(t *testing.T) {
    cases := map[string]string{
        "US":   "us-east",
        "EU":   "eu-west",
        "APAC": "asia-pacific",
        "ASIA": "asia-pacific",
        "":     "eu-west", // default fallback
    }
    for in, want := range cases {
        if got := mapRegionToRouter(in); got != want {
            t.Fatalf("mapRegionToRouter(%q) = %q, want %q", in, got, want)
        }
    }
}
