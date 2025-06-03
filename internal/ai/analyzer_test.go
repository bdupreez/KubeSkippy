package ai

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeskippy/kubeskippy/internal/types"
	"github.com/kubeskippy/kubeskippy/pkg/config"
)

// MockAIClient is a mock implementation of AIClient for testing
type MockAIClient struct {
	QueryFunc     func(ctx context.Context, prompt string, temperature float32) (string, error)
	Model         string
	Available     bool
	QueryResponse string
}

func (m *MockAIClient) Query(ctx context.Context, prompt string, temperature float32) (string, error) {
	if m.QueryFunc != nil {
		return m.QueryFunc(ctx, prompt, temperature)
	}
	if m.QueryResponse != "" {
		return m.QueryResponse, nil
	}
	return defaultMockResponse, nil
}

func (m *MockAIClient) GetModel() string {
	if m.Model != "" {
		return m.Model
	}
	return "mock/test-model"
}

func (m *MockAIClient) IsAvailable(ctx context.Context) bool {
	return m.Available
}

const defaultMockResponse = `SUMMARY:
The cluster is experiencing high CPU usage on several nodes, with pods showing increased restart counts.

ISSUES:
- High CPU usage on nodes
  Severity: High
  Impact: Performance degradation and potential pod evictions
  Root Cause: Resource-intensive workloads without proper limits

- Pod restart loops
  Severity: Critical
  Impact: Service availability issues
  Root Cause: OOMKilled due to memory limits

RECOMMENDATIONS:
1. Scale up the deployment to distribute load
   Target: deployment/api-server
   Reason: Current replicas are insufficient for the load
   Risk: Minimal, may increase resource costs
   Confidence: 0.85

2. Increase memory limits for failing pods
   Target: deployment/worker
   Reason: Pods are being OOMKilled
   Risk: Low, ensure nodes have capacity
   Confidence: 0.90

END`

func TestAnalyzer_AnalyzeClusterState(t *testing.T) {
	config := config.AIConfig{
		Provider:          "mock",
		Model:             "test-model",
		Temperature:       0.7,
		MinConfidence:     0.7,
		ValidateResponses: false,
	}

	mockClient := &MockAIClient{
		Available: true,
	}

	analyzer := &Analyzer{
		config:   config,
		client:   mockClient,
		prompts:  &PromptTemplates{ClusterAnalysis: defaultClusterAnalysisPrompt},
		validate: true,
	}

	// Test data
	metrics := &types.ClusterMetrics{
		Timestamp: time.Now(),
		Nodes: []types.NodeMetrics{
			{
				Name:        "node1",
				CPUUsage:    85.5,
				MemoryUsage: 70.2,
			},
		},
		Pods: []types.PodMetrics{
			{
				Name:         "pod1",
				Namespace:    "default",
				RestartCount: 5,
			},
		},
	}

	issues := []types.Issue{
		{
			ID:          "issue-1",
			Severity:    "High",
			Type:        "ResourceExhaustion",
			Description: "Node CPU usage above 80%",
		},
	}

	// Test successful analysis
	t.Run("successful analysis", func(t *testing.T) {
		analysis, err := analyzer.AnalyzeClusterState(context.Background(), metrics, issues)
		require.NoError(t, err)
		assert.NotNil(t, analysis)
		assert.NotEmpty(t, analysis.Summary)
		assert.Len(t, analysis.Issues, 2)
		assert.Len(t, analysis.Recommendations, 2)
		assert.Equal(t, "mock/test-model", analysis.ModelVersion)
	})

	// Test unavailable AI service
	t.Run("unavailable service", func(t *testing.T) {
		mockClient.Available = false
		_, err := analyzer.AnalyzeClusterState(context.Background(), metrics, issues)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not available")
		mockClient.Available = true
	})

	// Test confidence filtering
	t.Run("confidence filtering", func(t *testing.T) {
		// Set response with low confidence recommendation
		mockClient.QueryResponse = `SUMMARY:
Test summary

ISSUES:
- Test issue
  Severity: Low

RECOMMENDATIONS:
1. Low confidence action
   Target: deployment/test
   Confidence: 0.5

2. High confidence action
   Target: deployment/test2
   Confidence: 0.8

END`

		analysis, err := analyzer.AnalyzeClusterState(context.Background(), metrics, issues)
		require.NoError(t, err)
		assert.Len(t, analysis.Recommendations, 1) // Only high confidence
		assert.Equal(t, 0.8, analysis.Recommendations[0].Confidence)
	})
}

