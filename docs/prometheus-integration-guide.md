# KubeSkippy Prometheus Integration Guide

## Overview

KubeSkippy integrates with Prometheus to enable advanced metric-based healing policies using PromQL queries. This allows you to create sophisticated triggers based on any metrics available in your Prometheus instance.

## Configuration

### Enabling Prometheus Integration

1. **Environment Variable**: Set the Prometheus URL via environment variable:
   ```bash
   export METRICS_PROMETHEUS_URL=http://prometheus.monitoring.svc.cluster.local:9090
   ```

2. **ConfigMap**: Or configure in the operator's ConfigMap:
   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: kubeskippy-config
   data:
     config.yaml: |
       metrics:
         prometheus_url: "http://prometheus.monitoring.svc.cluster.local:9090"
   ```

3. **Demo Mode**: In the demo, use the `--with-prometheus` flag:
   ```bash
   ./demo/setup.sh --with-prometheus
   ```

## Writing PromQL Queries in Healing Policies

### Basic Structure

```yaml
apiVersion: kubeskippy.io/v1alpha1
kind: HealingPolicy
metadata:
  name: prometheus-based-healing
spec:
  resourceSelector:
    kind: Deployment
    namespace: default
    labelSelector:
      matchLabels:
        app: my-app
  triggers:
    - type: Metric
      metricTrigger:
        query: |
          rate(container_cpu_usage_seconds_total{pod=~"my-app-.*"}[5m])
        threshold: 0.8
        operator: ">"
        duration: 3m
  actions:
    - type: Scale
      scaleAction:
        replicas: "+2"
```

### Query Recognition

KubeSkippy automatically detects PromQL queries by looking for:
- Parentheses: `rate(...)`, `sum(...)`, `avg(...)`
- Curly braces: `{job="app", status="500"}`
- Time ranges: `[5m]`, `[1h]`, `[30s]`

If these patterns aren't detected, the query falls back to basic metrics (cpu_usage_percent, memory_usage_percent, etc.).

## Common Query Patterns

### 1. HTTP Error Rate

```yaml
query: |
  sum(rate(http_requests_total{job="my-app",status=~"5.."}[5m])) 
  / sum(rate(http_requests_total{job="my-app"}[5m]))
threshold: 0.05  # 5% error rate
operator: ">"
```

### 2. P95/P99 Latency

```yaml
query: |
  histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket{job="my-app"}[5m])) by (le))
threshold: 0.5  # 500ms
operator: ">"
```

### 3. Pod CPU Usage

```yaml
query: |
  avg(rate(container_cpu_usage_seconds_total{pod=~"my-app-.*"}[5m])) * 100
threshold: 80  # 80%
operator: ">"
```

### 4. Memory Usage

```yaml
query: |
  avg(container_memory_working_set_bytes{pod=~"my-app-.*"}) 
  / avg(container_spec_memory_limit_bytes{pod=~"my-app-.*"}) * 100
threshold: 90  # 90%
operator: ">"
```

### 5. Network Traffic

```yaml
query: |
  sum(rate(container_network_receive_bytes_total{pod=~"my-app-.*"}[5m]))
threshold: 1000000  # 1MB/s
operator: ">"
```

### 6. Disk IOPS

```yaml
query: |
  sum(rate(container_fs_reads_total{pod=~"my-app-.*"}[1m])) +
  sum(rate(container_fs_writes_total{pod=~"my-app-.*"}[1m]))
threshold: 1000  # 1000 IOPS
operator: ">"
```

### 7. Queue Depth

```yaml
query: |
  avg(kafka_consumer_lag{consumer_group="my-app"})
threshold: 10000  # 10k messages
operator: ">"
```

### 8. Database Connection Pool

```yaml
query: |
  (db_connections_active{app="my-app"} / db_connections_max{app="my-app"}) * 100
threshold: 85  # 85% utilized
operator: ">"
```

## Best Practices for Trigger Thresholds

### 1. Use Percentiles for Latency

Instead of average latency:
```yaml
# Bad - averages hide outliers
query: avg(http_request_duration_seconds)

# Good - captures tail latency
query: histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))
```

### 2. Rate vs. Instant Values

Use rates for counters:
```yaml
# Bad - raw counter value
query: http_requests_total{status="500"}

# Good - rate of errors
query: rate(http_requests_total{status="500"}[5m])
```

### 3. Aggregation Across Pods

Aggregate metrics across all pods:
```yaml
# Bad - single pod metric
query: container_cpu_usage_seconds_total{pod="my-app-abc123"}

# Good - all pods with regex
query: avg(rate(container_cpu_usage_seconds_total{pod=~"my-app-.*"}[5m]))
```

### 4. Time Windows

Choose appropriate time windows:
- **1m**: Very responsive, but noisy
- **5m**: Good balance for most metrics
- **15m**: Stable, but slower to react

```yaml
# Fast response for critical metrics
query: rate(http_requests_total{status="500"}[1m])

