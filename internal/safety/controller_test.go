package safety

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
	"github.com/kubeskippy/kubeskippy/internal/controller"
	"github.com/kubeskippy/kubeskippy/pkg/config"
)

// MockAuditLogger implements AuditLogger for testing
type MockAuditLogger struct {
	Actions     []AuditAction
	Validations []AuditValidation
	RateLimits  []AuditRateLimit
}

type AuditAction struct {
	Action  *v1alpha1.HealingAction
	Result  string
	Details map[string]interface{}
}

type AuditValidation struct {
	Action *v1alpha1.HealingAction
	Valid  bool
	Reason string
}

type AuditRateLimit struct {
	PolicyKey string
	Allowed   bool
	Current   int
	Limit     int
}

func (m *MockAuditLogger) LogAction(ctx context.Context, action *v1alpha1.HealingAction, result string, details map[string]interface{}) {
	m.Actions = append(m.Actions, AuditAction{Action: action, Result: result, Details: details})
}

func (m *MockAuditLogger) LogValidation(ctx context.Context, action *v1alpha1.HealingAction, valid bool, reason string) {
	m.Validations = append(m.Validations, AuditValidation{Action: action, Valid: valid, Reason: reason})
}

func (m *MockAuditLogger) LogRateLimit(ctx context.Context, policyKey string, allowed bool, current int, limit int) {
	m.RateLimits = append(m.RateLimits, AuditRateLimit{PolicyKey: policyKey, Allowed: allowed, Current: current, Limit: limit})
}

func TestController_ValidateAction(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	tests := []struct {
		name           string
		config         config.SafetyConfig
		action         *v1alpha1.HealingAction
		expectedValid  bool
		expectedReason string
		checkAudit     func(t *testing.T, logger *MockAuditLogger)
	}{
		{
			name: "valid action",
			config: config.SafetyConfig{
				DryRunMode: false,
			},
			action: &v1alpha1.HealingAction{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "default",
				},
				Spec: v1alpha1.HealingActionSpec{
					PolicyRef: v1alpha1.PolicyReference{
						Name:      "test-policy",
						Namespace: "default",
					},
					TargetResource: v1alpha1.TargetResource{
						Kind:      "Pod",
						Name:      "test-pod",
						Namespace: "default",
					},
					Action: v1alpha1.HealingActionTemplate{
						Name: "restart",
						Type: "restart",
					},
				},
			},
			expectedValid: true,
			checkAudit: func(t *testing.T, logger *MockAuditLogger) {
				require.Len(t, logger.Validations, 1)
				assert.True(t, logger.Validations[0].Valid)
			},
		},
		{
			name: "dry-run mode blocks non-dry-run actions",
			config: config.SafetyConfig{
				DryRunMode: true,
			},
			action: &v1alpha1.HealingAction{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "default",
				},
				Spec: v1alpha1.HealingActionSpec{
					PolicyRef: v1alpha1.PolicyReference{
						Name:      "test-policy",
						Namespace: "default",
					},
					TargetResource: v1alpha1.TargetResource{
						Kind:      "Pod",
						Name:      "test-pod",
						Namespace: "default",
					},
					Action: v1alpha1.HealingActionTemplate{
						Name: "restart",
						Type: "restart",
					},
					DryRun: false,
				},
			},
			expectedValid:  false,
			expectedReason: "System is in dry-run mode only",
		},
		{
			name: "protected namespace blocks action",
			config: config.SafetyConfig{
				ProtectedNamespaces: []string{"kube-system"},
			},
			action: &v1alpha1.HealingAction{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "default",
				},
				Spec: v1alpha1.HealingActionSpec{
					PolicyRef: v1alpha1.PolicyReference{
						Name:      "test-policy",
						Namespace: "default",
					},
					TargetResource: v1alpha1.TargetResource{
						Kind:      "Pod",
						Name:      "test-pod",
						Namespace: "kube-system",
					},
					Action: v1alpha1.HealingActionTemplate{
						Name: "restart",
						Type: "restart",
					},
				},
			},
			expectedValid:  false,
			expectedReason: "Resource is protected: namespace kube-system is protected",
		},
		{
			name:   "delete PV is blocked",
			config: config.SafetyConfig{},
			action: &v1alpha1.HealingAction{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "default",
				},
				Spec: v1alpha1.HealingActionSpec{
					PolicyRef: v1alpha1.PolicyReference{
						Name:      "test-policy",
						Namespace: "default",
					},
					TargetResource: v1alpha1.TargetResource{
						Kind: "PersistentVolume",
						Name: "test-pv",
					},
					Action: v1alpha1.HealingActionTemplate{
						Name: "delete",
						Type: "delete",
					},
				},
			},
			expectedValid:  false,
			expectedReason: "deleting PersistentVolumes is not allowed",
		},
		{
			name:   "scale action without config is invalid",
			config: config.SafetyConfig{},
			action: &v1alpha1.HealingAction{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "default",
				},
				Spec: v1alpha1.HealingActionSpec{
					PolicyRef: v1alpha1.PolicyReference{
						Name:      "test-policy",
						Namespace: "default",
					},
					TargetResource: v1alpha1.TargetResource{
						Kind:      "Deployment",
						Name:      "test-deployment",
						Namespace: "default",
					},
					Action: v1alpha1.HealingActionTemplate{
						Name: "scale",
						Type: "scale",
						// ScaleAction is nil
					},
				},
			},
			expectedValid:  false,
			expectedReason: "scale action missing configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithScheme(scheme).Build()
			store := NewInMemoryActionStore()
			auditLogger := &MockAuditLogger{}

			safetyCtrl := NewController(client, tt.config, store, auditLogger)

			result, err := safetyCtrl.ValidateAction(context.Background(), tt.action)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedValid, result.Valid)
			if tt.expectedReason != "" {
				assert.Equal(t, tt.expectedReason, result.Reason)
			}

			if tt.checkAudit != nil {
				tt.checkAudit(t, auditLogger)
			}
		})
	}
}

