package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPrometheusServer creates a test server that mimics Prometheus API
func mockPrometheusServer(t *testing.T) *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/query":
			query := r.URL.Query().Get("query")
			var response string

			// Mock responses for different queries
			switch query {
			case "up":
				response = `{
					"status": "success",
					"data": {
						"resultType": "vector",
						"result": [{
							"metric": {},
							"value": [1234567890, "1"]
						}]
					}
				}`
			case `container_memory_working_set_bytes{pod="test-pod"}`:
				response = `{
					"status": "success",
					"data": {
						"resultType": "vector",
						"result": [{
							"metric": {"pod": "test-pod"},
							"value": [1234567890, "1073741824"]
						}]
					}
				}`
			case `rate(http_requests_total[5m])`:
				response = `{
					"status": "success",
					"data": {
						"resultType": "vector",
						"result": [{
							"metric": {},
							"value": [1234567890, "100.5"]
						}]
					}
				}`
			default:
				response = `{
					"status": "success",
					"data": {
						"resultType": "vector",
						"result": []
					}
				}`
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(response))

		case "/api/v1/query_range":
			response := `{
				"status": "success",
				"data": {
					"resultType": "matrix",
					"result": [{
						"metric": {},
						"values": [
							[1234567890, "100"],
							[1234567900, "110"],
							[1234567910, "120"]
						]
					}]
				}
			}`
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(response))

		case "/api/v1/config":
			// Health check endpoint
			response := `{"status": "success"}`
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(response))

		default:
			http.NotFound(w, r)
		}
	})

	return httptest.NewServer(handler)
}

func TestNewPrometheusClient(t *testing.T) {
	server := mockPrometheusServer(t)
	defer server.Close()

	tests := []struct {
		name    string
		address string
		timeout time.Duration
		wantErr bool
	}{
		{
			name:    "valid configuration",
			address: server.URL,
			timeout: 10 * time.Second,
			wantErr: false,
		},
		{
			name:    "empty address",
			address: "",
			timeout: 10 * time.Second,
			wantErr: true,
		},
		{
			name:    "invalid address",
			address: "http://invalid-prometheus:9999",
			timeout: 1 * time.Second,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewPrometheusClient(tt.address, tt.timeout)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestPrometheusClient_Query(t *testing.T) {
	server := mockPrometheusServer(t)
	defer server.Close()

	client, err := NewPrometheusClient(server.URL, 10*time.Second)
	require.NoError(t, err)

	tests := []struct {
		name      string
		query     string
		wantValue float64
		wantErr   bool
	}{
		{
			name:      "simple up query",
			query:     "up",
			wantValue: 1.0,
			wantErr:   false,
		},
		{
			name:      "memory query with labels",
			query:     `container_memory_working_set_bytes{pod="test-pod"}`,
			wantValue: 1073741824.0, // 1GB in bytes
			wantErr:   false,
		},
		{
			name:      "rate query",
			query:     `rate(http_requests_total[5m])`,
			wantValue: 100.5,
			wantErr:   false,
		},
		{
			name:      "no data query",
			query:     `non_existent_metric`,
			wantValue: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			value, err := client.Query(ctx, tt.query)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantValue, value)
			}
		})
	}
}

func TestPrometheusClient_QueryRange(t *testing.T) {
	server := mockPrometheusServer(t)
	defer server.Close()

	client, err := NewPrometheusClient(server.URL, 10*time.Second)
	require.NoError(t, err)

	ctx := context.Background()
	values, err := client.QueryRange(ctx, "test_metric", 5*time.Minute)

	assert.NoError(t, err)
	assert.Len(t, values, 3)
	assert.Equal(t, []float64{100.0, 110.0, 120.0}, values)
}

func TestPrometheusClient_IsHealthy(t *testing.T) {
	server := mockPrometheusServer(t)
	defer server.Close()

	client, err := NewPrometheusClient(server.URL, 10*time.Second)
	require.NoError(t, err)

	ctx := context.Background()
	healthy := client.IsHealthy(ctx)
	assert.True(t, healthy)

	// Test with stopped server
	server.Close()
	healthy = client.IsHealthy(ctx)
	assert.False(t, healthy)
}

func TestBuildQuery(t *testing.T) {
	tests := []struct {
		name       string
		metricType string
		labels     map[string]string
		want       string
	}{
		{
			name:       "pod cpu without labels",
			metricType: "pod_cpu",
			labels:     nil,
			want:       `sum(rate(container_cpu_usage_seconds_total{container!=""}[5m])) by (pod, namespace)`,
		},
		{
			name:       "pod memory with namespace label",
			metricType: "pod_memory",
			labels:     map[string]string{"namespace": "default"},
			want:       `sum(container_memory_working_set_bytes{container!="",namespace="default"}) by (pod, namespace)`,
		},
		{
			name:       "custom promql query",
			metricType: `histogram_quantile(0.99, http_request_duration_seconds_bucket)`,
			labels:     nil,
			want:       `histogram_quantile(0.99, http_request_duration_seconds_bucket)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildQuery(tt.metricType, tt.labels)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCommonQueries(t *testing.T) {
	// Test that common queries are properly formatted
	assert.Contains(t, CommonQueries, "pod_cpu_usage_percent")
	assert.Contains(t, CommonQueries, "pod_memory_usage_bytes")
	assert.Contains(t, CommonQueries, "pod_restart_count")
	assert.Contains(t, CommonQueries, "http_error_rate")

	// Verify query has placeholder
	cpuQuery := CommonQueries["pod_cpu_usage_percent"]
	assert.Contains(t, cpuQuery, "%s")
}
