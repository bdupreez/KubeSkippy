package remediation

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
	"github.com/kubeskippy/kubeskippy/internal/controller"
)

// MockExecutor is a mock action executor for testing
type MockExecutor struct {
	ExecuteFunc  func(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) (*controller.ActionResult, error)
	ValidateFunc func(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) error
	DryRunFunc   func(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) (*controller.ActionResult, error)
}

func (m *MockExecutor) Execute(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) (*controller.ActionResult, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, target, action)
	}
	return &controller.ActionResult{Success: true, Message: "Mock execution"}, nil
}

func (m *MockExecutor) Validate(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) error {
	if m.ValidateFunc != nil {
		return m.ValidateFunc(ctx, target, action)
	}
	return nil
}

func (m *MockExecutor) DryRun(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) (*controller.ActionResult, error) {
	if m.DryRunFunc != nil {
		return m.DryRunFunc(ctx, target, action)
	}
	return &controller.ActionResult{Success: true, Message: "Mock dry-run"}, nil
}

func TestEngine_ExecuteAction(t *testing.T) {
	// Create test pod
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test:latest",
				},
			},
		},
	}

	// Create fake client
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod).
		Build()

	// Create recorder
	recorder := NewInMemoryActionRecorder(1 * time.Hour)

	// Create engine
	engine := NewEngine(fakeClient, recorder)

	// Test successful execution
	t.Run("successful execution", func(t *testing.T) {
		action := &v1alpha1.HealingAction{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-action",
				Namespace: "default",
			},
			Spec: v1alpha1.HealingActionSpec{
				TargetResource: v1alpha1.TargetResource{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       "test-pod",
					Namespace:  "default",
				},
				Action: v1alpha1.HealingActionTemplate{
					Name: "restart-pod",
					Type: "restart",
				},
			},
		}

		// Override restart executor with mock
		mockExecutor := &MockExecutor{
			ExecuteFunc: func(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) (*controller.ActionResult, error) {
				return &controller.ActionResult{
					Success: true,
					Message: "Pod restarted successfully",
					Changes: []v1alpha1.ResourceChange{
						{
							ResourceRef: "Pod/default/test-pod",
							ChangeType:  "delete",
							Field:       "pod",
							OldValue:    "test-pod",
							NewValue:    "recreated",
						},
					},
				}, nil
			},
		}
		engine.RegisterExecutor("restart", mockExecutor)

		result, err := engine.ExecuteAction(context.Background(), action)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, "Pod restarted successfully", result.Message)
		assert.Len(t, result.Changes, 1)

		// Check if action was recorded
		history, err := recorder.GetActionHistory(context.Background(), action.Name)
		require.NoError(t, err)
		assert.NotNil(t, history)
		assert.Equal(t, action.Name, history.ActionName)
	})

	// Test validation failure
	t.Run("validation failure", func(t *testing.T) {
		action := &v1alpha1.HealingAction{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-action-invalid",
				Namespace: "default",
			},
			Spec: v1alpha1.HealingActionSpec{
				TargetResource: v1alpha1.TargetResource{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       "test-pod",
					Namespace:  "default",
				},
				Action: v1alpha1.HealingActionTemplate{
					Name: "invalid-action",
					Type: "invalid",
				},
			},
		}

		// Override executor with validation failure
		mockExecutor := &MockExecutor{
			ValidateFunc: func(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) error {
				return fmt.Errorf("invalid action type")
			},
		}
		engine.RegisterExecutor("invalid", mockExecutor)

		result, err := engine.ExecuteAction(context.Background(), action)
		require.NoError(t, err) // No error returned, but result indicates failure
		assert.False(t, result.Success)
		assert.Contains(t, result.Message, "validation failed")
	})

	// Test non-existent resource
	t.Run("non-existent resource", func(t *testing.T) {
		action := &v1alpha1.HealingAction{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-action-notfound",
				Namespace: "default",
			},
			Spec: v1alpha1.HealingActionSpec{
				TargetResource: v1alpha1.TargetResource{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       "non-existent-pod",
					Namespace:  "default",
				},
				Action: v1alpha1.HealingActionTemplate{
					Name: "restart",
					Type: "restart",
				},
			},
		}

		result, err := engine.ExecuteAction(context.Background(), action)
		require.Error(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Message, "Failed to get target resource")
	})

	// Test unknown executor
	t.Run("unknown executor", func(t *testing.T) {
		action := &v1alpha1.HealingAction{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-action-unknown",
				Namespace: "default",
			},
			Spec: v1alpha1.HealingActionSpec{
				TargetResource: v1alpha1.TargetResource{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       "test-pod",
					Namespace:  "default",
				},
				Action: v1alpha1.HealingActionTemplate{
					Name: "unknown",
					Type: "unknown",
				},
			},
		}

		result, err := engine.ExecuteAction(context.Background(), action)
		require.Error(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Message, "Failed to get executor")
	})
}

