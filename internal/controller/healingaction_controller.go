package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
	"github.com/kubeskippy/kubeskippy/pkg/config"
)

// HealingActionReconciler reconciles a HealingAction object
type HealingActionReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	Config            *config.Config
	RemediationEngine RemediationEngine
	SafetyController  SafetyController
}

// +kubebuilder:rbac:groups=kubeskippy.io,resources=healingactions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeskippy.io,resources=healingactions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubeskippy.io,resources=healingactions/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets;daemonsets;replicasets,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop
func (r *HealingActionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling HealingAction")

	// Fetch the HealingAction instance
	action := &v1alpha1.HealingAction{}
	if err := r.Get(ctx, req.NamespacedName, action); err != nil {
		if errors.IsNotFound(err) {
			log.Info("HealingAction not found, likely deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get HealingAction")
		return ctrl.Result{}, err
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(action, FinalizerName) {
		controllerutil.AddFinalizer(action, FinalizerName)
		if err := r.Update(ctx, action); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		// Continue processing instead of requeuing
	}

	// Handle deletion
	if !action.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, log, action)
	}

	// Update observed generation
	if action.Status.ObservedGeneration != action.Generation {
		action.Status.ObservedGeneration = action.Generation
		if err := r.Status().Update(ctx, action); err != nil {
			log.Error(err, "Failed to update observed generation")
			return ctrl.Result{}, err
		}
	}

	// Process based on phase
	switch action.Status.Phase {
	case "", v1alpha1.HealingActionPhasePending:
		return r.handlePending(ctx, log, action)
	case v1alpha1.HealingActionPhaseApproved:
		return r.handleApproved(ctx, log, action)
	case v1alpha1.HealingActionPhaseInProgress:
		return r.handleInProgress(ctx, log, action)
	case v1alpha1.HealingActionPhaseSucceeded, v1alpha1.HealingActionPhaseFailed, v1alpha1.HealingActionPhaseCancelled:
		// Terminal states - nothing to do
		return ctrl.Result{}, nil
	default:
		log.Error(nil, "Unknown phase", "phase", action.Status.Phase)
		return ctrl.Result{}, nil
	}
}

