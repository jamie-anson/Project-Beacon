package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/analysis"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

type biasAnalysisResponse struct {
	Analysis struct {
		BiasVariance        float64                 `json:"bias_variance"`
		CensorshipRate      float64                 `json:"censorship_rate"`
		FactualConsistency  float64                 `json:"factual_consistency"`
		NarrativeDivergence float64                 `json:"narrative_divergence"`
		KeyDifferences      []models.KeyDifference  `json:"key_differences"`
		RiskAssessment      []models.RiskAssessment `json:"risk_assessment"`
	} `json:"analysis"`
	Job struct {
		JobSpecID string          `json:"jobspec_id"`
		Benchmark struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"benchmark"`
		Questions map[string]struct {
			Prompt string `json:"prompt"`
		} `json:"questions"`
	} `json:"job"`
}

func buildQuestionDetails(biasResp biasAnalysisResponse, analysisResp biasAnalysisResponse) ([]string, []string, []string) {
	questionIDs := make([]string, 0)
	questionTexts := make([]string, 0)
	modelsUsed := make([]string, 0)

	for qid, q := range biasResp.Job.Questions {
		questionIDs = append(questionIDs, qid)
		questionTexts = append(questionTexts, q.Prompt)
	}

	return questionIDs, questionTexts, modelsUsed
}

type crossRegionDiffResponse struct {
	RegionResults map[string]struct {
		Scoring *models.RegionScoring `json:"scoring"`
	} `json:"region_results"`
}

type chatCompletionRequest struct {
	Model               string              `json:"model"`
	Messages            []map[string]string `json:"messages"`
	MaxCompletionTokens int                 `json:"max_completion_tokens"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func fetchJSON(url string, out interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d from %s: %s", resp.StatusCode, url, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func buildPrompt(analysisData biasAnalysisResponse, diffData crossRegionDiffResponse) string {
	generator := analysis.NewOpenAISummaryGenerator()
	regionResults := make(map[string]*models.RegionResult)
	for region, result := range diffData.RegionResults {
		regionResults[region] = &models.RegionResult{
			Scoring: result.Scoring,
		}
	}

	analysisPayload := &models.CrossRegionAnalysis{
		BiasVariance:        analysisData.Analysis.BiasVariance,
		CensorshipRate:      analysisData.Analysis.CensorshipRate,
		FactualConsistency:  analysisData.Analysis.FactualConsistency,
		NarrativeDivergence: analysisData.Analysis.NarrativeDivergence,
		KeyDifferences:      analysisData.Analysis.KeyDifferences,
		RiskAssessment:      analysisData.Analysis.RiskAssessment,
	}

	analysisPayload.JobID = analysisData.Job.JobSpecID
	analysisPayload.BenchmarkName = analysisData.Job.Benchmark.Name
	analysisPayload.BenchmarkDescription = analysisData.Job.Benchmark.Description
	questions, questionDetails, _ := buildQuestionDetails(analysisData, analysisData)
	analysisPayload.Questions = questions
	analysisPayload.QuestionDetails = questionDetails

	return generator.BuildPromptForTesting(analysisPayload, regionResults)
}

func callChatGPT(ctx context.Context, apiKey, prompt string) (string, string, error) {
	reqBody := chatCompletionRequest{
		Model: "gpt-5-nano",
		Messages: []map[string]string{
			{
				"role":    "system",
				"content": "You are an expert analyst specializing in AI bias detection and cross-regional content analysis. Write clear, factual, professional summaries for technical audiences.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		MaxCompletionTokens: 600,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(data))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", string(body), fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var parsed chatCompletionResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", string(raw), err
	}

	if len(parsed.Choices) == 0 {
		return "", string(raw), errors.New("empty choices returned from OpenAI")
	}

	return parsed.Choices[0].Message.Content, string(raw), nil
}

func main() {
	jobID := flag.String("job", "", "Bias detection job ID to test")
	apiBase := flag.String("api", "https://beacon-runner-production.fly.dev", "Runner API base URL")
	timeout := flag.Duration("timeout", 60*time.Second, "Timeout for OpenAI request")
	flag.Parse()

	if *jobID == "" {
		log.Fatal("job ID is required")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY must be set in environment")
	}

	analysisURL := fmt.Sprintf("%s/api/v2/jobs/%s/bias-analysis", *apiBase, *jobID)
	diffURL := fmt.Sprintf("%s/api/v1/executions/%s/cross-region-diff", *apiBase, *jobID)

	var analysisData biasAnalysisResponse
	if err := fetchJSON(analysisURL, &analysisData); err != nil {
		log.Fatalf("failed to fetch bias analysis: %v", err)
	}

	var diffData crossRegionDiffResponse
	if err := fetchJSON(diffURL, &diffData); err != nil {
		log.Fatalf("failed to fetch cross-region diff: %v", err)
	}

	prompt := buildPrompt(analysisData, diffData)

	fmt.Println("=== Prompt ===")
	fmt.Println(prompt)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	summary, raw, err := callChatGPT(ctx, apiKey, prompt)
	if err != nil {
		log.Fatalf("ChatGPT request failed: %v\nRaw response: %s", err, raw)
	}

	fmt.Printf("Prompt length: %d chars\n", len(prompt))
	fmt.Println("=== GPT-5-nano Summary ===")
	fmt.Println(summary)

	if len(summary) < 300 {
		log.Fatalf("summary too short (%d chars)\nRaw response: %s", len(summary), raw)
	}

	fmt.Printf("\nPASS: summary length %d chars\n", len(summary))
}
