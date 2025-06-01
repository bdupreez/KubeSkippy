package metrics

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	// validLabelNameRegex validates Prometheus label names
	validLabelNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

	// labelValueEscaper escapes special characters in label values
	labelValueEscaper = strings.NewReplacer(
		`"`, `\"`, // Escape quotes
		`\`, `\\`, // Escape backslashes
		"\n", `\n`, // Escape newlines
		"\t", `\t`, // Escape tabs
	)
)

// PrometheusClient wraps Prometheus API client
type PrometheusClient struct {
	api     promv1.API
	timeout time.Duration
}

// NewPrometheusClient creates a new Prometheus client
func NewPrometheusClient(address string, timeout time.Duration) (*PrometheusClient, error) {
	if address == "" {
		return nil, fmt.Errorf("prometheus address is required")
	}

	config := api.Config{
		Address: address,
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus client: %w", err)
	}

	return &PrometheusClient{
		api:     promv1.NewAPI(client),
		timeout: timeout,
	}, nil
}

// Query executes a PromQL query and returns the result as a float64
func (p *PrometheusClient) Query(ctx context.Context, query string) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	log := log.FromContext(ctx)
	log.V(1).Info("Executing Prometheus query", "query", query)

	result, warnings, err := p.api.Query(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("prometheus query failed: %w", err)
	}

	if len(warnings) > 0 {
		log.Info("Prometheus query warnings", "warnings", warnings)
	}

	// Extract value from result
	value, err := p.extractValue(result)
	if err != nil {
		return 0, fmt.Errorf("failed to extract value: %w", err)
	}

	return value, nil
}

// QueryRange executes a range query (useful for checking if condition held for duration)
func (p *PrometheusClient) QueryRange(ctx context.Context, query string, duration time.Duration) ([]float64, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	log := log.FromContext(ctx)
	log.V(1).Info("Executing Prometheus range query", "query", query, "duration", duration)

	end := time.Now()
	start := end.Add(-duration)
	step := duration / 10 // 10 data points

	result, warnings, err := p.api.QueryRange(ctx, query, promv1.Range{
		Start: start,
		End:   end,
		Step:  step,
	})
	if err != nil {
		return nil, fmt.Errorf("prometheus range query failed: %w", err)
	}

	if len(warnings) > 0 {
		log.Info("Prometheus query warnings", "warnings", warnings)
	}

	// Extract values from result
	values, err := p.extractRangeValues(result)
	if err != nil {
		return nil, fmt.Errorf("failed to extract range values: %w", err)
	}

	return values, nil
}

// IsHealthy checks if Prometheus is reachable
func (p *PrometheusClient) IsHealthy(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := p.api.Config(ctx)
	return err == nil
}

// extractValue extracts a single float64 value from Prometheus query result
func (p *PrometheusClient) extractValue(result model.Value) (float64, error) {
	switch v := result.(type) {
	case model.Vector:
		if len(v) == 0 {
			return 0, fmt.Errorf("query returned no data")
		}
		// Take the first result
		return float64(v[0].Value), nil

	case *model.Scalar:
		return float64(v.Value), nil

	default:
		return 0, fmt.Errorf("unexpected result type: %T", result)
	}
}

// extractRangeValues extracts multiple values from a range query result
func (p *PrometheusClient) extractRangeValues(result model.Value) ([]float64, error) {
	switch v := result.(type) {
	case model.Matrix:
		if len(v) == 0 {
			return nil, fmt.Errorf("range query returned no data")
		}

		// Extract values from the first series
		series := v[0]
		values := make([]float64, len(series.Values))
		for i, pair := range series.Values {
			values[i] = float64(pair.Value)
		}
		return values, nil

	default:
		return nil, fmt.Errorf("unexpected range result type: %T", result)
	}
}

// CommonQueries provides pre-built PromQL queries for common metrics
var CommonQueries = map[string]string{
	// Pod metrics
	"pod_cpu_usage_percent":  `100 * sum(rate(container_cpu_usage_seconds_total{pod="%s",container!=""}[5m])) by (pod)`,
	"pod_memory_usage_bytes": `sum(container_memory_working_set_bytes{pod="%s",container!=""}) by (pod)`,
	"pod_restart_count":      `sum(kube_pod_container_status_restarts_total{pod="%s"}) by (pod)`,

	// Node metrics
	"node_cpu_usage_percent":    `100 * (1 - avg(rate(node_cpu_seconds_total{mode="idle"}[5m])))`,
	"node_memory_usage_percent": `100 * (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes))`,

	// Application metrics (examples)
	"http_error_rate":  `sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))`,
	"http_latency_p99": `histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))`,

	// Container metrics
	"container_cpu_throttled_percent": `100 * sum(rate(container_cpu_cfs_throttled_periods_total[5m])) / sum(rate(container_cpu_cfs_periods_total[5m]))`,
}

// validateLabelName checks if a label name is valid for Prometheus
func validateLabelName(name string) bool {
	return validLabelNameRegex.MatchString(name)
}

// escapeLabelValue safely escapes a label value for use in PromQL
func escapeLabelValue(value string) string {
	return labelValueEscaper.Replace(value)
}

// BuildQuery helps construct common PromQL queries safely
func BuildQuery(metricType string, labels map[string]string) string {
	// This is a simplified query builder with security validation

	baseQuery := ""
	switch metricType {
	case "pod_cpu":
		baseQuery = `sum(rate(container_cpu_usage_seconds_total{container!=""}[5m])) by (pod, namespace)`
	case "pod_memory":
		baseQuery = `sum(container_memory_working_set_bytes{container!=""}) by (pod, namespace)`
	case "pod_restarts":
		baseQuery = `sum(kube_pod_container_status_restarts_total) by (pod, namespace)`
	default:
		return metricType // Assume it's already a PromQL query
	}

	// Add label filters with proper validation and escaping
	if len(labels) > 0 {
		var filters []string
		for k, v := range labels {
			// Validate label name
			if !validateLabelName(k) {
				log.Log.Info("Invalid label name, skipping", "label", k)
				continue
			}

			// Escape label value and add to filters
			escapedValue := escapeLabelValue(v)
			filters = append(filters, fmt.Sprintf(`%s="%s"`, k, escapedValue))
		}

		if len(filters) > 0 {
			filterStr := strings.Join(filters, ",")
			// Insert filters into the query safely
			idx := strings.Index(baseQuery, "}")
			if idx > 0 {
				baseQuery = baseQuery[:idx] + "," + filterStr + baseQuery[idx:]
			}
		}
	}

	return baseQuery
}
