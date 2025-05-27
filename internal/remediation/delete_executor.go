package remediation

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
	"github.com/kubeskippy/kubeskippy/internal/controller"
)

// DeleteExecutor handles delete actions
type DeleteExecutor struct {
	client client.Client
}

// NewDeleteExecutor creates a new delete executor
func NewDeleteExecutor(client client.Client) *DeleteExecutor {
	return &DeleteExecutor{
		client: client,
	}
}

// Execute performs the delete action
func (d *DeleteExecutor) Execute(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) (*controller.ActionResult, error) {
	log := log.FromContext(ctx)
	startTime := time.Now()

	// Get delete configuration
	config := action.DeleteAction
	if config == nil {
		config = &v1alpha1.DeleteAction{
			GracePeriodSeconds: 30,
		}
	}

	// Check if resource has finalizers
	finalizers := target.GetFinalizers()
	if len(finalizers) > 0 && !config.Force {
		return &controller.ActionResult{
			Success:   false,
			Message:   fmt.Sprintf("Resource has finalizers and force delete is not enabled: %v", finalizers),
			StartTime: startTime,
			EndTime:   time.Now(),
		}, fmt.Errorf("resource has finalizers: %v", finalizers)
	}

	// Create delete options
	deleteOptions := &client.DeleteOptions{}
	if config.GracePeriodSeconds > 0 {
		gracePeriod := int64(config.GracePeriodSeconds)
		deleteOptions.GracePeriodSeconds = &gracePeriod
	}

	// If immediate deletion is requested
	if config.GracePeriodSeconds == 0 {
		gracePeriod := int64(0)
		deleteOptions.GracePeriodSeconds = &gracePeriod
	}

	// Propagation policy
	if config.PropagationPolicy != "" {
		switch config.PropagationPolicy {
		case "Orphan":
			propagation := metav1.DeletePropagationOrphan
			deleteOptions.PropagationPolicy = &propagation
		case "Background":
			propagation := metav1.DeletePropagationBackground
			deleteOptions.PropagationPolicy = &propagation
		case "Foreground":
			propagation := metav1.DeletePropagationForeground
			deleteOptions.PropagationPolicy = &propagation
		}
	}

	// Record the deletion
	changes := []v1alpha1.ResourceChange{
		{
			ResourceRef: fmt.Sprintf("%s/%s/%s", target.GetObjectKind().GroupVersionKind().Kind, target.GetNamespace(), target.GetName()),
			ChangeType:  "delete",
			Field:       "resource",
			OldValue:    target.GetName(),
			NewValue:    "deleted",
			Timestamp:   &metav1.Time{Time: time.Now()},
		},
	}

	log.Info("Deleting resource",
		"resource", fmt.Sprintf("%s/%s", target.GetNamespace(), target.GetName()),
		"gracePeriod", config.GracePeriodSeconds,
		"force", config.Force)

	// Force delete by removing finalizers if requested
	if config.Force && len(finalizers) > 0 {
		log.Info("Force deleting: removing finalizers", "finalizers", finalizers)
		target.SetFinalizers([]string{})
		if err := d.client.Update(ctx, target); err != nil && !errors.IsNotFound(err) {
			log.Error(err, "Failed to remove finalizers")
		}
	}

	// Delete the resource
	if err := d.client.Delete(ctx, target, deleteOptions); err != nil {
		if errors.IsNotFound(err) {
			// Already deleted
			return &controller.ActionResult{
				Success:   true,
				Message:   "Resource already deleted",
				Changes:   changes,
				StartTime: startTime,
				EndTime:   time.Now(),
			}, nil
		}
		return &controller.ActionResult{
			Success:   false,
			Message:   fmt.Sprintf("Failed to delete resource: %v", err),
			Error:     err,
			Changes:   changes,
			StartTime: startTime,
			EndTime:   time.Now(),
		}, err
	}

	log.Info("Resource deleted successfully",
		"resource", fmt.Sprintf("%s/%s", target.GetNamespace(), target.GetName()))

	return &controller.ActionResult{
		Success:   true,
		Message:   fmt.Sprintf("Successfully deleted %s/%s", target.GetNamespace(), target.GetName()),
		Changes:   changes,
		StartTime: startTime,
		EndTime:   time.Now(),
		Metrics: map[string]string{
			"grace_period_seconds": fmt.Sprintf("%d", config.GracePeriodSeconds),
			"force":                fmt.Sprintf("%v", config.Force),
			"propagation_policy":   config.PropagationPolicy,
		},
	}, nil
}