func TestController_CheckRateLimit(t *testing.T) {
	tests := []struct {
		name            string
		config          config.SafetyConfig
		policy          *v1alpha1.HealingPolicy
		existingActions []ActionRecord
		expectedAllowed bool
	}{
		{
			name: "under rate limit",
			config: config.SafetyConfig{
				MaxActionsPerHour: 10,
			},
			policy: &v1alpha1.HealingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-policy",
					Namespace: "default",
				},
			},
			existingActions: []ActionRecord{
				{PolicyKey: "default/test-policy", Timestamp: time.Now().Add(-30 * time.Minute)},
				{PolicyKey: "default/test-policy", Timestamp: time.Now().Add(-20 * time.Minute)},
			},
			expectedAllowed: true,
		},
		{
			name: "at rate limit",
			config: config.SafetyConfig{
				MaxActionsPerHour: 3,
			},
			policy: &v1alpha1.HealingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-policy",
					Namespace: "default",
				},
			},
			existingActions: []ActionRecord{
				{PolicyKey: "default/test-policy", Timestamp: time.Now().Add(-30 * time.Minute)},
				{PolicyKey: "default/test-policy", Timestamp: time.Now().Add(-20 * time.Minute)},
				{PolicyKey: "default/test-policy", Timestamp: time.Now().Add(-10 * time.Minute)},
			},
			expectedAllowed: false,
		},
		{
			name: "policy overrides global limit",
			config: config.SafetyConfig{
				MaxActionsPerHour: 10,
			},
			policy: &v1alpha1.HealingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-policy",
					Namespace: "default",
				},
				Spec: v1alpha1.HealingPolicySpec{
					SafetyRules: v1alpha1.SafetyRules{
						MaxActionsPerHour: 2,
					},
				},
			},
			existingActions: []ActionRecord{
				{PolicyKey: "default/test-policy", Timestamp: time.Now().Add(-30 * time.Minute)},
				{PolicyKey: "default/test-policy", Timestamp: time.Now().Add(-20 * time.Minute)},
			},
			expectedAllowed: false,
		},
		{
			name: "old actions don't count",
			config: config.SafetyConfig{
				MaxActionsPerHour: 2,
			},
			policy: &v1alpha1.HealingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-policy",
					Namespace: "default",
				},
			},
			existingActions: []ActionRecord{
				{PolicyKey: "default/test-policy", Timestamp: time.Now().Add(-2 * time.Hour)},
				{PolicyKey: "default/test-policy", Timestamp: time.Now().Add(-90 * time.Minute)},
				{PolicyKey: "default/test-policy", Timestamp: time.Now().Add(-30 * time.Minute)},
			},
			expectedAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = v1alpha1.AddToScheme(scheme)

			client := fake.NewClientBuilder().WithScheme(scheme).Build()
			store := NewInMemoryActionStore()
			auditLogger := &MockAuditLogger{}

			// Populate store with existing actions
			for _, record := range tt.existingActions {
				err := store.RecordAction(context.Background(), record)
				require.NoError(t, err)
			}

			safetyCtrl := NewController(client, tt.config, store, auditLogger)

			allowed, err := safetyCtrl.CheckRateLimit(context.Background(), tt.policy)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedAllowed, allowed)

			// Check audit log
			require.Len(t, auditLogger.RateLimits, 1)
			assert.Equal(t, tt.expectedAllowed, auditLogger.RateLimits[0].Allowed)
		})
	}
}