func TestEngine_DryRun(t *testing.T) {
	// Create test deployment
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "test"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test:latest",
						},
					},
				},
			},
		},
	}

	// Create fake client
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(deployment).
		Build()

	// Create engine
	engine := NewEngine(fakeClient, nil)

	// Test dry-run
	action := &v1alpha1.HealingAction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-scale-action",
			Namespace: "default",
		},
		Spec: v1alpha1.HealingActionSpec{
			TargetResource: v1alpha1.TargetResource{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "test-deployment",
				Namespace:  "default",
			},
			Action: v1alpha1.HealingActionTemplate{
				Name: "scale-up",
				Type: "scale",
				ScaleAction: &v1alpha1.ScaleAction{
					Direction: "up",
					Replicas:  2,
				},
			},
		},
	}

	// Override scale executor with mock
	mockExecutor := &MockExecutor{
		DryRunFunc: func(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) (*controller.ActionResult, error) {
			return &controller.ActionResult{
				Success: true,
				Message: "Dry-run: Would scale deployment from 3 to 5 replicas",
				Changes: []v1alpha1.ResourceChange{
					{
						ResourceRef: "Deployment/default/test-deployment",
						ChangeType:  "update",
						Field:       "spec.replicas",
						OldValue:    "3",
						NewValue:    "5",
					},
				},
				Metrics: map[string]string{
					"dry_run": "true",
				},
			}, nil
		},
	}
	engine.RegisterExecutor("scale", mockExecutor)

	result, err := engine.DryRun(context.Background(), action)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Message, "Dry-run")
	assert.Equal(t, "true", result.Metrics["dry_run"])
}

func TestEngine_Rollback(t *testing.T) {
	// Create test configmap
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-configmap",
			Namespace: "default",
		},
		Data: map[string]string{
			"key1": "value1",
		},
	}

	// Create fake client
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(configMap).
		Build()

	// Create recorder
	recorder := NewInMemoryActionRecorder(1 * time.Hour)

	// Create engine
	engine := NewEngine(fakeClient, recorder)

	// Create action
	action := &v1alpha1.HealingAction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-patch-action",
			Namespace: "default",
		},
		Spec: v1alpha1.HealingActionSpec{
			TargetResource: v1alpha1.TargetResource{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       "test-configmap",
				Namespace:  "default",
			},
			Action: v1alpha1.HealingActionTemplate{
				Name: "patch-config",
				Type: "patch",
			},
		},
	}

	// Record a fake action with original state
	originalConfigMap := configMap.DeepCopy()
	result := &controller.ActionResult{
		Success: true,
		Changes: []v1alpha1.ResourceChange{
			{
				ResourceRef: "ConfigMap/default/test-configmap",
				ChangeType:  "update",
				Field:       "data.key1",
				OldValue:    "value1",
				NewValue:    "modified",
			},
		},
		StartTime: time.Now(),
	}
	
	err := recorder.RecordAction(context.Background(), action, result, originalConfigMap)
	require.NoError(t, err)

	// Modify the configmap
	configMap.Data["key1"] = "modified"
	err = fakeClient.Update(context.Background(), configMap)
	require.NoError(t, err)

	// Test rollback
	err = engine.Rollback(context.Background(), action)
	require.NoError(t, err)

	// Verify configmap was restored
	var restored corev1.ConfigMap
	err = fakeClient.Get(context.Background(), client.ObjectKey{
		Namespace: "default",
		Name:      "test-configmap",
	}, &restored)
	require.NoError(t, err)
	assert.Equal(t, "value1", restored.Data["key1"])
}

func TestEngine_ConcurrentActions(t *testing.T) {
	// Create fake client
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	// Create engine
	engine := NewEngine(fakeClient, nil)

	// Track an action
	action := &v1alpha1.HealingAction{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-concurrent-action",
		},
	}

	ctx := engine.trackAction(action)
	assert.NotNil(t, ctx)

	// Verify action is tracked
	activeActions := engine.GetActiveActions()
	assert.Contains(t, activeActions, action.Name)

	// Untrack the action
	engine.untrackAction(action.Name)

	// Verify action is no longer tracked
	activeActions = engine.GetActiveActions()
	assert.NotContains(t, activeActions, action.Name)
}

func TestEngine_GetTargetResource(t *testing.T) {
	// Create test pod
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

	// Create fake client
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod).
		Build()

	// Create engine
	engine := NewEngine(fakeClient, nil)

	// Test getting resource
	target := &v1alpha1.TargetResource{
		APIVersion: "v1",
		Kind:       "Pod",
		Name:       "test-pod",
		Namespace:  "default",
	}

	obj, err := engine.getTargetResource(context.Background(), target)
	require.NoError(t, err)
	assert.NotNil(t, obj)
	
	// Verify it's unstructured
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	require.True(t, ok)
	assert.Equal(t, "test-pod", unstructuredObj.GetName())
	assert.Equal(t, "default", unstructuredObj.GetNamespace())
	assert.Equal(t, schema.GroupVersionKind{Version: "v1", Kind: "Pod"}, unstructuredObj.GroupVersionKind())
}