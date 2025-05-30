# KubeSkippy Demo

## Prerequisites

- **Go toolchain**: Required for generating Kubernetes CRDs and running the demo.
  - Install with Homebrew (macOS):  
    ```sh
    brew install go
    ```
  - Or follow instructions for your OS: https://golang.org/doc/install

> If Go is not installed, the setup will fail to generate required files.  
> You can verify Go is installed by running: `go version`

This demo showcases KubeSkippy's autonomous healing capabilities by simulating various application issues and watching the operator automatically detect and remediate them.

## Quick Start

```bash
# 1. Setup the demo environment
./setup.sh

# 2. (Optional) Setup with Prometheus for advanced metrics
./setup.sh --with-prometheus

# 3. (Optional) Setup with full monitoring stack (Prometheus + Grafana)
./setup.sh --with-monitoring

# 4. Watch healing in action (new terminal)
./monitor.sh

# 5. Quick demo with all features
./quick-demo.sh
```

## Demo Applications

The demo includes four problematic applications that trigger different healing policies:

| Application | Issue | Healing Actions |
|------------|-------|-----------------|
| **crashloop-app** | Exits with error code 1 | • Apply debug patches<br>• Restart pods |
| **memory-leak-app** | Memory grows to 500MB then crashes | • Restart pods<br>• Scale deployment |
| **cpu-spike-app** | Random CPU spikes | • Scale horizontally<br>• Apply CPU limits |
| **flaky-web-app** | 20% error rate (500/502/504) | • Restart pods<br>• Scale up service |
| **pattern-failure-app** | Complex multi-condition failures | • AI pattern recognition<br>• Strategic healing |

## Healing Policies

### 1. Crashloop Pod Healing
- **Triggers**: Restart count > 3 or CrashLoopBackOff status
- **Actions**: Applies debug environment variables and restarts pods
- **Mode**: Automatic

### 2. Memory Leak Healing
- **Triggers**: Memory usage > 85% for 3 minutes
- **Actions**: Rolling restart and horizontal scaling
- **Mode**: Automatic

### 3. CPU Spike Healing
- **Triggers**: CPU usage > 80% for 2 minutes
- **Actions**: Horizontal scaling and CPU limit adjustment
- **Mode**: Automatic
- **Note**: May not trigger in demo if CPU usage stays below threshold

### 4. Service Degradation Healing
- **Triggers**: Error rate > 5% or availability < 99.5%
- **Actions**: Restart pods and scale up deployment
- **Mode**: Automatic

### 5. AI-Driven Healing
- **Triggers**: Multiple metrics and event patterns with AI analysis
- **Actions**: Intelligent remediation based on AI pattern recognition
- **Mode**: Automatic (AI-powered actions execute automatically)
- **Special Features**: Pattern recognition, confidence scoring, strategic decision-making

### 6. AI-Intelligent Healing (Enhanced)
- **Triggers**: Complex pattern recognition, predictive analysis
- **Actions**: Confidence-based actions with reasoning annotations
- **Mode**: Automatic with advanced AI capabilities
- **Special Features**: Multi-dimensional analysis, alternative strategy evaluation

### 7. Prometheus-Based Healing (Optional)
- **Triggers**: PromQL queries for advanced metrics
- **Metrics**: HTTP error rates, P99 latency, custom app metrics
- **Actions**: Context-aware healing based on real application behavior
- **Mode**: Automatic (requires --with-prometheus or --with-monitoring setup)

## Managing AI-Driven Healing

The AI-driven healing policy runs in `automatic` mode by default, executing AI-powered healing actions automatically. You can switch to dryrun mode if needed:

```bash
# Switch to dryrun mode (actions logged but not executed)
kubectl patch healingpolicy ai-driven-healing -n demo-apps \
  --type merge -p '{"spec":{"mode":"dryrun"}}'

# Switch back to automatic mode (healing actions will be executed)
kubectl patch healingpolicy ai-driven-healing -n demo-apps \
  --type merge -p '{"spec":{"mode":"automatic"}}'

# Check current mode
kubectl get healingpolicy ai-driven-healing -n demo-apps -o jsonpath='{.spec.mode}'
echo
```

## Monitoring the Demo

### Watch Real-time Status
```bash
./monitor.sh
```

This shows:
- Pod status and restart counts
- Resource usage (CPU/Memory)
- Active healing policies
- Healing actions being created
- Recent events
- Operator logs
- Monitoring stack status (Prometheus/Grafana)

### Visual Monitoring with Grafana (Optional)
If you deployed with `--with-monitoring`, access Grafana for visual dashboards:

```bash
# 1. Start port forwarding (if not already running)
kubectl port-forward -n monitoring svc/grafana 3000:3000

# 2. Check if port-forward is active
ps aux | grep "port-forward.*grafana"

# 3. Access Grafana in your browser
http://localhost:3000

# 4. Login credentials
Username: admin
Password: admin

# 5. Find the dashboard
# Option A: Navigate to Dashboards → KubeSkippy Healing Overview
# Option B: Direct link: http://localhost:3000/d/kubeskippy-overview
```

