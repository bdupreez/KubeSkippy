package safety

import (
	"context"
	"fmt"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
	"github.com/kubeskippy/kubeskippy/internal/controller"
	"github.com/kubeskippy/kubeskippy/pkg/config"
)

// Controller implements the SafetyController interface
type Controller struct {
	client       client.Client
	config       config.SafetyConfig
	store        ActionStore
	auditLogger  AuditLogger
	
	// Circuit breakers per policy
	circuitBreakers map[string]*controller.CircuitBreaker
	cbMutex         sync.RWMutex
}

// NewController creates a new safety controller
func NewController(client client.Client, config config.SafetyConfig, store ActionStore, auditLogger AuditLogger) *Controller {
	if store == nil {
		store = NewInMemoryActionStore()
	}
	
	if auditLogger == nil {
		auditLogger = &defaultAuditLogger{}
	}

	return &Controller{
		client:          client,
		config:          config,
		store:           store,
		auditLogger:     auditLogger,
		circuitBreakers: make(map[string]*controller.CircuitBreaker),
	}
}

// ValidateAction checks if an action is safe to execute
func (c *Controller) ValidateAction(ctx context.Context, action *v1alpha1.HealingAction) (*controller.ValidationResult, error) {
	log := log.FromContext(ctx)
	
	result := &controller.ValidationResult{
		Valid:    true,
		Warnings: []string{},
	}

	// Check if in dry-run mode
	if c.config.DryRunMode && !action.Spec.DryRun {
		result.Valid = false
		result.Reason = "System is in dry-run mode only"
		c.auditLogger.LogValidation(ctx, action, false, result.Reason)
		return result, nil
	}

	// Get the target resource
	target, err := c.getTargetResource(ctx, action)
	if err != nil {
		result.Valid = false
		result.Reason = fmt.Sprintf("Failed to get target resource: %v", err)
		c.auditLogger.LogValidation(ctx, action, false, result.Reason)
		return result, nil
	}

	// Check if resource is protected
	if protected, reason := c.IsProtectedResource(target); protected {
		result.Valid = false
		result.Reason = fmt.Sprintf("Resource is protected: %s", reason)
		c.auditLogger.LogValidation(ctx, action, false, result.Reason)
		return result, nil
	}

	// Check circuit breaker
	cb := c.getOrCreateCircuitBreaker(action.Spec.PolicyRef.Name)
	if err := cb.Call(ctx, func() error { return nil }); err != nil {
		result.Valid = false
		result.Reason = fmt.Sprintf("Circuit breaker is open: %v", err)
		result.Warnings = append(result.Warnings, "Too many failures detected")
		c.auditLogger.LogValidation(ctx, action, false, result.Reason)
		return result, nil
	}

	// Validate action type specific rules
	if err := c.validateActionType(action, target); err != nil {
		result.Valid = false
		result.Reason = err.Error()
		c.auditLogger.LogValidation(ctx, action, false, result.Reason)
		return result, nil
	}

	// Check if approval is enforced globally
	if c.config.RequireApproval && !action.Spec.DryRun {
		if action.Spec.ApprovalRequired || action.Status.Approval == nil || !action.Status.Approval.Approved {
			result.Warnings = append(result.Warnings, "Action requires manual approval")
		}
	}

	// Add warnings for risky operations
	if action.Spec.Action.Type == "delete" {
		result.Warnings = append(result.Warnings, "Delete operations are potentially destructive")
	}

	log.Info("Action validation completed", 
		"action", action.Name,
		"valid", result.Valid,
		"warnings", len(result.Warnings))

	c.auditLogger.LogValidation(ctx, action, result.Valid, result.Reason)
	return result, nil
}

// CheckRateLimit verifies action frequency limits
func (c *Controller) CheckRateLimit(ctx context.Context, policy *v1alpha1.HealingPolicy) (bool, error) {
	policyKey := getPolicyKey(policy)
	
	// Determine the rate limit
	limit := c.config.MaxActionsPerHour
	if policy.Spec.SafetyRules.MaxActionsPerHour > 0 {
		limit = int(policy.Spec.SafetyRules.MaxActionsPerHour)
	}

	// Get action count in the last hour
	since := time.Now().Add(-1 * time.Hour)
	count, err := c.store.GetActionCount(ctx, policyKey, since)
	if err != nil {
		return false, fmt.Errorf("failed to get action count: %w", err)
	}

	allowed := count < limit
	c.auditLogger.LogRateLimit(ctx, policyKey, allowed, count, limit)

	if !allowed {
		log.FromContext(ctx).Info("Rate limit exceeded",
			"policy", policyKey,
			"current", count,
			"limit", limit)
	}

	return allowed, nil
}

