package controller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
)

func TestPolicyMatcher_Matches(t *testing.T) {
	tests := []struct {
		name     string
		policy   *v1alpha1.HealingPolicy
		object   client.Object
		expected bool
	}{
		{
			name: "matches namespace",
			policy: &v1alpha1.HealingPolicy{
				Spec: v1alpha1.HealingPolicySpec{
					Selector: v1alpha1.ResourceSelector{
						Namespaces: []string{"default", "test"},
						Resources: []v1alpha1.ResourceFilter{
							{
								APIVersion: "v1",
								Kind:       "Pod",
							},
						},
					},
				},
			},
			object: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
			},
			expected: true,
		},
		{
			name: "does not match namespace",
			policy: &v1alpha1.HealingPolicy{
				Spec: v1alpha1.HealingPolicySpec{
					Selector: v1alpha1.ResourceSelector{
						Namespaces: []string{"production"},
						Resources: []v1alpha1.ResourceFilter{
							{
								APIVersion: "v1",
								Kind:       "Pod",
							},
						},
					},
				},
			},
			object: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
			},
			expected: false,
		},
		{
			name: "matches label selector",
			policy: &v1alpha1.HealingPolicy{
				Spec: v1alpha1.HealingPolicySpec{
					Selector: v1alpha1.ResourceSelector{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "nginx",
							},
						},
						Resources: []v1alpha1.ResourceFilter{
							{
								APIVersion: "v1",
								Kind:       "Pod",
							},
						},
					},
				},
			},
			object: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
					Labels: map[string]string{
						"app": "nginx",
					},
				},
			},
			expected: true,
		},
		{
			name: "excludes by name",
			policy: &v1alpha1.HealingPolicy{
				Spec: v1alpha1.HealingPolicySpec{
					Selector: v1alpha1.ResourceSelector{
						Resources: []v1alpha1.ResourceFilter{
							{
								APIVersion:   "v1",
								Kind:         "Pod",
								ExcludeNames: []string{"system-pod", "test-pod"},
							},
						},
					},
				},
			},
			object: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewPolicyMatcher(tt.policy)
			result, err := matcher.Matches(tt.object)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsProtectedResource(t *testing.T) {
	tests := []struct {
		name                string
		object              client.Object
		protectedNamespaces []string
		protectedLabels     map[string]string
		expected            bool
	}{
		{
			name: "protected namespace",
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "kube-system",
				},
			},
			protectedNamespaces: []string{"kube-system", "kube-public"},
			expected:            true,
		},
		{
			name: "protected label",
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
					Labels: map[string]string{
						"kubeskippy.io/protected": "true",
					},
				},
			},
			protectedLabels: map[string]string{
				"kubeskippy.io/protected": "true",
			},
			expected: true,
		},
		{
			name: "protected annotation",
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
					Annotations: map[string]string{
						AnnotationProtected: "true",
					},
				},
			},
			expected: true,
		},
		{
			name: "not protected",
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
			},
			protectedNamespaces: []string{"kube-system"},
			expected:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsProtectedResource(tt.object, tt.protectedNamespaces, tt.protectedLabels)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		name       string
		attempt    int32
		baseDelay  time.Duration
		multiplier float64
		expected   time.Duration
	}{
		{
			name:       "first attempt",
			attempt:    1,
			baseDelay:  time.Second,
			multiplier: 2.0,
			expected:   time.Second,
		},
		{
			name:       "second attempt",
			attempt:    2,
			baseDelay:  time.Second,
			multiplier: 2.0,
			expected:   2 * time.Second,
		},
		{
			name:       "third attempt",
			attempt:    3,
			baseDelay:  time.Second,
			multiplier: 2.0,
			expected:   4 * time.Second,
		},
		{
			name:       "max delay cap",
			attempt:    20,
			baseDelay:  time.Second,
			multiplier: 2.0,
			expected:   30 * time.Minute,
		},
		{
			name:       "zero attempt",
			attempt:    0,
			baseDelay:  time.Second,
			multiplier: 2.0,
			expected:   time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateBackoff(tt.attempt, tt.baseDelay, tt.multiplier)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, 5*time.Second)

	// Initial state should be closed
	assert.Equal(t, CircuitBreakerClosed, cb.GetState())

	// Simulate failures to open the circuit
	for i := 0; i < 3; i++ {
		err := cb.Call(nil, func() error {
			return assert.AnError
		})
		assert.Error(t, err)
	}

	// Circuit should now be open
	assert.Equal(t, CircuitBreakerOpen, cb.GetState())

	// Calls should fail immediately when open
	err := cb.Call(nil, func() error {
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")
}

func TestResourceKey(t *testing.T) {
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	key := ResourceKey(pod)
	assert.Equal(t, ":v1:Pod|default|test-pod", key)

	// Test parsing
	gvk, ns, name, err := ParseResourceKey(key)
	assert.NoError(t, err)
	assert.Equal(t, ":v1:Pod", gvk)
	assert.Equal(t, "default", ns)
	assert.Equal(t, "test-pod", name)
}

func TestCreateHealingAction(t *testing.T) {
	policy := &v1alpha1.HealingPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-policy",
			Namespace: "default",
			UID:       "policy-uid",
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeskippy.io/v1alpha1",
			Kind:       "HealingPolicy",
		},
		Spec: v1alpha1.HealingPolicySpec{
			Mode: "automatic",
		},
	}

	target := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "target-pod",
			Namespace: "default",
			UID:       "pod-uid",
		},
	}

	actionTemplate := &v1alpha1.HealingActionTemplate{
		Name:             "restart",
		Type:             "restart",
		RequiresApproval: false,
	}

	action := CreateHealingAction(policy, target, actionTemplate, false)

	assert.NotNil(t, action)
	assert.Equal(t, "default", action.Namespace)
	assert.Contains(t, action.GenerateName, "test-policy-restart-")
	assert.Equal(t, "kubeskippy", action.Labels[LabelManagedBy])
	assert.Equal(t, "test-policy", action.Labels[LabelPolicyName])
	assert.Equal(t, "restart", action.Labels[LabelActionType])
	assert.Equal(t, v1alpha1.HealingActionPhasePending, action.Labels[LabelActionPhase])

	assert.Equal(t, "test-policy", action.Spec.PolicyRef.Name)
	assert.Equal(t, "default", action.Spec.PolicyRef.Namespace)
	assert.Equal(t, "v1", action.Spec.TargetResource.APIVersion)
	assert.Equal(t, "Pod", action.Spec.TargetResource.Kind)
	assert.Equal(t, "target-pod", action.Spec.TargetResource.Name)

	assert.False(t, action.Spec.ApprovalRequired)
	assert.False(t, action.Spec.DryRun)
	assert.NotNil(t, action.Spec.RetryPolicy)
	assert.Equal(t, int32(3), action.Spec.RetryPolicy.MaxAttempts)
}

