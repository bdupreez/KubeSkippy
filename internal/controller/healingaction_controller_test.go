package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
	"github.com/kubeskippy/kubeskippy/pkg/config"
)

// MockRemediationEngine implements RemediationEngine interface for testing
type MockRemediationEngine struct {
	ExecuteActionFunc     func(ctx context.Context, action *v1alpha1.HealingAction) (*ActionResult, error)
	DryRunFunc            func(ctx context.Context, action *v1alpha1.HealingAction) (*ActionResult, error)
	RollbackFunc          func(ctx context.Context, action *v1alpha1.HealingAction) error
	GetActionExecutorFunc func(actionType string) (ActionExecutor, error)
}

func (m *MockRemediationEngine) ExecuteAction(ctx context.Context, action *v1alpha1.HealingAction) (*ActionResult, error) {
	if m.ExecuteActionFunc != nil {
		return m.ExecuteActionFunc(ctx, action)
	}
	return &ActionResult{Success: true, Message: "Mock success"}, nil
}

func (m *MockRemediationEngine) DryRun(ctx context.Context, action *v1alpha1.HealingAction) (*ActionResult, error) {
	if m.DryRunFunc != nil {
		return m.DryRunFunc(ctx, action)
	}
	return &ActionResult{Success: true, Message: "Mock dry-run success"}, nil
}

func (m *MockRemediationEngine) Rollback(ctx context.Context, action *v1alpha1.HealingAction) error {
	if m.RollbackFunc != nil {
		return m.RollbackFunc(ctx, action)
	}
	return nil
}

func (m *MockRemediationEngine) GetActionExecutor(actionType string) (ActionExecutor, error) {
	if m.GetActionExecutorFunc != nil {
		return m.GetActionExecutorFunc(actionType)
	}
	return nil, nil
}


// reconcileUntilPhase simulates multiple reconciliations until the action reaches the expected phase or a terminal state
func reconcileUntilPhase(t *testing.T, r *HealingActionReconciler, req reconcile.Request, expectedPhase string, maxIterations int) (*v1alpha1.HealingAction, error) {
	var lastPhase string
	for i := 0; i < maxIterations; i++ {
		// Get current action state
		currentAction := &v1alpha1.HealingAction{}
		err := r.Client.Get(context.Background(), req.NamespacedName, currentAction)
		if err != nil {
			return nil, err
		}
		
		currentPhase := currentAction.Status.Phase
		if currentPhase == "" {
			currentPhase = "<empty>"
		}
		
		// Reconcile
		result, err := r.Reconcile(context.Background(), req)
		if err != nil {
			return nil, err
		}

		// Get the updated action
		action := &v1alpha1.HealingAction{}
		err = r.Client.Get(context.Background(), req.NamespacedName, action)
		if err != nil {
			return nil, err
		}

		newPhase := action.Status.Phase
		if newPhase == "" {
			newPhase = "<empty>"
		}
		
		t.Logf("Iteration %d: %s -> %s (Expected: %s, Result: %+v)", 
			i, currentPhase, newPhase, expectedPhase, result)

		// Check if we've reached the expected phase or a terminal state
		if action.Status.Phase == expectedPhase ||
			action.Status.Phase == v1alpha1.HealingActionPhaseSucceeded ||
			action.Status.Phase == v1alpha1.HealingActionPhaseFailed ||
			action.Status.Phase == v1alpha1.HealingActionPhaseCancelled {
			return action, nil
		}
		
		// If phase hasn't changed after 2 iterations, something's wrong
		if action.Status.Phase == lastPhase && i > 1 {
			t.Logf("WARNING: Phase stuck at %s after %d iterations", action.Status.Phase, i+1)
			break
		}
		lastPhase = action.Status.Phase
	}

	action := &v1alpha1.HealingAction{}
	err := r.Client.Get(context.Background(), req.NamespacedName, action)
	return action, err
}

func TestHealingActionReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	tests := []struct {
		name            string
		action          *v1alpha1.HealingAction
		remediationFunc func(ctx context.Context, action *v1alpha1.HealingAction) (*ActionResult, error)
		validateFunc    func(ctx context.Context, action *v1alpha1.HealingAction) (*ValidationResult, error)
		expectedPhase   string
		maxReconciles   int
		setupFunc       func(t *testing.T, action *v1alpha1.HealingAction)
	}{
		{
			name: "pending action without approval required transitions to succeeded",
			action: &v1alpha1.HealingAction{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "default",
				},
				Spec: v1alpha1.HealingActionSpec{
					ApprovalRequired: false,
					Action: v1alpha1.HealingActionTemplate{
						Name: "restart",
						Type: "restart",
					},
					Timeout: metav1.Duration{Duration: 10 * time.Minute},
				},
				Status: v1alpha1.HealingActionStatus{
					// Start with empty phase for new actions
				},
			},
			remediationFunc: func(ctx context.Context, action *v1alpha1.HealingAction) (*ActionResult, error) {
				return &ActionResult{
					Success: true,
					Message: "Action completed successfully",
				}, nil
			},
			expectedPhase: v1alpha1.HealingActionPhaseSucceeded,
			maxReconciles: 10,
		},
		{
			name: "pending action with approval required stays pending",
			action: &v1alpha1.HealingAction{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "default",
				},
				Spec: v1alpha1.HealingActionSpec{
					ApprovalRequired: true,
					Action: v1alpha1.HealingActionTemplate{
						Name: "restart",
						Type: "restart",
					},
				},
				Status: v1alpha1.HealingActionStatus{
					// Start with empty phase for new actions
				},
			},
			expectedPhase: v1alpha1.HealingActionPhasePending,
			maxReconciles: 2,
		},
		{
			name: "approved action with validation failure transitions to failed",
			action: &v1alpha1.HealingAction{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "default",
				},
				Spec: v1alpha1.HealingActionSpec{
					Action: v1alpha1.HealingActionTemplate{
						Name: "restart",
						Type: "restart",
					},
				},
				Status: v1alpha1.HealingActionStatus{
					Phase: v1alpha1.HealingActionPhaseApproved,
				},
			},
			validateFunc: func(ctx context.Context, action *v1alpha1.HealingAction) (*ValidationResult, error) {
				return &ValidationResult{
					Valid:  false,
					Reason: "Resource is protected",
				}, nil
			},
			expectedPhase: v1alpha1.HealingActionPhaseFailed,
			maxReconciles: 5,
		},
		{
			name: "in-progress action executes successfully",
			action: &v1alpha1.HealingAction{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "default",
				},
				Spec: v1alpha1.HealingActionSpec{
					Action: v1alpha1.HealingActionTemplate{
						Name: "restart",
						Type: "restart",
					},
					Timeout: metav1.Duration{Duration: 10 * time.Minute},
				},
				Status: v1alpha1.HealingActionStatus{
					Phase:     v1alpha1.HealingActionPhaseInProgress,
					StartTime: &metav1.Time{Time: time.Now()},
				},
			},
			remediationFunc: func(ctx context.Context, action *v1alpha1.HealingAction) (*ActionResult, error) {
				return &ActionResult{
					Success: true,
					Message: "Action completed successfully",
				}, nil
			},
			expectedPhase: v1alpha1.HealingActionPhaseSucceeded,
			maxReconciles: 5,
		},
		{
			name: "in-progress action with failure and retry",
			action: &v1alpha1.HealingAction{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "default",
				},
				Spec: v1alpha1.HealingActionSpec{
					Action: v1alpha1.HealingActionTemplate{
						Name: "restart",
						Type: "restart",
					},
					Timeout: metav1.Duration{Duration: 10 * time.Minute},
					RetryPolicy: &v1alpha1.RetryPolicy{
						MaxAttempts:       3,
						BackoffDelay:      metav1.Duration{Duration: 1 * time.Second},
						BackoffMultiplier: 2.0,
					},
				},
				Status: v1alpha1.HealingActionStatus{
					Phase:     v1alpha1.HealingActionPhaseInProgress,
					StartTime: &metav1.Time{Time: time.Now()},
					Attempts:  0,
				},
			},
			remediationFunc: func(ctx context.Context, action *v1alpha1.HealingAction) (*ActionResult, error) {
				// Fail first 2 attempts, succeed on third
				if action.Status.Attempts < 2 {
					return nil, errors.New("temporary failure")
				}
				return &ActionResult{
					Success: true,
					Message: "Action completed successfully",
				}, nil
			},
			expectedPhase: v1alpha1.HealingActionPhaseSucceeded,
			maxReconciles: 10,
		},
		{
			name: "in-progress action with max retries exceeded",
			action: &v1alpha1.HealingAction{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "default",
				},
				Spec: v1alpha1.HealingActionSpec{
					Action: v1alpha1.HealingActionTemplate{
						Name: "restart",
						Type: "restart",
					},
					Timeout: metav1.Duration{Duration: 10 * time.Minute},
					RetryPolicy: &v1alpha1.RetryPolicy{
						MaxAttempts:       2,
						BackoffDelay:      metav1.Duration{Duration: 1 * time.Second},
						BackoffMultiplier: 2.0,
					},
				},
				Status: v1alpha1.HealingActionStatus{
					Phase:     v1alpha1.HealingActionPhaseInProgress,
					StartTime: &metav1.Time{Time: time.Now()},
					Attempts:  0,
				},
			},
			remediationFunc: func(ctx context.Context, action *v1alpha1.HealingAction) (*ActionResult, error) {
				return nil, errors.New("permanent failure")
			},
			expectedPhase: v1alpha1.HealingActionPhaseFailed,
			maxReconciles: 10,
		},
		{
			name: "dry-run action executes successfully",
			action: &v1alpha1.HealingAction{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "default",
				},
				Spec: v1alpha1.HealingActionSpec{
					Action: v1alpha1.HealingActionTemplate{
						Name: "restart",
						Type: "restart",
					},
					DryRun:  true,
					Timeout: metav1.Duration{Duration: 10 * time.Minute},
				},
				Status: v1alpha1.HealingActionStatus{
					Phase:     v1alpha1.HealingActionPhaseInProgress,
					StartTime: &metav1.Time{Time: time.Now()},
				},
			},
			remediationFunc: func(ctx context.Context, action *v1alpha1.HealingAction) (*ActionResult, error) {
				return &ActionResult{
					Success: true,
					Message: "Dry-run completed successfully",
				}, nil
			},
			expectedPhase: v1alpha1.HealingActionPhaseSucceeded,
			maxReconciles: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.action).
				WithStatusSubresource(tt.action).
				Build()

			// Create mocks
			remediationEngine := &MockRemediationEngine{
				ExecuteActionFunc: tt.remediationFunc,
				DryRunFunc:        tt.remediationFunc, // Use same function for dry-run
			}

			safetyController := &MockSafetyController{
				ValidateActionFunc: tt.validateFunc,
			}

			// Create reconciler
			r := &HealingActionReconciler{
				Client:            fakeClient,
				Scheme:            scheme,
				Config:            config.NewDefaultConfig(),
				RemediationEngine: remediationEngine,
				SafetyController:  safetyController,
			}

			// Run setup if provided
			if tt.setupFunc != nil {
				tt.setupFunc(t, tt.action)
			}

			// Reconcile until expected phase
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.action.Name,
					Namespace: tt.action.Namespace,
				},
			}

			finalAction, err := reconcileUntilPhase(t, r, req, tt.expectedPhase, tt.maxReconciles)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedPhase, finalAction.Status.Phase)

			// Additional assertions based on phase
			switch finalAction.Status.Phase {
			case v1alpha1.HealingActionPhaseSucceeded:
				assert.NotNil(t, finalAction.Status.Result)
				assert.True(t, finalAction.Status.Result.Success)
				assert.NotNil(t, finalAction.Status.CompletionTime)
			case v1alpha1.HealingActionPhaseFailed:
				assert.NotNil(t, finalAction.Status.Result)
				assert.False(t, finalAction.Status.Result.Success)
				assert.NotNil(t, finalAction.Status.CompletionTime)
			case v1alpha1.HealingActionPhasePending:
				if tt.action.Spec.ApprovalRequired {
					assert.NotNil(t, finalAction.Status.Approval)
					assert.True(t, finalAction.Status.Approval.Required)
					assert.False(t, finalAction.Status.Approval.Approved)
				}
			}
		})
	}
}

