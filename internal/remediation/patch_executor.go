package remediation

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
	"github.com/kubeskippy/kubeskippy/internal/controller"
)

// PatchExecutor handles patch actions
type PatchExecutor struct {
	client client.Client
}

// NewPatchExecutor creates a new patch executor
func NewPatchExecutor(client client.Client) *PatchExecutor {
	return &PatchExecutor{
		client: client,
	}
}

// Execute performs the patch action
func (p *PatchExecutor) Execute(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) (*controller.ActionResult, error) {
	log := log.FromContext(ctx)
	startTime := time.Now()

	// Get patch configuration
	config := action.PatchAction
	if config == nil {
		return &controller.ActionResult{
			Success:   false,
			Message:   "Patch action configuration is missing",
			StartTime: startTime,
			EndTime:   time.Now(),
		}, fmt.Errorf("patch action configuration is missing")
	}

	// Create unstructured object for patching
	unstructuredTarget, err := p.toUnstructured(target)
	if err != nil {
		return &controller.ActionResult{
			Success:   false,
			Message:   fmt.Sprintf("Failed to convert to unstructured: %v", err),
			Error:     err,
			StartTime: startTime,
			EndTime:   time.Now(),
		}, err
	}

	// Store original values for change tracking
	originalValues := make(map[string]interface{})
	changes := []v1alpha1.ResourceChange{}

	// Apply patches
	for _, patch := range config.Patches {
		// Get original value
		originalValue, exists, err := unstructured.NestedFieldCopy(unstructuredTarget.Object, patch.Path...)
		if err != nil {
			log.Error(err, "Failed to get original value", "path", patch.Path)
			originalValue = nil
		}
		
		if exists {
			originalValues[pathToString(patch.Path)] = originalValue
		}

		// Parse the new value
		var newValue interface{}
		if err := json.Unmarshal([]byte(patch.Value), &newValue); err != nil {
			// If JSON parsing fails, treat as string
			newValue = patch.Value
		}

		// Apply the patch
		if err := unstructured.SetNestedField(unstructuredTarget.Object, newValue, patch.Path...); err != nil {
			return &controller.ActionResult{
				Success:   false,
				Message:   fmt.Sprintf("Failed to set field %s: %v", pathToString(patch.Path), err),
				Error:     err,
				Changes:   changes,
				StartTime: startTime,
				EndTime:   time.Now(),
			}, err
		}

		// Record the change
		changes = append(changes, v1alpha1.ResourceChange{
			ResourceRef: fmt.Sprintf("%s/%s/%s", target.GetObjectKind().GroupVersionKind().Kind, target.GetNamespace(), target.GetName()),
			ChangeType:  "update",
			Field:       pathToString(patch.Path),
			OldValue:    fmt.Sprintf("%v", originalValue),
			NewValue:    patch.Value,
			Timestamp:   &metav1.Time{Time: time.Now()},
		})
	}

	// Update the resource
	if err := p.client.Update(ctx, unstructuredTarget); err != nil {
		return &controller.ActionResult{
			Success:   false,
			Message:   fmt.Sprintf("Failed to update resource: %v", err),
			Error:     err,
			Changes:   changes,
			StartTime: startTime,
			EndTime:   time.Now(),
		}, err
	}

	log.Info("Resource patched successfully",
		"resource", fmt.Sprintf("%s/%s", target.GetNamespace(), target.GetName()),
		"patches", len(config.Patches))

	return &controller.ActionResult{
		Success:   true,
		Message:   fmt.Sprintf("Successfully patched %s/%s with %d patches", target.GetNamespace(), target.GetName(), len(config.Patches)),
		Changes:   changes,
		StartTime: startTime,
		EndTime:   time.Now(),
		Metrics: map[string]string{
			"patch_count":  fmt.Sprintf("%d", len(config.Patches)),
			"patch_type":   string(config.Type),
		},
	}, nil
}

// Validate checks if the patch action can be executed
func (p *PatchExecutor) Validate(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) error {
	// Validate patch configuration
	if action.PatchAction == nil {
		return fmt.Errorf("patch action configuration is missing")
	}

	config := action.PatchAction

	// Validate patch type
	switch config.Type {
	case "json", "merge", "strategic":
		// Valid patch types
	default:
		return fmt.Errorf("invalid patch type: %s", config.Type)
	}

	// Validate patches
	if len(config.Patches) == 0 {
		return fmt.Errorf("no patches specified")
	}

	// For JSON patches, validate the path format
	if config.Type == "json" {
		for i, patch := range config.Patches {
			if len(patch.Path) == 0 {
				return fmt.Errorf("patch %d has empty path", i)
			}
		}
	}

	// Try to convert to unstructured to ensure it's possible
	if _, err := p.toUnstructured(target); err != nil {
		return fmt.Errorf("cannot convert target to unstructured: %w", err)
	}

	return nil
}

// DryRun simulates the patch action
func (p *PatchExecutor) DryRun(ctx context.Context, target client.Object, action *v1alpha1.HealingActionTemplate) (*controller.ActionResult, error) {
	// Validate the action
	if err := p.Validate(ctx, target, action); err != nil {
		return &controller.ActionResult{
			Success: false,
			Message: fmt.Sprintf("Validation failed: %v", err),
		}, err
	}

	config := action.PatchAction

	// Create unstructured object for simulation
	unstructuredTarget, err := p.toUnstructured(target)
	if err != nil {
		return &controller.ActionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to convert to unstructured: %v", err),
		}, err
	}

	// Simulate patches
	simulatedChanges := []v1alpha1.ResourceChange{}
	
	for _, patch := range config.Patches {
		// Get current value
		currentValue, exists, _ := unstructured.NestedFieldCopy(unstructuredTarget.Object, patch.Path...)
		
		oldValue := "<not set>"
		if exists {
			oldValue = fmt.Sprintf("%v", currentValue)
		}

		simulatedChanges = append(simulatedChanges, v1alpha1.ResourceChange{
			ResourceRef: fmt.Sprintf("%s/%s/%s", target.GetObjectKind().GroupVersionKind().Kind, target.GetNamespace(), target.GetName()),
			ChangeType:  "update",
			Field:       pathToString(patch.Path),
			OldValue:    oldValue,
			NewValue:    patch.Value,
		})
	}

	return &controller.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Dry-run: Would apply %d patches to %s/%s", len(config.Patches), target.GetNamespace(), target.GetName()),
		Changes: simulatedChanges,
		Metrics: map[string]string{
			"patch_count": fmt.Sprintf("%d", len(config.Patches)),
			"patch_type":  string(config.Type),
			"dry_run":     "true",
		},
	}, nil
}

// toUnstructured converts a client.Object to unstructured
func (p *PatchExecutor) toUnstructured(obj client.Object) (*unstructured.Unstructured, error) {
	// If already unstructured, return as is
	if u, ok := obj.(*unstructured.Unstructured); ok {
		return u, nil
	}

	// Convert to unstructured
	content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	u := &unstructured.Unstructured{Object: content}
	u.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	u.SetNamespace(obj.GetNamespace())
	u.SetName(obj.GetName())
	
	return u, nil
}

// pathToString converts a path array to a dot-notation string
func pathToString(path []string) string {
	result := ""
	for i, p := range path {
		if i > 0 {
			result += "."
		}
		result += p
	}
	return result
}

// compareValues compares two values for equality
func compareValues(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}