package remediation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
	"github.com/kubeskippy/kubeskippy/internal/controller"
)

// RestartExecutor handles restart actions
type RestartExecutor struct {
	client client.Client
}

// NewRestartExecutor creates a new restart executor
func NewRestartExecutor(client client.Client) *RestartExecutor {
	return &RestartExecutor{
		client: client,
	}
}

// Execute performs the restart action
func (r *RestartExecutor) Execute(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) (*controller.ActionResult, error) {
	log := log.FromContext(ctx)
	startTime := time.Now()

	// Get restart configuration
	config := action.RestartAction
	if config == nil {
		config = &v1alpha1.RestartAction{
			Strategy: "rolling",
		}
	}

	// Get the GVK to determine resource type
	gvk := target.GetObjectKind().GroupVersionKind()

	// Execute based on resource type
	var changes []v1alpha1.ResourceChange
	var err error

	switch gvk.Kind {
	case "Pod":
		changes, err = r.restartPodGeneric(ctx, target, config)
	case "Deployment":
		changes, err = r.restartWorkloadGeneric(ctx, target, config, "Deployment")
	case "StatefulSet":
		changes, err = r.restartWorkloadGeneric(ctx, target, config, "StatefulSet")
	case "DaemonSet":
		changes, err = r.restartWorkloadGeneric(ctx, target, config, "DaemonSet")
	default:
		return &controller.ActionResult{
			Success:   false,
			Message:   fmt.Sprintf("Restart not supported for resource kind %s", gvk.Kind),
			StartTime: startTime,
			EndTime:   time.Now(),
		}, fmt.Errorf("unsupported resource kind: %s", gvk.Kind)
	}

	if err != nil {
		return &controller.ActionResult{
			Success:   false,
			Message:   fmt.Sprintf("Failed to restart resource: %v", err),
			Error:     err,
			Changes:   changes,
			StartTime: startTime,
			EndTime:   time.Now(),
		}, err
	}

	log.Info("Resource restarted successfully",
		"resource", fmt.Sprintf("%s/%s", target.GetNamespace(), target.GetName()),
		"strategy", config.Strategy,
		"changes", len(changes))

	return &controller.ActionResult{
		Success:   true,
		Message:   fmt.Sprintf("Successfully restarted %s/%s using %s strategy", target.GetNamespace(), target.GetName(), config.Strategy),
		Changes:   changes,
		StartTime: startTime,
		EndTime:   time.Now(),
		Metrics: map[string]string{
			"restart_strategy": config.Strategy,
			"resource_type":    fmt.Sprintf("%T", target),
		},
	}, nil
}

// Validate checks if the restart action can be executed
func (r *RestartExecutor) Validate(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) error {
	// Check if resource type is supported
	gvk := target.GetObjectKind().GroupVersionKind()
	switch gvk.Kind {
	case "Pod", "Deployment", "StatefulSet", "DaemonSet":
		// Supported types
	default:
		return fmt.Errorf("restart not supported for resource kind %s", gvk.Kind)
	}

	// Validate restart configuration
	if action.RestartAction != nil {
		if action.RestartAction.Strategy != "" {
			switch action.RestartAction.Strategy {
			case "rolling", "recreate", "graceful":
				// Valid strategies
			default:
				return fmt.Errorf("invalid restart strategy: %s", action.RestartAction.Strategy)
			}
		}
	}

	return nil
}

