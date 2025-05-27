package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
	"github.com/kubeskippy/kubeskippy/pkg/config"
)

// MockMetricsCollector implements MetricsCollector interface for testing
type MockMetricsCollector struct {
	CollectMetricsFunc     func(ctx context.Context, policy *v1alpha1.HealingPolicy) (*ClusterMetrics, error)
	EvaluateTriggerFunc    func(ctx context.Context, trigger *v1alpha1.HealingTrigger, metrics *ClusterMetrics) (bool, string, error)
	GetResourceMetricsFunc func(ctx context.Context, resource *v1alpha1.TargetResource) (*ResourceMetrics, error)
}

func (m *MockMetricsCollector) CollectMetrics(ctx context.Context, policy *v1alpha1.HealingPolicy) (*ClusterMetrics, error) {
	if m.CollectMetricsFunc != nil {
		return m.CollectMetricsFunc(ctx, policy)
	}
	return &ClusterMetrics{Timestamp: time.Now()}, nil
}

func (m *MockMetricsCollector) EvaluateTrigger(ctx context.Context, trigger *v1alpha1.HealingTrigger, metrics *ClusterMetrics) (bool, string, error) {
	if m.EvaluateTriggerFunc != nil {
		return m.EvaluateTriggerFunc(ctx, trigger, metrics)
	}
	return false, "", nil
}

func (m *MockMetricsCollector) GetResourceMetrics(ctx context.Context, resource *v1alpha1.TargetResource) (*ResourceMetrics, error) {
	if m.GetResourceMetricsFunc != nil {
		return m.GetResourceMetricsFunc(ctx, resource)
	}
	return &ResourceMetrics{}, nil
}

// MockSafetyController implements SafetyController interface for testing
type MockSafetyController struct {
	ValidateActionFunc      func(ctx context.Context, action *v1alpha1.HealingAction) (*ValidationResult, error)
	CheckRateLimitFunc      func(ctx context.Context, policy *v1alpha1.HealingPolicy) (bool, error)
	IsProtectedResourceFunc func(resource runtime.Object) (bool, string)
	RecordActionFunc        func(ctx context.Context, action *v1alpha1.HealingAction, result *ActionResult)
}

func (m *MockSafetyController) ValidateAction(ctx context.Context, action *v1alpha1.HealingAction) (*ValidationResult, error) {
	if m.ValidateActionFunc != nil {
		return m.ValidateActionFunc(ctx, action)
	}
	return &ValidationResult{Valid: true}, nil
}

func (m *MockSafetyController) CheckRateLimit(ctx context.Context, policy *v1alpha1.HealingPolicy) (bool, error) {
	if m.CheckRateLimitFunc != nil {
		return m.CheckRateLimitFunc(ctx, policy)
	}
	return true, nil
}

func (m *MockSafetyController) IsProtectedResource(resource runtime.Object) (bool, string) {
	if m.IsProtectedResourceFunc != nil {
		return m.IsProtectedResourceFunc(resource)
	}
	return false, ""
}

func (m *MockSafetyController) RecordAction(ctx context.Context, action *v1alpha1.HealingAction, result *ActionResult) {
	if m.RecordActionFunc != nil {
		m.RecordActionFunc(ctx, action, result)
	}
}

