# K8s AI Auto-Healing Operator Design

## Architecture Overview

```
┌─────────────────┐     ┌──────────────┐     ┌─────────────┐
│  Prometheus     │────▶│   Operator   │────▶│  Local AI   │
│  Metrics Server │     │              │     │  (Ollama)   │
└─────────────────┘     └──────┬───────┘     └─────────────┘
                               │
                               ▼
                      ┌─────────────────┐
                      │  Remediation    │
                      │  Engine         │
                      └─────────────────┘
```

## Core Components

### 1. Metrics Collector
- Pull metrics from Prometheus/metrics-server
- Watch Kubernetes events
- Monitor pod/node health status
- Track error patterns

### 2. AI Analysis Engine
- Format cluster state into prompts
- Send to local Ollama instance
- Parse AI recommendations
- Validate suggestions against safety rules

### 3. Safety Controller
- Maintain list of protected resources
- Validate all actions before execution
- Implement circuit breaker pattern
- Audit log all decisions

### 4. Remediation Engine
- Execute approved healing actions
- Support dry-run mode
- Rollback capabilities
- Action types:
  - Pod restarts
  - HPA adjustments
  - Node cordon/drain (with approval)
  - PVC expansion
  - Resource limit adjustments

## Safety Mechanisms

### Protected Resources (Never Delete/Modify)
```yaml
protectedResources:
  namespaces:
    - kube-system
    - kube-public
    - kube-node-lease
    - cert-manager
    - ingress-nginx
    - monitoring
  resources:
    - apiVersion: "*"
      kind: "CustomResourceDefinition"
    - apiVersion: "v1"
      kind: "PersistentVolume"
  labels:
    - "app.kubernetes.io/managed-by=Helm"
    - "nanny.ai/protected=true"
```

### Action Approval Levels
1. **Auto-approve** (safe actions):
   - Restart pods with CrashLoopBackOff
   - Increase HPA max replicas (within limits)
   - Delete completed jobs > 24h old

2. **Requires Confirmation**:
   - Node operations
   - PVC modifications
   - Namespace-wide actions

3. **Never Allow**:
   - Delete statefulsets
   - Modify RBAC
   - Delete PVs

## Implementation Phases

### Phase 1: Monitoring Only
- Collect metrics
- Generate reports
- No actions taken

### Phase 2: Dry Run Mode
- Suggest actions
- Log what would happen
- Build confidence

### Phase 3: Limited Auto-Healing
- Auto-restart crashed pods
- Clean up completed resources
- Simple, safe actions only

### Phase 4: Full Auto-Healing
- Complex remediation
- Predictive healing
- Cost optimization

## Local AI Integration

### Ollama Setup
```yaml
aiConfig:
  provider: ollama
  model: llama2:13b  # or mistral, codellama
  endpoint: http://ollama-service:11434
  timeout: 30s
  maxTokens: 2048
```

### Prompt Engineering
```
System: You are a Kubernetes cluster healing assistant. Analyze the cluster state and suggest ONLY safe remediation actions.

Context:
- Cluster Version: {version}
- Node Count: {nodes}
- Problem Summary: {issues}

Rules:
- Never suggest deleting stateful workloads
- Prefer restart over delete
- Consider resource limits before scaling

Question: What actions would heal these issues?
```

## Monitoring Integration

### Prometheus Queries
```promql
# Pod restart detection
rate(kube_pod_container_status_restarts_total[15m]) > 0

# Memory pressure
(node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) < 0.1

# CPU throttling
rate(container_cpu_cfs_throttled_periods_total[5m]) > 0.5
```

## Testing Strategy

### Unit Tests
- Mock Kubernetes client
- Test safety rules
- Validate AI prompt generation

### Integration Tests
- Use Kind cluster
- Inject failures
- Verify healing actions

### Chaos Testing
- Use Chaos Mesh
- Verify operator resilience
- Test rollback mechanisms

## Deployment Architecture

### GitOps Structure
```
k8s-ai-nanny/
├── .github/workflows/      # CI pipelines
├── config/
│   ├── crd/               # CRDs
│   ├── rbac/              # RBAC rules
│   └── samples/           # Example configs
├── helm/
│   └── ai-nanny/          # Helm chart
├── environments/
│   ├── dev/               # Dev overrides
│   ├── staging/           # Staging config
│   └── prod/              # Prod config
└── tests/
    ├── e2e/               # End-to-end tests
    └── chaos/             # Chaos scenarios
```

### CI/CD Pipeline
1. **PR Pipeline**: Lint, test, build
2. **Main Pipeline**: Build, push images, update manifests
3. **ArgoCD**: Auto-sync to clusters
4. **Monitoring**: Grafana dashboards for operator metrics

## Metrics & Observability

### Custom Metrics
```go
healingActionsTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "nanny_healing_actions_total",
        Help: "Total number of healing actions taken",
    },
    []string{"action_type", "namespace", "status"},
)

aiAnalysisLatency = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name: "nanny_ai_analysis_duration_seconds",
        Help: "Latency of AI analysis",
    },
    []string{"model"},
)
```

## Next Steps
1. Set up development environment
2. Create GitOps repository structure
3. Implement basic operator with CRDs
4. Add Ollama integration
5. Build safety controls
6. Implement monitoring integration
7. Create remediation engine
8. Set up CI/CD pipeline