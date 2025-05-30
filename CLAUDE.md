# Claude Assistant Context for KubeSkippy

This file provides context for Claude AI when working on the KubeSkippy project.

## Project Overview

KubeSkippy is a Kubernetes operator that provides self-healing capabilities for applications. It uses policy-based healing with optional AI-powered analysis to automatically detect, diagnose, and remediate issues.

## Key Components

1. **Custom Resources**:
   - `HealingPolicy`: Defines what to monitor and how to respond
   - `HealingAction`: Represents a remediation action to be taken

2. **Controllers**:
   - `HealingPolicyController`: Monitors policies and creates healing actions
   - `HealingActionController`: Executes healing actions with safety controls

3. **Core Systems**:
   - Metrics collection (Prometheus integration)
   - AI analysis (Ollama/OpenAI)
   - Remediation engine (restart, scale, patch, delete)
   - Safety controller (rate limiting, validation)
   - Enhanced Grafana monitoring with AI metrics

## Important Commands

```bash
# Run tests
make test

# Run specific tests
go test ./internal/controller/... -v
go test ./internal/metrics/... -v

# Run E2E tests
cd tests/e2e && ./run-tests.sh

# Build
make build

# Run locally
make run

# Deploy to cluster
make deploy

# Run demo with monitoring
cd demo && ./setup.sh --with-monitoring

# Access Grafana dashboard
# URL: http://localhost:3000 (admin/admin)
# Enhanced dashboard includes AI metrics section
```

## Coding Standards

1. **Error Handling**: Always wrap errors with context using `fmt.Errorf`
2. **Logging**: Use structured logging with `logr`
3. **Testing**: Write table-driven tests, use fake clients for controllers
4. **Comments**: Add comments for exported types and functions

## Architecture Notes

### Controller Reconciliation
- Controllers should handle ONE state transition per reconciliation
- Always update Status subresource before updating object metadata
- Use finalizers for cleanup logic
- Return appropriate Result (Requeue, RequeueAfter) based on state

### Status Updates Pattern
```go
// Correct order - status first, then object
if err := r.Status().Update(ctx, action); err != nil {
    return ctrl.Result{}, err
}
if err := r.Update(ctx, action); err != nil {
    return ctrl.Result{}, err
}
```

### Test Patterns
- Use `fake.NewClientBuilder().WithStatusSubresource()` for controller tests
- Mock external dependencies (AI clients, metrics collectors)
- Simulate multiple reconciliations for state transitions

## Recent Updates (2025-05-30)

### AI Strategic Healing Implementation ✅
- **AI Strategic Deletes**: Intelligent pod removal with high priority (Priority 5)
- **AI Resource Optimization**: Smart scaling decisions based on predictive analysis (Priority 8)
- **AI Emergency Deletes**: Cascade prevention with highest priority (Priority 1)
- **AI System Patches**: Intelligent configuration optimization (Priority 15)
- **Predictive Healing**: Early intervention at 30% memory, 40% CPU thresholds

### Continuous Failure Generation ✅
- **continuous-memory-degradation**: Gradual memory increase with 60s cycles
- **continuous-cpu-oscillation**: Sine wave CPU patterns with escalation
- **continuous-network-degradation**: Progressive network latency increases
- **chaos-monkey-component**: Random unpredictable failures every 30s

### Enhanced AI Analytics ✅
- **Strategic Action Metrics**: Dedicated tracking for delete, scale, patch, restart actions
- **AI Confidence Scoring**: Real-time confidence levels in Grafana dashboard
- **Predictive vs Traditional**: Clear differentiation showing AI superiority
- **Rate Limit Optimization**: Traditional policies reduced (1-2 actions/hour) to showcase AI

### Demo Environment Automation ✅
- **Default AI Setup**: `./setup.sh` includes AI strategic healing by default
- **Enhanced Grafana Dashboard**: 🤖 AI Analysis & Healing section with strategic action tracking
- **Continuous Activity**: 70+ healing actions with 15 delete actions consistently generated
- **Parallel Deployment**: Optimized 5-minute setup with monitoring stack included

### Current State
- ✅ AI Strategic Deletes actively working (15+ actions demonstrated)
- ✅ AI Resource Optimization scaling based on predictive analysis
- ✅ Continuous failure apps generating predictable patterns for AI learning
- ✅ Enhanced Grafana dashboard showing AI vs traditional action ratios
- ✅ Prometheus metrics recording all strategic AI actions with proper labels

## Known Optimizations

1. **Traditional Policy Rates**: Reduced to 1-2 actions/hour to showcase AI capabilities
2. **AI Action Approval**: Strategic deletes now automatic for demo purposes (requiresApproval: false)
3. **Metrics Visibility**: All AI strategic actions properly labeled for dashboard filtering

## AI Integration

The project supports two AI backends:
- **Ollama**: For local LLM inference (default)
- **OpenAI**: For cloud-based analysis

AI is used for:
- **Strategic Decision Making**: AI Strategic Deletes, Resource Optimization
- **Predictive Healing**: Early intervention before traditional thresholds
- **Cascade Prevention**: Emergency interventions to prevent system failures
- **Root cause analysis**: Pattern recognition and anomaly detection
- **Confidence Scoring**: Multi-dimensional analysis with reasoning annotations

## Safety Considerations

1. **Protected Resources**: System namespaces and labeled resources are protected
2. **Rate Limiting**: Configurable per policy to prevent action storms
3. **Dry Run Mode**: Test policies without executing actions
4. **Validation**: All actions are validated before execution

## Common Tasks

### Adding a New Action Type
1. Add to `api/v1alpha1/healingpolicy_types.go`
2. Implement executor in `internal/remediation/`
3. Register in `internal/remediation/engine.go`
4. Add tests

### Adding a New Trigger Type
1. Add to `api/v1alpha1/healingpolicy_types.go`
2. Implement evaluation in `internal/controller/healingpolicy_controller.go`
3. Add metrics collection if needed
4. Add tests

## Testing Philosophy

- Unit tests for business logic
- Integration tests for controllers
- E2E tests for user scenarios
- Mock external dependencies
- Test error cases thoroughly

## Performance Guidelines

- Reconciliation should complete in <5 seconds
- Metrics collection should be cached when possible
- AI calls should have timeouts (default: 30s)
- Use pagination for large resource lists