func TestHealingPolicyReconciler_Reconcile(t *testing.T) {
	// Create scheme
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	tests := []struct {
		name           string
		policy         *v1alpha1.HealingPolicy
		existingObjs   []client.Object
		metricsFunc    func(ctx context.Context, policy *v1alpha1.HealingPolicy) (*ClusterMetrics, error)
		triggerFunc    func(ctx context.Context, trigger *v1alpha1.HealingTrigger, metrics *ClusterMetrics) (bool, string, error)
		rateLimitFunc  func(ctx context.Context, policy *v1alpha1.HealingPolicy) (bool, error)
		validateFunc   func(ctx context.Context, action *v1alpha1.HealingAction) (*ValidationResult, error)
		expectedResult reconcile.Result
		expectedError  bool
		checkFunc      func(t *testing.T, client client.Client)
	}{
		{
			name: "policy not found",
			policy: &v1alpha1.HealingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-policy",
					Namespace: "default",
				},
			},
			existingObjs:   []client.Object{},
			expectedResult: reconcile.Result{},
			expectedError:  false,
		},
		{
			name: "monitor mode - no actions created",
			policy: &v1alpha1.HealingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-policy",
					Namespace: "default",
				},
				Spec: v1alpha1.HealingPolicySpec{
					Mode: "monitor",
					Selector: v1alpha1.ResourceSelector{
						Resources: []v1alpha1.ResourceFilter{
							{APIVersion: "v1", Kind: "Pod"},
						},
					},
					Triggers: []v1alpha1.HealingTrigger{
						{
							Name: "high-restarts",
							Type: "metric",
						},
					},
					Actions: []v1alpha1.HealingActionTemplate{
						{
							Name: "restart",
							Type: "restart",
						},
					},
				},
			},
			existingObjs: []client.Object{
				&v1alpha1.HealingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-policy",
						Namespace: "default",
					},
					Spec: v1alpha1.HealingPolicySpec{
						Mode: "monitor",
						Selector: v1alpha1.ResourceSelector{
							Resources: []v1alpha1.ResourceFilter{
								{APIVersion: "v1", Kind: "Pod"},
							},
						},
						Triggers: []v1alpha1.HealingTrigger{
							{Name: "high-restarts", Type: "metric"},
						},
						Actions: []v1alpha1.HealingActionTemplate{
							{Name: "restart", Type: "restart"},
						},
					},
				},
			},
			expectedResult: reconcile.Result{RequeueAfter: 5 * time.Minute},
			expectedError:  false,
			checkFunc: func(t *testing.T, c client.Client) {
				// No healing actions should be created in monitor mode
				actionList := &v1alpha1.HealingActionList{}
				err := c.List(context.Background(), actionList)
				require.NoError(t, err)
				assert.Empty(t, actionList.Items)
			},
		},
		{
			name: "rate limited - no actions created",
			policy: &v1alpha1.HealingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-policy",
					Namespace: "default",
				},
				Spec: v1alpha1.HealingPolicySpec{
					Mode: "automatic",
					Selector: v1alpha1.ResourceSelector{
						Resources: []v1alpha1.ResourceFilter{
							{APIVersion: "v1", Kind: "Pod"},
						},
					},
					Triggers: []v1alpha1.HealingTrigger{
						{Name: "high-restarts", Type: "metric"},
					},
					Actions: []v1alpha1.HealingActionTemplate{
						{Name: "restart", Type: "restart"},
					},
				},
			},
			existingObjs: []client.Object{
				&v1alpha1.HealingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-policy",
						Namespace: "default",
					},
					Spec: v1alpha1.HealingPolicySpec{
						Mode: "automatic",
						Selector: v1alpha1.ResourceSelector{
							Resources: []v1alpha1.ResourceFilter{
								{APIVersion: "v1", Kind: "Pod"},
							},
						},
						Triggers: []v1alpha1.HealingTrigger{
							{Name: "high-restarts", Type: "metric"},
						},
						Actions: []v1alpha1.HealingActionTemplate{
							{Name: "restart", Type: "restart"},
						},
					},
				},
			},
			rateLimitFunc: func(ctx context.Context, policy *v1alpha1.HealingPolicy) (bool, error) {
				return false, nil // Rate limited
			},
			expectedResult: reconcile.Result{RequeueAfter: 1 * time.Minute},
			expectedError:  false,
			checkFunc: func(t *testing.T, c client.Client) {
				// No healing actions should be created when rate limited
				actionList := &v1alpha1.HealingActionList{}
				err := c.List(context.Background(), actionList)
				require.NoError(t, err)
				assert.Empty(t, actionList.Items)
			},
		},
		{
			name: "trigger activated - action created",
			policy: &v1alpha1.HealingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-policy",
					Namespace: "default",
				},
				Spec: v1alpha1.HealingPolicySpec{
					Mode: "automatic",
					Selector: v1alpha1.ResourceSelector{
						Resources: []v1alpha1.ResourceFilter{
							{APIVersion: "v1", Kind: "Pod"},
						},
					},
					Triggers: []v1alpha1.HealingTrigger{
						{
							Name:           "high-restarts",
							Type:           "metric",
							CooldownPeriod: metav1.Duration{Duration: 5 * time.Minute},
						},
					},
					Actions: []v1alpha1.HealingActionTemplate{
						{
							Name: "restart",
							Type: "restart",
						},
					},
				},
			},
			existingObjs: []client.Object{
				&v1alpha1.HealingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-policy",
						Namespace: "default",
					},
					Spec: v1alpha1.HealingPolicySpec{
						Mode: "automatic",
						Selector: v1alpha1.ResourceSelector{
							Resources: []v1alpha1.ResourceFilter{
								{APIVersion: "v1", Kind: "Pod"},
							},
						},
						Triggers: []v1alpha1.HealingTrigger{
							{
								Name:           "high-restarts",
								Type:           "metric",
								CooldownPeriod: metav1.Duration{Duration: 5 * time.Minute},
							},
						},
						Actions: []v1alpha1.HealingActionTemplate{
							{Name: "restart", Type: "restart"},
						},
					},
				},
				&corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Pod",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
					},
				},
			},
			triggerFunc: func(ctx context.Context, trigger *v1alpha1.HealingTrigger, metrics *ClusterMetrics) (bool, string, error) {
				return true, "High restart count detected", nil
			},
			expectedResult: reconcile.Result{RequeueAfter: 1 * time.Minute},
			expectedError:  false,
			checkFunc: func(t *testing.T, c client.Client) {
				// A healing action should be created
				actionList := &v1alpha1.HealingActionList{}
				err := c.List(context.Background(), actionList)
				require.NoError(t, err)
				require.Len(t, actionList.Items, 1)

				action := &actionList.Items[0]
				assert.Equal(t, "restart", action.Spec.Action.Type)
				assert.Equal(t, "Pod", action.Spec.TargetResource.Kind)
				assert.Equal(t, "test-pod", action.Spec.TargetResource.Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.existingObjs...).
				Build()

			// Create mocks
			metricsCollector := &MockMetricsCollector{
				CollectMetricsFunc:  tt.metricsFunc,
				EvaluateTriggerFunc: tt.triggerFunc,
			}

			safetyController := &MockSafetyController{
				CheckRateLimitFunc: tt.rateLimitFunc,
				ValidateActionFunc: tt.validateFunc,
			}

			// Create reconciler
			r := &HealingPolicyReconciler{
				Client:           fakeClient,
				Scheme:           scheme,
				Config:           config.NewDefaultConfig(),
				MetricsCollector: metricsCollector,
				SafetyController: safetyController,
			}

			// Reconcile
			result, err := r.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.policy.Name,
					Namespace: tt.policy.Namespace,
				},
			})

			// Check result
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedResult, result)

			// Run additional checks
			if tt.checkFunc != nil {
				tt.checkFunc(t, fakeClient)
			}
		})
	}
}

func TestHealingPolicyReconciler_findMatchingResources(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	policy := &v1alpha1.HealingPolicy{
		Spec: v1alpha1.HealingPolicySpec{
			Selector: v1alpha1.ResourceSelector{
				Namespaces: []string{"default"},
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test",
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
	}

	pods := []client.Object{
		&corev1.Pod{
			TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: "default",
				Labels:    map[string]string{"app": "test"},
			},
		},
		&corev1.Pod{
			TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod2",
				Namespace: "default",
				Labels:    map[string]string{"app": "other"},
			},
		},
		&corev1.Pod{
			TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod3",
				Namespace: "kube-system",
				Labels:    map[string]string{"app": "test"},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pods...).
		Build()

	r := &HealingPolicyReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	resources, err := r.findMatchingResources(context.Background(), policy)
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "pod1", resources[0].GetName())
}