func TestController_IsProtectedResource(t *testing.T) {
	tests := []struct {
		name              string
		config            config.SafetyConfig
		resource          *GenericResource
		expectedProtected bool
		expectedReason    string
	}{
		{
			name:   "not protected",
			config: config.SafetyConfig{},
			resource: &GenericResource{
				namespace: "default",
				name:      "test-pod",
			},
			expectedProtected: false,
		},
		{
			name: "protected namespace",
			config: config.SafetyConfig{
				ProtectedNamespaces: []string{"kube-system", "kube-public"},
			},
			resource: &GenericResource{
				namespace: "kube-system",
				name:      "test-pod",
			},
			expectedProtected: true,
			expectedReason:    "namespace kube-system is protected",
		},
		{
			name: "protected label",
			config: config.SafetyConfig{
				ProtectedLabels: map[string]string{
					"critical": "true",
				},
			},
			resource: &GenericResource{
				namespace: "default",
				name:      "test-pod",
				labels: map[string]string{
					"app":      "test",
					"critical": "true",
				},
			},
			expectedProtected: true,
			expectedReason:    "has protected label critical=true",
		},
		{
			name:   "protected annotation",
			config: config.SafetyConfig{},
			resource: &GenericResource{
				namespace: "default",
				name:      "test-pod",
				annotations: map[string]string{
					controller.AnnotationProtected: "true",
				},
			},
			expectedProtected: true,
			expectedReason:    "has protected annotation",
		},
		{
			name:   "healing disabled",
			config: config.SafetyConfig{},
			resource: &GenericResource{
				namespace: "default",
				name:      "test-pod",
				annotations: map[string]string{
					controller.AnnotationHealingDisabled: "true",
				},
			},
			expectedProtected: true,
			expectedReason:    "healing is disabled via annotation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			client := fake.NewClientBuilder().WithScheme(scheme).Build()

			safetyCtrl := NewController(client, tt.config, nil, nil)

			protected, reason := safetyCtrl.IsProtectedResource(tt.resource)
			assert.Equal(t, tt.expectedProtected, protected)
			if tt.expectedReason != "" {
				assert.Equal(t, tt.expectedReason, reason)
			}
		})
	}
}

func TestController_RecordAction(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	store := NewInMemoryActionStore()
	auditLogger := &MockAuditLogger{}
	config := config.SafetyConfig{
		CircuitBreaker: config.CircuitBreakerConfig{
			FailureThreshold: 3,
			SuccessThreshold: 2,
			Timeout:          5 * time.Minute,
		},
	}

	safetyCtrl := NewController(client, config, store, auditLogger)

	action := &v1alpha1.HealingAction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-action",
			Namespace: "default",
		},
		Spec: v1alpha1.HealingActionSpec{
			PolicyRef: v1alpha1.PolicyReference{
				Name:      "test-policy",
				Namespace: "default",
			},
			TargetResource: v1alpha1.TargetResource{
				Kind:      "Pod",
				Name:      "test-pod",
				Namespace: "default",
			},
			Action: v1alpha1.HealingActionTemplate{
				Name: "restart",
				Type: "restart",
			},
			DryRun: true,
		},
		Status: v1alpha1.HealingActionStatus{
			Approval: &v1alpha1.ApprovalStatus{
				ApprovedBy: "user@example.com",
			},
		},
	}

	result := &controller.ActionResult{
		Success:   true,
		Message:   "Action completed",
		StartTime: time.Now().Add(-1 * time.Minute),
		EndTime:   time.Now(),
	}

	safetyCtrl.RecordAction(context.Background(), action, result)

	// Verify action was recorded
	records, err := store.GetRecentActions(context.Background(), "default/test-policy", 10)
	require.NoError(t, err)
	require.Len(t, records, 1)

	record := records[0]
	assert.Equal(t, "default/test-policy", record.PolicyKey)
	assert.Equal(t, "restart", record.ActionName)
	assert.Equal(t, "restart", record.ActionType)
	assert.Equal(t, "Pod/default/test-pod", record.TargetKey)
	assert.True(t, record.Success)
	assert.True(t, record.DryRun)
	assert.Equal(t, "user@example.com", record.ApprovedBy)
	assert.Greater(t, record.DurationMS, int64(0))

	// Verify audit log
	require.Len(t, auditLogger.Actions, 1)
	assert.Equal(t, "success=true", auditLogger.Actions[0].Result)
}

