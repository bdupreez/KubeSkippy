package remediation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
	"github.com/kubeskippy/kubeskippy/internal/controller"
)

// InMemoryActionRecorder implements ActionRecorder using in-memory storage
type InMemoryActionRecorder struct {
	mu      sync.RWMutex
	history map[string]*ActionHistory
	maxAge  time.Duration
}

// NewInMemoryActionRecorder creates a new in-memory action recorder
func NewInMemoryActionRecorder(maxAge time.Duration) *InMemoryActionRecorder {
	return &InMemoryActionRecorder{
		history: make(map[string]*ActionHistory),
		maxAge:  maxAge,
	}
}

// RecordAction records an action execution for audit and rollback
func (r *InMemoryActionRecorder) RecordAction(ctx context.Context, action *v1alpha1.HealingAction, result *controller.ActionResult, originalState runtime.Object) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Create history entry
	history := &ActionHistory{
		ActionName:    action.Name,
		OriginalState: originalState,
		Changes:       result.Changes,
		ExecutedAt:    result.StartTime,
	}

	// Store in history
	r.history[action.Name] = history

	log.FromContext(ctx).Info("Recorded action for rollback",
		"action", action.Name,
		"changes", len(result.Changes))

	return nil
}

// GetActionHistory retrieves the history for a specific action
func (r *InMemoryActionRecorder) GetActionHistory(ctx context.Context, actionName string) (*ActionHistory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	history, exists := r.history[actionName]
	if !exists {
		return nil, fmt.Errorf("no history found for action %s", actionName)
	}

	// Check if history is too old
	if time.Since(history.ExecutedAt) > r.maxAge {
		return nil, fmt.Errorf("action history for %s is too old (executed at %v)", actionName, history.ExecutedAt)
	}

	return history, nil
}

// CleanupOldHistory removes old action history entries
func (r *InMemoryActionRecorder) CleanupOldHistory(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := time.Now().Add(-r.maxAge)
	deleted := 0

	for name, history := range r.history {
		if history.ExecutedAt.Before(cutoff) {
			delete(r.history, name)
			deleted++
		}
	}

	if deleted > 0 {
		log.FromContext(ctx).Info("Cleaned up old action history",
			"deleted", deleted,
			"remaining", len(r.history))
	}
}

// StartCleanupLoop starts a background loop to clean up old history
func (r *InMemoryActionRecorder) StartCleanupLoop(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.CleanupOldHistory(ctx)
			}
		}
	}()
}

// GetAllHistory returns all recorded action history (for debugging/monitoring)
func (r *InMemoryActionRecorder) GetAllHistory() map[string]*ActionHistory {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create a copy to avoid race conditions
	result := make(map[string]*ActionHistory)
	for k, v := range r.history {
		result[k] = v
	}

	return result
}

// GetHistoryCount returns the number of recorded actions
func (r *InMemoryActionRecorder) GetHistoryCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.history)
}
