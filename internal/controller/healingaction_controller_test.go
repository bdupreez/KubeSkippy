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

func TestHealingActionReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	tests := []struct {
		name            string
		action          *v1alpha1.HealingAction
		remediationFunc func(ctx context.Context, action *v1alpha1.HealingAction) (*ActionResult, error)
		validateFunc    func(ctx context.Context, action *v1alpha1.HealingAction) (*ValidationResult, error)
		expectedPhase   string
		expectedResult  reconcile.Result
		expectedError   bool
	}{
		{
			name: "pending action without approval required",
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
				},
				Status: v1alpha1.HealingActionStatus{
					Phase: v1alpha1.HealingActionPhasePending,
				},
			},
			expectedPhase:  v1alpha1.HealingActionPhaseApproved,
			expectedResult: reconcile.Result{Requeue: true},
		},
		{
			name: "pending action with approval required",
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
					Phase: v1alpha1.HealingActionPhasePending,
				},
			},
			expectedPhase:  v1alpha1.HealingActionPhasePending,
			expectedResult: reconcile.Result{RequeueAfter: 30 * time.Second},
		},
		{
			name: "approved action - validation fails",
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
			expectedPhase:  v1alpha1.HealingActionPhaseFailed,
			expectedResult: reconcile.Result{},
		},
		{
			name: "in-progress action - success",
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
					Changes: []v1alpha1.ResourceChange{
						{
							Field:     "status.phase",
							OldValue:  "Running",
							NewValue:  "Pending",
							Timestamp: &metav1.Time{Time: time.Now()},
						},
					},
				}, nil
			},
			expectedPhase:  v1alpha1.HealingActionPhaseSucceeded,
			expectedResult: reconcile.Result{},
		},
		{
			name: "in-progress action - failure with retry",
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
						BackoffDelay:      metav1.Duration{Duration: 30 * time.Second},
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
				return nil, errors.New("temporary failure")
			},
			expectedPhase:  v1alpha1.HealingActionPhaseInProgress,
			expectedResult: reconcile.Result{RequeueAfter: 30 * time.Second},
		},
		{
			name: "in-progress action - max retries exceeded",
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
						BackoffDelay:      metav1.Duration{Duration: 30 * time.Second},
						BackoffMultiplier: 2.0,
					},
				},
				Status: v1alpha1.HealingActionStatus{
					Phase:     v1alpha1.HealingActionPhaseInProgress,
					StartTime: &metav1.Time{Time: time.Now()},
					Attempts:  2, // Will be incremented to 3
				},
			},
			remediationFunc: func(ctx context.Context, action *v1alpha1.HealingAction) (*ActionResult, error) {
				return nil, errors.New("permanent failure")
			},
			expectedPhase:  v1alpha1.HealingActionPhaseFailed,
			expectedResult: reconcile.Result{},
		},
		{
			name: "dry-run action",
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
			expectedPhase:  v1alpha1.HealingActionPhaseSucceeded,
			expectedResult: reconcile.Result{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.action).
				Build()

			// Create mocks
			remediationEngine := &MockRemediationEngine{
				ExecuteActionFunc: tt.remediationFunc,
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

			// Reconcile
			result, err := r.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.action.Name,
					Namespace: tt.action.Namespace,
				},
			})

			// Check result
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedResult, result)

			// Check phase if action still exists
			updatedAction := &v1alpha1.HealingAction{}
			err = fakeClient.Get(context.Background(), types.NamespacedName{
				Name:      tt.action.Name,
				Namespace: tt.action.Namespace,
			}, updatedAction)

			if err == nil {
				assert.Equal(t, tt.expectedPhase, updatedAction.Status.Phase)
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
		Build()

	r := &HealingActionReconciler{
		Client:            fakeClient,
		Scheme:            scheme,
		Config:            config.NewDefaultConfig(),
		RemediationEngine: &MockRemediationEngine{},
		SafetyController:  &MockSafetyController{},
	}

	// Reconcile
	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      action.Name,
			Namespace: action.Namespace,
		},
	})
	require.NoError(t, err)

	// Check that action failed due to timeout
	updatedAction := &v1alpha1.HealingAction{}
	err = fakeClient.Get(context.Background(), types.NamespacedName{
		Name:      action.Name,
		Namespace: action.Namespace,
	}, updatedAction)
	require.NoError(t, err)

	assert.Equal(t, v1alpha1.HealingActionPhaseFailed, updatedAction.Status.Phase)
	assert.NotNil(t, updatedAction.Status.Result)
	assert.False(t, updatedAction.Status.Result.Success)
	assert.Contains(t, updatedAction.Status.Result.Message, "timed out")
}