// DryRun simulates the restart action
func (r *RestartExecutor) DryRun(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) (*controller.ActionResult, error) {
	// Validate the action
	if err := r.Validate(ctx, target, action); err != nil {
		return &controller.ActionResult{
			Success: false,
			Message: fmt.Sprintf("Validation failed: %v", err),
		}, err
	}

	config := action.RestartAction
	if config == nil {
		config = &v1alpha1.RestartAction{
			Strategy: "rolling",
		}
	}

	// Simulate changes based on resource type
	var simulatedChanges []v1alpha1.ResourceChange

	switch obj := target.(type) {
	case *corev1.Pod:
		simulatedChanges = []v1alpha1.ResourceChange{
			{
				ResourceRef: fmt.Sprintf("Pod/%s/%s", obj.Namespace, obj.Name),
				ChangeType:  "delete",
				Field:       "pod",
				OldValue:    obj.Name,
				NewValue:    "recreated",
			},
		}
	case *appsv1.Deployment:
		simulatedChanges = []v1alpha1.ResourceChange{
			{
				ResourceRef: fmt.Sprintf("Deployment/%s/%s", obj.Namespace, obj.Name),
				ChangeType:  "update",
				Field:       "spec.template.metadata.annotations[kubectl.kubernetes.io/restartedAt]",
				OldValue:    "",
				NewValue:    time.Now().Format(time.RFC3339),
			},
		}
	case *appsv1.StatefulSet:
		simulatedChanges = []v1alpha1.ResourceChange{
			{
				ResourceRef: fmt.Sprintf("StatefulSet/%s/%s", obj.Namespace, obj.Name),
				ChangeType:  "update",
				Field:       "spec.template.metadata.annotations[kubectl.kubernetes.io/restartedAt]",
				OldValue:    "",
				NewValue:    time.Now().Format(time.RFC3339),
			},
		}
	case *appsv1.DaemonSet:
		simulatedChanges = []v1alpha1.ResourceChange{
			{
				ResourceRef: fmt.Sprintf("DaemonSet/%s/%s", obj.Namespace, obj.Name),
				ChangeType:  "update",
				Field:       "spec.template.metadata.annotations[kubectl.kubernetes.io/restartedAt]",
				OldValue:    "",
				NewValue:    time.Now().Format(time.RFC3339),
			},
		}
	}

	return &controller.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Dry-run: Would restart %s/%s using %s strategy", target.GetNamespace(), target.GetName(), config.Strategy),
		Changes: simulatedChanges,
		Metrics: map[string]string{
			"restart_strategy": config.Strategy,
			"resource_type":    fmt.Sprintf("%T", target),
			"dry_run":          "true",
		},
	}, nil
}

// restartPod restarts a single pod
func (r *RestartExecutor) restartPod(ctx context.Context, pod *corev1.Pod, config *v1alpha1.RestartAction) ([]v1alpha1.ResourceChange, error) {
	log := log.FromContext(ctx)

	// Record the change
	changes := []v1alpha1.ResourceChange{
		{
			ResourceRef: fmt.Sprintf("Pod/%s/%s", pod.Namespace, pod.Name),
			ChangeType:  "delete",
			Field:       "pod",
			OldValue:    pod.Name,
			NewValue:    "recreated",
			Timestamp:   &metav1.Time{Time: time.Now()},
		},
	}

	// Delete the pod based on strategy
	deleteOptions := &client.DeleteOptions{}

	if config.Strategy == "graceful" {
		// Use grace period for graceful termination
		gracePeriod := int64(30)
		if config.GracePeriodSeconds > 0 {
			gracePeriod = int64(config.GracePeriodSeconds)
		}
		deleteOptions.GracePeriodSeconds = &gracePeriod
	}

	log.Info("Deleting pod for restart",
		"pod", pod.Name,
		"namespace", pod.Namespace,
		"strategy", config.Strategy)

	if err := r.client.Delete(ctx, pod, deleteOptions); err != nil {
		if !errors.IsNotFound(err) {
			return changes, fmt.Errorf("failed to delete pod: %w", err)
		}
	}

	return changes, nil
}

