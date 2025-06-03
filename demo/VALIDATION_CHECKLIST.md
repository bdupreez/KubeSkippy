# KubeSkippy Demo Validation Checklist

## Pre-Demo Cleanup
- [ ] No existing Kind cluster: `kind get clusters | grep -q kubeskippy-demo && echo "EXISTS"`
- [ ] No lingering port forwards: `pgrep -f "kubectl port-forward"`
- [ ] Clean docker state: `docker ps -a | grep kubeskippy`

## Demo Setup Validation
- [ ] Setup script completes without errors
- [ ] All namespaces created: `kubeskippy-system`, `demo-apps`, `monitoring`
- [ ] CRDs installed successfully
- [ ] Operator image built and loaded

## Component Health Checks
### Infrastructure
- [ ] Metrics-server running: `kubectl get deploy -n kube-system metrics-server`
- [ ] Kube-state-metrics running: `kubectl get deploy -n kube-system kube-state-metrics`
- [ ] Pod metrics available: `kubectl top pods -n demo-apps` (should work after 2 min)

### Monitoring Stack
- [ ] Prometheus running: `kubectl get pods -n monitoring -l app=prometheus`
- [ ] Prometheus targets healthy: `curl -s localhost:9090/api/v1/targets | jq '.data.activeTargets[].health' | grep -c up`
- [ ] Grafana running: `kubectl get pods -n monitoring -l app=grafana`
- [ ] Grafana accessible: `curl -s -u admin:admin localhost:3000/api/health | grep -q ok`

### KubeSkippy Operator
- [ ] Operator running: `kubectl get pods -n kubeskippy-system -l control-plane=controller-manager`
- [ ] Operator logs clean: `kubectl logs -n kubeskippy-system deployment/kubeskippy-controller-manager | grep -i error | wc -l` (should be 0 or minimal)
- [ ] Metrics service exists: `kubectl get svc -n kubeskippy-system kubeskippy-controller-manager-metrics`

### AI Backend (Ollama)
- [ ] Ollama running: `kubectl get pods -n kubeskippy-system -l app=ollama`
- [ ] Model loaded: `kubectl logs -n kubeskippy-system job/ollama-model-loader | grep -q "successfully loaded"`
- [ ] AI endpoint responsive: `kubectl exec -n kubeskippy-system deploy/ollama -- curl -s localhost:11434/api/tags | grep -q llama2`

### Demo Applications
- [ ] All apps deployed: `kubectl get deployments -n demo-apps --no-headers | wc -l` (should be 8+)
- [ ] Continuous failure apps running:
  - [ ] continuous-memory-degradation
  - [ ] continuous-cpu-oscillation
  - [ ] continuous-network-degradation
  - [ ] chaos-monkey-component
- [ ] Apps have correct labels: `kubectl get pods -n demo-apps -l demo=kubeskippy --no-headers | wc -l` (should be 5+)

### Healing Policies
- [ ] AI policies created: `kubectl get healingpolicies -n demo-apps | grep -c ai-` (should be 4+)
- [ ] Policies in automatic mode: `kubectl get healingpolicies -n demo-apps -o json | jq '.items[].spec.mode' | grep -c automatic`

## Healing Actions Validation
- [ ] Actions being created: `kubectl get healingactions -n demo-apps --no-headers | wc -l` (should increase over time)
- [ ] AI-driven actions present: `kubectl get healingactions -n demo-apps | grep -c "ai-"`
- [ ] Delete actions present: `kubectl get healingactions -n demo-apps -o json | jq '.items[].spec.type' | grep -c delete` (target: 15+)
- [ ] Recent action timestamps: `kubectl get healingactions -n demo-apps --sort-by=.metadata.creationTimestamp | tail -5`

## Grafana Dashboard Validation

### Access Check
- [ ] Dashboard loads: http://localhost:3000 (admin/admin)
- [ ] Enhanced dashboard exists: "KubeSkippy Enhanced AI Healing Overview"
- [ ] No authentication errors
- [ ] Prometheus datasource connected

### Panel Data Validation
#### Overview Section
- [ ] Total Healing Actions: Shows count > 0
- [ ] Success Rate: Shows percentage
- [ ] Active Healing Policies: Shows count
- [ ] AI Backend Status: Shows "UP" or 1

#### Healing Actions Section  
- [ ] Healing Actions Over Time: Shows time series data
- [ ] Healing Actions by Type: Shows breakdown (restart, scale, patch, delete)
- [ ] Healing Actions Table: Lists recent actions with details
- [ ] Policy Evaluation Rate: Shows evaluations/min

#### Application Health Section
- [ ] Target Application CPU Usage: Shows metrics for demo apps
- [ ] Target Application Memory Usage: Shows metrics for demo apps
- [ ] Pod Restart Counts: Shows restart data
- [ ] Application Status: Shows pod states

#### AI Analysis Section
- [ ] AI Confidence Level: Shows gauge 0-100%
- [ ] AI vs Traditional Effectiveness: Shows comparison metrics
- [ ] AI Action Distribution: Shows pie chart of action types
- [ ] AI Decision Timeline: Shows AI actions over time
- [ ] AI Reasoning Table: Shows reasoning steps (may need AI actions first)

#### System Metrics Section
- [ ] Cluster CPU Usage: Shows node metrics
- [ ] Cluster Memory Usage: Shows node metrics
- [ ] Network I/O: Shows network metrics
- [ ] Prometheus Targets: Shows healthy targets

## Metrics Validation
Check key metrics exist in Prometheus:
- [ ] `kubeskippy_healing_actions_total`: http://localhost:9090/api/v1/query?query=kubeskippy_healing_actions_total
- [ ] `kubeskippy_ai_confidence_score`: http://localhost:9090/api/v1/query?query=kubeskippy_ai_confidence_score
- [ ] `kubeskippy_policy_evaluations_total`: http://localhost:9090/api/v1/query?query=kubeskippy_policy_evaluations_total
- [ ] `container_cpu_usage_seconds_total`: http://localhost:9090/api/v1/query?query=container_cpu_usage_seconds_total
- [ ] `container_memory_usage_bytes`: http://localhost:9090/api/v1/query?query=container_memory_usage_bytes

## Performance Validation
- [ ] Demo apps creating sufficient load: CPU/Memory spikes visible
- [ ] Healing actions response time: < 2 minutes from trigger to action
- [ ] No excessive resource usage: `kubectl top nodes`
- [ ] No pod evictions or OOM kills: `kubectl get events -A | grep -i evict`

## Expected Outcomes (After 10 minutes)
- [ ] 20+ healing actions created
- [ ] 5+ delete actions executed  
- [ ] AI confidence scores > 70%
- [ ] Multiple action types demonstrated
- [ ] Continuous failure patterns visible in metrics
- [ ] AI vs traditional comparison shows AI superiority

## Common Issues to Check
- [ ] Port forwarding active: `./start-port-forwards.sh`
- [ ] Metrics delay: Wait 3-5 minutes for initial metrics
- [ ] Model loading: Check ollama job completion
- [ ] Policy cooldowns: Check if policies are in cooldown period
- [ ] Resource limits: Ensure cluster has sufficient resources

## Validation Commands
```bash
# Quick health check
./monitor-demo.sh

# Check healing velocity
watch -n 5 'kubectl get healingactions -n demo-apps --no-headers | wc -l'

# Monitor AI activity
kubectl logs -f -n kubeskippy-system deployment/kubeskippy-controller-manager | grep -i "ai\|confidence\|reasoning"

# Check metric emission
curl -s localhost:8080/metrics | grep kubeskippy
```