package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOllamaClient(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			// Model list endpoint
			response := map[string]interface{}{
				"models": []map[string]interface{}{
					{
						"name":        "llama2",
						"modified_at": time.Now(),
						"size":        1000000,
					},
				},
			}
			json.NewEncoder(w).Encode(response)

		case "/api/generate":
			// Generate endpoint
			var req OllamaRequest
			json.NewDecoder(r.Body).Decode(&req)
			
			response := OllamaResponse{
				Model:     req.Model,
				CreatedAt: time.Now(),
				Response:  "Test response from Ollama",
				Done:      true,
			}
			json.NewEncoder(w).Encode(response)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Test client creation
	t.Run("create client", func(t *testing.T) {
		client, err := NewOllamaClient(server.URL, "llama2", 30*time.Second)
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, "ollama/llama2", client.GetModel())
	})

	// Test availability check
	t.Run("check availability", func(t *testing.T) {
		client := &OllamaClient{
			endpoint:   server.URL,
			model:      "llama2",
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}
		
		available := client.IsAvailable(context.Background())
		assert.True(t, available)
	})

	// Test query
	t.Run("query", func(t *testing.T) {
		client := &OllamaClient{
			endpoint:   server.URL,
			model:      "llama2",
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}
		
		response, err := client.Query(context.Background(), "Test prompt", 0.7)
		require.NoError(t, err)
		assert.Equal(t, "Test response from Ollama", response)
	})
}

func TestOpenAIClient(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authorization header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch r.URL.Path {
		case "/v1/chat/completions":
			var req OpenAIRequest
			json.NewDecoder(r.Body).Decode(&req)
			
			// Check if this is a test request
			if req.MaxTokens == 1 {
				// Availability check
				response := OpenAIResponse{
					ID:      "test",
					Model:   req.Model,
					Created: time.Now().Unix(),
					Choices: []Choice{
						{
							Message: Message{
								Role:    "assistant",
								Content: "test",
							},
							FinishReason: "stop",
						},
					},
					Usage: Usage{
						TotalTokens: 1,
					},
				}
				json.NewEncoder(w).Encode(response)
				return
			}
			
			// Normal query
			response := OpenAIResponse{
				ID:      "chatcmpl-test",
				Model:   req.Model,
				Created: time.Now().Unix(),
				Choices: []Choice{
					{
						Message: Message{
							Role:    "assistant",
							Content: "Test response from OpenAI",
						},
						FinishReason: "stop",
					},
				},
				Usage: Usage{
					PromptTokens:     10,
					CompletionTokens: 5,
					TotalTokens:      15,
				},
			}
			json.NewEncoder(w).Encode(response)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Test client creation
	t.Run("create client", func(t *testing.T) {
		client := &OpenAIClient{
			apiKey:     "test-api-key",
			model:      "gpt-3.5-turbo",
			endpoint:   server.URL + "/v1/chat/completions",
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}
		
		assert.Equal(t, "openai/gpt-3.5-turbo", client.GetModel())
		assert.True(t, client.IsAvailable(context.Background()))
	})

	// Test query
	t.Run("query", func(t *testing.T) {
		client := &OpenAIClient{
			apiKey:     "test-api-key",
			model:      "gpt-3.5-turbo",
			endpoint:   server.URL + "/v1/chat/completions",
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}
		
		response, err := client.Query(context.Background(), "Test prompt", 0.7)
		require.NoError(t, err)
		assert.Equal(t, "Test response from OpenAI", response)
	})

	// Test error handling
	t.Run("unauthorized", func(t *testing.T) {
		client := &OpenAIClient{
			apiKey:     "invalid-key",
			model:      "gpt-3.5-turbo",
			endpoint:   server.URL + "/v1/chat/completions",
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}
		
		_, err := client.Query(context.Background(), "Test prompt", 0.7)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})

	// Test token estimation
	t.Run("estimate tokens", func(t *testing.T) {
		client := &OpenAIClient{}
		
		// Test with ~40 character string
		tokens := client.EstimateTokens("This is a test string with about 40 chars")
		assert.InDelta(t, 10, tokens, 2) // Should be around 10 tokens
	})
}

func TestStreamQuery(t *testing.T) {
	// Test Ollama streaming
	t.Run("ollama streaming", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/tags" {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"models": []map[string]interface{}{
						{"name": "llama2"},
					},
				})
				return
			}
			
			// Stream response
			chunks := []OllamaResponse{
				{Response: "Hello", Done: false},
				{Response: " world", Done: false},
				{Response: "!", Done: true},
			}
			
			for _, chunk := range chunks {
				json.NewEncoder(w).Encode(chunk)
			}
		}))
		defer server.Close()

		client := &OllamaClient{
			endpoint:   server.URL,
			model:      "llama2",
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}

		var result string
		err := client.StreamQuery(context.Background(), "test", 0.7, func(chunk string) error {
			result += chunk
			return nil
		})
		
		require.NoError(t, err)
		assert.Equal(t, "Hello world!", result)
	})
}