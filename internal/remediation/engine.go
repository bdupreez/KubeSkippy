package remediation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
	"github.com/kubeskippy/kubeskippy/internal/controller"
)

// Engine implements the RemediationEngine interface
type Engine struct {
	client    client.Client
	executors map[string]controller.ActionExecutor
	recorder  ActionRecorder
	mu        sync.RWMutex

	// For tracking in-flight actions
	activeActions map[string]*ActionContext
	actionsMu     sync.RWMutex
}

// ActionContext tracks the state of an in-flight action
type ActionContext struct {
	Action      *v1alpha1.HealingAction
	StartTime   time.Time
	CancelFunc  context.CancelFunc
	OriginalObj runtime.Object
}

// ActionRecorder records action history for audit and rollback
type ActionRecorder interface {
	RecordAction(ctx context.Context, action *v1alpha1.HealingAction, result *controller.ActionResult, originalState runtime.Object) error
	GetActionHistory(ctx context.Context, actionName string) (*ActionHistory, error)
}

// ActionHistory contains historical information about an action
type ActionHistory struct {
	ActionName    string
	OriginalState runtime.Object
	Changes       []v1alpha1.ResourceChange
	ExecutedAt    time.Time
}

// NewEngine creates a new remediation engine
func NewEngine(client client.Client, recorder ActionRecorder) *Engine {
	engine := &Engine{
		client:        client,
		executors:     make(map[string]controller.ActionExecutor),
		recorder:      recorder,
		activeActions: make(map[string]*ActionContext),
	}

	// Register default executors
	engine.RegisterExecutor("restart", NewRestartExecutor(client))
	engine.RegisterExecutor("scale", NewScaleExecutor(client))
	engine.RegisterExecutor("patch", NewPatchExecutor(client))
	engine.RegisterExecutor("delete", NewDeleteExecutor(client))

	return engine
}

// RegisterExecutor registers an action executor for a specific action type
func (e *Engine) RegisterExecutor(actionType string, executor controller.ActionExecutor) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.executors[actionType] = executor
}

// ExecuteAction performs the healing action
func (e *Engine) ExecuteAction(ctx context.Context, action *v1alpha1.HealingAction) (*controller.ActionResult, error) {
	log := log.FromContext(ctx)
	log.Info("Executing healing action", 
		"action", action.Name,
		"type", action.Spec.Action.Type,
		"target", fmt.Sprintf("%s/%s/%s", 
			action.Spec.TargetResource.Kind,
			action.Spec.TargetResource.Namespace,
			action.Spec.TargetResource.Name))

	// Track active action
	actionCtx := e.trackAction(action)
	defer e.untrackAction(action.Name)

	// Create cancelable context
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	actionCtx.CancelFunc = cancel
	defer cancel()

	// Get the executor
	executor, err := e.GetActionExecutor(action.Spec.Action.Type)
	if err != nil {
		return &controller.ActionResult{
			Success:   false,
			Message:   fmt.Sprintf("Failed to get executor: %v", err),
			Error:     err,
			StartTime: actionCtx.StartTime,
			EndTime:   time.Now(),
		}, err
	}

	// Get the target resource
	target, err := e.getTargetResource(ctx, &action.Spec.TargetResource)
	if err != nil {
		return &controller.ActionResult{
			Success:   false,
			Message:   fmt.Sprintf("Failed to get target resource: %v", err),
			Error:     err,
			StartTime: actionCtx.StartTime,
			EndTime:   time.Now(),
		}, err
	}

	// Store original state for potential rollback
	actionCtx.OriginalObj = target.DeepCopyObject()

	// Validate the action
	if err := executor.Validate(ctx, target, &action.Spec.Action); err != nil {
		return &controller.ActionResult{
			Success:   false,
			Message:   fmt.Sprintf("Action validation failed: %v", err),
			Error:     err,
			StartTime: actionCtx.StartTime,
			EndTime:   time.Now(),
		}, nil
	}

	// Execute the action
	result, err := executor.Execute(ctx, target, &action.Spec.Action)
	if result == nil {
		result = &controller.ActionResult{
			StartTime: actionCtx.StartTime,
			EndTime:   time.Now(),
		}
	}
	result.StartTime = actionCtx.StartTime
	result.EndTime = time.Now()

	// Record the action for audit and potential rollback
	if e.recorder != nil {
		if recordErr := e.recorder.RecordAction(ctx, action, result, actionCtx.OriginalObj); recordErr != nil {
			log.Error(recordErr, "Failed to record action")
		}
	}

	if err != nil {
		result.Success = false
		result.Error = err
		if result.Message == "" {
			result.Message = fmt.Sprintf("Action execution failed: %v", err)
		}
		return result, err
	}

	if !result.Success {
		return result, fmt.Errorf(result.Message)
	}

	log.Info("Healing action completed successfully",
		"action", action.Name,
		"duration", result.EndTime.Sub(result.StartTime))

	return result, nil
}