func TestHealingActionHelpers(t *testing.T) {
	action := &v1alpha1.HealingAction{
		Spec: v1alpha1.HealingActionSpec{
			ApprovalRequired: true,
		},
		Status: v1alpha1.HealingActionStatus{
			Phase: v1alpha1.HealingActionPhasePending,
			Approval: &v1alpha1.ApprovalStatus{
				Required: true,
				Approved: false,
			},
		},
	}

	// Test IsComplete
	assert.False(t, action.IsComplete())

	action.Status.Phase = v1alpha1.HealingActionPhaseSucceeded
	assert.True(t, action.IsComplete())

	action.Status.Phase = v1alpha1.HealingActionPhaseFailed
	assert.True(t, action.IsComplete())

	action.Status.Phase = v1alpha1.HealingActionPhaseCancelled
	assert.True(t, action.IsComplete())

	// Test NeedsApproval
	action.Status.Phase = v1alpha1.HealingActionPhasePending
	assert.True(t, action.NeedsApproval())

	action.Status.Approval.Approved = true
	assert.False(t, action.NeedsApproval())

	action.Spec.ApprovalRequired = false
	assert.False(t, action.NeedsApproval())

	// Test SetPhase
	action.SetPhase(v1alpha1.HealingActionPhaseInProgress, "Starting", "Action is starting")
	assert.Equal(t, v1alpha1.HealingActionPhaseInProgress, action.Status.Phase)
	assert.NotNil(t, action.Status.StartTime)

	action.SetPhase(v1alpha1.HealingActionPhaseSucceeded, "Completed", "Action completed successfully")
	assert.Equal(t, v1alpha1.HealingActionPhaseSucceeded, action.Status.Phase)
	assert.NotNil(t, action.Status.CompletionTime)
}
