package remediation

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
	kubetypes "github.com/kubeskippy/kubeskippy/internal/types"
)

func TestInMemoryActionRecorder(t *testing.T) {
	recorder := NewInMemoryActionRecorder(1 * time.Hour)

	// Create test action
	action := &v1alpha1.HealingAction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-action",
			Namespace: "default",
		},
	}

	// Create test result
	result := &kubetypes.ActionResult{
		Success:   true,
		Message:   "Test action completed",
		StartTime: time.Now(),
		Changes: []v1alpha1.ResourceChange{
			{
				ResourceRef: "Pod/default/test-pod",
				ChangeType:  "update",
				Field:       "spec.replicas",
				OldValue:    "1",
				NewValue:    "3",
			},
		},
	}

	// Create original state
	originalState := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	// Test recording action
	t.Run("record action", func(t *testing.T) {
		err := recorder.RecordAction(context.Background(), action, result, originalState)
		require.NoError(t, err)

		// Verify action was recorded
		history, err := recorder.GetActionHistory(context.Background(), action.Name)
		require.NoError(t, err)
		assert.NotNil(t, history)
		assert.Equal(t, action.Name, history.ActionName)
		assert.NotNil(t, history.OriginalState)
		assert.Len(t, history.Changes, 1)
	})

	// Test getting non-existent action
	t.Run("non-existent action", func(t *testing.T) {
		history, err := recorder.GetActionHistory(context.Background(), "non-existent")
		assert.Error(t, err)
		assert.Nil(t, history)
	})

	// Test overwriting action
	t.Run("overwrite action", func(t *testing.T) {
		// Record same action again with different result
		newResult := &kubetypes.ActionResult{
			Success:   false,
			Message:   "Failed action",
			StartTime: time.Now(),
		}

		err := recorder.RecordAction(context.Background(), action, newResult, nil)
		require.NoError(t, err)

		// Verify new history
		history, err := recorder.GetActionHistory(context.Background(), action.Name)
		require.NoError(t, err)
		assert.Nil(t, history.OriginalState) // Should be nil from second recording
		assert.Empty(t, history.Changes)
	})

	// Test history count
	t.Run("history count", func(t *testing.T) {
		count := recorder.GetHistoryCount()
		assert.Equal(t, 1, count) // Should only have one action

		// Add another action
		action2 := &v1alpha1.HealingAction{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-action-2",
			},
		}
		err := recorder.RecordAction(context.Background(), action2, result, nil)
		require.NoError(t, err)

		count = recorder.GetHistoryCount()
		assert.Equal(t, 2, count)
	})

	// Test getting all history
	t.Run("get all history", func(t *testing.T) {
		allHistory := recorder.GetAllHistory()
		assert.Len(t, allHistory, 2)
		assert.Contains(t, allHistory, "test-action")
		assert.Contains(t, allHistory, "test-action-2")
	})
}

func TestInMemoryActionRecorder_Cleanup(t *testing.T) {
	// Create recorder with short max age
	recorder := NewInMemoryActionRecorder(100 * time.Millisecond)

	// Record an action
	action := &v1alpha1.HealingAction{
		ObjectMeta: metav1.ObjectMeta{
			Name: "old-action",
		},
	}

	result := &kubetypes.ActionResult{
		Success:   true,
		StartTime: time.Now().Add(-200 * time.Millisecond), // Old timestamp
	}

	err := recorder.RecordAction(context.Background(), action, result, nil)
	require.NoError(t, err)

	// Verify action exists
	history, err := recorder.GetActionHistory(context.Background(), action.Name)
	assert.Error(t, err) // Should error because it's too old
	assert.Nil(t, history)

	// Test cleanup
	recorder.CleanupOldHistory(context.Background())

	// Verify action was cleaned up
	count := recorder.GetHistoryCount()
	assert.Equal(t, 0, count)
}

func TestInMemoryActionRecorder_CleanupLoop(t *testing.T) {
	// Create recorder with short max age
	recorder := NewInMemoryActionRecorder(50 * time.Millisecond)

	// Start cleanup loop with short interval
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	recorder.StartCleanupLoop(ctx, 25*time.Millisecond)

	// Record an action
	action := &v1alpha1.HealingAction{
		ObjectMeta: metav1.ObjectMeta{
			Name: "temp-action",
		},
	}

	// Manually set old timestamp in history
	recorder.history[action.Name] = &ActionHistory{
		ActionName: action.Name,
		ExecutedAt: time.Now().Add(-100 * time.Millisecond),
	}

	// Verify action exists initially
	assert.Equal(t, 1, recorder.GetHistoryCount())

	// Wait for cleanup to run
	time.Sleep(100 * time.Millisecond)

	// Verify action was cleaned up
	assert.Equal(t, 0, recorder.GetHistoryCount())
}

func TestActionHistory(t *testing.T) {
	// Test action history structure
	history := &ActionHistory{
		ActionName: "test-action",
		OriginalState: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pod",
			},
		},
		Changes: []v1alpha1.ResourceChange{
			{
				ResourceRef: "Pod/default/test-pod",
				ChangeType:  "update",
				Field:       "spec.replicas",
				OldValue:    "1",
				NewValue:    "3",
			},
		},
		ExecutedAt: time.Now(),
	}

	assert.Equal(t, "test-action", history.ActionName)
	assert.NotNil(t, history.OriginalState)
	assert.Len(t, history.Changes, 1)
	assert.False(t, history.ExecutedAt.IsZero())
}