// handlePending handles actions in pending state
func (r *HealingActionReconciler) handlePending(ctx context.Context, log logr.Logger, action *v1alpha1.HealingAction) (ctrl.Result, error) {
	log.Info("Handling pending action")

	// Update label for phase
	if action.Labels == nil {
		action.Labels = make(map[string]string)
	}
	action.Labels[LabelActionPhase] = v1alpha1.HealingActionPhasePending

	// Check if approval is required
	if action.Spec.ApprovalRequired {
		log.Info("Action requires approval")

		if action.Status.Approval == nil {
			action.Status.Approval = &v1alpha1.ApprovalStatus{
				Required: true,
				Approved: false,
			}
		}

		// Check if approved
		if !action.Status.Approval.Approved {
			// Still waiting for approval
			action.SetPhase(v1alpha1.HealingActionPhasePending, "WaitingForApproval",
				"Action is waiting for manual approval")

			// Update status first
			if err := r.Status().Update(ctx, action); err != nil {
				log.Error(err, "Failed to update status")
				return ctrl.Result{}, err
			}

			// Update labels
			if err := r.Update(ctx, action); err != nil {
				log.Error(err, "Failed to update action")
				return ctrl.Result{}, err
			}

			// Check again in 30 seconds
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		// Approved - move to approved phase
		action.SetPhase(v1alpha1.HealingActionPhaseApproved, "Approved",
			fmt.Sprintf("Action approved by %s", action.Status.Approval.ApprovedBy))
	} else {
		// No approval required - move directly to approved
		action.SetPhase(v1alpha1.HealingActionPhaseApproved, "AutoApproved",
			"Action automatically approved")
	}

	// Update status first
	if err := r.Status().Update(ctx, action); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// Then update labels
	if action.Labels == nil {
		action.Labels = make(map[string]string)
	}
	action.Labels[LabelActionPhase] = v1alpha1.HealingActionPhaseApproved
	if err := r.Update(ctx, action); err != nil {
		log.Error(err, "Failed to update action")
		return ctrl.Result{}, err
	}

	return ctrl.Result{Requeue: true}, nil
}

// handleApproved handles actions that have been approved
func (r *HealingActionReconciler) handleApproved(ctx context.Context, log logr.Logger, action *v1alpha1.HealingAction) (ctrl.Result, error) {
	log.Info("Handling approved action")

	// Validate action one more time before execution
	validation, err := r.SafetyController.ValidateAction(ctx, action)
	if err != nil {
		log.Error(err, "Failed to validate action")
		action.SetPhase(v1alpha1.HealingActionPhaseFailed, ReasonValidationError, err.Error())
		if err := r.Status().Update(ctx, action); err != nil {
			log.Error(err, "Failed to update status")
		}
		return ctrl.Result{}, nil
	}

	if !validation.Valid {
		log.Info("Action validation failed", "reason", validation.Reason)
		action.SetPhase(v1alpha1.HealingActionPhaseFailed, ReasonValidationError, validation.Reason)
		action.Status.Result = &v1alpha1.ActionResult{
			Success: false,
			Message: validation.Reason,
			Error:   "Validation failed",
		}
		if err := r.Status().Update(ctx, action); err != nil {
			log.Error(err, "Failed to update status")
		}
		return ctrl.Result{}, nil
	}

	// Move to in-progress
	action.SetPhase(v1alpha1.HealingActionPhaseInProgress, "Executing", "Starting action execution")
	action.Status.StartTime = &metav1.Time{Time: time.Now()}
	action.Status.Attempts = 0
	
	// Update status first
	if err := r.Status().Update(ctx, action); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}
	
	// Then update labels
	if action.Labels == nil {
		action.Labels = make(map[string]string)
	}
	action.Labels[LabelActionPhase] = v1alpha1.HealingActionPhaseInProgress
	if err := r.Update(ctx, action); err != nil {
		log.Error(err, "Failed to update action")
		return ctrl.Result{}, err
	}

	return ctrl.Result{Requeue: true}, nil
}

// handleInProgress handles actions that are being executed
func (r *HealingActionReconciler) handleInProgress(ctx context.Context, log logr.Logger, action *v1alpha1.HealingAction) (ctrl.Result, error) {
	log.Info("Handling in-progress action", "attempts", action.Status.Attempts)

	// Check timeout
	if action.Status.StartTime != nil {
		elapsed := time.Since(action.Status.StartTime.Time)
		if elapsed > action.Spec.Timeout.Duration {
			log.Info("Action timed out")
			action.SetPhase(v1alpha1.HealingActionPhaseFailed, "Timeout", "Action execution timed out")
			action.Status.Result = &v1alpha1.ActionResult{
				Success: false,
				Message: "Action timed out",
				Error:   fmt.Sprintf("Exceeded timeout of %v", action.Spec.Timeout.Duration),
			}
			return r.completeAction(ctx, log, action)
		}
	}

	// Execute the action
	action.Status.Attempts++
	action.Status.LastAttemptTime = &metav1.Time{Time: time.Now()}

	var result *ActionResult
	var err error

	if action.Spec.DryRun {
		log.Info("Executing dry-run")
		result, err = r.RemediationEngine.DryRun(ctx, action)
	} else {
		log.Info("Executing action")
		result, err = r.RemediationEngine.ExecuteAction(ctx, action)
	}

	if err != nil {
		log.Error(err, "Action execution failed")

		// Check if we should retry
		if action.Spec.RetryPolicy != nil && action.Status.Attempts < action.Spec.RetryPolicy.MaxAttempts {
			backoff := CalculateBackoff(
				action.Status.Attempts,
				action.Spec.RetryPolicy.BackoffDelay.Duration,
				action.Spec.RetryPolicy.BackoffMultiplier,
			)

			log.Info("Will retry action", "attempt", action.Status.Attempts, "backoff", backoff)

			SetCondition(&action.Status.Conditions, "Retrying", metav1.ConditionTrue,
				"RetryScheduled", fmt.Sprintf("Will retry after %v", backoff))

			if err := r.Status().Update(ctx, action); err != nil {
				log.Error(err, "Failed to update status")
			}

			return ctrl.Result{RequeueAfter: backoff}, nil
		}

		// Max retries exceeded or no retry policy
		action.SetPhase(v1alpha1.HealingActionPhaseFailed, ReasonActionFailed,
			fmt.Sprintf("Action failed after %d attempts: %v", action.Status.Attempts, err))

		if result != nil {
			action.Status.Result = &v1alpha1.ActionResult{
				Success: result.Success,
				Message: result.Message,
				Error:   err.Error(),
				Metrics: result.Metrics,
				Changes: result.Changes,
			}
		} else {
			action.Status.Result = &v1alpha1.ActionResult{
				Success: false,
				Error:   err.Error(),
			}
		}

		return r.completeAction(ctx, log, action)
	}

	// Action succeeded
	log.Info("Action executed successfully")
	action.SetPhase(v1alpha1.HealingActionPhaseSucceeded, ReasonActionSucceeded,
		"Action completed successfully")

	action.Status.Result = &v1alpha1.ActionResult{
		Success: result.Success,
		Message: result.Message,
		Metrics: result.Metrics,
		Changes: result.Changes,
	}

	// Record the action with safety controller
	r.SafetyController.RecordAction(ctx, action, result)

	return r.completeAction(ctx, log, action)
}

