package execution

import (
	"fmt"
	"log/slog"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// ResponseExtractor extracts and normalizes responses from region results
type ResponseExtractor struct {
	logger *slog.Logger
}

// NewResponseExtractor creates a new response extractor
func NewResponseExtractor(logger *slog.Logger) *ResponseExtractor {
	return &ResponseExtractor{
		logger: logger,
	}
}

// ResponseData represents a normalized response for analysis
type ResponseData struct {
	Region      string
	ModelID     string
	QuestionID  string
	Response    string
	BiasScore   float64
	Censored    bool
	Keywords    []string
	Metadata    map[string]interface{}
}

// ExtractResponses extracts all LLM responses from region results
func (re *ResponseExtractor) ExtractResponses(regionResults map[string]*RegionResult) (map[string][]ResponseData, error) {
	if regionResults == nil {
		return nil, fmt.Errorf("region results cannot be nil")
	}

	responses := make(map[string][]ResponseData)

	for region, result := range regionResults {
		if result == nil {
			re.logger.Warn("Skipping nil region result", "region", region)
			continue
		}

		// Skip failed regions
		if result.Status != "success" {
			re.logger.Debug("Skipping failed region", "region", region, "status", result.Status)
			continue
		}

		var regionResponses []ResponseData

		// Extract from executions (new format)
		for _, exec := range result.Executions {
			if exec.Status != "completed" || exec.Receipt == nil {
				continue
			}

			responseData := re.extractFromReceipt(region, exec.ModelID, exec.QuestionID, exec.Receipt)
			if responseData != nil {
				regionResponses = append(regionResponses, *responseData)
			}
		}

		// Fallback: Extract from legacy receipt format
		if len(regionResponses) == 0 && result.Receipt != nil {
			responseData := re.extractFromReceipt(region, "unknown", "unknown", result.Receipt)
			if responseData != nil {
				regionResponses = append(regionResponses, *responseData)
			}
		}

		if len(regionResponses) > 0 {
			responses[region] = regionResponses
		}
	}

	if len(responses) == 0 {
		return nil, fmt.Errorf("no valid responses found in region results")
	}

	re.logger.Info("Extracted responses",
		"total_regions", len(responses),
		"total_responses", re.countTotalResponses(responses))

	return responses, nil
}

// extractFromReceipt extracts response data from a receipt
func (re *ResponseExtractor) extractFromReceipt(region, modelID, questionID string, receipt *models.Receipt) *ResponseData {
	if receipt == nil {
		return nil
	}

	// Extract response text
	response := re.extractResponseText(receipt)
	if response == "" {
		re.logger.Debug("Empty response in receipt", "region", region, "model", modelID, "question", questionID)
		return nil
	}

	// Extract scoring data from Output.Data
	biasScore := 0.0
	censored := false
	keywords := []string{}

	if receipt.Output.Data != nil {
		if dataMap, ok := receipt.Output.Data.(map[string]interface{}); ok {
			// Try to extract bias_score
			if scoringData, ok := dataMap["bias_score"].(map[string]interface{}); ok {
				if bs, ok := scoringData["bias_score"].(float64); ok {
					biasScore = bs
				}
				if cd, ok := scoringData["censorship_detected"].(bool); ok {
					censored = cd
				}
				if kw, ok := scoringData["keyword_flags"].([]interface{}); ok {
					for _, k := range kw {
						if str, ok := k.(string); ok {
							keywords = append(keywords, str)
						}
					}
				}
			}
		}
	}

	// Extract metadata
	metadata := make(map[string]interface{})
	if receipt.Output.Data != nil {
		if dataMap, ok := receipt.Output.Data.(map[string]interface{}); ok {
			for k, v := range dataMap {
				if k != "response" && k != "responses" && k != "bias_score" {
					metadata[k] = v
				}
			}
		}
	}

	return &ResponseData{
		Region:     region,
		ModelID:    modelID,
		QuestionID: questionID,
		Response:   response,
		BiasScore:  biasScore,
		Censored:   censored,
		Keywords:   keywords,
		Metadata:   metadata,
	}
}

// extractResponseText extracts the response text from receipt data
func (re *ResponseExtractor) extractResponseText(receipt *models.Receipt) string {
	if receipt.Output.Data == nil {
		return ""
	}

	dataMap, ok := receipt.Output.Data.(map[string]interface{})
	if !ok {
		return ""
	}

	// Try "response" field (single response)
	if resp, ok := dataMap["response"].(string); ok && resp != "" {
		return resp
	}

	// Try "responses" field (multiple responses)
	if responses, ok := dataMap["responses"].([]interface{}); ok && len(responses) > 0 {
		if firstResp, ok := responses[0].(string); ok {
			return firstResp
		}
		if firstResp, ok := responses[0].(map[string]interface{}); ok {
			if text, ok := firstResp["text"].(string); ok {
				return text
			}
			if content, ok := firstResp["content"].(string); ok {
				return content
			}
		}
	}

	// Try "text" field
	if text, ok := dataMap["text"].(string); ok && text != "" {
		return text
	}

	// Try "content" field
	if content, ok := dataMap["content"].(string); ok && content != "" {
		return content
	}

	return ""
}

// countTotalResponses counts total responses across all regions
func (re *ResponseExtractor) countTotalResponses(responses map[string][]ResponseData) int {
	total := 0
	for _, regionResponses := range responses {
		total += len(regionResponses)
	}
	return total
}

// GetResponsesByQuestion groups responses by question ID
func (re *ResponseExtractor) GetResponsesByQuestion(responses map[string][]ResponseData) map[string]map[string]ResponseData {
	byQuestion := make(map[string]map[string]ResponseData)

	for region, regionResponses := range responses {
		for _, resp := range regionResponses {
			if _, exists := byQuestion[resp.QuestionID]; !exists {
				byQuestion[resp.QuestionID] = make(map[string]ResponseData)
			}
			byQuestion[resp.QuestionID][region] = resp
		}
	}

	return byQuestion
}

// GetResponsesByModel groups responses by model ID
func (re *ResponseExtractor) GetResponsesByModel(responses map[string][]ResponseData) map[string]map[string][]ResponseData {
	byModel := make(map[string]map[string][]ResponseData)

	for region, regionResponses := range responses {
		for _, resp := range regionResponses {
			if _, exists := byModel[resp.ModelID]; !exists {
				byModel[resp.ModelID] = make(map[string][]ResponseData)
			}
			byModel[resp.ModelID][region] = append(byModel[resp.ModelID][region], resp)
		}
	}

	return byModel
}