func TestInMemoryActionStore(t *testing.T) {
	store := NewInMemoryActionStore()
	ctx := context.Background()

	// Test recording actions
	now := time.Now()
	records := []ActionRecord{
		{
			PolicyKey:  "default/policy1",
			ActionName: "restart",
			Timestamp:  now.Add(-3 * time.Hour),
		},
		{
			PolicyKey:  "default/policy1",
			ActionName: "scale",
			Timestamp:  now.Add(-2 * time.Hour),
		},
		{
			PolicyKey:  "default/policy1",
			ActionName: "restart",
			Timestamp:  now.Add(-30 * time.Minute),
		},
		{
			PolicyKey:  "default/policy2",
			ActionName: "patch",
			Timestamp:  now.Add(-10 * time.Minute),
		},
	}

	for _, record := range records {
		err := store.RecordAction(ctx, record)
		require.NoError(t, err)
	}

	// Test GetActionCount
	count, err := store.GetActionCount(ctx, "default/policy1", now.Add(-1*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	count, err = store.GetActionCount(ctx, "default/policy1", now.Add(-4*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// Test GetRecentActions
	recent, err := store.GetRecentActions(ctx, "default/policy1", 2)
	require.NoError(t, err)
	require.Len(t, recent, 2)
	assert.Equal(t, "restart", recent[0].ActionName)
	assert.Equal(t, "scale", recent[1].ActionName)

	// Test GetLastAction
	last, err := store.GetLastAction(ctx, "default/policy1")
	require.NoError(t, err)
	require.NotNil(t, last)
	assert.Equal(t, "restart", last.ActionName)

	// Test CleanupOldRecords
	err = store.CleanupOldRecords(ctx, now.Add(-90*time.Minute))
	require.NoError(t, err)

	count, err = store.GetActionCount(ctx, "default/policy1", now.Add(-4*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 1, count) // Only the most recent action remains
}

func TestCircuitBreakerIntegration(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	store := NewInMemoryActionStore()
	config := config.SafetyConfig{
		CircuitBreaker: config.CircuitBreakerConfig{
			FailureThreshold: 2,
			SuccessThreshold: 1,
			Timeout:          100 * time.Millisecond,
		},
	}

	safetyCtrl := NewController(client, config, store, nil)

	action := &v1alpha1.HealingAction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-action",
			Namespace: "default",
		},
		Spec: v1alpha1.HealingActionSpec{
			PolicyRef: v1alpha1.PolicyReference{
				Name:      "test-policy",
				Namespace: "default",
			},
			TargetResource: v1alpha1.TargetResource{
				Kind:      "Pod",
				Name:      "test-pod",
				Namespace: "default",
			},
			Action: v1alpha1.HealingActionTemplate{
				Name: "restart",
				Type: "restart",
			},
		},
	}

	// First validation should succeed
	result, err := safetyCtrl.ValidateAction(context.Background(), action)
	require.NoError(t, err)
	assert.True(t, result.Valid)

	// Record failures to trip circuit breaker
	for i := 0; i < 2; i++ {
		safetyCtrl.RecordAction(context.Background(), action, &controller.ActionResult{
			Success:   false,
			Error:     fmt.Errorf("test error"),
			StartTime: time.Now(),
			EndTime:   time.Now(),
		})
	}

	// Circuit breaker should now be open
	result, err = safetyCtrl.ValidateAction(context.Background(), action)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Reason, "Circuit breaker is open")

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Should be able to validate again (half-open state)
	result, err = safetyCtrl.ValidateAction(context.Background(), action)
	require.NoError(t, err)
	assert.True(t, result.Valid)
}
