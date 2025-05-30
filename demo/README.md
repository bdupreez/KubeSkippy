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

This demo showcases KubeSkippy's **AI-powered autonomous healing** capabilities by simulating continuous application failures and demonstrating how the operator uses artificial intelligence to predict, detect, and remediate issues with strategic decisions including **AI Strategic Deletes** and **Resource Optimization**.

## Quick Start

```bash
# 1. Setup AI-powered demo with continuous failures (RECOMMENDED)
./setup.sh

# 2. Setup without monitoring stack (basic mode)
./setup.sh --no-monitoring

# 3. Watch AI healing in action (new terminal)
./monitor.sh

# 4. Access enhanced Grafana dashboard with AI metrics
# http://localhost:3000 (admin/admin)
# Dashboard: "KubeSkippy Enhanced AI Healing Overview"

# 5. Quick demo status check
./quick-demo.sh
```

## Demo Applications

The demo includes **continuous failure applications** designed to showcase AI-powered predictive healing:

### Core Failure Applications
| Application | Issue Pattern | Traditional vs AI Healing |
|------------|---------------|---------------------------|
| **crashloop-app** | Exits with error code 1 | Traditional: Restarts after crash<br>**AI**: Predictive intervention |
| **memory-leak-app** | Memory grows to 500MB then crashes | Traditional: Restarts at 85% usage<br>**AI**: Strategic deletes at 30% |
| **cpu-spike-app** | Random CPU spikes | Traditional: Scales at 80% CPU<br>**AI**: Resource optimization at 40% |
| **flaky-web-app** | 20% error rate (500/502/504) | Traditional: Restarts after degradation<br>**AI**: Predictive scaling |

### AI Continuous Failure Generators
| Application | Failure Pattern | AI Strategic Actions |
|------------|-----------------|---------------------|
| **continuous-memory-degradation** | Gradual memory increase (60s cycles) | **AI Strategic Deletes**<br>**Resource Optimization** |
| **continuous-cpu-oscillation** | Sine wave CPU patterns with escalation | **Predictive Scaling**<br>**Intelligent Restarts** |
| **continuous-network-degradation** | Gradual network latency increases | **AI System Patches**<br>**Emergency Deletes** |
| **chaos-monkey-component** | Random unpredictable failures (30s intervals) | **Cascade Prevention**<br>**Strategic Interventions** |

## Healing Policies Architecture

### 🤖 AI Strategic Healing Policies (PRIMARY)

#### 1. AI Strategic Healing
- **Triggers**: Low thresholds for early intervention (CPU >50%, Memory >25%)
- **Actions**: 
  - **AI Strategic Deletes** (Priority 5)
  - **AI Resource Optimization** (Priority 8)
  - **AI Intelligent Restarts** (Priority 12)
  - **AI System Patches** (Priority 15)
- **Rate Limit**: 25 actions/hour for continuous demo activity
- **Mode**: Automatic with AI decision-making

#### 2. AI Cascade Prevention
- **Triggers**: Event-based cascade detection (Created/Started events)
- **Actions**:
  - **AI Emergency Deletes** (Priority 1 - Highest)
  - **AI Controlled Scaling** (Priority 3)
- **Rate Limit**: 30 actions/hour for emergency scenarios
- **Mode**: Automatic emergency intervention

#### 3. Predictive AI Healing
- **Triggers**: Predictive analysis with early warning thresholds (30% memory, 40% CPU)
- **Actions**: Early intervention before traditional policies trigger
- **Rate Limit**: 15 actions/hour
- **Mode**: Automatic predictive healing

### 📊 Traditional Healing Policies (REDUCED RATES)

#### 4. Crashloop Pod Healing
- **Triggers**: Restart count > 3 or CrashLoopBackOff status
- **Actions**: Debug patches and restarts
- **Rate Limit**: **2 actions/hour** (reduced to let AI handle more)

#### 5. Memory Leak Healing
- **Triggers**: Memory usage > 85% for 3 minutes
- **Actions**: Rolling restart and scaling
- **Rate Limit**: **1 action/hour** (reduced to let AI handle more)

#### 6. Service Degradation Healing
- **Triggers**: Error rate > 5% or availability < 99.5%
- **Actions**: Restart pods and scale up
- **Rate Limit**: **1 action/hour** (reduced to let AI handle more)

### 🔄 Continuous Activity Policies

#### 7. Continuous Activity Policy
- **Purpose**: Ensures constant healing demonstration
- **Triggers**: Event-based with 30s cooldowns
- **Rate Limit**: 50 actions/hour for demo visibility

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

### Enhanced AI Monitoring with Grafana (INCLUDED BY DEFAULT)
Access the enhanced Grafana dashboard with dedicated AI metrics:

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

# 5. Find the enhanced dashboard
# Option A: Navigate to Dashboards → KubeSkippy Enhanced AI Healing Overview
# Option B: Direct link: http://localhost:3000/d/kubeskippy-enhanced
```

The **Enhanced KubeSkippy AI Dashboard** includes:

#### 🤖 AI Analysis & Healing Section
- **AI-Driven Healing Actions**: Counter showing total AI-triggered actions
- **AI Backend Status**: Real-time Ollama/AI service availability
- **AI Healing Activity Timeline**: Time series of AI action rates and patterns
- **AI Actions Table**: Recent AI-driven healing actions with detailed status
- **AI Confidence Levels**: AI decision confidence scoring over time

#### 📊 Traditional Healing Metrics
- **Healing Actions Over Time**: Real-time count of all healing actions
- **Success Rate**: Percentage of successful vs failed healing actions
- **Active Policies**: Number of healing policies currently active
- **Healing Actions by Type**: Breakdown showing AI vs traditional actions
- **Target Application Health**: CPU and memory metrics for demo apps

#### 🎯 Strategic Action Tracking
- **AI Strategic Deletes**: Count and timeline of AI strategic delete actions
- **AI Resource Optimization**: Scale and patch actions by AI
- **Cascade Prevention**: Emergency interventions and their effectiveness

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