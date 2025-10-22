package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("ERROR: OPENAI_API_KEY not set")
		os.Exit(1)
	}

	fmt.Printf("API Key present: %s...%s\n", apiKey[:8], apiKey[len(apiKey)-8:])

	// Test with gpt-5-nano-2025-08-07
	requestBody := map[string]interface{}{
		"model": "gpt-5-nano-2025-08-07",
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are a helpful assistant.",
			},
			{
				"role":    "user",
				"content": "Say 'Hello, this is a test response from GPT-5-nano!' and nothing else.",
			},
		},
		"max_completion_tokens": 4000,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Printf("ERROR marshaling request: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nSending request to OpenAI API...")
	fmt.Printf("Model: gpt-5-nano-2025-08-07\n")
	fmt.Printf("Request size: %d bytes\n", len(jsonData))

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("ERROR creating request: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("ERROR making request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	fmt.Printf("Response status: %s (took %dms)\n", resp.Status, elapsed.Milliseconds())

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ERROR reading response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Response size: %d bytes\n\n", len(body))

	if resp.StatusCode != 200 {
		fmt.Printf("ERROR Response:\n%s\n", string(body))
		os.Exit(1)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("ERROR parsing JSON: %v\n", err)
		fmt.Printf("Raw response:\n%s\n", string(body))
		os.Exit(1)
	}

	// Pretty print the response
	prettyJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Printf("SUCCESS! Full response:\n%s\n", string(prettyJSON))

	// Extract the message content
	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					fmt.Printf("\n=== EXTRACTED MESSAGE ===\n%s\n", content)
					fmt.Printf("=== LENGTH: %d characters ===\n", len(content))
				}
			}
		}
	}
}
