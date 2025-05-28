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

# Run demo
cd demo && ./setup.sh
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

## Known Issues

1. **HealingPolicy controller tests**: Some tests expect specific requeue intervals that may need adjustment
2. **Prometheus mock**: Must handle POST requests with form data, not GET with query params

## AI Integration

The project supports two AI backends:
- **Ollama**: For local LLM inference (default)
- **OpenAI**: For cloud-based analysis

AI is used for:
- Root cause analysis
- Pattern recognition
- Healing recommendations
- Anomaly detection

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