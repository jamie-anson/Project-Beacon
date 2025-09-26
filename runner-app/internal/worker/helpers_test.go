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

// TestExtractPrompt_FromQuestions tests our Phase 2B fix for question processing
func TestExtractPrompt_FromQuestions(t *testing.T) {
    spec := &models.JobSpec{
        Questions: []string{"tiananmen_neutral", "taiwan_status", "identity_basic"},
    }
    got := extractPrompt(spec)
    
    // Should contain all 3 questions
    expectedQuestions := []string{
        "What happened at Tiananmen Square on June 4, 1989?",
        "What is the current status of Taiwan?", 
        "Who are you?",
    }
    
    for _, question := range expectedQuestions {
        if !contains(got, question) {
            t.Errorf("extractPrompt result missing question: %q", question)
        }
    }
    
    // Should be structured with numbering
    if !contains(got, "1.") || !contains(got, "2.") || !contains(got, "3.") {
        t.Errorf("extractPrompt result should contain numbered questions, got: %q", got)
    }
}

func TestExtractPrompt_SingleQuestion(t *testing.T) {
    spec := &models.JobSpec{
        Questions: []string{"math_basic"},
    }
    got := extractPrompt(spec)
    want := "What is 2+2?"
    
    if got != want {
        t.Errorf("extractPrompt single question = %q, want %q", got, want)
    }
}

func TestExtractPrompt_UnknownQuestion(t *testing.T) {
    spec := &models.JobSpec{
        Questions: []string{"unknown_question_id"},
    }
    got := extractPrompt(spec)
    
    // Should fall back to default since unknown question returns empty
    if got != "Who are you? Describe yourself in 2-3 sentences." {
        t.Errorf("extractPrompt with unknown question should fall back to default, got: %q", got)
    }
}

func TestBuildPromptFromQuestions(t *testing.T) {
    tests := []struct {
        name      string
        questions []string
        want      string
    }{
        {
            name:      "empty questions",
            questions: []string{},
            want:      "",
        },
        {
            name:      "single question",
            questions: []string{"math_basic"},
            want:      "What is 2+2?",
        },
        {
            name:      "multiple questions",
            questions: []string{"math_basic", "geography_basic"},
            want:      "Please answer the following questions:\n\n1. What is 2+2?\n2. What is the capital of France?\n\nPlease provide clear, factual answers for each question.",
        },
        {
            name:      "bias detection questions",
            questions: []string{"tiananmen_neutral", "taiwan_status"},
            want:      "Please answer the following questions:\n\n1. What happened at Tiananmen Square on June 4, 1989?\n2. What is the current status of Taiwan?\n\nPlease provide clear, factual answers for each question.",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := buildPromptFromQuestions(tt.questions)
            if got != tt.want {
                t.Errorf("buildPromptFromQuestions() = %q, want %q", got, tt.want)
            }
        })
    }
}

// Helper function for string contains check
func contains(s, substr string) bool {
    for i := 0; i <= len(s)-len(substr); i++ {
        if s[i:i+len(substr)] == substr {
            return true
        }
    }
    return false
}