// IsProtectedResource checks if a resource is protected
func (c *Controller) IsProtectedResource(resource runtime.Object) (bool, string) {
	obj, ok := resource.(client.Object)
	if !ok {
		// For testing purposes, check if it's a GenericResource
		if gr, ok := resource.(*GenericResource); ok {
			// Check protected namespaces
			for _, ns := range c.config.ProtectedNamespaces {
				if gr.GetNamespace() == ns {
					return true, fmt.Sprintf("namespace %s is protected", ns)
				}
			}

			// Check protected labels
			labels := gr.GetLabels()
			for k, v := range c.config.ProtectedLabels {
				if labels[k] == v {
					return true, fmt.Sprintf("has protected label %s=%s", k, v)
				}
			}

			// Check protected annotation
			annotations := gr.GetAnnotations()
			if annotations[controller.AnnotationProtected] == "true" {
				return true, "has protected annotation"
			}

			// Check if healing is disabled
			if annotations[controller.AnnotationHealingDisabled] == "true" {
				return true, "healing is disabled via annotation"
			}
		}
		return false, ""
	}

	// Check protected namespaces
	for _, ns := range c.config.ProtectedNamespaces {
		if obj.GetNamespace() == ns {
			return true, fmt.Sprintf("namespace %s is protected", ns)
		}
	}

	// Check protected labels
	labels := obj.GetLabels()
	for k, v := range c.config.ProtectedLabels {
		if labels[k] == v {
			return true, fmt.Sprintf("has protected label %s=%s", k, v)
		}
	}

	// Check protected annotation
	annotations := obj.GetAnnotations()
	if annotations[controller.AnnotationProtected] == "true" {
		return true, "has protected annotation"
	}

	// Check if healing is disabled
	if annotations[controller.AnnotationHealingDisabled] == "true" {
		return true, "healing is disabled via annotation"
	}

	return false, ""
}

// RecordAction logs an executed action
func (c *Controller) RecordAction(ctx context.Context, action *v1alpha1.HealingAction, result *controller.ActionResult) {
	policyKey := fmt.Sprintf("%s/%s", action.Spec.PolicyRef.Namespace, action.Spec.PolicyRef.Name)
	targetKey := fmt.Sprintf("%s/%s/%s", 
		action.Spec.TargetResource.Kind,
		action.Spec.TargetResource.Namespace,
		action.Spec.TargetResource.Name)

	record := ActionRecord{
		PolicyKey:  policyKey,
		ActionName: action.Spec.Action.Name,
		ActionType: action.Spec.Action.Type,
		TargetKey:  targetKey,
		Success:    result.Success,
		Timestamp:  result.StartTime,
		DurationMS: result.EndTime.Sub(result.StartTime).Milliseconds(),
		DryRun:     action.Spec.DryRun,
	}

	if !result.Success && result.Error != nil {
		record.Error = result.Error.Error()
	}

	if action.Status.Approval != nil && action.Status.Approval.ApprovedBy != "" {
		record.ApprovedBy = action.Status.Approval.ApprovedBy
	}

	if err := c.store.RecordAction(ctx, record); err != nil {
		log.FromContext(ctx).Error(err, "Failed to record action")
	}

	// Update circuit breaker based on result
	cb := c.getOrCreateCircuitBreaker(action.Spec.PolicyRef.Name)
	if result.Success {
		cb.Call(ctx, func() error { return nil })
	} else {
		cb.Call(ctx, func() error { return fmt.Errorf("action failed") })
	}

	// Log audit
	details := map[string]interface{}{
		"duration_ms": record.DurationMS,
		"dry_run":     record.DryRun,
		"target":      targetKey,
	}
	c.auditLogger.LogAction(ctx, action, fmt.Sprintf("success=%v", result.Success), details)
}

// getTargetResource retrieves the target resource for validation
func (c *Controller) getTargetResource(ctx context.Context, action *v1alpha1.HealingAction) (runtime.Object, error) {
	// In a real implementation, this would use dynamic client to fetch any resource type
	// For now, we'll return a minimal implementation
	
	// This is a simplified version - in production, you'd use dynamic client
	obj := &GenericResource{
		namespace: action.Spec.TargetResource.Namespace,
		name:      action.Spec.TargetResource.Name,
		labels:    make(map[string]string),
		annotations: make(map[string]string),
	}
	
	return obj, nil
}

// validateActionType performs action-type specific validation
func (c *Controller) validateActionType(action *v1alpha1.HealingAction, target runtime.Object) error {
	switch action.Spec.Action.Type {
	case "delete":
		// Never allow delete in certain cases
		if action.Spec.TargetResource.Kind == "PersistentVolume" {
			return fmt.Errorf("deleting PersistentVolumes is not allowed")
		}
		if action.Spec.TargetResource.Kind == "CustomResourceDefinition" {
			return fmt.Errorf("deleting CRDs is not allowed")
		}
		
	case "scale":
		// Validate scale actions have proper parameters
		if action.Spec.Action.ScaleAction == nil {
			return fmt.Errorf("scale action missing configuration")
		}
		
	case "patch":
		// Validate patch actions
		if action.Spec.Action.PatchAction == nil {
			return fmt.Errorf("patch action missing configuration")
		}
	}
	
	return nil
}