// Validate checks if the delete action can be executed
func (d *DeleteExecutor) Validate(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) error {
	// Check if resource is deletable
	gvk := target.GetObjectKind().GroupVersionKind()
	
	// Prevent deletion of certain critical resources
	criticalKinds := map[string]bool{
		"Namespace":                true,
		"Node":                     true,
		"PersistentVolume":         true,
		"CustomResourceDefinition": true,
		"ClusterRole":              true,
		"ClusterRoleBinding":       true,
	}

	if criticalKinds[gvk.Kind] {
		return fmt.Errorf("deletion of %s resources is not allowed", gvk.Kind)
	}

	// Check for protected namespaces
	protectedNamespaces := map[string]bool{
		"kube-system":     true,
		"kube-public":     true,
		"kube-node-lease": true,
		"default":         true,
	}

	if target.GetNamespace() != "" && protectedNamespaces[target.GetNamespace()] {
		return fmt.Errorf("deletion in protected namespace %s is not allowed", target.GetNamespace())
	}

	// Validate delete configuration if provided
	if action.DeleteAction != nil {
		config := action.DeleteAction
		
		if config.GracePeriodSeconds < 0 {
			return fmt.Errorf("gracePeriodSeconds cannot be negative")
		}

		if config.PropagationPolicy != "" {
			switch config.PropagationPolicy {
			case "Orphan", "Background", "Foreground":
				// Valid policies
			default:
				return fmt.Errorf("invalid propagation policy: %s", config.PropagationPolicy)
			}
		}
	}

	// Check if resource exists
	key := client.ObjectKey{
		Namespace: target.GetNamespace(),
		Name:      target.GetName(),
	}
	if err := d.client.Get(ctx, key, target); err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("resource not found")
		}
		return fmt.Errorf("failed to get resource: %w", err)
	}

	return nil
}

// DryRun simulates the delete action
func (d *DeleteExecutor) DryRun(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) (*controller.ActionResult, error) {
	// Validate the action
	if err := d.Validate(ctx, target, action); err != nil {
		return &controller.ActionResult{
			Success: false,
			Message: fmt.Sprintf("Validation failed: %v", err),
		}, err
	}

	config := action.DeleteAction
	if config == nil {
		config = &v1alpha1.DeleteAction{
			GracePeriodSeconds: 30,
		}
	}

	// Check for dependent resources
	dependents := d.checkDependentResources(ctx, target)
	
	// Simulate changes
	simulatedChanges := []v1alpha1.ResourceChange{
		{
			ResourceRef: fmt.Sprintf("%s/%s/%s", target.GetObjectKind().GroupVersionKind().Kind, target.GetNamespace(), target.GetName()),
			ChangeType:  "delete",
			Field:       "resource",
			OldValue:    target.GetName(),
			NewValue:    "would be deleted",
		},
	}

	message := fmt.Sprintf("Dry-run: Would delete %s/%s", target.GetNamespace(), target.GetName())
	if len(dependents) > 0 {
		message += fmt.Sprintf(" (warning: has %d dependent resources)", len(dependents))
	}

	return &controller.ActionResult{
		Success: true,
		Message: message,
		Changes: simulatedChanges,
		Metrics: map[string]string{
			"grace_period_seconds": fmt.Sprintf("%d", config.GracePeriodSeconds),
			"force":                fmt.Sprintf("%v", config.Force),
			"propagation_policy":   config.PropagationPolicy,
			"dependent_resources":  fmt.Sprintf("%d", len(dependents)),
			"dry_run":              "true",
		},
	}, nil
}

// checkDependentResources checks for resources that depend on the target
func (d *DeleteExecutor) checkDependentResources(ctx context.Context, target client.Object) []string {
	log := log.FromContext(ctx)
	var dependents []string

	// For pods, check if they're part of a ReplicaSet/Deployment
	if pod, ok := target.(*corev1.Pod); ok {
		for _, owner := range pod.OwnerReferences {
			dependents = append(dependents, fmt.Sprintf("%s/%s", owner.Kind, owner.Name))
		}
	}

	// For services, check for endpoints
	if svc, ok := target.(*corev1.Service); ok {
		endpoints := &corev1.Endpoints{}
		key := client.ObjectKey{
			Namespace: svc.Namespace,
			Name:      svc.Name,
		}
		if err := d.client.Get(ctx, key, endpoints); err == nil {
			dependents = append(dependents, fmt.Sprintf("Endpoints/%s", endpoints.Name))
		}
	}

	// Log if there are dependents
	if len(dependents) > 0 {
		log.Info("Target resource has dependents",
			"resource", fmt.Sprintf("%s/%s", target.GetNamespace(), target.GetName()),
			"dependents", dependents)
	}

	return dependents
}