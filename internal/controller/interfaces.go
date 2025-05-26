package controller

import (
	"context"
	"time"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MetricsCollector defines the interface for collecting cluster metrics
type MetricsCollector interface {
	// CollectMetrics gathers metrics for the given policy
	CollectMetrics(ctx context.Context, policy *v1alpha1.HealingPolicy) (*ClusterMetrics, error)

	// EvaluateTrigger checks if a trigger condition is met
	EvaluateTrigger(ctx context.Context, trigger *v1alpha1.HealingTrigger, metrics *ClusterMetrics) (bool, string, error)

	// GetResourceMetrics gets metrics for a specific resource
	GetResourceMetrics(ctx context.Context, resource *v1alpha1.TargetResource) (*ResourceMetrics, error)
}

// SafetyController validates and enforces safety rules
type SafetyController interface {
	// ValidateAction checks if an action is safe to execute
	ValidateAction(ctx context.Context, action *v1alpha1.HealingAction) (*ValidationResult, error)

	// CheckRateLimit verifies action frequency limits
	CheckRateLimit(ctx context.Context, policy *v1alpha1.HealingPolicy) (bool, error)

	// IsProtectedResource checks if a resource is protected
	IsProtectedResource(resource runtime.Object) (bool, string)

	// RecordAction logs an executed action
	RecordAction(ctx context.Context, action *v1alpha1.HealingAction, result *ActionResult)
}

// RemediationEngine executes healing actions
type RemediationEngine interface {
	// ExecuteAction performs the healing action
	ExecuteAction(ctx context.Context, action *v1alpha1.HealingAction) (*ActionResult, error)

	// DryRun simulates the action without executing
	DryRun(ctx context.Context, action *v1alpha1.HealingAction) (*ActionResult, error)

	// Rollback reverses a previously executed action
	Rollback(ctx context.Context, action *v1alpha1.HealingAction) error

	// GetActionExecutor returns the executor for a specific action type
	GetActionExecutor(actionType string) (ActionExecutor, error)
}

// ActionExecutor defines the interface for specific action implementations
type ActionExecutor interface {
	// Execute performs the action
	Execute(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) (*ActionResult, error)

	// Validate checks if the action can be executed
	Validate(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) error

	// DryRun simulates the action
	DryRun(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) (*ActionResult, error)
}

// AIAnalyzer interfaces with the AI system for analysis
type AIAnalyzer interface {
	// AnalyzeClusterState sends cluster state to AI for analysis
	AnalyzeClusterState(ctx context.Context, metrics *ClusterMetrics, issues []Issue) (*AIAnalysis, error)

	// ValidateRecommendation checks if an AI recommendation is safe
	ValidateRecommendation(ctx context.Context, recommendation *AIRecommendation) error

	// GetModel returns the current AI model configuration
	GetModel() string
}

// ClusterMetrics represents collected cluster metrics
type ClusterMetrics struct {
	Timestamp time.Time
	Nodes     []NodeMetrics
	Pods      []PodMetrics
	Resources map[string]interface{}
	Events    []EventMetrics
	Custom    map[string]float64
}

// NodeMetrics represents metrics for a node
type NodeMetrics struct {
	Name            string
	CPUUsage        float64
	MemoryUsage     float64
	DiskUsage       float64
	PodCount        int32
	Conditions      []string
	Labels          map[string]string
	LastUpdateTime  time.Time
}

// PodMetrics represents metrics for a pod
type PodMetrics struct {
	Name              string
	Namespace         string
	CPUUsage          float64
	MemoryUsage       float64
	RestartCount      int32
	Status            string
	Conditions        []string
	Labels            map[string]string
	OwnerReferences   []string
	LastUpdateTime    time.Time
}

// ResourceMetrics represents metrics for a specific resource
type ResourceMetrics struct {
	APIVersion string
	Kind       string
	Name       string
	Namespace  string
	Metrics    map[string]interface{}
	Events     []EventMetrics
}

// EventMetrics represents Kubernetes events
type EventMetrics struct {
	Type      string
	Reason    string
	Message   string
	Count     int32
	FirstSeen time.Time
	LastSeen  time.Time
	Object    string
}

// Issue represents a detected problem
type Issue struct {
	ID          string
	Severity    string
	Type        string
	Resource    string
	Description string
	Metrics     map[string]interface{}
	DetectedAt  time.Time
}

// ValidationResult contains the result of safety validation
type ValidationResult struct {
	Valid       bool
	Reason      string
	Warnings    []string
	Suggestions []string
}

// ActionResult contains the result of executing an action
type ActionResult struct {
	Success   bool
	Message   string
	Error     error
	Changes   []v1alpha1.ResourceChange
	Metrics   map[string]string
	StartTime time.Time
	EndTime   time.Time
}

// AIAnalysis represents the AI's analysis of cluster state
type AIAnalysis struct {
	Timestamp       time.Time
	Summary         string
	Issues          []AIIssue
	Recommendations []AIRecommendation
	Confidence      float64
	ModelVersion    string
}

// AIIssue represents an issue identified by AI
type AIIssue struct {
	ID          string
	Severity    string
	Description string
	Impact      string
	RootCause   string
}

// AIRecommendation represents an AI-suggested action
type AIRecommendation struct {
	ID          string
	Priority    int
	Action      string
	Target      string
	Reason      string
	Risk        string
	Confidence  float64
}