func TestAnalyzer_ValidateRecommendation(t *testing.T) {
	config := config.AIConfig{
		MinConfidence:     0.7,
		ValidateResponses: false,
	}

	analyzer := &Analyzer{
		config: config,
		client: &MockAIClient{Available: true},
	}

	tests := []struct {
		name           string
		recommendation *types.AIRecommendation
		expectError    bool
		errorContains  string
	}{
		{
			name: "valid recommendation",
			recommendation: &types.AIRecommendation{
				Action:     "scale deployment",
				Target:     "deployment/api-server",
				Confidence: 0.8,
			},
			expectError: false,
		},
		{
			name: "missing action",
			recommendation: &types.AIRecommendation{
				Target:     "deployment/api-server",
				Confidence: 0.8,
			},
			expectError:   true,
			errorContains: "no action specified",
		},
		{
			name: "missing target",
			recommendation: &types.AIRecommendation{
				Action:     "scale deployment",
				Confidence: 0.8,
			},
			expectError:   true,
			errorContains: "no target specified",
		},
		{
			name: "unsafe action",
			recommendation: &types.AIRecommendation{
				Action:     "delete-namespace kube-system",
				Target:     "namespace/kube-system",
				Confidence: 0.9,
			},
			expectError:   true,
			errorContains: "unsafe action detected",
		},
		{
			name: "low confidence",
			recommendation: &types.AIRecommendation{
				Action:     "scale deployment",
				Target:     "deployment/api-server",
				Confidence: 0.5,
			},
			expectError:   true,
			errorContains: "below threshold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := analyzer.ValidateRecommendation(context.Background(), tt.recommendation)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseAnalysisResponse(t *testing.T) {
	analyzer := &Analyzer{}

	tests := []struct {
		name     string
		response string
		validate func(t *testing.T, analysis *types.AIAnalysis)
	}{
		{
			name: "structured text response",
			response: `SUMMARY:
Test cluster analysis summary

ISSUES:
- High memory usage
  Severity: Critical
  Impact: Pod evictions likely
  Root Cause: Memory leak in application

- Network latency
  Severity: Medium
  Impact: Slow response times

RECOMMENDATIONS:
1. Restart affected pods
   Target: deployment/leaky-app
   Reason: Clear memory leak
   Risk: Temporary downtime
   Confidence: 0.85

2. Scale up network pods
   Target: daemonset/network-agent
   Confidence: 0.7

END`,
			validate: func(t *testing.T, analysis *types.AIAnalysis) {
				assert.Equal(t, "Test cluster analysis summary", analysis.Summary)
				assert.Len(t, analysis.Issues, 2)
				assert.Equal(t, "Critical", analysis.Issues[0].Severity)
				assert.Equal(t, "Memory leak in application", analysis.Issues[0].RootCause)
				assert.Len(t, analysis.Recommendations, 2)
				assert.Equal(t, 0.85, analysis.Recommendations[0].Confidence)
			},
		},
		{
			name: "JSON response",
			response: `{
				"summary": "JSON test summary",
				"confidence": 0.9,
				"issues": [
					{
						"id": "issue-1",
						"severity": "High",
						"description": "Test issue"
					}
				],
				"recommendations": [
					{
						"id": "rec-1",
						"action": "Test action",
						"confidence": 0.8
					}
				]
			}`,
			validate: func(t *testing.T, analysis *types.AIAnalysis) {
				assert.Equal(t, "JSON test summary", analysis.Summary)
				assert.Equal(t, 0.9, analysis.Confidence)
				assert.Len(t, analysis.Issues, 1)
				assert.Len(t, analysis.Recommendations, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis, err := analyzer.parseAnalysisResponse(tt.response)
			require.NoError(t, err)
			tt.validate(t, analysis)
		})
	}
}

func TestExtractConfidence(t *testing.T) {
	tests := []struct {
		text     string
		expected float64
	}{
		{"Confidence: 0.85", 0.85},
		{"confidence level: 0.7", 0.7},
		{"I am 95% confident", 0.95},
		{"80% confident in this", 0.80},
		{"no confidence mentioned", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			result := extractConfidence(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildClusterAnalysisPrompt(t *testing.T) {
	analyzer := &Analyzer{
		prompts: &PromptTemplates{
			ClusterAnalysis: "Metrics: %s\nIssues: %s\nTime: %s",
		},
	}

	metrics := &types.ClusterMetrics{
		Nodes: []types.NodeMetrics{
			{Name: "node1", CPUUsage: 50},
		},
	}

	issues := []types.Issue{
		{ID: "1", Description: "Test issue"},
	}

	prompt, err := analyzer.buildClusterAnalysisPrompt(metrics, issues)
	require.NoError(t, err)
	assert.Contains(t, prompt, "node1")
	assert.Contains(t, prompt, "Test issue")
}
