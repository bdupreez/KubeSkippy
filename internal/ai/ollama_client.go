package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// OllamaClient implements the AIClient interface for Ollama
type OllamaClient struct {
	endpoint   string
	model      string
	httpClient *http.Client
}

// OllamaRequest represents a request to the Ollama API
type OllamaRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	Temperature float32 `json:"temperature,omitempty"`
	Stream      bool    `json:"stream"`
}

// OllamaResponse represents a response from the Ollama API
type OllamaResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Response           string    `json:"response"`
	Done               bool      `json:"done"`
	Context            []int     `json:"context,omitempty"`
	TotalDuration      int64     `json:"total_duration,omitempty"`
	LoadDuration       int64     `json:"load_duration,omitempty"`
	PromptEvalCount    int       `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64     `json:"prompt_eval_duration,omitempty"`
	EvalCount          int       `json:"eval_count,omitempty"`
	EvalDuration       int64     `json:"eval_duration,omitempty"`
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient(endpoint, model string, timeout time.Duration) (*OllamaClient, error) {
	if endpoint == "" {
		endpoint = "http://localhost:11434" // Default Ollama endpoint
	}

	if model == "" {
		model = "llama2" // Default model
	}

	// Ensure endpoint doesn't have trailing slash
	endpoint = strings.TrimRight(endpoint, "/")

	client := &OllamaClient{
		endpoint: endpoint,
		model:    model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}

	// Check if Ollama is accessible
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !client.IsAvailable(ctx) {
		return nil, fmt.Errorf("Ollama service is not available at %s", endpoint)
	}

	// Check if model exists
	if err := client.checkModel(ctx); err != nil {
		return nil, fmt.Errorf("model %s not available: %w", model, err)
	}

	return client, nil
}

// Query sends a prompt to Ollama and returns the response
func (o *OllamaClient) Query(ctx context.Context, prompt string, temperature float32) (string, error) {
	log := log.FromContext(ctx)
	log.V(1).Info("Querying Ollama", "model", o.model, "prompt_length", len(prompt))

	// Prepare request
	request := OllamaRequest{
		Model:       o.model,
		Prompt:      prompt,
		Temperature: temperature,
		Stream:      false, // We'll use non-streaming for simplicity
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", o.endpoint+"/api/generate", bytes.NewReader(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if !ollamaResp.Done {
		return "", fmt.Errorf("incomplete response from Ollama")
	}

	log.V(1).Info("Ollama query completed",
		"response_length", len(ollamaResp.Response),
		"eval_count", ollamaResp.EvalCount,
		"duration_ms", ollamaResp.TotalDuration/1_000_000)

	return ollamaResp.Response, nil
}

// GetModel returns the model identifier
func (o *OllamaClient) GetModel() string {
	return fmt.Sprintf("ollama/%s", o.model)
}

// IsAvailable checks if the Ollama service is reachable
func (o *OllamaClient) IsAvailable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", o.endpoint+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// checkModel verifies that the specified model is available
func (o *OllamaClient) checkModel(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", o.endpoint+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to list models: status %d", resp.StatusCode)
	}

	var result struct {
		Models []struct {
			Name       string    `json:"name"`
			ModifiedAt time.Time `json:"modified_at"`
			Size       int64     `json:"size"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode models: %w", err)
	}

	// Check if our model exists
	for _, model := range result.Models {
		if model.Name == o.model || strings.HasPrefix(model.Name, o.model+":") {
			return nil
		}
	}

	// Try to pull the model
	log.FromContext(ctx).Info("Model not found locally, attempting to pull", "model", o.model)
	if err := o.pullModel(ctx); err != nil {
		return fmt.Errorf("model not found and pull failed: %w", err)
	}

	return nil
}

// pullModel attempts to pull a model from the Ollama registry
func (o *OllamaClient) pullModel(ctx context.Context) error {
	request := map[string]string{
		"name": o.model,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.endpoint+"/api/pull", bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Use a longer timeout for pulling models
	pullClient := &http.Client{
		Timeout: 30 * time.Minute, // Models can be large
	}

	resp, err := pullClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to pull model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to pull model: status %d: %s", resp.StatusCode, string(body))
	}

	// Read streaming response
	decoder := json.NewDecoder(resp.Body)
	for {
		var status map[string]interface{}
		if err := decoder.Decode(&status); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode pull status: %w", err)
		}

		// Check for completion
		if status["status"] == "success" {
			return nil
		}

		// Check for errors
		if errMsg, ok := status["error"].(string); ok {
			return fmt.Errorf("pull failed: %s", errMsg)
		}
	}

	return nil
}

// StreamQuery sends a prompt to Ollama and streams the response
func (o *OllamaClient) StreamQuery(ctx context.Context, prompt string, temperature float32, callback func(chunk string) error) error {
	log := log.FromContext(ctx)
	log.V(1).Info("Streaming query to Ollama", "model", o.model)

	// Prepare request
	request := OllamaRequest{
		Model:       o.model,
		Prompt:      prompt,
		Temperature: temperature,
		Stream:      true,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", o.endpoint+"/api/generate", bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read streaming response
	decoder := json.NewDecoder(resp.Body)
	for {
		var chunk OllamaResponse
		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode chunk: %w", err)
		}

		// Send chunk to callback
		if err := callback(chunk.Response); err != nil {
			return fmt.Errorf("callback error: %w", err)
		}

		// Check if done
		if chunk.Done {
			break
		}
	}

	return nil
}