The KubeSkippy dashboard includes:
- **Healing Actions Over Time**: Real-time count of healing actions
- **Success Rate**: Percentage of successful vs failed healing actions
- **Active Policies**: Number of healing policies currently active
- **Policy Evaluations**: Total evaluation count over time
- **Healing Actions Timeline**: Time-series graph of actions by type
- **Target Application Health**: CPU and memory metrics for demo apps
- **Action Results by Type**: Pie chart of success/failure distribution
- **Recent Healing Actions**: Log view of recent healing activity

### Access Prometheus (Optional)
For raw metrics and custom queries:

```bash
# Access Prometheus UI
kubectl port-forward -n monitoring svc/prometheus 9090:9090

# Open http://localhost:9090 in browser
```

### Check Healing Actions
```bash
# List all healing actions
kubectl get healingactions -n demo-apps

# Watch healing actions being created
kubectl get healingactions -n demo-apps -w

# Check specific healing action details
kubectl describe healingaction <action-name> -n demo-apps
```

### View Operator Logs
```bash
# Follow operator logs
kubectl logs -n kubeskippy-system deployment/kubeskippy-controller-manager -f

# Check trigger evaluations
kubectl logs -n kubeskippy-system deployment/kubeskippy-controller-manager | grep "Trigger"
```

## Demo Scenarios

### Scenario 1: Basic Healing Demo (2-3 minutes)
1. Run `./setup.sh` to start the demo
2. Watch `./monitor.sh` in another terminal
3. Within 1-2 minutes, you'll see:
   - Crashloop pods getting debug patches
   - Memory leak pods being restarted
   - Service degradation scaling up flaky-web-app
4. Check healing actions: `kubectl get healingactions -n demo-apps`

### Scenario 2: AI-Driven Healing (1 minute)
```bash
# AI-driven healing is already enabled by default
# Check current mode
kubectl get healingpolicy ai-driven-healing -n demo-apps -o jsonpath='{.spec.mode}'
echo

# Wait 30 seconds for triggers to evaluate
sleep 30

# Check AI-driven actions
kubectl get healingactions -n demo-apps | grep ai-driven

# Optional: Switch to dryrun if you want to disable automatic actions
# kubectl patch healingpolicy ai-driven-healing -n demo-apps \
#   --type merge -p '{"spec":{"mode":"dryrun"}}'
```

### Scenario 3: AI Intelligence Showcase
```bash
./showcase-ai.sh
```

This script will:
- Deploy complex pattern failure scenarios
- Enable enhanced AI healing policies
- Show real-time AI vs rule-based comparison
- Display AI confidence levels and decision reasoning

### Scenario 4: Quick Demo Script
```bash
./quick-demo.sh
```

This script will:
- Show current issues
- Display healing actions (AI-driven healing is already enabled)
- Show the status of all healing policies

## Expected Timeline

After starting the demo:
- **0-30 seconds**: Pods start, some begin crashing
- **30-60 seconds**: Metrics collection begins
- **1-2 minutes**: First healing actions for crashloop pods
- **2-3 minutes**: Memory and service degradation healing
- **3-5 minutes**: Multiple rounds of healing actions

## Cleanup

```bash
./cleanup.sh
```

This will:
- Delete the demo namespace
- Remove the KubeSkippy operator
- Delete the Kind cluster

## Troubleshooting

### Grafana Dashboard Issues
```bash
# Dashboard not loading
kubectl logs -n monitoring deployment/grafana | grep -i error

# Port forwarding not working
# Kill existing port-forward
pkill -f "port-forward.*grafana"
# Restart it
kubectl port-forward -n monitoring svc/grafana 3000:3000

# Dashboard shows "No Data"
# Check Prometheus is running
kubectl get pods -n monitoring
# Check datasource connectivity in Grafana UI
# Settings → Data Sources → Prometheus → Test
```

### No Healing Actions Created
```bash
# Check operator is running
kubectl get pods -n kubeskippy-system

# Check for errors
kubectl logs -n kubeskippy-system deployment/kubeskippy-controller-manager | grep ERROR

# Verify pods have correct labels
kubectl get pods -n demo-apps --show-labels
```

### Healing Actions Not Executing
```bash
# Check action phase
kubectl get healingactions -n demo-apps

# Look for safety controller blocks
kubectl logs -n kubeskippy-system deployment/kubeskippy-controller-manager | grep "Rate limit"
```

### High Resource Usage
The demo apps intentionally consume resources. If needed:
```bash
# Scale down apps
kubectl scale deployment --all --replicas=1 -n demo-apps

# Check resource usage
kubectl top pods -n demo-apps
```

## Key Commands Reference

```bash
# Policy management
kubectl get healingpolicies -n demo-apps
kubectl describe healingpolicy <name> -n demo-apps
kubectl patch healingpolicy <name> -n demo-apps --type merge -p '{"spec":{"mode":"dryrun"}}'

# Action monitoring  
kubectl get healingactions -n demo-apps
kubectl get healingactions -n demo-apps -w
kubectl delete healingactions --all -n demo-apps  # Clean up old actions

# Quick status check
./check-demo.sh
```