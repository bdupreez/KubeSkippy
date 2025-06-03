package types

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
	Name           string
	CPUUsage       float64
	MemoryUsage    float64
	DiskUsage      float64
	PodCount       int32
	Conditions     []string
	Labels         map[string]string
	LastUpdateTime time.Time
}

// PodMetrics represents metrics for a pod
type PodMetrics struct {
	Name            string
	Namespace       string
	CPUUsage        float64
	MemoryUsage     float64
	RestartCount    int32
	Status          string
	Conditions      []string
	Labels          map[string]string
	OwnerReferences []string
	LastUpdateTime  time.Time
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
	ReasoningSteps  []ReasoningStep
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
	ID         string
	Priority   int
	Action     string
	Target     string
	Reason     string
	Risk       string
	Confidence float64
	Reasoning  DecisionReasoning
}

// ReasoningStep represents a step in the AI's decision process
type ReasoningStep struct {
	Step        int
	Description string
	Evidence    []string
	Confidence  float64
	Timestamp   time.Time
}

// DecisionReasoning contains detailed reasoning for a specific recommendation
type DecisionReasoning struct {
	Observations      []string
	Analysis          []string
	Alternatives      []Alternative
	DecisionLogic     string
	ConfidenceFactors []ConfidenceFactor
}

// Alternative represents an alternative action considered by the AI
type Alternative struct {
	Action     string
	Pros       []string
	Cons       []string
	RiskLevel  string
	Confidence float64
	Rejected   bool
	Reason     string
}

// ConfidenceFactor represents factors that influence confidence level
type ConfidenceFactor struct {
	Factor   string
	Impact   string // "positive", "negative", "neutral"
	Weight   float64
	Evidence string
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

// Common annotations
const (
	AnnotationProtected       = "kubeskippy.io/protected"
	AnnotationHealingDisabled = "kubeskippy.io/healing-disabled"
)

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
	mu               sync.RWMutex
	failureCount     int
	successCount     int
	lastFailureTime  time.Time
	timeout          time.Duration
	failureThreshold int
	successThreshold int
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold, successThreshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            CircuitBreakerClosed,
		timeout:          timeout,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err := cb.canExecute(); err != nil {
		return err
	}

	err := fn()
	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

// canExecute checks if the circuit breaker allows execution
func (cb *CircuitBreaker) canExecute() error {
	switch cb.state {
	case CircuitBreakerOpen:
		if time.Since(cb.lastFailureTime) > cb.timeout {
			cb.state = CircuitBreakerHalfOpen
			cb.successCount = 0
			return nil
		}
		return fmt.Errorf("circuit breaker is open")
	case CircuitBreakerHalfOpen:
		return nil
	case CircuitBreakerClosed:
		return nil
	default:
		return nil
	}
}

// recordFailure records a failure and updates state
func (cb *CircuitBreaker) recordFailure() {
	cb.lastFailureTime = time.Now()
	cb.failureCount++
	cb.successCount = 0

	switch cb.state {
	case CircuitBreakerClosed:
		if cb.failureCount >= cb.failureThreshold {
			cb.state = CircuitBreakerOpen
		}
	case CircuitBreakerHalfOpen:
		cb.state = CircuitBreakerOpen
		cb.failureCount = 1
	}
}

// recordSuccess records a success and updates state
func (cb *CircuitBreaker) recordSuccess() {
	cb.failureCount = 0
	cb.successCount++

	switch cb.state {
	case CircuitBreakerClosed:
		// Stay closed
	case CircuitBreakerHalfOpen:
		if cb.successCount >= cb.successThreshold {
			cb.state = CircuitBreakerClosed
		}
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}