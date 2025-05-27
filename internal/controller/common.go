package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Constants for operator behavior
const (
	// Annotation keys
	AnnotationLastApplied     = "kubeskippy.io/last-applied"
	AnnotationProtected       = "kubeskippy.io/protected"
	AnnotationHealingDisabled = "kubeskippy.io/healing-disabled"

	// Label keys
	LabelManagedBy   = "kubeskippy.io/managed-by"
	LabelPolicyName  = "kubeskippy.io/policy-name"
	LabelActionName  = "kubeskippy.io/action-name"
	LabelActionType  = "kubeskippy.io/action-type"
	LabelActionPhase = "kubeskippy.io/action-phase"

	// Finalizer
	FinalizerName = "kubeskippy.io/finalizer"

	// Condition reasons
	ReasonPolicyCreated   = "PolicyCreated"
	ReasonPolicyUpdated   = "PolicyUpdated"
	ReasonPolicyDeleted   = "PolicyDeleted"
	ReasonActionCreated   = "ActionCreated"
	ReasonActionExecuted  = "ActionExecuted"
	ReasonActionFailed    = "ActionFailed"
	ReasonActionSucceeded = "ActionSucceeded"
	ReasonValidationError = "ValidationError"
	ReasonRateLimited     = "RateLimited"
)

// PolicyMatcher matches resources against a policy selector
type PolicyMatcher struct {
	policy *v1alpha1.HealingPolicy
}

// NewPolicyMatcher creates a new PolicyMatcher
func NewPolicyMatcher(policy *v1alpha1.HealingPolicy) *PolicyMatcher {
	return &PolicyMatcher{policy: policy}
}

// Matches checks if a resource matches the policy selector
func (pm *PolicyMatcher) Matches(obj client.Object) (bool, error) {
	// Check namespace
	if len(pm.policy.Spec.Selector.Namespaces) > 0 {
		found := false
		for _, ns := range pm.policy.Spec.Selector.Namespaces {
			if obj.GetNamespace() == ns {
				found = true
				break
			}
		}
		if !found {
			return false, nil
		}
	}

	// Check labels
	if pm.policy.Spec.Selector.LabelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(pm.policy.Spec.Selector.LabelSelector)
		if err != nil {
			return false, fmt.Errorf("invalid label selector: %w", err)
		}
		if !selector.Matches(labels.Set(obj.GetLabels())) {
			return false, nil
		}
	}

	// Check resource type
	gvk := obj.GetObjectKind().GroupVersionKind()
	apiVersion, kind := gvk.ToAPIVersionAndKind()

	found := false
	for _, rf := range pm.policy.Spec.Selector.Resources {
		if rf.APIVersion == apiVersion && rf.Kind == kind {
			// Check exclude names
			for _, exclude := range rf.ExcludeNames {
				if obj.GetName() == exclude {
					return false, nil
				}
			}
			found = true
			break
		}
	}

	return found, nil
}

// ResourceKey generates a unique key for a resource
func ResourceKey(obj client.Object) string {
	gvk := obj.GetObjectKind().GroupVersionKind()
	return fmt.Sprintf("%s:%s:%s|%s|%s",
		gvk.Group,
		gvk.Version,
		gvk.Kind,
		obj.GetNamespace(),
		obj.GetName())
}

// ParseResourceKey parses a resource key
func ParseResourceKey(key string) (gvk string, namespace string, name string, err error) {
	parts := strings.Split(key, "|")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid resource key: %s", key)
	}
	return parts[0], parts[1], parts[2], nil
}

// IsProtectedResource checks if a resource is protected
func IsProtectedResource(obj client.Object, protectedNamespaces []string, protectedLabels map[string]string) bool {
	// Check protected namespaces
	for _, ns := range protectedNamespaces {
		if obj.GetNamespace() == ns {
			return true
		}
	}

	// Check protected labels
	labels := obj.GetLabels()
	for k, v := range protectedLabels {
		if labels[k] == v {
			return true
		}
	}

	// Check protected annotation
	annotations := obj.GetAnnotations()
	if annotations[AnnotationProtected] == "true" {
		return true
	}

	return false
}

// CalculateBackoff calculates exponential backoff duration
func CalculateBackoff(attempt int32, baseDelay time.Duration, multiplier float64) time.Duration {
	if attempt <= 0 {
		return baseDelay
	}

	delay := float64(baseDelay)
	for i := int32(1); i < attempt; i++ {
		delay *= multiplier
	}

	maxDelay := 30 * time.Minute
	if time.Duration(delay) > maxDelay {
		return maxDelay
	}

	return time.Duration(delay)
}

