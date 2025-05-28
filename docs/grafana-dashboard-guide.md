# KubeSkippy Grafana Dashboard Guide

This guide explains how to access and use the KubeSkippy Grafana dashboard for monitoring healing operations.

## Prerequisites

Deploy KubeSkippy with the monitoring stack:
```bash
cd demo
./setup.sh --with-monitoring
```

## Accessing Grafana

### 1. Start Port Forwarding
```bash
kubectl port-forward -n monitoring svc/grafana 3000:3000
```

### 2. Access the Dashboard
- URL: http://localhost:3000
- Username: `admin`
- Password: `admin`

### 3. Navigate to KubeSkippy Dashboard
- Method 1: Click "Dashboards" → "KubeSkippy Demo Overview"
- Method 2: Direct URL: http://localhost:3000/d/kubeskippy-demo

## Dashboard Panels Explained

### Overview Metrics (Top Row)
1. **Total Demo Pods**
   - Shows total demo application pods
   - Real-time count from Kubernetes

2. **Unhealthy Pods**
   - Count of pods not in Running state
   - Color coding: Green (0), Yellow (1-2), Red (3+)

3. **AI-Driven Healing Actions**
   - Total count of AI-triggered healing actions
   - Shows 0 until AI analysis generates actions

4. **AI Backend Status**
   - Status of Ollama/AI backend service
   - Online/Offline indicator with color coding

### Container Details Section
#### Pod Status and Restart Monitoring
- **Pod Status Distribution**: Pie chart showing pod phases
- **Pod Restart Monitoring**: Table with pod names, status, and restart counts
- Real-time view of application health and stability

### Resource Usage Monitoring (like monitor.sh)
#### CPU & Memory Usage
- **Pod CPU Usage %**: Time series graph of CPU utilization per pod
- **Pod Memory Usage**: Time series graph of memory consumption per pod
- Matches the comprehensive monitoring from the ./monitor.sh script

### 🤖 AI Analysis & Healing Section
#### AI Healing Activity Timeline
- **Real-time AI Actions**: Rate of AI-triggered healing actions per second
- **Pending/Completed Counters**: Track AI action lifecycle
- **Time series visualization**: See AI activity patterns over time

#### AI Healing Actions - Recent Activity
- **Table view**: Recent AI-driven healing actions with details
- **Action types**: restart, scale, delete, patch operations
- **Status tracking**: pending, completed, failed states

### Summary Visualizations

#### Action Results by Type
- Pie chart showing distribution of action statuses
- Categories: success, failed, pending
- Quick visual of overall system health

#### Recent Healing Actions (Logs Panel)
- Real-time log stream of healing actions
- Filtered for KubeSkippy system namespace
- Shows recent healing activity and decisions

## Common Use Cases

### 1. Monitoring Healing Effectiveness
- Watch the Success Rate panel
- Check if actions are completing successfully
- Identify any patterns in failures

### 2. Understanding System Load
- Monitor Policy Evaluations for system activity
- Check Healing Actions Timeline for spikes
- Correlate with Target Application Health

### 3. Debugging Issues
- Use Recent Healing Actions logs for real-time info
- Check Action Results for failure patterns
- Cross-reference with application health metrics

## Customizing the Dashboard

### Adding Custom Queries
1. Click panel title → Edit
2. Modify the PromQL query
3. Common KubeSkippy metrics:
   - `kubeskippy_healing_actions_total{trigger_type="ai-driven"}`
   - `kubeskippy_policy_evaluations_total`
   - `kubeskippy_healing_duration_seconds`
   - `kube_pod_status_phase{namespace="demo-apps"}`

### Creating Alerts
1. Navigate to panel → Edit → Alert
2. Set threshold conditions
3. Configure notification channels

## Troubleshooting

### No Data Showing
```bash
# Check Prometheus is running
kubectl get pods -n monitoring

# Verify metrics are being scraped
kubectl port-forward -n monitoring svc/prometheus 9090:9090
# Visit http://localhost:9090/targets
```

### Dashboard Not Loading
```bash
# Check Grafana logs
kubectl logs -n monitoring deployment/grafana

# Restart Grafana
kubectl rollout restart deployment/grafana -n monitoring
```

### Port Forwarding Issues
```bash
# Kill existing port-forward
pkill -f "port-forward.*grafana"

# Start fresh
kubectl port-forward -n monitoring svc/grafana 3000:3000
```

## Advanced Features

### Prometheus Integration
Access Prometheus directly for custom queries:
```bash
kubectl port-forward -n monitoring svc/prometheus 9090:9090
# Visit http://localhost:9090
```

### Example PromQL Queries
```promql
# AI-driven healing actions by type in last hour
sum by (action_type) (increase(kubeskippy_healing_actions_total{trigger_type=~".*ai.*"}[1h]))

# AI vs Traditional healing action ratio
sum(kubeskippy_healing_actions_total{trigger_type=~".*ai.*"}) /
sum(kubeskippy_healing_actions_total)

# Pod health monitoring (like monitor.sh)
count(kube_pod_status_phase{namespace="demo-apps", phase="Running"}) /
count(kube_pod_status_phase{namespace="demo-apps"})

# AI backend availability
up{job="ollama"} or kube_pod_status_phase{namespace="demo-apps", pod=~"ollama.*", phase="Running"}
```

## Best Practices

1. **Regular Monitoring**
   - Check dashboard during demo runs
   - Monitor for unexpected spikes
   - Verify healing actions match expectations

2. **Performance Tuning**
   - Adjust refresh rate based on needs (default: 30s)
   - Use time range selector for historical analysis
   - Export interesting patterns for documentation

3. **Integration with Demos**
   - Keep dashboard visible during presentations
   - Use full-screen mode for better visibility
   - Highlight specific panels for different scenarios