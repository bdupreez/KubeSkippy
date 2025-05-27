package safety

import (
	"context"
	"time"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
)

// ActionStore defines the interface for storing and retrieving action history
type ActionStore interface {
	// RecordAction stores an executed action
	RecordAction(ctx context.Context, record ActionRecord) error

	// GetActionCount returns the number of actions for a policy in the time window
	GetActionCount(ctx context.Context, policyKey string, since time.Time) (int, error)

	// GetRecentActions returns recent actions for a policy
	GetRecentActions(ctx context.Context, policyKey string, limit int) ([]ActionRecord, error)

	// GetLastAction returns the most recent action for a policy
	GetLastAction(ctx context.Context, policyKey string) (*ActionRecord, error)

	// CleanupOldRecords removes records older than the retention period
	CleanupOldRecords(ctx context.Context, before time.Time) error
}

// ActionRecord represents a recorded healing action
type ActionRecord struct {
	ID         string
	PolicyKey  string
	ActionName string
	ActionType string
	TargetKey  string
	Success    bool
	Error      string
	Timestamp  time.Time
	DurationMS int64
	ApprovedBy string
	DryRun     bool
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	// LogAction logs an action execution
	LogAction(ctx context.Context, action *v1alpha1.HealingAction, result string, details map[string]interface{})

	// LogValidation logs a validation result
	LogValidation(ctx context.Context, action *v1alpha1.HealingAction, valid bool, reason string)

	// LogRateLimit logs a rate limit event
	LogRateLimit(ctx context.Context, policyKey string, allowed bool, current int, limit int)
}

// ValidationContext provides additional context for validation
type ValidationContext struct {
	// DryRun indicates if this is a dry-run validation
	DryRun bool

	// Force bypasses certain safety checks (use with caution)
	Force bool

	// Username of the approver (if manual approval)
	ApprovedBy string

	// Additional metadata
	Metadata map[string]string
}

// RateLimitResult contains rate limit check details
type RateLimitResult struct {
	Allowed       bool
	CurrentCount  int
	Limit         int
	WindowStart   time.Time
	WindowEnd     time.Time
	NextAllowedAt *time.Time
}
