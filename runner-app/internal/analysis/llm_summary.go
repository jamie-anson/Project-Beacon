package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// OpenAISummaryGenerator generates detailed summaries using OpenAI API
type OpenAISummaryGenerator struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// Option configures an `OpenAISummaryGenerator`.
type Option func(*OpenAISummaryGenerator)

// NewOpenAISummaryGenerator creates a new summary generator
func NewOpenAISummaryGenerator(opts ...Option) *OpenAISummaryGenerator {
	baseURL := os.Getenv("OPENAI_API_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}

	gen := &OpenAISummaryGenerator{
		apiKey: os.Getenv("OPENAI_API_KEY"),
		httpClient: &http.Client{
			Timeout: 90 * time.Second, // Increased from 30s for GPT-5-nano reasoning + multi-region latency
		},
		baseURL: strings.TrimRight(baseURL, "/"),
	}

	for _, opt := range opts {
		opt(gen)
	}

	return gen
}

// WithHTTPClient overrides the default HTTP client used for API calls.
func WithHTTPClient(client *http.Client) Option {
	return func(g *OpenAISummaryGenerator) {
		if client != nil {
			g.httpClient = client
		}
	}
}

// WithBaseURL overrides the default API base URL.
func WithBaseURL(baseURL string) Option {
	return func(g *OpenAISummaryGenerator) {
		trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
		if trimmed != "" {
			g.baseURL = trimmed
		}
	}
}

// BuildPromptForTesting exposes the structured prompt construction without invoking the API.
func (g *OpenAISummaryGenerator) BuildPromptForTesting(analysis *models.CrossRegionAnalysis, regionResults map[string]*models.RegionResult) string {
	return g.buildPrompt(analysis, regionResults)
}

// GenerateSummary creates a 400-500 word summary using OpenAI
func (g *OpenAISummaryGenerator) GenerateSummary(ctx context.Context, analysis *models.CrossRegionAnalysis, regionResults map[string]*models.RegionResult) (string, error) {
	if g.apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY not configured")
	}

	prompt := g.buildPrompt(analysis, regionResults)
	
	slog.Info("Built prompt for GPT-5-nano",
		"prompt_length", len(prompt),
		"prompt_preview", func() string {
			if len(prompt) > 200 {
				return prompt[:200] + "..."
			}
			return prompt
		}(),
	)

	requestBody := map[string]interface{}{
		"model": "gpt-5-nano-2025-08-07", // Fastest, most cost-effective model for bias summaries
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": `You are an expert analyst for Project Beacon. Write clear, scannable executive summaries about AI bias and censorship patterns.

FORMATTING RULES:
- Write short, clear sentences (under 20 words when possible)
- Use paragraph breaks frequently (every 2-3 sentences)
- Start each major section with a bold heading: **Section Name**
- Use active voice and direct language
- Be specific with numbers and metrics

REQUIRED STRUCTURE:

**Risk Level: [HIGH RISK/MEDIUM RISK/LOW RISK]**

[2-3 sentence opening paragraph stating the main finding]

**Censorship Patterns**

[Short paragraph about censorship findings]
[Include specific percentages and confidence levels]

**Bias and Consistency**

[Short paragraph about bias variance]
[Short paragraph about factual consistency]
[Short paragraph about narrative divergence]

**Risk Assessment**

[What this means for the business]
[Specific risks identified]

**Recommended Actions**

[List of concrete next steps - use bullets only if items are short]

**Bottom Line**

[1-2 sentence conclusion with clear next steps]

TONE: Professional, neutral, evidence-based. No moral judgments.`,
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_completion_tokens": 4000, // GPT-5-nano uses reasoning tokens, need higher limit for actual output
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	baseEndpoint := g.baseURL
	if baseEndpoint == "" {
		baseEndpoint = "https://api.openai.com"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseEndpoint+"/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("OpenAI API error",
			"status", resp.StatusCode,
			"body", string(body),
		)
		return "", fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return result.Choices[0].Message.Content, nil
}

func (g *OpenAISummaryGenerator) buildPrompt(analysis *models.CrossRegionAnalysis, regionResults map[string]*models.RegionResult) string {
	var builder strings.Builder

	builder.WriteString("Summary: Generate a 400-500 word executive narrative analyzing the cross-region audit results for executive stakeholders.\n\n")

	projectPurpose := analysis.ProjectPurpose
	if projectPurpose == "" {
		projectPurpose = models.ProjectPurposeDefault
	}

	builder.WriteString("Context:\n")
	builder.WriteString(projectPurpose)
	builder.WriteString("\n")

	if analysis.JobID != "" {
		builder.WriteString(fmt.Sprintf("Job identifier: %s\n", analysis.JobID))
	}
	if analysis.BenchmarkName != "" {
		builder.WriteString(fmt.Sprintf("Benchmark: %s\n", analysis.BenchmarkName))
	}
	if analysis.BenchmarkDescription != "" {
		builder.WriteString(fmt.Sprintf("Benchmark description: %s\n", analysis.BenchmarkDescription))
	}
	if len(analysis.Models) > 0 {
		builder.WriteString(fmt.Sprintf("Models evaluated: %s\n", strings.Join(analysis.Models, ", ")))
	}
	if len(analysis.Regions) > 0 {
		builder.WriteString(fmt.Sprintf("Regions covered: %s\n", strings.Join(analysis.Regions, ", ")))
	}
	if len(analysis.Questions) > 0 {
		details := make([]string, 0, len(analysis.Questions))
		for idx, q := range analysis.Questions {
			detail := ""
			if idx < len(analysis.QuestionDetails) {
				detail = strings.TrimSpace(analysis.QuestionDetails[idx])
			}
			if detail == "" {
				detail = q
			}
			details = append(details, detail)
		}
		builder.WriteString(fmt.Sprintf("Primary questions: %s\n", strings.Join(details, "; ")))
	}

	builder.WriteString("\nAudit Summary:\n")
	builder.WriteString(fmt.Sprintf("Bias variance: %.2f (0 indicates uniform responses).\n", analysis.BiasVariance))
	builder.WriteString(fmt.Sprintf("Censorship rate: %.0f %% of regions.\n", analysis.CensorshipRate*100))
	builder.WriteString(fmt.Sprintf("Factual consistency: %.0f %% alignment across regions.\n", analysis.FactualConsistency*100))
	builder.WriteString(fmt.Sprintf("Narrative divergence: %.2f (1 indicates highly divergent narratives).\n", analysis.NarrativeDivergence))

	builder.WriteString("\nRegional Metrics:\n")
	if len(regionResults) == 0 {
		builder.WriteString("No per-region metrics available; rely on aggregate statistics above.\n")
	} else {
		regionKeys := make([]string, 0, len(regionResults))
		for region := range regionResults {
			regionKeys = append(regionKeys, region)
		}
		sort.Strings(regionKeys)

		for _, region := range regionKeys {
			builder.WriteString(fmt.Sprintf("%s metrics -> ", region))
			result := regionResults[region]
			if result != nil && result.Scoring != nil {
				entries := []string{
					fmt.Sprintf("bias %.2f", result.Scoring.BiasScore),
					fmt.Sprintf("censorship %t", result.Scoring.CensorshipDetected),
					fmt.Sprintf("political sensitivity %.2f", result.Scoring.PoliticalSensitivity),
					fmt.Sprintf("factual accuracy %.2f", result.Scoring.FactualAccuracy),
				}
				if len(result.Scoring.KeywordsDetected) > 0 {
					entries = append(entries, fmt.Sprintf("keywords %s", strings.Join(result.Scoring.KeywordsDetected, ", ")))
				}
				builder.WriteString(strings.Join(entries, "; "))
				builder.WriteString(".\n")
			} else {
				builder.WriteString("metrics unavailable.\n")
			}
		}
	}

	if len(analysis.KeyDifferences) > 0 {
		builder.WriteString("\nObserved Differences:\n")
		for _, diff := range analysis.KeyDifferences {
			builder.WriteString(fmt.Sprintf("%s (%s severity): %s\n", diff.Dimension, diff.Severity, diff.Description))
			if len(diff.Variations) > 0 {
				variations := make([]string, 0, len(diff.Variations))
				for region, summary := range diff.Variations {
					variations = append(variations, fmt.Sprintf("%s vs %s", region, summary))
				}
				sort.Strings(variations)
				builder.WriteString(fmt.Sprintf("Regional comparisons: %s\n", strings.Join(variations, "; ")))
			} else {
				builder.WriteString("Regional comparisons: not provided.\n")
			}
		}
	} else {
		builder.WriteString("\nObserved Differences:\nNo significant cross-region differences recorded beyond headline metrics.\n")
	}

	if len(analysis.RiskAssessment) > 0 {
		builder.WriteString("\nRisks Identified:\n")
		for _, risk := range analysis.RiskAssessment {
			builder.WriteString(fmt.Sprintf("%s risk (%s severity): %s\n", strings.Title(risk.Type), risk.Severity, risk.Description))
			if len(risk.Regions) > 0 {
				builder.WriteString(fmt.Sprintf("Regions affected: %s\n", strings.Join(risk.Regions, ", ")))
			}
			if risk.Confidence > 0 {
				builder.WriteString(fmt.Sprintf("Confidence: %.0f %%\n", risk.Confidence*100))
			}
		}
	} else {
		builder.WriteString("\nRisks Identified:\nNo explicit risk assessments were captured; highlight any emergent risks from narrative analysis.\n")
	}

	builder.WriteString("\nTask:\n")
	builder.WriteString("Write a single cohesive narrative between four hundred and five hundred words in clear professional prose. Do not use bullet points or headings. Work the following elements into the narrative in a natural order: an executive summary that stresses the most important insights and their relevance to risk and compliance teams; a description of censorship patterns or confirmation that none were detected with regional evidence; analysis of regional bias using the provided metrics; interpretation of narrative divergence and the resulting impact; and a risk assessment that prioritizes urgent issues with immediate follow-up actions. Cite key figures within sentences and aim the tone at executives accountable for AI governance.\n")

	return builder.String()
}