func TestHealingActionReconciler_handleTimeout(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	// Create an action that has timed out
	action := &v1alpha1.HealingAction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "timeout-action",
			Namespace: "default",
		},
		Spec: v1alpha1.HealingActionSpec{
			Action: v1alpha1.HealingActionTemplate{
				Name: "restart",
				Type: "restart",
			},
			Timeout: metav1.Duration{Duration: 1 * time.Minute},
		},
		Status: v1alpha1.HealingActionStatus{
			Phase:     v1alpha1.HealingActionPhaseInProgress,
			StartTime: &metav1.Time{Time: time.Now().Add(-2 * time.Minute)}, // Started 2 minutes ago
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(action).
		WithStatusSubresource(action).
		Build()

	r := &HealingActionReconciler{
		Client:            fakeClient,
		Scheme:            scheme,
		Config:            config.NewDefaultConfig(),
		RemediationEngine: &MockRemediationEngine{},
		SafetyController:  &MockSafetyController{},
	}

	// Reconcile
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      action.Name,
			Namespace: action.Namespace,
		},
	}

	finalAction, err := reconcileUntilPhase(t, r, req, v1alpha1.HealingActionPhaseFailed, 5)
	require.NoError(t, err)

	// Check that action failed due to timeout
	assert.Equal(t, v1alpha1.HealingActionPhaseFailed, finalAction.Status.Phase)
	assert.NotNil(t, finalAction.Status.Result)
	assert.False(t, finalAction.Status.Result.Success)
	assert.Contains(t, finalAction.Status.Result.Message, "timed out")
	assert.NotNil(t, finalAction.Status.CompletionTime)
}