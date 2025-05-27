package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kubeskippy/kubeskippy/internal/controller"
	"github.com/kubeskippy/kubeskippy/pkg/config"
)

// Analyzer implements the AIAnalyzer interface
type Analyzer struct {
	config   config.AIConfig
	client   AIClient
	prompts  *PromptTemplates
	validate bool
}

// AIClient defines the interface for AI backend implementations
type AIClient interface {
	// Query sends a prompt to the AI and returns the response
	Query(ctx context.Context, prompt string, temperature float32) (string, error)

	// GetModel returns the model identifier
	GetModel() string

	// IsAvailable checks if the AI service is reachable
	IsAvailable(ctx context.Context) bool
}

// PromptTemplates contains templates for different AI queries
type PromptTemplates struct {
	ClusterAnalysis   string
	IssueAnalysis     string
	ActionValidation  string
	RootCauseAnalysis string
}

// NewAnalyzer creates a new AI analyzer
func NewAnalyzer(config config.AIConfig) (*Analyzer, error) {
	var client AIClient
	var err error

	// Create appropriate client based on provider
	switch config.Provider {
	case "ollama":
		client, err = NewOllamaClient(config.Endpoint, config.Model, config.Timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to create Ollama client: %w", err)
		}

	case "openai":
		if config.APIKey == "" {
			return nil, fmt.Errorf("OpenAI API key is required")
		}
		client, err = NewOpenAIClient(config.APIKey, config.Model, config.Timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", config.Provider)
	}

	// Initialize prompt templates
	prompts := &PromptTemplates{
		ClusterAnalysis:   defaultClusterAnalysisPrompt,
		IssueAnalysis:     defaultIssueAnalysisPrompt,
		ActionValidation:  defaultActionValidationPrompt,
		RootCauseAnalysis: defaultRootCausePrompt,
	}

	return &Analyzer{
		config:   config,
		client:   client,
		prompts:  prompts,
		validate: true,
	}, nil
}

// AnalyzeClusterState analyzes the cluster state and provides recommendations
func (a *Analyzer) AnalyzeClusterState(ctx context.Context, metrics *controller.ClusterMetrics, issues []controller.Issue) (*controller.AIAnalysis, error) {
	log := log.FromContext(ctx)
	log.Info("Analyzing cluster state with AI", "provider", a.config.Provider, "model", a.client.GetModel())

	// Check if AI is available
	if !a.client.IsAvailable(ctx) {
		return nil, fmt.Errorf("AI service is not available")
	}

	// Prepare the analysis prompt
	prompt, err := a.buildClusterAnalysisPrompt(metrics, issues)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	// Query the AI
	response, err := a.client.Query(ctx, prompt, a.config.Temperature)
	if err != nil {
		return nil, fmt.Errorf("AI query failed: %w", err)
	}

	// Parse the AI response
	analysis, err := a.parseAnalysisResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// Add metadata
	analysis.Timestamp = time.Now()
	analysis.ModelVersion = a.client.GetModel()

	// Validate recommendations if enabled
	if a.validate {
		analysis = a.validateAnalysis(ctx, analysis, metrics)
	}

	log.Info("AI analysis completed",
		"issues", len(analysis.Issues),
		"recommendations", len(analysis.Recommendations),
		"confidence", analysis.Confidence)

	return analysis, nil
}

// ValidateRecommendation validates an AI recommendation for safety
func (a *Analyzer) ValidateRecommendation(ctx context.Context, recommendation *controller.AIRecommendation) error {
	log := log.FromContext(ctx)

	// Basic validation
	if recommendation.Action == "" {
		return fmt.Errorf("recommendation has no action specified")
	}

	if recommendation.Target == "" {
		return fmt.Errorf("recommendation has no target specified")
	}

	// Check against known unsafe actions
	unsafeActions := []string{"delete-namespace", "delete-node", "delete-pv", "delete-crd"}
	for _, unsafe := range unsafeActions {
		if strings.Contains(strings.ToLower(recommendation.Action), unsafe) {
			return fmt.Errorf("unsafe action detected: %s", unsafe)
		}
	}

	// Validate confidence threshold
	if recommendation.Confidence < float64(a.config.MinConfidence) {
		return fmt.Errorf("recommendation confidence %.2f is below threshold %.2f",
			recommendation.Confidence, a.config.MinConfidence)
	}

	// Additional validation using AI if configured
	if a.config.ValidateResponses {
		prompt := a.buildValidationPrompt(recommendation)
		response, err := a.client.Query(ctx, prompt, 0.1) // Low temperature for validation
		if err != nil {
			log.Error(err, "Failed to validate recommendation with AI")
			return fmt.Errorf("validation query failed: %w", err)
		}

		if !strings.Contains(strings.ToLower(response), "safe") {
			return fmt.Errorf("AI validation failed: %s", response)
		}
	}

	return nil
}

// GetModel returns the current AI model configuration
func (a *Analyzer) GetModel() string {
	return a.client.GetModel()
}

// buildClusterAnalysisPrompt creates the prompt for cluster analysis
func (a *Analyzer) buildClusterAnalysisPrompt(metrics *controller.ClusterMetrics, issues []controller.Issue) (string, error) {
	// Convert metrics to JSON for structured input
	metricsJSON, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return "", err
	}

	issuesJSON, err := json.MarshalIndent(issues, "", "  ")
	if err != nil {
		return "", err
	}

	prompt := fmt.Sprintf(a.prompts.ClusterAnalysis,
		string(metricsJSON),
		string(issuesJSON),
		time.Now().Format(time.RFC3339))

	return prompt, nil
}