# Stable for capacity planning
query: avg_over_time(cpu_usage[15m])
```

### 5. Threshold Guidelines

| Metric Type | Conservative | Balanced | Aggressive |
|-------------|--------------|----------|-------------|
| CPU Usage | 90% | 80% | 70% |
| Memory Usage | 95% | 85% | 75% |
| Error Rate | 5% | 2% | 1% |
| P99 Latency | 2s | 1s | 500ms |
| Queue Depth | 10k | 5k | 1k |

## Complex Multi-Metric Triggers

### Combining CPU and Memory

Create multiple triggers that must all be true:

```yaml
triggers:
  - type: Metric
    metricTrigger:
      query: |
        avg(rate(container_cpu_usage_seconds_total{pod=~"my-app-.*"}[5m])) * 100
      threshold: 70
      operator: ">"
      duration: 5m
  - type: Metric
    metricTrigger:
      query: |
        avg(container_memory_working_set_bytes{pod=~"my-app-.*"}) 
        / avg(container_spec_memory_limit_bytes{pod=~"my-app-.*"}) * 100
      threshold: 80
      operator: ">"
      duration: 5m
```

### Service Quality Indicators (SQI)

Combine availability and performance:

```yaml
query: |
  (
    sum(rate(http_requests_total{job="my-app",status!~"5.."}[5m])) 
    / sum(rate(http_requests_total{job="my-app"}[5m]))
  ) * (
    histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{job="my-app"}[5m])) by (le)) < 1
  )
threshold: 0.95  # 95% requests successful AND under 1s
operator: "<"
```

### Business Metrics

Monitor business KPIs:

```yaml
query: |
  sum(rate(orders_completed_total[1h])) / sum(rate(orders_created_total[1h]))
threshold: 0.9  # 90% order completion rate
operator: "<"
```

## Debugging PromQL Queries

### 1. Test in Prometheus UI

Always test your queries in the Prometheus UI first:
```
http://prometheus:9090/graph
```

### 2. Check Metric Names

List available metrics:
```promql
{__name__=~"container_cpu.*"}
```

### 3. Verify Labels

Check label values:
```promql
group by (pod) (container_cpu_usage_seconds_total)
```

### 4. Use Recording Rules

For complex queries, create Prometheus recording rules:

```yaml
groups:
  - name: kubeskippy
    rules:
      - record: app:error_rate:5m
        expr: |
          sum by (app) (rate(http_requests_total{status=~"5.."}[5m])) 
          / sum by (app) (rate(http_requests_total[5m]))
```

Then use in healing policy:
```yaml
query: app:error_rate:5m{app="my-app"}
threshold: 0.02
```

## Common Issues and Solutions

### Issue: Query Returns No Data

**Solution**: Check that metrics exist and pods are being scraped:
```bash
kubectl logs -n monitoring prometheus-0 | grep "my-app"
```

### Issue: Query Too Expensive

**Solution**: Optimize by:
1. Adding more specific label selectors
2. Pre-aggregating with recording rules
3. Reducing time range
4. Using `without` instead of `by` for high-cardinality metrics

### Issue: Flapping Triggers

**Solution**: Add hysteresis:
1. Increase duration requirement
2. Use moving averages: `avg_over_time(...[10m])`
3. Implement different thresholds for scale-up vs scale-down

## Example: Complete Service Degradation Policy

```yaml
apiVersion: kubeskippy.io/v1alpha1
kind: HealingPolicy
metadata:
  name: service-degradation-advanced
spec:
  resourceSelector:
    kind: Deployment
    namespace: production
    labelSelector:
      matchLabels:
        tier: frontend
  
  triggers:
    # High error rate
    - type: Metric
      metricTrigger:
        query: |
          sum(rate(http_requests_total{job="frontend",status=~"5.."}[5m])) 
          / sum(rate(http_requests_total{job="frontend"}[5m]))
        threshold: 0.05
        operator: ">"
        duration: 2m
    
    # High P99 latency
    - type: Metric
      metricTrigger:
        query: |
          histogram_quantile(0.99, 
            sum(rate(http_request_duration_seconds_bucket{job="frontend"}[5m])) 
            by (le)
          )
        threshold: 2.0
        operator: ">"
        duration: 2m
    
    # Connection pool exhaustion
    - type: Metric
      metricTrigger:
        query: |
          max(db_connections_active{app="frontend"} / db_connections_max{app="frontend"}) * 100
        threshold: 90
        operator: ">"
        duration: 1m
  
  actions:
    # First try scaling
    - type: Scale
      scaleAction:
        replicas: "+2"
        min: 3
        max: 10
    
    # Then restart if issues persist
    - type: Restart
      restartAction:
        strategy: "RollingRestart"
  
  # Safety settings
  safetySettings:
    enabled: true
    maxActionsPerHour: 5
    cooldownPeriod: 10m
    dryRun: false
```

## Monitoring KubeSkippy Itself

Use these queries to monitor KubeSkippy's performance:

```promql
# Healing actions triggered
sum(rate(kubeskippy_healing_actions_total[1h])) by (policy, action)

# Successful vs failed actions
sum(rate(kubeskippy_healing_actions_total{status="success"}[1h])) 
/ sum(rate(kubeskippy_healing_actions_total[1h]))

# Policy evaluation latency
histogram_quantile(0.95, 
  sum(rate(kubeskippy_policy_evaluation_duration_seconds_bucket[5m])) 
  by (le, policy)
)
```

## Next Steps

1. **Start Simple**: Begin with basic CPU/memory queries
2. **Test Thoroughly**: Use the demo environment to validate policies
3. **Monitor Impact**: Track the effectiveness of your healing actions
4. **Iterate**: Refine thresholds based on real-world behavior
5. **Share**: Contribute your policies to the community

For more examples, check out:
- `/demo/policies/prometheus-based-healing.yaml`
- `/tests/e2e/prometheus_integration_test.go`
- Prometheus documentation: https://prometheus.io/docs/prometheus/latest/querying/