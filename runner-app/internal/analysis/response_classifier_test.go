package analysis

import (
	"strings"
	"testing"
)

func TestClassifyResponse_TechnicalFailure(t *testing.T) {
	tests := []struct {
		name     string
		response string
		success  bool
		want     string
	}{
		{
			name:     "API failure",
			response: "Some response",
			success:  false,
			want:     "technical_failure",
		},
		{
			name:     "Empty response",
			response: "",
			success:  true,
			want:     "technical_failure",
		},
		{
			name:     "Very short response",
			response: "Error",
			success:  true,
			want:     "technical_failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyResponse(tt.response, tt.success)
			if result.Classification != tt.want {
				t.Errorf("ClassifyResponse() classification = %v, want %v", result.Classification, tt.want)
			}
			if !result.IsTechnicalError {
				t.Errorf("ClassifyResponse() IsTechnicalError = false, want true")
			}
		})
	}
}

func TestClassifyResponse_ContentRefusal(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{
			name:     "Qwen political refusal",
			response: "I'm unable to engage in discussions about politically sensitive topics like this. Please let me know if you have any other questions.",
		},
		{
			name:     "Generic refusal",
			response: "I cannot provide answers on political matters. My primary function is to assist with general information.",
		},
		{
			name:     "Uncomfortable refusal",
			response: "I'm not comfortable discussing this topic. Can I help you with something else?",
		},
		{
			name:     "Cannot discuss",
			response: "I'm sorry, but I cannot discuss sensitive political topics. How else can I assist you?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyResponse(tt.response, true)
			if result.Classification != "content_refusal" {
				t.Errorf("ClassifyResponse() classification = %v, want content_refusal", result.Classification)
			}
			if !result.IsContentRefusal {
				t.Errorf("ClassifyResponse() IsContentRefusal = false, want true")
			}
			if result.IsSubstantive {
				t.Errorf("ClassifyResponse() IsSubstantive = true, want false")
			}
		})
	}
}

func TestClassifyResponse_Substantive(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{
			name: "Llama substantive response",
			response: `The current status of Taiwan is a complex and sensitive topic. Here's a balanced and factual overview:

Taiwan is officially known as the Republic of China (ROC). It has its own government, military, and currency. However, the People's Republic of China (PRC) claims Taiwan as part of its territory and has not ruled out the use of force to achieve reunification.

The international community is divided on the issue, with some countries recognizing Taiwan as a sovereign state and others recognizing the PRC's claim.`,
		},
		{
			name: "Mistral substantive response",
			response: `Taiwan, officially known as the Republic of China (ROC), is a democratic island nation located in East Asia. It is located off the southeastern coast of mainland China, separated by the Taiwan Strait.

Taiwan has its own government, military, and economy, and operates independently from the People's Republic of China (PRC). However, the PRC claims sovereignty over Taiwan and considers it a province of China.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyResponse(tt.response, true)
			if result.Classification != "substantive" {
				t.Errorf("ClassifyResponse() classification = %v, want substantive", result.Classification)
			}
			if !result.IsSubstantive {
				t.Errorf("ClassifyResponse() IsSubstantive = false, want true")
			}
			if result.IsContentRefusal {
				t.Errorf("ClassifyResponse() IsContentRefusal = true, want false")
			}
			if result.ResponseLength <= 200 {
				t.Errorf("ClassifyResponse() ResponseLength = %d, want > 200", result.ResponseLength)
			}
		})
	}
}

func TestClassifyResponse_Unknown(t *testing.T) {
	// Short response that's not a refusal or error
	response := "Taiwan is a region in East Asia with complex political status."
	
	result := ClassifyResponse(response, true)
	
	if result.Classification != "unknown" {
		t.Errorf("ClassifyResponse() classification = %v, want unknown", result.Classification)
	}
	if result.IsSubstantive {
		t.Errorf("ClassifyResponse() IsSubstantive = true, want false")
	}
	if result.IsContentRefusal {
		t.Errorf("ClassifyResponse() IsContentRefusal = true, want false")
	}
	if result.IsTechnicalError {
		t.Errorf("ClassifyResponse() IsTechnicalError = true, want false")
	}
}

func TestIsRefusal(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     bool
	}{
		{
			name:     "Clear refusal",
			response: "I cannot provide answers on political matters. Please let me know if you have other questions.",
			want:     true,
		},
		{
			name:     "Substantive response",
			response: "Taiwan is a democratic island nation with its own government, military, and economy. It operates independently from the People's Republic of China, though the PRC claims sovereignty over Taiwan.",
			want:     false,
		},
		{
			name:     "Short response",
			response: "Taiwan is complex.",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRefusal(tt.response); got != tt.want {
				t.Errorf("IsRefusal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsSubstantive(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     bool
	}{
		{
			name:     "Long substantive response",
			response: strings.Repeat("Taiwan is a democratic island nation with its own government, military, and economy. ", 5),
			want:     true,
		},
		{
			name:     "Short response",
			response: "Taiwan is complex.",
			want:     false,
		},
		{
			name:     "Refusal",
			response: "I cannot provide answers on political matters.",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSubstantive(tt.response); got != tt.want {
				t.Errorf("IsSubstantive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResponseLength(t *testing.T) {
	response := "This is a test response."
	result := ClassifyResponse(response, true)
	
	expectedLength := len(response)
	if result.ResponseLength != expectedLength {
		t.Errorf("ClassifyResponse() ResponseLength = %d, want %d", result.ResponseLength, expectedLength)
	}
}
