package remediation

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
	"github.com/kubeskippy/kubeskippy/internal/controller"
)

// ScaleExecutor handles scale actions
type ScaleExecutor struct {
	client client.Client
}

// NewScaleExecutor creates a new scale executor
func NewScaleExecutor(client client.Client) *ScaleExecutor {
	return &ScaleExecutor{
		client: client,
	}
}

// Execute performs the scale action
func (s *ScaleExecutor) Execute(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) (*controller.ActionResult, error) {
	log := log.FromContext(ctx)
	startTime := time.Now()

	// Get scale configuration
	config := action.ScaleAction
	if config == nil {
		return &controller.ActionResult{
			Success:   false,
			Message:   "Scale action configuration is missing",
			StartTime: startTime,
			EndTime:   time.Now(),
		}, fmt.Errorf("scale action configuration is missing")
	}

	// Get current replicas
	currentReplicas, err := s.getCurrentReplicas(target)
	if err != nil {
		return &controller.ActionResult{
			Success:   false,
			Message:   fmt.Sprintf("Failed to get current replicas: %v", err),
			Error:     err,
			StartTime: startTime,
			EndTime:   time.Now(),
		}, err
	}

	// Calculate new replicas
	newReplicas := currentReplicas
	switch config.Direction {
	case "up":
		newReplicas = currentReplicas + config.Replicas
		if config.MaxReplicas > 0 && newReplicas > config.MaxReplicas {
			newReplicas = config.MaxReplicas
		}
	case "down":
		newReplicas = currentReplicas - config.Replicas
		if newReplicas < config.MinReplicas {
			newReplicas = config.MinReplicas
		}
	case "absolute":
		newReplicas = config.Replicas
		if config.MaxReplicas > 0 && newReplicas > config.MaxReplicas {
			newReplicas = config.MaxReplicas
		}
		if newReplicas < config.MinReplicas {
			newReplicas = config.MinReplicas
		}
	default:
		return &controller.ActionResult{
			Success:   false,
			Message:   fmt.Sprintf("Invalid scale direction: %s", config.Direction),
			StartTime: startTime,
			EndTime:   time.Now(),
		}, fmt.Errorf("invalid scale direction: %s", config.Direction)
	}

	// Check if scaling is needed
	if newReplicas == currentReplicas {
		return &controller.ActionResult{
			Success: true,
			Message: fmt.Sprintf("No scaling needed, already at %d replicas", currentReplicas),
			StartTime: startTime,
			EndTime:   time.Now(),
			Metrics: map[string]string{
				"current_replicas": fmt.Sprintf("%d", currentReplicas),
				"target_replicas":  fmt.Sprintf("%d", newReplicas),
				"action":           "no-op",
			},
		}, nil
	}

	// Perform the scaling
	changes, err := s.scaleResource(ctx, target, newReplicas)
	if err != nil {
		return &controller.ActionResult{
			Success:   false,
			Message:   fmt.Sprintf("Failed to scale resource: %v", err),
			Error:     err,
			Changes:   changes,
			StartTime: startTime,
			EndTime:   time.Now(),
		}, err
	}

	log.Info("Resource scaled successfully",
		"resource", fmt.Sprintf("%s/%s", target.GetNamespace(), target.GetName()),
		"from", currentReplicas,
		"to", newReplicas)

	return &controller.ActionResult{
		Success:   true,
		Message:   fmt.Sprintf("Successfully scaled %s/%s from %d to %d replicas", target.GetNamespace(), target.GetName(), currentReplicas, newReplicas),
		Changes:   changes,
		StartTime: startTime,
		EndTime:   time.Now(),
		Metrics: map[string]string{
			"previous_replicas": fmt.Sprintf("%d", currentReplicas),
			"new_replicas":      fmt.Sprintf("%d", newReplicas),
			"scale_direction":   config.Direction,
		},
	}, nil
}

// Validate checks if the scale action can be executed
func (s *ScaleExecutor) Validate(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) error {
	// Check if resource type is supported
	switch target.(type) {
	case *appsv1.Deployment, *appsv1.ReplicaSet, *appsv1.StatefulSet:
		// Supported types
	default:
		return fmt.Errorf("scale not supported for resource type %T", target)
	}

	// Validate scale configuration
	if action.ScaleAction == nil {
		return fmt.Errorf("scale action configuration is missing")
	}

	config := action.ScaleAction
	
	// Validate direction
	switch config.Direction {
	case "up", "down", "absolute":
		// Valid directions
	default:
		return fmt.Errorf("invalid scale direction: %s", config.Direction)
	}

	// Validate replica counts
	if config.MinReplicas < 0 {
		return fmt.Errorf("minReplicas cannot be negative")
	}

	if config.MaxReplicas > 0 && config.MaxReplicas < config.MinReplicas {
		return fmt.Errorf("maxReplicas must be greater than or equal to minReplicas")
	}

	if config.Direction == "absolute" && config.Replicas < 0 {
		return fmt.Errorf("replicas cannot be negative for absolute scaling")
	}

	return nil
}

