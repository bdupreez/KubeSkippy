package controller

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
	"github.com/kubeskippy/kubeskippy/pkg/config"
)

// HealingPolicyReconciler reconciles a HealingPolicy object
type HealingPolicyReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	Config           *config.Config
	MetricsCollector MetricsCollector
	SafetyController SafetyController
	AIAnalyzer       AIAnalyzer
}

// +kubebuilder:rbac:groups=kubeskippy.io,resources=healingpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeskippy.io,resources=healingpolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubeskippy.io,resources=healingpolicies/finalizers,verbs=update
// +kubebuilder:rbac:groups=kubeskippy.io,resources=healingactions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeskippy.io,resources=healingactions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=pods;services;nodes;persistentvolumeclaims;configmaps;secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets;daemonsets;replicasets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;patch

// Reconcile is part of the main kubernetes reconciliation loop
func (r *HealingPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling HealingPolicy")

	// Fetch the HealingPolicy instance
	policy := &v1alpha1.HealingPolicy{}
	if err := r.Get(ctx, req.NamespacedName, policy); err != nil {
		if errors.IsNotFound(err) {
			log.Info("HealingPolicy not found, likely deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get HealingPolicy")
		return ctrl.Result{}, err
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(policy, FinalizerName) {
		controllerutil.AddFinalizer(policy, FinalizerName)
		if err := r.Update(ctx, policy); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Handle deletion
	if !policy.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, log, policy)
	}

	// Update status observed generation
	if policy.Status.ObservedGeneration != policy.Generation {
		policy.Status.ObservedGeneration = policy.Generation
		if err := r.Status().Update(ctx, policy); err != nil {
			log.Error(err, "Failed to update observed generation")
			return ctrl.Result{}, err
		}
	}

	// Evaluate the policy
	_, err := r.evaluatePolicy(ctx, log, policy)
	if err != nil {
		log.Error(err, "Failed to evaluate policy")
		SetCondition(&policy.Status.Conditions, v1alpha1.ConditionTypeReady,
			metav1.ConditionFalse, ReasonValidationError, err.Error())
		if err := r.Status().Update(ctx, policy); err != nil {
			log.Error(err, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, err
	}

	// Update status
	policy.Status.LastEvaluated = metav1.Now()
	SetCondition(&policy.Status.Conditions, v1alpha1.ConditionTypeReady,
		metav1.ConditionTrue, ReasonPolicyUpdated, "Policy evaluated successfully")

	if err := r.Status().Update(ctx, policy); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// Requeue based on policy mode and evaluation interval
	requeueAfter := 1 * time.Minute
	if policy.Spec.Mode == "monitor" {
		requeueAfter = 5 * time.Minute
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// evaluatePolicy evaluates triggers and creates healing actions if needed
func (r *HealingPolicyReconciler) evaluatePolicy(ctx context.Context, log logr.Logger, policy *v1alpha1.HealingPolicy) (*EvaluationResult, error) {
	log.Info("Evaluating policy", "mode", policy.Spec.Mode)

	// Check if policy is in monitor-only mode
	if policy.Spec.Mode == "monitor" {
		log.Info("Policy is in monitor mode, skipping action creation")
		return &EvaluationResult{Mode: "monitor"}, nil
	}

	// Collect metrics
	metrics, err := r.MetricsCollector.CollectMetrics(ctx, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to collect metrics: %w", err)
	}

	// Check rate limits
	if allowed, err := r.SafetyController.CheckRateLimit(ctx, policy); err != nil {
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	} else if !allowed {
		log.Info("Rate limit exceeded, skipping evaluation")
		return &EvaluationResult{RateLimited: true}, nil
	}

	// Evaluate triggers
	activeTriggers := []string{}
	triggeredActions := []TriggeredAction{}

	for _, trigger := range policy.Spec.Triggers {
		// Check cooldown
		if !r.checkCooldown(policy, trigger.Name, trigger.CooldownPeriod.Duration) {
			log.V(1).Info("Trigger in cooldown", "trigger", trigger.Name)
			continue
		}

		// Evaluate trigger
		triggered, reason, err := r.MetricsCollector.EvaluateTrigger(ctx, &trigger, metrics)
		if err != nil {
			log.Error(err, "Failed to evaluate trigger", "trigger", trigger.Name)
			continue
		}

		log.Info("Trigger evaluation result", "trigger", trigger.Name, "type", trigger.Type, "triggered", triggered, "reason", reason)

		if triggered {
			log.Info("Trigger activated", "trigger", trigger.Name, "reason", reason)
			activeTriggers = append(activeTriggers, trigger.Name)

			// Find matching resources
			resources, err := r.findMatchingResources(ctx, policy)
			if err != nil {
				log.Error(err, "Failed to find matching resources")
				continue
			}

			// Create triggered actions
			for _, resource := range resources {
				for _, actionTemplate := range policy.Spec.Actions {
					triggeredActions = append(triggeredActions, TriggeredAction{
						Trigger:  trigger.Name,
						Resource: resource,
						Action:   actionTemplate,
						Reason:   reason,
					})
				}
			}
		}
	}

	// Update active triggers in status
	policy.Status.ActiveTriggers = activeTriggers

	// Process triggered actions
	if len(triggeredActions) > 0 {
		// Get AI recommendations if configured
		if r.AIAnalyzer != nil && r.Config.AI.Provider != "" {
			aiResult, err := r.getAIRecommendations(ctx, metrics, triggeredActions)
			if err != nil {
				log.Error(err, "Failed to get AI recommendations")
			} else {
				triggeredActions = r.filterActionsWithAI(triggeredActions, aiResult)
			}
		}

		// Sort actions by priority
		sort.Slice(triggeredActions, func(i, j int) bool {
			return triggeredActions[i].Action.Priority > triggeredActions[j].Action.Priority
		})

		// Create healing actions
		createdCount := 0
		for _, ta := range triggeredActions {
			if createdCount >= 5 { // Limit actions per evaluation
				break
			}

			action := CreateHealingAction(
				policy,
				ta.Resource,
				&ta.Action,
				policy.Spec.Mode == "dryrun",
			)

			// Validate action with safety controller
			validation, err := r.SafetyController.ValidateAction(ctx, action)
			if err != nil {
				log.Error(err, "Failed to validate action")
				continue
			}

			if !validation.Valid {
				log.Info("Action validation failed", "reason", validation.Reason,
					"warnings", validation.Warnings)
				continue
			}

			// Create the action
			if err := r.Create(ctx, action); err != nil {
				log.Error(err, "Failed to create healing action")
				continue
			}

			log.Info("Created healing action",
				"action", action.Name,
				"type", action.Spec.Action.Type,
				"target", fmt.Sprintf("%s/%s", action.Spec.TargetResource.Kind, action.Spec.TargetResource.Name))

			createdCount++
			policy.Status.ActionsTaken++
			policy.Status.LastActionTime = metav1.Now()
		}
	}

	return &EvaluationResult{
		ActiveTriggers:   activeTriggers,
		ActionsCreated:   len(triggeredActions),
		MetricsCollected: true,
	}, nil
}

// findMatchingResources finds resources that match the policy selector
func (r *HealingPolicyReconciler) findMatchingResources(ctx context.Context, policy *v1alpha1.HealingPolicy) ([]client.Object, error) {
	matcher := NewPolicyMatcher(policy)
	var resources []client.Object

	for _, rf := range policy.Spec.Selector.Resources {
		// Map common resource types
		var list client.ObjectList
		switch rf.Kind {
		case "Pod":
			list = &corev1.PodList{}
		case "Deployment":
			list = &appsv1.DeploymentList{}
		case "StatefulSet":
			list = &appsv1.StatefulSetList{}
		case "DaemonSet":
			list = &appsv1.DaemonSetList{}
		case "Service":
			list = &corev1.ServiceList{}
		case "PersistentVolumeClaim":
			list = &corev1.PersistentVolumeClaimList{}
		default:
			// Skip unknown resource types for now
			continue
		}

		// List resources
		listOpts := []client.ListOption{}
		if len(policy.Spec.Selector.Namespaces) > 0 {
			// List in specific namespaces
			for _, ns := range policy.Spec.Selector.Namespaces {
				nsListOpts := append(listOpts, client.InNamespace(ns))
				if err := r.List(ctx, list, nsListOpts...); err != nil {
					return nil, fmt.Errorf("failed to list %s: %w", rf.Kind, err)
				}

				// Extract items and check if they match
				items, err := meta.ExtractList(list)
				if err != nil {
					return nil, fmt.Errorf("failed to extract list: %w", err)
				}

				for _, item := range items {
					obj := item.(client.Object)
					if matches, err := matcher.Matches(obj); err != nil {
						return nil, err
					} else if matches {
						resources = append(resources, obj)
					}
				}
			}
		} else {
			// List in all namespaces
			if err := r.List(ctx, list, listOpts...); err != nil {
				return nil, fmt.Errorf("failed to list %s: %w", rf.Kind, err)
			}

			// Extract items and check if they match
			items, err := meta.ExtractList(list)
			if err != nil {
				return nil, fmt.Errorf("failed to extract list: %w", err)
			}

			for _, item := range items {
				obj := item.(client.Object)
				if matches, err := matcher.Matches(obj); err != nil {
					return nil, err
				} else if matches {
					resources = append(resources, obj)
				}
			}
		}
	}

	return resources, nil
}

// checkCooldown checks if a trigger is in cooldown
func (r *HealingPolicyReconciler) checkCooldown(policy *v1alpha1.HealingPolicy, triggerName string, cooldown time.Duration) bool {
	// Check last action time for this trigger
	// In a real implementation, this would check a more detailed history
	if !policy.Status.LastActionTime.IsZero() {
		elapsed := time.Since(policy.Status.LastActionTime.Time)
		if elapsed < cooldown {
			return false
		}
	}
	return true
}

// getAIRecommendations gets AI recommendations for triggered actions
func (r *HealingPolicyReconciler) getAIRecommendations(ctx context.Context, metrics *ClusterMetrics, actions []TriggeredAction) (*AIAnalysis, error) {
	// Convert triggered actions to issues
	issues := make([]Issue, len(actions))
	for i, action := range actions {
		issues[i] = Issue{
			ID:          fmt.Sprintf("%s-%s", action.Trigger, action.Resource.GetName()),
			Severity:    "medium",
			Type:        action.Trigger,
			Resource:    ResourceKey(action.Resource),
			Description: action.Reason,
			DetectedAt:  time.Now(),
		}
	}

	// Get AI analysis
	return r.AIAnalyzer.AnalyzeClusterState(ctx, metrics, issues)
}

// filterActionsWithAI filters actions based on AI recommendations
func (r *HealingPolicyReconciler) filterActionsWithAI(actions []TriggeredAction, aiResult *AIAnalysis) []TriggeredAction {
	// In a real implementation, this would match AI recommendations
	// with triggered actions and filter based on confidence
	return actions
}

// handleDeletion handles cleanup when a policy is deleted
func (r *HealingPolicyReconciler) handleDeletion(ctx context.Context, log logr.Logger, policy *v1alpha1.HealingPolicy) (ctrl.Result, error) {
	log.Info("Handling policy deletion")

	// Delete all associated healing actions
	actionList := &v1alpha1.HealingActionList{}
	if err := r.List(ctx, actionList, client.InNamespace(policy.Namespace),
		client.MatchingLabels{LabelPolicyName: policy.Name}); err != nil {
		log.Error(err, "Failed to list healing actions")
		return ctrl.Result{}, err
	}

	for _, action := range actionList.Items {
		if err := r.Delete(ctx, &action); err != nil && !errors.IsNotFound(err) {
			log.Error(err, "Failed to delete healing action", "action", action.Name)
			return ctrl.Result{}, err
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(policy, FinalizerName)
	if err := r.Update(ctx, policy); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *HealingPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Create indices for efficient lookups
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1alpha1.HealingAction{}, "spec.policyRef.name", func(rawObj client.Object) []string {
		action := rawObj.(*v1alpha1.HealingAction)
		return []string{action.Spec.PolicyRef.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.HealingPolicy{}).
		Owns(&v1alpha1.HealingAction{}).
		Complete(r)
}

// EvaluationResult contains the result of policy evaluation
type EvaluationResult struct {
	Mode             string
	ActiveTriggers   []string
	ActionsCreated   int
	MetricsCollected bool
	RateLimited      bool
}

// TriggeredAction represents an action triggered by a policy
type TriggeredAction struct {
	Trigger  string
	Resource client.Object
	Action   v1alpha1.HealingActionTemplate
	Reason   string
}
