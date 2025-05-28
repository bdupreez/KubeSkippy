package config

import (
	"time"
)

// Config holds the operator configuration
type Config struct {
	// MetricsAddr is the address the metric endpoint binds to
	MetricsAddr string `json:"metricsAddr,omitempty"`

	// ProbeAddr is the address the probe endpoint binds to
	ProbeAddr string `json:"probeAddr,omitempty"`

	// EnableLeaderElection enables leader election for controller manager
	EnableLeaderElection bool `json:"enableLeaderElection,omitempty"`

	// Namespace to watch (empty means all namespaces)
	WatchNamespace string `json:"watchNamespace,omitempty"`

	// MetricsCollector configuration
	Metrics MetricsConfig `json:"metrics,omitempty"`

	// AI configuration
	AI AIConfig `json:"ai,omitempty"`

	// Safety configuration
	Safety SafetyConfig `json:"safety,omitempty"`

	// Remediation configuration
	Remediation RemediationConfig `json:"remediation,omitempty"`

	// Logging configuration
	Logging LoggingConfig `json:"logging,omitempty"`
}

// MetricsConfig configures the metrics collector
type MetricsConfig struct {
	// PrometheusURL is the Prometheus server URL
	PrometheusURL string `json:"prometheusURL,omitempty"`

	// MetricsServerEnabled enables metrics-server integration
	MetricsServerEnabled bool `json:"metricsServerEnabled,omitempty"`

	// CollectionInterval is how often to collect metrics
	CollectionInterval time.Duration `json:"collectionInterval,omitempty"`

	// RetentionPeriod is how long to keep metrics
	RetentionPeriod time.Duration `json:"retentionPeriod,omitempty"`

	// CustomQueries for additional Prometheus queries
	CustomQueries map[string]string `json:"customQueries,omitempty"`
}

// AIConfig configures the AI integration
type AIConfig struct {
	// Provider (ollama, openai, etc.)
	Provider string `json:"provider,omitempty"`

	// Model to use
	Model string `json:"model,omitempty"`

	// Endpoint URL
	Endpoint string `json:"endpoint,omitempty"`

	// APIKey for authentication (if needed)
	APIKey string `json:"apiKey,omitempty"`

	// Timeout for AI requests
	Timeout time.Duration `json:"timeout,omitempty"`

	// MaxTokens limit
	MaxTokens int `json:"maxTokens,omitempty"`

	// Temperature for generation
	Temperature float32 `json:"temperature,omitempty"`

	// SystemPrompt base prompt
	SystemPrompt string `json:"systemPrompt,omitempty"`

	// MinConfidence for accepting AI recommendations
	MinConfidence float32 `json:"minConfidence,omitempty"`

	// ValidateResponses enables response validation
	ValidateResponses bool `json:"validateResponses,omitempty"`
}

// SafetyConfig configures safety controls
type SafetyConfig struct {
	// DryRunMode enables dry-run only operation
	DryRunMode bool `json:"dryRunMode,omitempty"`

	// MaxActionsPerHour global limit
	MaxActionsPerHour int `json:"maxActionsPerHour,omitempty"`

	// RequireApproval for all actions
	RequireApproval bool `json:"requireApproval,omitempty"`

	// ProtectedNamespaces that cannot be modified
	ProtectedNamespaces []string `json:"protectedNamespaces,omitempty"`

	// ProtectedLabels that mark protected resources
	ProtectedLabels map[string]string `json:"protectedLabels,omitempty"`

	// CircuitBreaker configuration
	CircuitBreaker CircuitBreakerConfig `json:"circuitBreaker,omitempty"`

	// AuditLog configuration
	AuditLog AuditLogConfig `json:"auditLog,omitempty"`
}

// CircuitBreakerConfig configures the circuit breaker
type CircuitBreakerConfig struct {
	// Enabled flag
	Enabled bool `json:"enabled,omitempty"`

	// FailureThreshold before opening
	FailureThreshold int `json:"failureThreshold,omitempty"`

	// SuccessThreshold before closing
	SuccessThreshold int `json:"successThreshold,omitempty"`

	// Timeout when open
	Timeout time.Duration `json:"timeout,omitempty"`

	// HalfOpenMaxActions in half-open state
	HalfOpenMaxActions int `json:"halfOpenMaxActions,omitempty"`
}

// AuditLogConfig configures audit logging
type AuditLogConfig struct {
	// Enabled flag
	Enabled bool `json:"enabled,omitempty"`

	// FilePath for audit logs
	FilePath string `json:"filePath,omitempty"`

	// MaxSize in MB before rotation
	MaxSize int `json:"maxSize,omitempty"`

	// MaxBackups to keep
	MaxBackups int `json:"maxBackups,omitempty"`

	// MaxAge in days
	MaxAge int `json:"maxAge,omitempty"`

	// IncludeMetrics in audit logs
	IncludeMetrics bool `json:"includeMetrics,omitempty"`
}

// RemediationConfig configures the remediation engine
type RemediationConfig struct {
	// DefaultTimeout for actions
	DefaultTimeout time.Duration `json:"defaultTimeout,omitempty"`

	// MaxRetries for failed actions
	MaxRetries int `json:"maxRetries,omitempty"`

	// RetryBackoff configuration
	RetryBackoff time.Duration `json:"retryBackoff,omitempty"`

	// EnableRollback allows automatic rollback on failure
	EnableRollback bool `json:"enableRollback,omitempty"`

	// ParallelActions maximum concurrent actions
	ParallelActions int `json:"parallelActions,omitempty"`

	// ActionDefaults per action type
	ActionDefaults map[string]ActionConfig `json:"actionDefaults,omitempty"`
}