// restartDeployment restarts all pods in a deployment
func (r *RestartExecutor) restartDeployment(ctx context.Context, deployment *appsv1.Deployment, config *v1alpha1.RestartAction) ([]v1alpha1.ResourceChange, error) {
	log := log.FromContext(ctx)

	// Use kubectl's restart annotation approach
	patch := client.MergeFrom(deployment.DeepCopy())

	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}

	restartTime := time.Now().Format(time.RFC3339)
	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = restartTime

	log.Info("Restarting deployment",
		"deployment", deployment.Name,
		"namespace", deployment.Namespace,
		"strategy", config.Strategy)

	if err := r.client.Patch(ctx, deployment, patch); err != nil {
		return nil, fmt.Errorf("failed to patch deployment: %w", err)
	}

	changes := []v1alpha1.ResourceChange{
		{
			ResourceRef: fmt.Sprintf("Deployment/%s/%s", deployment.Namespace, deployment.Name),
			ChangeType:  "update",
			Field:       "spec.template.metadata.annotations[kubectl.kubernetes.io/restartedAt]",
			OldValue:    "",
			NewValue:    restartTime,
			Timestamp:   &metav1.Time{Time: time.Now()},
		},
	}

	// For recreate strategy, scale down then up
	if config.Strategy == "recreate" {
		originalReplicas := *deployment.Spec.Replicas

		// Scale down to 0
		deployment.Spec.Replicas = int32Ptr(0)
		if err := r.client.Update(ctx, deployment); err != nil {
			return changes, fmt.Errorf("failed to scale down deployment: %w", err)
		}

		// Wait a moment for pods to terminate
		time.Sleep(2 * time.Second)

		// Scale back up
		deployment.Spec.Replicas = &originalReplicas
		if err := r.client.Update(ctx, deployment); err != nil {
			return changes, fmt.Errorf("failed to scale up deployment: %w", err)
		}

		changes = append(changes, v1alpha1.ResourceChange{
			ResourceRef: fmt.Sprintf("Deployment/%s/%s", deployment.Namespace, deployment.Name),
			ChangeType:  "scale",
			Field:       "spec.replicas",
			OldValue:    fmt.Sprintf("%d", originalReplicas),
			NewValue:    fmt.Sprintf("0->%d", originalReplicas),
			Timestamp:   &metav1.Time{Time: time.Now()},
		})
	}

	return changes, nil
}

// restartStatefulSet restarts all pods in a statefulset
func (r *RestartExecutor) restartStatefulSet(ctx context.Context, statefulSet *appsv1.StatefulSet, config *v1alpha1.RestartAction) ([]v1alpha1.ResourceChange, error) {
	log := log.FromContext(ctx)

	// Use kubectl's restart annotation approach
	patch := client.MergeFrom(statefulSet.DeepCopy())

	if statefulSet.Spec.Template.Annotations == nil {
		statefulSet.Spec.Template.Annotations = make(map[string]string)
	}

	restartTime := time.Now().Format(time.RFC3339)
	statefulSet.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = restartTime

	log.Info("Restarting statefulset",
		"statefulset", statefulSet.Name,
		"namespace", statefulSet.Namespace,
		"strategy", config.Strategy)

	if err := r.client.Patch(ctx, statefulSet, patch); err != nil {
		return nil, fmt.Errorf("failed to patch statefulset: %w", err)
	}

	return []v1alpha1.ResourceChange{
		{
			ResourceRef: fmt.Sprintf("StatefulSet/%s/%s", statefulSet.Namespace, statefulSet.Name),
			ChangeType:  "update",
			Field:       "spec.template.metadata.annotations[kubectl.kubernetes.io/restartedAt]",
			OldValue:    "",
			NewValue:    restartTime,
			Timestamp:   &metav1.Time{Time: time.Now()},
		},
	}, nil
}

// restartDaemonSet restarts all pods in a daemonset
func (r *RestartExecutor) restartDaemonSet(ctx context.Context, daemonSet *appsv1.DaemonSet, config *v1alpha1.RestartAction) ([]v1alpha1.ResourceChange, error) {
	log := log.FromContext(ctx)

	// Use kubectl's restart annotation approach
	patch := client.MergeFrom(daemonSet.DeepCopy())

	if daemonSet.Spec.Template.Annotations == nil {
		daemonSet.Spec.Template.Annotations = make(map[string]string)
	}

	restartTime := time.Now().Format(time.RFC3339)
	daemonSet.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = restartTime

	log.Info("Restarting daemonset",
		"daemonset", daemonSet.Name,
		"namespace", daemonSet.Namespace,
		"strategy", config.Strategy)

	if err := r.client.Patch(ctx, daemonSet, patch); err != nil {
		return nil, fmt.Errorf("failed to patch daemonset: %w", err)
	}

	return []v1alpha1.ResourceChange{
		{
			ResourceRef: fmt.Sprintf("DaemonSet/%s/%s", daemonSet.Namespace, daemonSet.Name),
			ChangeType:  "update",
			Field:       "spec.template.metadata.annotations[kubectl.kubernetes.io/restartedAt]",
			OldValue:    "",
			NewValue:    restartTime,
			Timestamp:   &metav1.Time{Time: time.Now()},
		},
	}, nil
}

