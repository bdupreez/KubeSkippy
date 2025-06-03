# KubeSkippy Demo Validation Guide

## Overview
This guide provides a systematic approach to running and validating the KubeSkippy demo, ensuring all components work correctly and the Grafana dashboard displays meaningful data.

## Prerequisites
Before running the demo, ensure:
1. **Docker is running** - Open Docker Desktop on macOS or start Docker service on Linux
2. **Required tools installed**: docker, kubectl, kind, curl, kustomize
3. **Sufficient resources**: At least 8GB RAM available for the demo cluster

## Running the Demo

### Option 1: Automated Setup with Validation (Recommended)
```bash
cd demo
./run-and-validate-demo.sh
```
This script will:
- Set up the entire demo environment
- Validate all components are working
- Automatically fix common issues
- Re-validate until everything passes

### Option 2: Manual Setup
```bash
cd demo
./setup.sh                    # Set up the demo
./validate-demo.sh           # Validate everything works
./monitor-demo.sh            # Monitor real-time status
```

## Expected Timeline

### 0-2 Minutes: Initial Setup
- Cluster creation
- CRD installation
- Namespace creation
- Operator deployment

### 2-5 Minutes: Component Initialization
- Metrics-server becomes available
- Prometheus starts collecting metrics
- Grafana becomes accessible
- Demo applications start

### 5-10 Minutes: AI Activation
- Ollama model loading (background)
- First healing policies evaluated
- Initial healing actions created
- Metrics start appearing in Prometheus

### 10+ Minutes: Full Demo Activity
- 20+ healing actions created
- Multiple delete actions executed
- AI confidence scores stabilize
- All dashboard panels populated

## Grafana Dashboard Validation

### Accessing the Dashboard
1. Navigate to http://localhost:3000
2. Login with admin/admin
3. Find "KubeSkippy Enhanced AI Healing Overview" dashboard

### Key Panels to Validate

#### Overview Section (Top Row)
- **Total Healing Actions**: Should show increasing count
- **Success Rate**: Should be >90%
- **Active Healing Policies**: Should show 6-8 policies
- **AI Backend Status**: Should show "1" (UP)

#### Healing Actions Section
- **Healing Actions Over Time**: Graph showing action creation rate
- **Healing Actions by Type**: Pie chart with restart, scale, patch, delete
- **Healing Actions Table**: List of recent actions with timestamps
- **Policy Evaluation Rate**: Shows evaluations per minute

#### Application Health Section
- **CPU Usage**: Shows spikes from continuous-cpu-oscillation
- **Memory Usage**: Shows growth from memory-degradation apps
- **Pod Restart Counts**: Increases for crashloop apps
- **Application Status**: Shows pod states

#### AI Analysis Section
- **AI Confidence Level**: Gauge showing 70-95%
- **AI vs Traditional Effectiveness**: Comparison metrics
- **AI Action Distribution**: Breakdown of AI decisions
- **AI Decision Timeline**: Time series of AI actions

## Common Issues and Fixes

### No Healing Actions Created
**Symptoms**: `kubectl get healingactions -n demo-apps` returns empty
**Fix**: 
1. Check operator logs: `kubectl logs -n kubeskippy-system deployment/kubeskippy-controller-manager`
2. Verify policies are in automatic mode: `kubectl get healingpolicies -n demo-apps -o yaml | grep mode`
3. Force a trigger: `kubectl delete pod -n demo-apps -l app=chaos-monkey-component`

### Grafana Shows No Data
**Symptoms**: Panels show "No data" or empty graphs
**Fix**:
1. Verify Prometheus is scraping: http://localhost:9090/targets
2. Check metrics exist: http://localhost:9090/api/v1/label/__name__/values
3. Ensure time range in Grafana is set to "Last 15 minutes"

### AI Backend Not Responding
**Symptoms**: AI Backend Status shows 0 or DOWN
**Fix**:
1. Check Ollama pod: `kubectl get pods -n kubeskippy-system -l app=ollama`
2. Check model loading: `kubectl logs -n kubeskippy-system job/ollama-model-loader`
3. Model loading can take 5-10 minutes on first run

### Port Forwarding Issues
**Symptoms**: Cannot access Grafana or Prometheus
**Fix**:
1. Kill existing forwards: `pkill -f "kubectl port-forward"`
2. Restart: `./start-port-forwards.sh`
3. Verify: `ps aux | grep port-forward`

## Validation Checklist

### Core Functionality
- [ ] All pods running in demo-apps namespace
- [ ] Healing actions being created (>10 after 10 minutes)
- [ ] Delete actions present (>3 after 10 minutes)
- [ ] AI confidence scores >70%

### Dashboard Data
- [ ] Overview metrics populated
- [ ] Time series graphs showing data
- [ ] Tables listing recent actions
- [ ] AI analysis panels showing metrics

### Performance
- [ ] Actions created within 2 minutes of triggers
- [ ] No excessive CPU/memory usage
- [ ] No pod evictions or crashes (except intentional demo apps)

## Demo Talking Points

When presenting the demo, highlight:

1. **AI Intelligence**: Show the AI confidence gauge and reasoning
2. **Proactive Healing**: Point out predictive actions before failures
3. **Strategic Deletes**: Emphasize AI making hard decisions (deleting pods)
4. **Comparison**: Show AI vs traditional effectiveness metrics
5. **Real-time Analysis**: Demonstrate live healing as it happens

## Troubleshooting Commands

```bash
# Quick health check
./monitor-demo.sh

# Watch healing velocity
watch -n 5 'kubectl get healingactions -n demo-apps --no-headers | wc -l'

# Check AI activity
kubectl logs -f -n kubeskippy-system deployment/kubeskippy-controller-manager | grep -E "(AI|confidence|reasoning)"

# Verify metrics
curl -s localhost:8080/metrics | grep kubeskippy

# Force healing action
kubectl delete pod -n demo-apps $(kubectl get pod -n demo-apps -o name | head -1)
```

## Clean Up

To completely remove the demo:
```bash
./cleanup-demo.sh
```

This removes:
- Kind cluster
- Docker images
- Temporary files
- Port forwards