// DryRun simulates the scale action
func (s *ScaleExecutor) DryRun(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) (*controller.ActionResult, error) {
	// Validate the action
	if err := s.Validate(ctx, target, action); err != nil {
		return &controller.ActionResult{
			Success: false,
			Message: fmt.Sprintf("Validation failed: %v", err),
		}, err
	}

	config := action.ScaleAction

	// Get current replicas
	currentReplicas, err := s.getCurrentReplicas(target)
	if err != nil {
		return &controller.ActionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to get current replicas: %v", err),
		}, err
	}

	// Calculate new replicas
	newReplicas := currentReplicas
	switch config.Direction {
	case "up":
		newReplicas = currentReplicas + config.Replicas
		if config.MaxReplicas > 0 && newReplicas > config.MaxReplicas {
			newReplicas = config.MaxReplicas
		}
	case "down":
		newReplicas = currentReplicas - config.Replicas
		if newReplicas < config.MinReplicas {
			newReplicas = config.MinReplicas
		}
	case "absolute":
		newReplicas = config.Replicas
	}

	// Simulate changes
	var resourceType string
	switch target.(type) {
	case *appsv1.Deployment:
		resourceType = "Deployment"
	case *appsv1.ReplicaSet:
		resourceType = "ReplicaSet"
	case *appsv1.StatefulSet:
		resourceType = "StatefulSet"
	}

	simulatedChanges := []v1alpha1.ResourceChange{
		{
			ResourceRef: fmt.Sprintf("%s/%s/%s", resourceType, target.GetNamespace(), target.GetName()),
			ChangeType:  "update",
			Field:       "spec.replicas",
			OldValue:    fmt.Sprintf("%d", currentReplicas),
			NewValue:    fmt.Sprintf("%d", newReplicas),
		},
	}

	return &controller.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Dry-run: Would scale %s/%s from %d to %d replicas", target.GetNamespace(), target.GetName(), currentReplicas, newReplicas),
		Changes: simulatedChanges,
		Metrics: map[string]string{
			"current_replicas": fmt.Sprintf("%d", currentReplicas),
			"target_replicas":  fmt.Sprintf("%d", newReplicas),
			"scale_direction":  config.Direction,
			"dry_run":          "true",
		},
	}, nil
}

// getCurrentReplicas gets the current replica count for a resource
func (s *ScaleExecutor) getCurrentReplicas(target client.Object) (int32, error) {
	switch obj := target.(type) {
	case *appsv1.Deployment:
		if obj.Spec.Replicas == nil {
			return 1, nil // Default for deployments
		}
		return *obj.Spec.Replicas, nil
	case *appsv1.ReplicaSet:
		if obj.Spec.Replicas == nil {
			return 1, nil
		}
		return *obj.Spec.Replicas, nil
	case *appsv1.StatefulSet:
		if obj.Spec.Replicas == nil {
			return 1, nil
		}
		return *obj.Spec.Replicas, nil
	default:
		return 0, fmt.Errorf("cannot get replicas for resource type %T", target)
	}
}

// scaleResource performs the actual scaling operation
func (s *ScaleExecutor) scaleResource(ctx context.Context, target client.Object, newReplicas int32) ([]v1alpha1.ResourceChange, error) {
	log := log.FromContext(ctx)
	
	currentReplicas, _ := s.getCurrentReplicas(target)
	
	// Create change record
	var resourceType string
	var changes []v1alpha1.ResourceChange

	switch obj := target.(type) {
	case *appsv1.Deployment:
		resourceType = "Deployment"
		obj.Spec.Replicas = &newReplicas
		if err := s.client.Update(ctx, obj); err != nil {
			return nil, fmt.Errorf("failed to update deployment: %w", err)
		}
		
	case *appsv1.ReplicaSet:
		resourceType = "ReplicaSet"
		obj.Spec.Replicas = &newReplicas
		if err := s.client.Update(ctx, obj); err != nil {
			return nil, fmt.Errorf("failed to update replicaset: %w", err)
		}
		
	case *appsv1.StatefulSet:
		resourceType = "StatefulSet"
		obj.Spec.Replicas = &newReplicas
		if err := s.client.Update(ctx, obj); err != nil {
			return nil, fmt.Errorf("failed to update statefulset: %w", err)
		}
		
	default:
		return nil, fmt.Errorf("unsupported resource type for scaling: %T", target)
	}

	changes = append(changes, v1alpha1.ResourceChange{
		ResourceRef: fmt.Sprintf("%s/%s/%s", resourceType, target.GetNamespace(), target.GetName()),
		ChangeType:  "update",
		Field:       "spec.replicas",
		OldValue:    fmt.Sprintf("%d", currentReplicas),
		NewValue:    fmt.Sprintf("%d", newReplicas),
		Timestamp:   &metav1.Time{Time: time.Now()},
	})

	log.Info("Scaled resource",
		"type", resourceType,
		"name", target.GetName(),
		"namespace", target.GetNamespace(),
		"from", currentReplicas,
		"to", newReplicas)

	// Check if HPA exists and might interfere
	s.checkHPA(ctx, target)

	return changes, nil
}

// checkHPA checks if there's an HPA that might interfere with manual scaling
func (s *ScaleExecutor) checkHPA(ctx context.Context, target client.Object) {
	log := log.FromContext(ctx)

	// List HPAs in the namespace
	hpaList := &autoscalingv1.HorizontalPodAutoscalerList{}
	if err := s.client.List(ctx, hpaList, client.InNamespace(target.GetNamespace())); err != nil {
		log.V(1).Info("Failed to list HPAs", "error", err)
		return
	}

	// Check if any HPA targets this resource
	for _, hpa := range hpaList.Items {
		if hpa.Spec.ScaleTargetRef.Name == target.GetName() {
			gvk := target.GetObjectKind().GroupVersionKind()
			if hpa.Spec.ScaleTargetRef.Kind == gvk.Kind {
				log.Info("Warning: HPA exists for this resource and may override manual scaling",
					"hpa", hpa.Name,
					"resource", target.GetName())
			}
		}
	}
}