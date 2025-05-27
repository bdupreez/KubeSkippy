# KubeSkippy Demo

This demo showcases KubeSkippy's auto-healing capabilities with various problematic applications and healing policies.

## Prerequisites

- Docker installed and running
- kubectl installed
- Kind (Kubernetes in Docker) installed
- make installed
- At least 8GB of available RAM

## Quick Start

1. **Start the demo cluster:**
   ```bash
   make demo-up
   ```

2. **Deploy the operator:**
   ```bash
   make demo-deploy-operator
   ```

3. **Deploy demo applications:**
   ```bash
   make demo-deploy-apps
   ```

4. **Apply healing policies:**
   ```bash
   make demo-apply-policies
   ```

5. **Watch the magic happen:**
   ```bash
   make demo-watch
   ```

## Demo Applications

### 1. CrashLoop App
- **Issue**: Application crashes every 30 seconds
- **Symptoms**: High restart count, CrashLoopBackOff status
- **Healing**: Automatic restart with debug mode enabled

### 2. Memory Leak App
- **Issue**: Gradually consumes more memory
- **Symptoms**: Increasing memory usage, eventual OOMKilled
- **Healing**: Rolling restart when memory exceeds 85%

### 3. CPU Spike App
- **Issue**: Periodic CPU spikes causing throttling
- **Symptoms**: High CPU usage, performance degradation
- **Healing**: Horizontal scaling and resource limit adjustment

### 4. Flaky Web App
- **Issue**: Random HTTP errors and timeouts
- **Symptoms**: Intermittent 5xx errors, failed health checks
- **Healing**: Rolling restart or temporary scaling

## Healing Policies

### Standard Policies

1. **crashloop-pod-healing**: Handles pods in CrashLoopBackOff
2. **memory-leak-healing**: Manages high memory usage
3. **cpu-spike-healing**: Responds to CPU spikes
4. **service-degradation-healing**: Maintains service SLOs

### AI-Driven Policy

The `ai-driven-healing` policy uses AI to analyze complex issues:
- Analyzes metrics, events, and patterns
- Provides intelligent recommendations
- Requires approval for safety

## Monitoring the Demo

### View Healing Actions
```bash
kubectl get healingactions -n demo-apps -w
```

### Check Policy Status
```bash
kubectl get healingpolicies -n demo-apps
```

### View Operator Logs
```bash
kubectl logs -n kubeskippy-system deployment/kubeskippy-controller-manager -f
```

### Describe a Healing Action
```bash
kubectl describe healingaction -n demo-apps <action-name>
```

## Demo Scenarios

### Scenario 1: Automatic Pod Recovery
1. Watch the crashloop-app pods
2. Observe automatic restarts after 3 failures
3. Check the healing action created

### Scenario 2: Memory Leak Mitigation
1. Monitor memory usage of memory-leak-app
2. Watch for automatic restart at 85% usage
3. Verify memory is reclaimed

### Scenario 3: CPU Spike Handling
1. Observe CPU spikes in cpu-spike-app
2. Watch for automatic scaling
3. Check new resource limits applied

### Scenario 4: AI Analysis
1. Enable AI-driven healing
2. Create a complex failure scenario
3. Review AI recommendations
4. Approve suggested actions

## Troubleshooting

### Ollama Connection Issues
```bash
# Check Ollama status
kubectl get pods -n kubeskippy-system -l app=ollama

# View Ollama logs
kubectl logs -n kubeskippy-system -l app=ollama
```

### Healing Not Triggered
1. Check policy is enabled: `kubectl get healingpolicy <name> -o yaml`
2. Verify metrics collection: `kubectl logs -n kubeskippy-system deployment/kubeskippy-controller-manager | grep metrics`
3. Check safety rules aren't blocking: Look for rate limit messages

### Demo Cleanup
```bash
make demo-down
```

## Advanced Usage

### Custom Healing Policy
Create your own healing policy:
```yaml
apiVersion: kubeskippy.io/v1alpha1
kind: HealingPolicy
metadata:
  name: custom-healing
  namespace: demo-apps
spec:
  # Your custom configuration
```

### Dry Run Mode
Test policies without executing actions:
```bash
kubectl patch healingpolicy <name> -n demo-apps --type merge -p '{"spec":{"safetyRules":{"dryRun":true}}}'
```

### Manual Healing Action
Trigger healing manually:
```bash
kubectl create -f - <<EOF
apiVersion: kubeskippy.io/v1alpha1
kind: HealingAction
metadata:
  name: manual-restart
  namespace: demo-apps
spec:
  policyName: manual
  target:
    apiVersion: v1
    kind: Pod
    name: <pod-name>
  action:
    type: restart
EOF
```

## Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│ Metrics Server  │────▶│ KubeSkippy       │────▶│ Remediation     │
│ Prometheus      │     │ Controller       │     │ Engine          │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                               │                          │
                               ▼                          ▼
                        ┌──────────────────┐     ┌─────────────────┐
                        │ AI Analyzer      │     │ Safety          │
                        │ (Ollama/OpenAI)  │     │ Controller      │
                        └──────────────────┘     └─────────────────┘
```

## Contributing

To add new demo scenarios:
1. Create a new app in `demo/apps/`
2. Create a healing policy in `demo/policies/`
3. Update this README with the scenario
4. Submit a PR!

## Learn More

- [KubeSkippy Documentation](../README.md)
- [Healing Policy Reference](../docs/healing-policy.md)
- [Safety Rules Guide](../docs/safety-rules.md)
- [AI Integration Guide](../docs/ai-integration.md)