// getOrCreateCircuitBreaker gets or creates a circuit breaker for a policy
func (c *Controller) getOrCreateCircuitBreaker(policyName string) *controller.CircuitBreaker {
	c.cbMutex.RLock()
	cb, exists := c.circuitBreakers[policyName]
	c.cbMutex.RUnlock()

	if exists {
		return cb
	}

	c.cbMutex.Lock()
	defer c.cbMutex.Unlock()

	// Double-check after acquiring write lock
	if cb, exists = c.circuitBreakers[policyName]; exists {
		return cb
	}

	// Create new circuit breaker
	cb = controller.NewCircuitBreaker(
		c.config.CircuitBreaker.FailureThreshold,
		c.config.CircuitBreaker.SuccessThreshold,
		c.config.CircuitBreaker.Timeout,
	)
	c.circuitBreakers[policyName] = cb

	return cb
}

// getPolicyKey generates a unique key for a policy
func getPolicyKey(policy *v1alpha1.HealingPolicy) string {
	return fmt.Sprintf("%s/%s", policy.Namespace, policy.Name)
}

// GenericResource is a minimal implementation for testing
type GenericResource struct {
	namespace   string
	name        string
	labels      map[string]string
	annotations map[string]string
}

func (g *GenericResource) GetObjectKind() schema.ObjectKind {
	return schema.EmptyObjectKind
}

func (g *GenericResource) DeepCopyObject() runtime.Object {
	return &GenericResource{
		namespace:   g.namespace,
		name:        g.name,
		labels:      g.labels,
		annotations: g.annotations,
	}
}

func (g *GenericResource) GetNamespace() string {
	return g.namespace
}

func (g *GenericResource) GetName() string {
	return g.name
}

func (g *GenericResource) GetLabels() map[string]string {
	return g.labels
}

func (g *GenericResource) GetAnnotations() map[string]string {
	return g.annotations
}

func (g *GenericResource) SetNamespace(namespace string) {
	g.namespace = namespace
}

func (g *GenericResource) SetName(name string) {
	g.name = name
}

func (g *GenericResource) SetLabels(labels map[string]string) {
	g.labels = labels
}

func (g *GenericResource) SetAnnotations(annotations map[string]string) {
	g.annotations = annotations
}

func (g *GenericResource) GetUID() types.UID {
	return ""
}

func (g *GenericResource) SetUID(uid types.UID) {}

func (g *GenericResource) GetResourceVersion() string {
	return ""
}

func (g *GenericResource) SetResourceVersion(version string) {}

func (g *GenericResource) GetGeneration() int64 {
	return 0
}

func (g *GenericResource) SetGeneration(generation int64) {}

func (g *GenericResource) GetSelfLink() string {
	return ""
}

func (g *GenericResource) SetSelfLink(selfLink string) {}

func (g *GenericResource) GetCreationTimestamp() metav1.Time {
	return metav1.Time{}
}

func (g *GenericResource) SetCreationTimestamp(timestamp metav1.Time) {}

func (g *GenericResource) GetDeletionTimestamp() *metav1.Time {
	return nil
}

func (g *GenericResource) SetDeletionTimestamp(timestamp *metav1.Time) {}

func (g *GenericResource) GetDeletionGracePeriodSeconds() *int64 {
	return nil
}

func (g *GenericResource) SetDeletionGracePeriodSeconds(i *int64) {}

func (g *GenericResource) GetFinalizers() []string {
	return nil
}

func (g *GenericResource) SetFinalizers(finalizers []string) {}

func (g *GenericResource) GetOwnerReferences() []metav1.OwnerReference {
	return nil
}

func (g *GenericResource) SetOwnerReferences(references []metav1.OwnerReference) {}

func (g *GenericResource) GetClusterName() string {
	return ""
}

func (g *GenericResource) SetClusterName(clusterName string) {}

func (g *GenericResource) GetManagedFields() []metav1.ManagedFieldsEntry {
	return nil
}

func (g *GenericResource) SetManagedFields(managedFields []metav1.ManagedFieldsEntry) {}

// defaultAuditLogger provides basic audit logging
type defaultAuditLogger struct{}

func (d *defaultAuditLogger) LogAction(ctx context.Context, action *v1alpha1.HealingAction, result string, details map[string]interface{}) {
	log := log.FromContext(ctx)
	log.Info("Audit: Action executed",
		"action", action.Name,
		"type", action.Spec.Action.Type,
		"result", result,
		"details", details)
}

func (d *defaultAuditLogger) LogValidation(ctx context.Context, action *v1alpha1.HealingAction, valid bool, reason string) {
	log := log.FromContext(ctx)
	log.Info("Audit: Action validated",
		"action", action.Name,
		"valid", valid,
		"reason", reason)
}

func (d *defaultAuditLogger) LogRateLimit(ctx context.Context, policyKey string, allowed bool, current int, limit int) {
	log := log.FromContext(ctx)
	log.Info("Audit: Rate limit check",
		"policy", policyKey,
		"allowed", allowed,
		"current", current,
		"limit", limit)
}

// StartCleanupLoop starts a background loop to clean up old records
func (c *Controller) StartCleanupLoop(ctx context.Context, retention time.Duration) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				before := time.Now().Add(-retention)
				if err := c.store.CleanupOldRecords(ctx, before); err != nil {
					log.FromContext(ctx).Error(err, "Failed to cleanup old records")
				}
			}
		}
	}()
}