// parseAnalysisResponse parses the AI response into structured analysis
func (a *Analyzer) parseAnalysisResponse(response string) (*controller.AIAnalysis, error) {
	// First, try to parse as JSON (if AI returns structured response)
	var analysis controller.AIAnalysis
	if err := json.Unmarshal([]byte(response), &analysis); err == nil {
		return &analysis, nil
	}

	// Otherwise, parse the text response
	analysis = controller.AIAnalysis{
		Summary:    extractSection(response, "SUMMARY", "ISSUES"),
		Confidence: extractConfidence(response),
	}

	// Extract issues
	issuesText := extractSection(response, "ISSUES", "RECOMMENDATIONS")
	analysis.Issues = parseIssues(issuesText)

	// Extract recommendations
	recsText := extractSection(response, "RECOMMENDATIONS", "END")
	analysis.Recommendations = parseRecommendations(recsText)

	// Default confidence if not found
	if analysis.Confidence == 0 {
		analysis.Confidence = 0.7
	}

	return &analysis, nil
}

// validateAnalysis validates and filters AI analysis results
func (a *Analyzer) validateAnalysis(ctx context.Context, analysis *controller.AIAnalysis, metrics *controller.ClusterMetrics) *controller.AIAnalysis {
	log := log.FromContext(ctx)

	// Filter recommendations below confidence threshold
	validRecs := []controller.AIRecommendation{}
	for _, rec := range analysis.Recommendations {
		if rec.Confidence >= float64(a.config.MinConfidence) {
			// Additional safety checks
			if err := a.ValidateRecommendation(ctx, &rec); err != nil {
				log.Info("Filtered out recommendation", "action", rec.Action, "reason", err.Error())
				continue
			}
			validRecs = append(validRecs, rec)
		}
	}
	analysis.Recommendations = validRecs

	// Adjust overall confidence based on filtering
	if len(validRecs) < len(analysis.Recommendations)/2 {
		analysis.Confidence *= 0.8
	}

	return analysis
}

// buildValidationPrompt creates a prompt to validate a recommendation
func (a *Analyzer) buildValidationPrompt(recommendation *controller.AIRecommendation) string {
	return fmt.Sprintf(a.prompts.ActionValidation,
		recommendation.Action,
		recommendation.Target,
		recommendation.Reason,
		recommendation.Risk)
}

// Helper functions for parsing

func extractSection(text, startMarker, endMarker string) string {
	start := strings.Index(text, startMarker)
	if start == -1 {
		return ""
	}
	start += len(startMarker)

	// Skip colon and whitespace after marker
	if start < len(text) && text[start] == ':' {
		start++
	}

	end := strings.Index(text[start:], endMarker)
	if end == -1 {
		return strings.TrimSpace(text[start:])
	}

	return strings.TrimSpace(text[start : start+end])
}

func extractConfidence(text string) float64 {
	// Look for confidence patterns like "Confidence: 0.85" or "85% confident"
	patterns := []string{
		"confidence: ",
		"confidence level: ",
		"% confident",
	}

	text = strings.ToLower(text)
	for _, pattern := range patterns {
		if idx := strings.Index(text, pattern); idx != -1 {
			// Extract number after pattern
			var conf float64
			if strings.Contains(pattern, "%") {
				// Look for number before %
				start := idx - 3
				if start < 0 {
					start = 0
				}
				fmt.Sscanf(text[start:idx], "%f", &conf)
				return conf / 100.0
			} else {
				fmt.Sscanf(text[idx+len(pattern):], "%f", &conf)
				return conf
			}
		}
	}

	return 0.0
}

