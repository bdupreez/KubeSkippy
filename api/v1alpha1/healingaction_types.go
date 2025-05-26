package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HealingActionSpec defines the desired state of HealingAction
type HealingActionSpec struct {
	// PolicyRef references the HealingPolicy that created this action
	PolicyRef PolicyReference `json:"policyRef"`

	// TargetResource identifies what to heal
	TargetResource TargetResource `json:"targetResource"`

	// Action to perform
	Action HealingActionTemplate `json:"action"`

	// ApprovalRequired indicates if manual approval is needed
	ApprovalRequired bool `json:"approvalRequired,omitempty"`

	// DryRun indicates this is a simulation
	DryRun bool `json:"dryRun,omitempty"`

	// Timeout for the action
	// +kubebuilder:default="10m"
	Timeout metav1.Duration `json:"timeout,omitempty"`

	// RetryPolicy for failed actions
	RetryPolicy *RetryPolicy `json:"retryPolicy,omitempty"`
}

// PolicyReference links to the source HealingPolicy
type PolicyReference struct {
	// Name of the HealingPolicy
	Name string `json:"name"`

	// Namespace of the HealingPolicy
	Namespace string `json:"namespace"`

	// UID of the HealingPolicy for validation
	UID string `json:"uid,omitempty"`
}

// TargetResource identifies the resource to heal
type TargetResource struct {
	// APIVersion of the resource
	APIVersion string `json:"apiVersion"`

	// Kind of the resource
	Kind string `json:"kind"`

	// Name of the resource
	Name string `json:"name"`

	// Namespace of the resource
	Namespace string `json:"namespace,omitempty"`

	// UID of the resource for validation
	UID string `json:"uid,omitempty"`
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	// MaxAttempts before giving up
	// +kubebuilder:default=3
	MaxAttempts int32 `json:"maxAttempts,omitempty"`

	// BackoffDelay between attempts
	// +kubebuilder:default="30s"
	BackoffDelay metav1.Duration `json:"backoffDelay,omitempty"`

	// BackoffMultiplier for exponential backoff
	// +kubebuilder:default=2.0
	BackoffMultiplier float64 `json:"backoffMultiplier,omitempty"`
}

// HealingActionStatus defines the observed state of HealingAction
type HealingActionStatus struct {
	// Phase of the action
	// +kubebuilder:validation:Enum=Pending;Approved;InProgress;Succeeded;Failed;Cancelled
	Phase string `json:"phase,omitempty"`

	// StartTime when execution began
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime when execution finished
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Attempts made so far
	Attempts int32 `json:"attempts,omitempty"`

	// LastAttemptTime of the most recent attempt
	LastAttemptTime *metav1.Time `json:"lastAttemptTime,omitempty"`

	// Result of the action
	Result *ActionResult `json:"result,omitempty"`

	// Approval information
	Approval *ApprovalStatus `json:"approval,omitempty"`

	// Conditions of the action
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration for tracking updates
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// ActionResult captures the outcome of a healing action
type ActionResult struct {
	// Success indicates if the action succeeded
	Success bool `json:"success"`

	// Message describing the result
	Message string `json:"message,omitempty"`

	// Error if the action failed
	Error string `json:"error,omitempty"`

	// Metrics captured during execution
	Metrics map[string]string `json:"metrics,omitempty"`

	// Changes made to the target resource
	Changes []ResourceChange `json:"changes,omitempty"`
}

// ResourceChange describes a modification made
type ResourceChange struct {
	// Field that was changed
	Field string `json:"field"`

	// OldValue before the change
	OldValue string `json:"oldValue,omitempty"`

	// NewValue after the change
	NewValue string `json:"newValue,omitempty"`

	// Timestamp of the change
	Timestamp metav1.Time `json:"timestamp"`
}

// ApprovalStatus tracks manual approval state
type ApprovalStatus struct {
	// Required indicates if approval is needed
	Required bool `json:"required"`

	// Approved indicates if approval was granted
	Approved bool `json:"approved"`

	// ApprovedBy identifies who approved
	ApprovedBy string `json:"approvedBy,omitempty"`

	// ApprovedAt timestamp
	ApprovedAt *metav1.Time `json:"approvedAt,omitempty"`

	// Reason for approval/rejection
	Reason string `json:"reason,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=ha
// +kubebuilder:printcolumn:name="Target",type="string",JSONPath=".spec.targetResource.kind"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Success",type="boolean",JSONPath=".status.result.success"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// HealingAction is the Schema for the healingactions API
type HealingAction struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HealingActionSpec   `json:"spec,omitempty"`
	Status HealingActionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HealingActionList contains a list of HealingAction
type HealingActionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HealingAction `json:"items"`
}

// Phase constants
const (
	HealingActionPhasePending    = "Pending"
	HealingActionPhaseApproved   = "Approved"
	HealingActionPhaseInProgress = "InProgress"
	HealingActionPhaseSucceeded  = "Succeeded"
	HealingActionPhaseFailed     = "Failed"
	HealingActionPhaseCancelled  = "Cancelled"
)

// Condition types
const (
	ConditionTypeReady     = "Ready"
	ConditionTypeApproved  = "Approved"
	ConditionTypeCompleted = "Completed"
)

func init() {
	SchemeBuilder.Register(&HealingAction{}, &HealingActionList{})
}

// IsComplete returns true if the action has finished (successfully or not)
func (ha *HealingAction) IsComplete() bool {
	return ha.Status.Phase == HealingActionPhaseSucceeded ||
		ha.Status.Phase == HealingActionPhaseFailed ||
		ha.Status.Phase == HealingActionPhaseCancelled
}

// NeedsApproval returns true if the action requires approval
func (ha *HealingAction) NeedsApproval() bool {
	return ha.Spec.ApprovalRequired && !ha.Status.Approval.Approved
}

// SetPhase updates the action phase and sets appropriate conditions
func (ha *HealingAction) SetPhase(phase string, reason, message string) {
	ha.Status.Phase = phase
	now := metav1.Now()

	switch phase {
	case HealingActionPhaseInProgress:
		if ha.Status.StartTime == nil {
			ha.Status.StartTime = &now
		}
	case HealingActionPhaseSucceeded, HealingActionPhaseFailed, HealingActionPhaseCancelled:
		if ha.Status.CompletionTime == nil {
			ha.Status.CompletionTime = &now
		}
	}

	// Update conditions based on phase
	condition := metav1.Condition{
		Type:               ConditionTypeReady,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: ha.Generation,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	}

	if phase == HealingActionPhaseSucceeded {
		condition.Status = metav1.ConditionTrue
	}

	meta.SetStatusCondition(&ha.Status.Conditions, condition)
}