// DryRun simulates the action without executing
func (e *Engine) DryRun(ctx context.Context, action *v1alpha1.HealingAction) (*controller.ActionResult, error) {
	log := log.FromContext(ctx)
	log.Info("Performing dry-run for healing action",
		"action", action.Name,
		"type", action.Spec.Action.Type)

	startTime := time.Now()

	// Get the executor
	executor, err := e.GetActionExecutor(action.Spec.Action.Type)
	if err != nil {
		return &controller.ActionResult{
			Success:   false,
			Message:   fmt.Sprintf("Failed to get executor: %v", err),
			Error:     err,
			StartTime: startTime,
			EndTime:   time.Now(),
		}, err
	}

	// Get the target resource
	target, err := e.getTargetResource(ctx, &action.Spec.TargetResource)
	if err != nil {
		return &controller.ActionResult{
			Success:   false,
			Message:   fmt.Sprintf("Failed to get target resource: %v", err),
			Error:     err,
			StartTime: startTime,
			EndTime:   time.Now(),
		}, err
	}

	// Validate the action
	if err := executor.Validate(ctx, target, &action.Spec.Action); err != nil {
		return &controller.ActionResult{
			Success:   false,
			Message:   fmt.Sprintf("Action validation failed: %v", err),
			Error:     err,
			StartTime: startTime,
			EndTime:   time.Now(),
		}, nil
	}

	// Perform dry-run
	result, err := executor.DryRun(ctx, target, &action.Spec.Action)
	if result == nil {
		result = &controller.ActionResult{
			StartTime: startTime,
			EndTime:   time.Now(),
		}
	}
	result.StartTime = startTime
	result.EndTime = time.Now()

	if err != nil {
		result.Success = false
		result.Error = err
		if result.Message == "" {
			result.Message = fmt.Sprintf("Dry-run failed: %v", err)
		}
		return result, err
	}

	// Add dry-run indicator to result
	if result.Metrics == nil {
		result.Metrics = make(map[string]string)
	}
	result.Metrics["dry_run"] = "true"

	log.Info("Dry-run completed",
		"action", action.Name,
		"result", result.Success,
		"message", result.Message)

	return result, nil
}

// Rollback reverses a previously executed action
func (e *Engine) Rollback(ctx context.Context, action *v1alpha1.HealingAction) error {
	log := log.FromContext(ctx)
	log.Info("Rolling back healing action", "action", action.Name)

	if e.recorder == nil {
		return fmt.Errorf("no action recorder configured for rollback")
	}

	// Get action history
	history, err := e.recorder.GetActionHistory(ctx, action.Name)
	if err != nil {
		return fmt.Errorf("failed to get action history: %w", err)
	}

	if history == nil || history.OriginalState == nil {
		return fmt.Errorf("no rollback information available for action %s", action.Name)
	}

	// Convert original state back to unstructured
	originalUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(history.OriginalState)
	if err != nil {
		return fmt.Errorf("failed to convert original state: %w", err)
	}

	original := &unstructured.Unstructured{Object: originalUnstructured}

	// Check if resource still exists
	current := &unstructured.Unstructured{}
	current.SetGroupVersionKind(original.GroupVersionKind())
	key := client.ObjectKey{
		Namespace: original.GetNamespace(),
		Name:      original.GetName(),
	}

	if err := e.client.Get(ctx, key, current); err != nil {
		if errors.IsNotFound(err) {
			// Resource was deleted, recreate it
			if err := e.client.Create(ctx, original); err != nil {
				return fmt.Errorf("failed to recreate resource: %w", err)
			}
			log.Info("Resource recreated during rollback", "resource", key)
			return nil
		}
		return fmt.Errorf("failed to get current resource state: %w", err)
	}

	// Update resource to original state
	original.SetResourceVersion(current.GetResourceVersion())
	if err := e.client.Update(ctx, original); err != nil {
		return fmt.Errorf("failed to restore resource: %w", err)
	}

	log.Info("Rollback completed successfully", "action", action.Name)
	return nil
}

// GetActionExecutor returns the executor for a specific action type
func (e *Engine) GetActionExecutor(actionType string) (controller.ActionExecutor, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	executor, exists := e.executors[actionType]
	if !exists {
		return nil, fmt.Errorf("no executor registered for action type: %s", actionType)
	}

	return executor, nil
}

// getTargetResource retrieves the target resource from the cluster
func (e *Engine) getTargetResource(ctx context.Context, target *v1alpha1.TargetResource) (client.Object, error) {
	// Parse GVK
	gv, err := schema.ParseGroupVersion(target.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid apiVersion: %w", err)
	}

	gvk := schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    target.Kind,
	}

	// Create unstructured object
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)

	// Get the resource
	key := client.ObjectKey{
		Namespace: target.Namespace,
		Name:      target.Name,
	}

	if err := e.client.Get(ctx, key, obj); err != nil {
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	return obj, nil
}

// trackAction tracks an active action
func (e *Engine) trackAction(action *v1alpha1.HealingAction) *ActionContext {
	e.actionsMu.Lock()
	defer e.actionsMu.Unlock()

	ctx := &ActionContext{
		Action:    action,
		StartTime: time.Now(),
	}

	e.activeActions[action.Name] = ctx
	return ctx
}

// untrackAction removes an action from active tracking
func (e *Engine) untrackAction(actionName string) {
	e.actionsMu.Lock()
	defer e.actionsMu.Unlock()

	if ctx, exists := e.activeActions[actionName]; exists {
		if ctx.CancelFunc != nil {
			ctx.CancelFunc()
		}
		delete(e.activeActions, actionName)
	}
}

// GetActiveActions returns the list of currently active actions
func (e *Engine) GetActiveActions() []string {
	e.actionsMu.RLock()
	defer e.actionsMu.RUnlock()

	actions := make([]string, 0, len(e.activeActions))
	for name := range e.activeActions {
		actions = append(actions, name)
	}

	return actions
}

// CancelAction cancels an active action
func (e *Engine) CancelAction(actionName string) error {
	e.actionsMu.RLock()
	ctx, exists := e.activeActions[actionName]
	e.actionsMu.RUnlock()

	if !exists {
		return fmt.Errorf("action %s is not active", actionName)
	}

	if ctx.CancelFunc != nil {
		ctx.CancelFunc()
		return nil
	}

	return fmt.Errorf("action %s has no cancel function", actionName)
}