func parseIssues(text string) []controller.AIIssue {
	issues := []controller.AIIssue{}
	lines := strings.Split(text, "\n")

	var currentIssue *controller.AIIssue
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for issue markers
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || strings.HasPrefix(line, "• ") {
			if currentIssue != nil {
				issues = append(issues, *currentIssue)
			}
			currentIssue = &controller.AIIssue{
				ID:          fmt.Sprintf("ai-issue-%d", len(issues)+1),
				Description: strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* "), "• "),
			}
		} else if currentIssue != nil {
			// Continue description or parse attributes
			if strings.Contains(line, "Severity:") {
				currentIssue.Severity = strings.TrimSpace(strings.Split(line, ":")[1])
			} else if strings.Contains(line, "Impact:") {
				currentIssue.Impact = strings.TrimSpace(strings.Split(line, ":")[1])
			} else if strings.Contains(line, "Root Cause:") {
				currentIssue.RootCause = strings.TrimSpace(strings.Split(line, ":")[1])
			}
		}
	}

	if currentIssue != nil {
		issues = append(issues, *currentIssue)
	}

	return issues
}

func parseRecommendations(text string) []controller.AIRecommendation {
	recommendations := []controller.AIRecommendation{}
	lines := strings.Split(text, "\n")

	var currentRec *controller.AIRecommendation
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for recommendation markers
		if strings.HasPrefix(line, "1.") || strings.HasPrefix(line, "2.") || strings.HasPrefix(line, "3.") ||
			strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			if currentRec != nil {
				recommendations = append(recommendations, *currentRec)
			}

			// Extract action from line
			action := line
			for _, prefix := range []string{"1.", "2.", "3.", "4.", "5.", "- ", "* "} {
				action = strings.TrimPrefix(action, prefix)
			}

			currentRec = &controller.AIRecommendation{
				ID:         fmt.Sprintf("ai-rec-%d", len(recommendations)+1),
				Priority:   len(recommendations) + 1,
				Action:     strings.TrimSpace(action),
				Confidence: 0.8, // Default confidence
			}
		} else if currentRec != nil {
			// Parse attributes
			if strings.Contains(line, "Target:") {
				currentRec.Target = strings.TrimSpace(strings.Split(line, ":")[1])
			} else if strings.Contains(line, "Reason:") {
				currentRec.Reason = strings.TrimSpace(strings.Split(line, ":")[1])
			} else if strings.Contains(line, "Risk:") {
				currentRec.Risk = strings.TrimSpace(strings.Split(line, ":")[1])
			} else if strings.Contains(line, "Confidence:") {
				fmt.Sscanf(strings.Split(line, ":")[1], "%f", &currentRec.Confidence)
			}
		}
	}

	if currentRec != nil {
		recommendations = append(recommendations, *currentRec)
	}

	return recommendations
}

// Default prompt templates

const defaultClusterAnalysisPrompt = `You are a Kubernetes cluster healing expert. Analyze the following cluster state and provide recommendations.

CLUSTER METRICS:
%s

DETECTED ISSUES:
%s

Current Time: %s

Please provide your analysis in the following format:

SUMMARY:
[Provide a brief summary of the cluster health and main concerns]

ISSUES:
[List each identified issue with severity, impact, and root cause]
- Issue description
  Severity: [Critical/High/Medium/Low]
  Impact: [Description of impact]
  Root Cause: [Likely root cause]

RECOMMENDATIONS:
[List actionable recommendations in priority order]
1. [Specific action to take]
   Target: [Resource type/name to act on]
   Reason: [Why this action will help]
   Risk: [Any risks associated]
   Confidence: [0.0-1.0]

END

Focus on practical, safe actions that can be automated. Avoid destructive operations unless absolutely necessary.`

const defaultIssueAnalysisPrompt = `Analyze the following Kubernetes issue and provide root cause analysis:

ISSUE: %s
RESOURCE: %s
METRICS: %s

Provide a detailed root cause analysis including:
1. Most likely root cause
2. Contributing factors
3. Recommended remediation steps
4. Preventive measures`

const defaultActionValidationPrompt = `Validate the safety of the following Kubernetes healing action:

ACTION: %s
TARGET: %s
REASON: %s
RISK: %s

Is this action safe to execute automatically? Consider:
1. Potential for data loss
2. Impact on application availability
3. Cluster stability
4. Security implications

Respond with either "SAFE" or "UNSAFE" followed by explanation.`

const defaultRootCausePrompt = `Perform root cause analysis for the following Kubernetes issue:

SYMPTOMS: %s
TIMELINE: %s
AFFECTED RESOURCES: %s

Identify:
1. Root cause
2. Contributing factors
3. Chain of events
4. Remediation steps`