// ActionConfig configures specific action types
type ActionConfig struct {
	// Enabled flag
	Enabled bool `json:"enabled,omitempty"`

	// Timeout override
	Timeout time.Duration `json:"timeout,omitempty"`

	// RequireApproval override
	RequireApproval bool `json:"requireApproval,omitempty"`

	// MaxConcurrent for this action type
	MaxConcurrent int `json:"maxConcurrent,omitempty"`

	// Parameters specific to action type
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// LoggingConfig configures logging
type LoggingConfig struct {
	// Level (debug, info, warn, error)
	Level string `json:"level,omitempty"`

	// Format (json, text)
	Format string `json:"format,omitempty"`

	// Development mode
	Development bool `json:"development,omitempty"`

	// DisableCaller info
	DisableCaller bool `json:"disableCaller,omitempty"`

	// DisableStacktrace for errors
	DisableStacktrace bool `json:"disableStacktrace,omitempty"`

	// Encoding (json, console)
	Encoding string `json:"encoding,omitempty"`

	// OutputPaths for logs
	OutputPaths []string `json:"outputPaths,omitempty"`
}

// NewDefaultConfig returns a Config with sensible defaults
func NewDefaultConfig() *Config {
	return &Config{
		MetricsAddr:          ":8080",
		ProbeAddr:            ":8081",
		EnableLeaderElection: true,
		WatchNamespace:       "",
		Metrics: MetricsConfig{
			PrometheusURL:        "http://prometheus.monitoring:9090",
			MetricsServerEnabled: true,
			CollectionInterval:   30 * time.Second,
			RetentionPeriod:      24 * time.Hour,
		},
		AI: AIConfig{
			Provider:          "ollama",
			Model:             "llama2:13b",
			Endpoint:          "http://ollama:11434",
			Timeout:           30 * time.Second,
			MaxTokens:         2048,
			Temperature:       0.7,
			SystemPrompt:      DefaultSystemPrompt,
			MinConfidence:     0.7,
			ValidateResponses: true,
		},
		Safety: SafetyConfig{
			DryRunMode:        false,
			MaxActionsPerHour: 100,
			RequireApproval:   false,
			ProtectedNamespaces: []string{
				"kube-system",
				"kube-public",
				"kube-node-lease",
				"cert-manager",
			},
			ProtectedLabels: map[string]string{
				"kubeskippy.io/protected": "true",
			},
			CircuitBreaker: CircuitBreakerConfig{
				Enabled:            true,
				FailureThreshold:   5,
				SuccessThreshold:   2,
				Timeout:            5 * time.Minute,
				HalfOpenMaxActions: 1,
			},
			AuditLog: AuditLogConfig{
				Enabled:        true,
				FilePath:       "/var/log/kubeskippy/audit.log",
				MaxSize:        100,
				MaxBackups:     10,
				MaxAge:         30,
				IncludeMetrics: false,
			},
		},
		Remediation: RemediationConfig{
			DefaultTimeout:  5 * time.Minute,
			MaxRetries:      3,
			RetryBackoff:    30 * time.Second,
			EnableRollback:  true,
			ParallelActions: 5,
			ActionDefaults: map[string]ActionConfig{
				"restart": {
					Enabled:         true,
					Timeout:         3 * time.Minute,
					RequireApproval: false,
					MaxConcurrent:   3,
				},
				"scale": {
					Enabled:         true,
					Timeout:         2 * time.Minute,
					RequireApproval: false,
					MaxConcurrent:   1,
				},
				"patch": {
					Enabled:         true,
					Timeout:         1 * time.Minute,
					RequireApproval: true,
					MaxConcurrent:   1,
				},
				"delete": {
					Enabled:         false,
					Timeout:         30 * time.Second,
					RequireApproval: true,
					MaxConcurrent:   1,
				},
			},
		},
		Logging: LoggingConfig{
			Level:             "info",
			Format:            "json",
			Development:       false,
			DisableCaller:     false,
			DisableStacktrace: false,
			Encoding:          "json",
			OutputPaths:       []string{"stdout"},
		},
	}
}

// DefaultSystemPrompt is the default AI system prompt
const DefaultSystemPrompt = `You are a Kubernetes cluster healing assistant. Your role is to analyze cluster state and suggest ONLY safe remediation actions.

Guidelines:
1. Never suggest deleting stateful workloads or persistent volumes
2. Prefer restart over delete for pods
3. Consider resource limits before suggesting scaling
4. Always explain the reasoning behind recommendations
5. Indicate confidence level and potential risks
6. Suggest monitoring after actions

When analyzing issues:
- Identify root causes, not just symptoms
- Consider cascade effects of actions
- Prioritize stability over optimization
- Recommend incremental changes

Output format:
- Summary: Brief description of the issue
- Root Cause: Likely cause analysis
- Recommendation: Specific action to take
- Risk: Potential negative effects
- Confidence: Low/Medium/High`

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Add validation logic here
	return nil
}
