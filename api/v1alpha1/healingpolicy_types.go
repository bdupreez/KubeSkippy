package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HealingPolicySpec defines the desired state of HealingPolicy
type HealingPolicySpec struct {
	// Selector defines which resources this policy applies to
	Selector ResourceSelector `json:"selector"`

	// Triggers define conditions that activate healing
	Triggers []HealingTrigger `json:"triggers"`

	// Actions define what healing actions to take
	Actions []HealingActionTemplate `json:"actions"`

	// SafetyRules define constraints on healing actions
	SafetyRules SafetyRules `json:"safetyRules,omitempty"`

	// Mode defines whether actions are automatic or require approval
	// +kubebuilder:validation:Enum=monitor;dryrun;automatic;manual
	// +kubebuilder:default=monitor
	Mode string `json:"mode,omitempty"`
}

// ResourceSelector defines how to select resources for healing
type ResourceSelector struct {
	// Namespaces to include (empty means all namespaces)
	Namespaces []string `json:"namespaces,omitempty"`

	// LabelSelector for resources
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`

	// Resource types to monitor
	Resources []ResourceFilter `json:"resources"`
}

// ResourceFilter defines a specific resource type to monitor
type ResourceFilter struct {
	// APIVersion of the resource
	APIVersion string `json:"apiVersion"`

	// Kind of the resource
	Kind string `json:"kind"`

	// ExcludeNames to ignore specific resource names
	ExcludeNames []string `json:"excludeNames,omitempty"`
}

// HealingTrigger defines when to initiate healing
type HealingTrigger struct {
	// Name of this trigger
	Name string `json:"name"`

	// Type of trigger
	// +kubebuilder:validation:Enum=metric;event;condition
	Type string `json:"type"`

	// MetricTrigger for Prometheus-based triggers
	MetricTrigger *MetricTrigger `json:"metricTrigger,omitempty"`

	// EventTrigger for Kubernetes event-based triggers
	EventTrigger *EventTrigger `json:"eventTrigger,omitempty"`

	// ConditionTrigger for resource condition-based triggers
	ConditionTrigger *ConditionTrigger `json:"conditionTrigger,omitempty"`

	// CooldownPeriod prevents trigger from firing too frequently
	// +kubebuilder:default="5m"
	CooldownPeriod metav1.Duration `json:"cooldownPeriod,omitempty"`
}

// MetricTrigger defines Prometheus metric-based triggers
type MetricTrigger struct {
	// Query is the PromQL query
	Query string `json:"query"`

	// Threshold for the metric
	Threshold float64 `json:"threshold"`

	// Operator for comparison
	// +kubebuilder:validation:Enum=">";"<";">=";"<="
	Operator string `json:"operator"`

	// Duration the condition must be true
	// +kubebuilder:default="2m"
	Duration metav1.Duration `json:"duration,omitempty"`
}

// EventTrigger defines Kubernetes event-based triggers
type EventTrigger struct {
	// Reason to match
	Reason string `json:"reason,omitempty"`

	// Type of event (Normal, Warning)
	Type string `json:"type,omitempty"`

	// Count threshold
	Count int32 `json:"count,omitempty"`

	// Window to count events in
	// +kubebuilder:default="5m"
	Window metav1.Duration `json:"window,omitempty"`
}

// ConditionTrigger defines resource condition-based triggers
type ConditionTrigger struct {
	// Type of condition
	Type string `json:"type"`

	// Status to match
	Status string `json:"status"`

	// Duration the condition must exist
	// +kubebuilder:default="2m"
	Duration metav1.Duration `json:"duration,omitempty"`
}

// HealingActionTemplate defines a healing action to take
type HealingActionTemplate struct {
	// Name of this action
	Name string `json:"name"`

	// Type of action
	// +kubebuilder:validation:Enum=restart;scale;patch;delete;custom
	Type string `json:"type"`

	// Description for logging/auditing
	Description string `json:"description,omitempty"`

	// RestartAction for pod restarts
	RestartAction *RestartAction `json:"restartAction,omitempty"`

	// ScaleAction for scaling operations
	ScaleAction *ScaleAction `json:"scaleAction,omitempty"`

	// PatchAction for resource patches
	PatchAction *PatchAction `json:"patchAction,omitempty"`

	// DeleteAction for resource deletion
	DeleteAction *DeleteAction `json:"deleteAction,omitempty"`

	// Priority of this action (higher executes first)
	// +kubebuilder:default=50
	Priority int32 `json:"priority,omitempty"`

	// RequiresApproval overrides policy mode
	RequiresApproval bool `json:"requiresApproval,omitempty"`
}

// RestartAction defines pod restart parameters
type RestartAction struct {
	// Strategy for restart
	// +kubebuilder:validation:Enum=recreate;rolling;graceful
	// +kubebuilder:default=rolling
	Strategy string `json:"strategy,omitempty"`

	// MaxConcurrent pods to restart at once
	// +kubebuilder:default=1
	MaxConcurrent int32 `json:"maxConcurrent,omitempty"`

	// GracePeriodSeconds for graceful shutdown
	// +kubebuilder:default=30
	GracePeriodSeconds int32 `json:"gracePeriodSeconds,omitempty"`
}

// ScaleAction defines scaling parameters
type ScaleAction struct {
	// Direction of scaling
	// +kubebuilder:validation:Enum=up;down;absolute
	Direction string `json:"direction"`

	// Replicas to scale by or to
	Replicas int32 `json:"replicas"`

	// MinReplicas constraint
	// +kubebuilder:default=0
	MinReplicas int32 `json:"minReplicas,omitempty"`

	// MaxReplicas constraint
	// +kubebuilder:default=100
	MaxReplicas int32 `json:"maxReplicas,omitempty"`
}

// PatchAction defines resource patching
type PatchAction struct {
	// Type of patch
	// +kubebuilder:validation:Enum=strategic;merge;json
	Type string `json:"type"`

	// Patch content
	Patch string `json:"patch"`

	// Patches for structured patching
	Patches []PatchOperation `json:"patches,omitempty"`
}

// PatchOperation defines a single patch operation
type PatchOperation struct {
	// Path to the field to patch
	Path []string `json:"path"`

	// Value to set
	Value string `json:"value"`
}

// DeleteAction defines resource deletion parameters
type DeleteAction struct {
	// GracePeriodSeconds before force deletion
	// +kubebuilder:default=30
	GracePeriodSeconds int32 `json:"gracePeriodSeconds,omitempty"`

	// Force deletion even with finalizers
	Force bool `json:"force,omitempty"`

	// PropagationPolicy for deletion
	// +kubebuilder:validation:Enum=Orphan;Background;Foreground
	PropagationPolicy string `json:"propagationPolicy,omitempty"`
}

// SafetyRules define constraints on healing actions
type SafetyRules struct {
	// MaxActionsPerHour limits action frequency
	// +kubebuilder:default=10
	MaxActionsPerHour int32 `json:"maxActionsPerHour,omitempty"`

	// ProtectedResources that should never be modified
	ProtectedResources []ResourceFilter `json:"protectedResources,omitempty"`

	// RequireHealthCheck before marking action successful
	RequireHealthCheck bool `json:"requireHealthCheck,omitempty"`

	// HealthCheckTimeout for post-action validation
	// +kubebuilder:default="5m"
	HealthCheckTimeout metav1.Duration `json:"healthCheckTimeout,omitempty"`
}

// HealingPolicyStatus defines the observed state of HealingPolicy
type HealingPolicyStatus struct {
	// LastEvaluated timestamp
	LastEvaluated metav1.Time `json:"lastEvaluated,omitempty"`

	// ActiveTriggers currently firing
	ActiveTriggers []string `json:"activeTriggers,omitempty"`

	// ActionsTaken in the current period
	ActionsTaken int32 `json:"actionsTaken,omitempty"`

	// LastActionTime of the most recent action
	LastActionTime metav1.Time `json:"lastActionTime,omitempty"`

	// Conditions of the policy
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration for tracking updates
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=hp
// +kubebuilder:printcolumn:name="Mode",type="string",JSONPath=".spec.mode"
// +kubebuilder:printcolumn:name="Actions Taken",type="integer",JSONPath=".status.actionsTaken"
// +kubebuilder:printcolumn:name="Last Action",type="date",JSONPath=".status.lastActionTime"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// HealingPolicy is the Schema for the healingpolicies API
type HealingPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HealingPolicySpec   `json:"spec,omitempty"`
	Status HealingPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HealingPolicyList contains a list of HealingPolicy
type HealingPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HealingPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HealingPolicy{}, &HealingPolicyList{})
}