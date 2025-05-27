# KubeSkippy Demo

This demo showcases KubeSkippy's autonomous healing capabilities by simulating various application issues and watching the operator automatically detect and remediate them.

## Quick Start

```bash
# 1. Setup the demo environment
./setup.sh

# 2. (Optional) Setup with Prometheus for advanced metrics
./setup.sh --with-prometheus

# 3. Watch healing in action (new terminal)
./monitor.sh

# 4. Quick demo with all features
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
- **Triggers**: Multiple metrics and event patterns
- **Actions**: Intelligent remediation based on AI analysis
- **Mode**: Dryrun (can be changed to automatic)

### 6. Prometheus-Based Healing (Optional)
- **Triggers**: PromQL queries for advanced metrics
- **Metrics**: HTTP error rates, P99 latency, custom app metrics
- **Actions**: Context-aware healing based on real application behavior
- **Mode**: Automatic (requires --with-prometheus setup)

## Managing AI-Driven Healing

The AI-driven healing policy runs in `dryrun` mode by default to prevent unintended actions. You can switch modes:

```bash
# Enable automatic mode (healing actions will be executed)
kubectl patch healingpolicy ai-driven-healing -n demo-apps \
  --type merge -p '{"spec":{"mode":"automatic"}}'

# Switch back to dryrun mode (actions logged but not executed)
kubectl patch healingpolicy ai-driven-healing -n demo-apps \
  --type merge -p '{"spec":{"mode":"dryrun"}}'

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
# Enable AI-driven healing
kubectl patch healingpolicy ai-driven-healing -n demo-apps \
  --type merge -p '{"spec":{"mode":"automatic"}}'

# Wait 30 seconds for triggers to evaluate
sleep 30

# Check AI-driven actions
kubectl get healingactions -n demo-apps | grep ai-driven

# Switch back to dryrun
kubectl patch healingpolicy ai-driven-healing -n demo-apps \
  --type merge -p '{"spec":{"mode":"dryrun"}}'
```

### Scenario 3: Quick Demo Script
```bash
./quick-demo.sh
```

This script will:
- Show current issues
- Enable AI-driven healing temporarily
- Display healing actions
- Revert AI-driven healing to dryrun

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