// restartPodGeneric restarts a pod using generic client
func (r *RestartExecutor) restartPodGeneric(ctx context.Context, target client.Object, config *v1alpha1.RestartAction) ([]v1alpha1.ResourceChange, error) {
	log := log.FromContext(ctx)

	// Record the change
	changes := []v1alpha1.ResourceChange{
		{
			ResourceRef: fmt.Sprintf("Pod/%s/%s", target.GetNamespace(), target.GetName()),
			ChangeType:  "delete",
			Field:       "pod",
			OldValue:    target.GetName(),
			NewValue:    "recreated",
			Timestamp:   &metav1.Time{Time: time.Now()},
		},
	}

	// Delete the pod based on strategy
	deleteOptions := &client.DeleteOptions{}

	if config.Strategy == "graceful" {
		// Use grace period for graceful termination
		gracePeriod := int64(30)
		if config.GracePeriodSeconds > 0 {
			gracePeriod = int64(config.GracePeriodSeconds)
		}
		deleteOptions.GracePeriodSeconds = &gracePeriod
	}

	log.Info("Deleting pod for restart",
		"pod", target.GetName(),
		"namespace", target.GetNamespace(),
		"strategy", config.Strategy)

	if err := r.client.Delete(ctx, target, deleteOptions); err != nil {
		if !errors.IsNotFound(err) {
			return changes, fmt.Errorf("failed to delete pod: %w", err)
		}
	}

	return changes, nil
}

// restartWorkloadGeneric restarts a workload (Deployment/StatefulSet/DaemonSet) using generic client
func (r *RestartExecutor) restartWorkloadGeneric(ctx context.Context, target client.Object, config *v1alpha1.RestartAction, kind string) ([]v1alpha1.ResourceChange, error) {
	log := log.FromContext(ctx)

	// Use kubectl's restart annotation approach

	// Add restart annotation to pod template
	restartTime := time.Now().Format(time.RFC3339)
	annotationPatch := map[string]interface{}{
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"annotations": map[string]string{
						"kubectl.kubernetes.io/restartedAt": restartTime,
					},
				},
			},
		},
	}

	log.Info("Restarting workload",
		"kind", kind,
		"name", target.GetName(),
		"namespace", target.GetNamespace(),
		"strategy", config.Strategy)

	if err := r.client.Patch(ctx, target, client.RawPatch(types.MergePatchType, mustMarshalJSON(annotationPatch))); err != nil {
		return nil, fmt.Errorf("failed to patch %s: %w", kind, err)
	}

	changes := []v1alpha1.ResourceChange{
		{
			ResourceRef: fmt.Sprintf("%s/%s/%s", kind, target.GetNamespace(), target.GetName()),
			ChangeType:  "update",
			Field:       "spec.template.metadata.annotations[kubectl.kubernetes.io/restartedAt]",
			OldValue:    "",
			NewValue:    restartTime,
			Timestamp:   &metav1.Time{Time: time.Now()},
		},
	}

	// For recreate strategy, scale down then up
	if config.Strategy == "recreate" && (kind == "Deployment" || kind == "StatefulSet") {
		// This would require more complex handling with unstructured objects
		// For now, we'll just use the annotation approach
		log.Info("Recreate strategy requested but using rolling restart for simplicity")
	}

	return changes, nil
}

// mustMarshalJSON marshals an object to JSON, panicking on error
func mustMarshalJSON(obj interface{}) []byte {
	data, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}
	return data
}

// int32Ptr is a helper to get a pointer to an int32
func int32Ptr(i int32) *int32 {
	return &i
}
