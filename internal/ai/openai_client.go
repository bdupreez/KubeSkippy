package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// OpenAIClient implements the AIClient interface for OpenAI
type OpenAIClient struct {
	apiKey     string
	model      string
	endpoint   string
	httpClient *http.Client
}

// OpenAIRequest represents a request to the OpenAI API
type OpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float32   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse represents a response from the OpenAI API
type OpenAIResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a response choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAIError represents an error response
type OpenAIError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(apiKey, model string, timeout time.Duration) (*OpenAIClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	if model == "" {
		model = "gpt-3.5-turbo" // Default model
	}

	client := &OpenAIClient{
		apiKey:   apiKey,
		model:    model,
		endpoint: "https://api.openai.com/v1/chat/completions",
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}

	// Validate the API key with a simple request
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !client.IsAvailable(ctx) {
		return nil, fmt.Errorf("OpenAI API is not accessible with provided credentials")
	}

	return client, nil
}

// Query sends a prompt to OpenAI and returns the response
func (o *OpenAIClient) Query(ctx context.Context, prompt string, temperature float32) (string, error) {
	log := log.FromContext(ctx)
	log.V(1).Info("Querying OpenAI", "model", o.model, "prompt_length", len(prompt))

	// Prepare request
	request := OpenAIRequest{
		Model: o.model,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are a Kubernetes cluster healing expert assistant. Provide detailed, actionable recommendations for cluster issues.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: temperature,
		MaxTokens:   2000, // Reasonable limit for responses
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", o.endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	// Execute request
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var apiError OpenAIError
		if err := json.Unmarshal(body, &apiError); err == nil && apiError.Error.Message != "" {
			return "", fmt.Errorf("OpenAI API error: %s (type: %s, code: %s)",
				apiError.Error.Message, apiError.Error.Type, apiError.Error.Code)
		}
		return "", fmt.Errorf("OpenAI returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	response := openAIResp.Choices[0].Message.Content

	log.V(1).Info("OpenAI query completed",
		"response_length", len(response),
		"total_tokens", openAIResp.Usage.TotalTokens,
		"finish_reason", openAIResp.Choices[0].FinishReason)

	return response, nil
}

// GetModel returns the model identifier
func (o *OpenAIClient) GetModel() string {
	return fmt.Sprintf("openai/%s", o.model)
}

// IsAvailable checks if the OpenAI service is reachable
func (o *OpenAIClient) IsAvailable(ctx context.Context) bool {
	// Create a minimal request to check API availability
	request := OpenAIRequest{
		Model: o.model,
		Messages: []Message{
			{
				Role:    "user",
				Content: "test",
			},
		},
		MaxTokens: 1, // Minimal tokens to reduce cost
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return false
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// We expect either 200 (success) or 401 (invalid API key)
	// Both indicate the service is reachable
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized
}

// StreamQuery sends a prompt to OpenAI and streams the response
func (o *OpenAIClient) StreamQuery(ctx context.Context, prompt string, temperature float32, callback func(chunk string) error) error {
	log := log.FromContext(ctx)
	log.V(1).Info("Streaming query to OpenAI", "model", o.model)

	// Prepare request with streaming enabled
	request := map[string]interface{}{
		"model": o.model,
		"messages": []Message{
			{
				Role:    "system",
				Content: "You are a Kubernetes cluster healing expert assistant. Provide detailed, actionable recommendations for cluster issues.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		"temperature": temperature,
		"max_tokens":  2000,
		"stream":      true,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", o.endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	// Execute request
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OpenAI returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read server-sent events stream
	reader := resp.Body
	buffer := make([]byte, 4096)

	for {
		n, err := reader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read stream: %w", err)
		}

		// Process the chunk
		chunk := string(buffer[:n])

		// Parse server-sent events
		lines := bytes.Split([]byte(chunk), []byte("\n"))
		for _, line := range lines {
			if bytes.HasPrefix(line, []byte("data: ")) {
				data := bytes.TrimPrefix(line, []byte("data: "))
				if string(data) == "[DONE]" {
					return nil
				}

				var streamResp struct {
					Choices []struct {
						Delta struct {
							Content string `json:"content"`
						} `json:"delta"`
					} `json:"choices"`
				}

				if err := json.Unmarshal(data, &streamResp); err == nil {
					if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
						if err := callback(streamResp.Choices[0].Delta.Content); err != nil {
							return fmt.Errorf("callback error: %w", err)
						}
					}
				}
			}
		}
	}

	return nil
}

// EstimateTokens provides a rough estimate of token count
func (o *OpenAIClient) EstimateTokens(text string) int {
	// Rough estimation: ~4 characters per token for English text
	return len(text) / 4
}