// completeAction updates the action to its final state
func (r *HealingActionReconciler) completeAction(ctx context.Context, log logr.Logger, action *v1alpha1.HealingAction) (ctrl.Result, error) {
	now := metav1.Now()
	action.Status.CompletionTime = &now
	
	// Ensure labels map exists
	if action.Labels == nil {
		action.Labels = make(map[string]string)
	}
	action.Labels[LabelActionPhase] = action.Status.Phase

	// Create an event
	eventType := corev1.EventTypeNormal
	reason := ReasonActionSucceeded
	message := fmt.Sprintf("Healing action %s completed successfully", action.Spec.Action.Type)

	if action.Status.Phase == v1alpha1.HealingActionPhaseFailed {
		eventType = corev1.EventTypeWarning
		reason = ReasonActionFailed
		message = fmt.Sprintf("Healing action %s failed: %s",
			action.Spec.Action.Type,
			action.Status.Result.Error)
	}

	r.recordEvent(action, eventType, reason, message)

	// Update status first (contains phase and completion time)
	if err := r.Status().Update(ctx, action); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// Then update labels
	if err := r.Update(ctx, action); err != nil {
		log.Error(err, "Failed to update action")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// handleDeletion handles cleanup when an action is deleted
func (r *HealingActionReconciler) handleDeletion(ctx context.Context, log logr.Logger, action *v1alpha1.HealingAction) (ctrl.Result, error) {
	log.Info("Handling action deletion")

	// If action was in progress, try to cancel/rollback
	if action.Status.Phase == v1alpha1.HealingActionPhaseInProgress {
		if r.Config.Remediation.EnableRollback {
			log.Info("Attempting to rollback in-progress action")
			if err := r.RemediationEngine.Rollback(ctx, action); err != nil {
				log.Error(err, "Failed to rollback action")
			}
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(action, FinalizerName)
	if err := r.Update(ctx, action); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// recordEvent records a Kubernetes event
func (r *HealingActionReconciler) recordEvent(action *v1alpha1.HealingAction, eventType, reason, message string) {
	// In a real implementation, this would use the event recorder
	// For now, we'll just log it
	log := log.FromContext(context.Background())
	log.Info("Recording event",
		"type", eventType,
		"reason", reason,
		"message", message,
		"action", action.Name)
}

// SetupWithManager sets up the controller with the Manager
func (r *HealingActionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.HealingAction{}).
		Complete(r)
}
