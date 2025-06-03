package controller

import (
	"context"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
	"github.com/kubeskippy/kubeskippy/internal/types"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MetricsCollector defines the interface for collecting cluster metrics
type MetricsCollector interface {
	// CollectMetrics gathers metrics for the given policy
	CollectMetrics(ctx context.Context, policy *v1alpha1.HealingPolicy) (*types.ClusterMetrics, error)

	// EvaluateTrigger checks if a trigger condition is met
	EvaluateTrigger(ctx context.Context, trigger *v1alpha1.HealingTrigger, metrics *types.ClusterMetrics) (bool, string, error)

	// GetResourceMetrics gets metrics for a specific resource
	GetResourceMetrics(ctx context.Context, resource *v1alpha1.TargetResource) (*types.ResourceMetrics, error)
}

// SafetyController validates and enforces safety rules
type SafetyController interface {
	// ValidateAction checks if an action is safe to execute
	ValidateAction(ctx context.Context, action *v1alpha1.HealingAction) (*types.ValidationResult, error)

	// CheckRateLimit verifies action frequency limits
	CheckRateLimit(ctx context.Context, policy *v1alpha1.HealingPolicy) (bool, error)

	// IsProtectedResource checks if a resource is protected
	IsProtectedResource(resource runtime.Object) (bool, string)

	// RecordAction logs an executed action
	RecordAction(ctx context.Context, action *v1alpha1.HealingAction, result *types.ActionResult)
}

// RemediationEngine executes healing actions
type RemediationEngine interface {
	// ExecuteAction performs the healing action
	ExecuteAction(ctx context.Context, action *v1alpha1.HealingAction) (*types.ActionResult, error)

	// DryRun simulates the action without executing
	DryRun(ctx context.Context, action *v1alpha1.HealingAction) (*types.ActionResult, error)

	// Rollback reverses a previously executed action
	Rollback(ctx context.Context, action *v1alpha1.HealingAction) error

	// GetActionExecutor returns the executor for a specific action type
	GetActionExecutor(actionType string) (types.ActionExecutor, error)
}

// ActionExecutor defines the interface for specific action implementations
type ActionExecutor interface {
	// Execute performs the action
	Execute(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) (*types.ActionResult, error)

	// Validate checks if the action can be executed
	Validate(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) error

	// DryRun simulates the action
	DryRun(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) (*types.ActionResult, error)
}

// AIAnalyzer interfaces with the AI system for analysis
type AIAnalyzer interface {
	// AnalyzeClusterState sends cluster state to AI for analysis
	AnalyzeClusterState(ctx context.Context, metrics *types.ClusterMetrics, issues []types.Issue) (*types.AIAnalysis, error)

	// ValidateRecommendation checks if an AI recommendation is safe
	ValidateRecommendation(ctx context.Context, recommendation *types.AIRecommendation) error

	// GetModel returns the current AI model configuration
	GetModel() string
}