// CreateHealingAction creates a HealingAction from a policy and trigger
func CreateHealingAction(
	policy *v1alpha1.HealingPolicy,
	target client.Object,
	actionTemplate *v1alpha1.HealingActionTemplate,
	dryRun bool,
) *v1alpha1.HealingAction {
	now := metav1.Now()
	gvk := target.GetObjectKind().GroupVersionKind()
	apiVersion, kind := gvk.ToAPIVersionAndKind()

	action := &v1alpha1.HealingAction{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-%s-", policy.Name, actionTemplate.Name),
			Namespace:    policy.Namespace,
			Labels: map[string]string{
				LabelManagedBy:   "kubeskippy",
				LabelPolicyName:  policy.Name,
				LabelActionName:  actionTemplate.Name,
				LabelActionType:  actionTemplate.Type,
				LabelActionPhase: v1alpha1.HealingActionPhasePending,
			},
			Annotations: map[string]string{
				AnnotationLastApplied: now.Format(time.RFC3339),
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: policy.APIVersion,
					Kind:       policy.Kind,
					Name:       policy.Name,
					UID:        policy.UID,
					Controller: ptr(true),
				},
			},
		},
		Spec: v1alpha1.HealingActionSpec{
			PolicyRef: v1alpha1.PolicyReference{
				Name:      policy.Name,
				Namespace: policy.Namespace,
				UID:       string(policy.UID),
			},
			TargetResource: v1alpha1.TargetResource{
				APIVersion: apiVersion,
				Kind:       kind,
				Name:       target.GetName(),
				Namespace:  target.GetNamespace(),
				UID:        string(target.GetUID()),
			},
			Action:           *actionTemplate,
			ApprovalRequired: actionTemplate.RequiresApproval || policy.Spec.Mode == "manual",
			DryRun:           dryRun || policy.Spec.Mode == "dryrun",
			Timeout:          metav1.Duration{Duration: 10 * time.Minute},
			RetryPolicy: &v1alpha1.RetryPolicy{
				MaxAttempts:       3,
				BackoffDelay:      metav1.Duration{Duration: 30 * time.Second},
				BackoffMultiplier: 2.0,
			},
		},
		Status: v1alpha1.HealingActionStatus{
			Phase:              v1alpha1.HealingActionPhasePending,
			ObservedGeneration: 0,
		},
	}

	// Initialize approval status if required
	if action.Spec.ApprovalRequired {
		action.Status.Approval = &v1alpha1.ApprovalStatus{
			Required: true,
			Approved: false,
		}
	}

	return action
}

// SetCondition sets a condition on an object
func SetCondition(conditions *[]metav1.Condition, conditionType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

// GetCondition gets a condition from an object
func GetCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	return meta.FindStatusCondition(conditions, conditionType)
}

// LoggerWithValues adds common key-value pairs to a logger
func LoggerWithValues(log logr.Logger, obj client.Object) logr.Logger {
	return log.WithValues(
		"name", obj.GetName(),
		"namespace", obj.GetNamespace(),
		"kind", obj.GetObjectKind().GroupVersionKind().Kind,
		"uid", obj.GetUID(),
	)
}

// NamespacedName creates a types.NamespacedName from an object
func NamespacedName(obj client.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}

// ptr returns a pointer to the given value
func ptr[T any](v T) *T {
	return &v
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState string

const (
	CircuitBreakerClosed   CircuitBreakerState = "closed"
	CircuitBreakerOpen     CircuitBreakerState = "open"
	CircuitBreakerHalfOpen CircuitBreakerState = "half-open"
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	state            CircuitBreakerState
	failures         int
	successes        int
	lastFailureTime  time.Time
	lastAttemptTime  time.Time
	failureThreshold int
	successThreshold int
	timeout          time.Duration
	halfOpenActions  int
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold, successThreshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            CircuitBreakerClosed,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
	}
}

// Call executes a function through the circuit breaker
func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
	if err := cb.canExecute(); err != nil {
		return err
	}

	cb.lastAttemptTime = time.Now()
	err := fn()

	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}

	return err
}

// canExecute checks if the circuit breaker allows execution
func (cb *CircuitBreaker) canExecute() error {
	switch cb.state {
	case CircuitBreakerOpen:
		if time.Since(cb.lastFailureTime) > cb.timeout {
			cb.state = CircuitBreakerHalfOpen
			cb.halfOpenActions = 0
			return nil
		}
		return fmt.Errorf("circuit breaker is open")
	case CircuitBreakerHalfOpen:
		if cb.halfOpenActions >= 1 {
			return fmt.Errorf("circuit breaker is half-open, waiting for result")
		}
		cb.halfOpenActions++
		return nil
	default:
		return nil
	}
}

// recordFailure records a failure
func (cb *CircuitBreaker) recordFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitBreakerClosed:
		if cb.failures >= cb.failureThreshold {
			cb.state = CircuitBreakerOpen
		}
	case CircuitBreakerHalfOpen:
		cb.state = CircuitBreakerOpen
		cb.failures = 0
	}
}

// recordSuccess records a success
func (cb *CircuitBreaker) recordSuccess() {
	cb.successes++

	switch cb.state {
	case CircuitBreakerClosed:
		cb.failures = 0
	case CircuitBreakerHalfOpen:
		if cb.successes >= cb.successThreshold {
			cb.state = CircuitBreakerClosed
			cb.failures = 0
			cb.successes = 0
		}
	}
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	return cb.state
}
