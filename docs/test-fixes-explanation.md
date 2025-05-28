# Test Fixes Explanation

This document explains the test issues that were fixed and the solutions applied.

## Issues Fixed

### 1. HealingAction Controller State Transitions

**Problem**: Tests were failing because the controller wasn't properly updating the action's phase status.

**Root Cause**: The controller was trying to update both the object and its status subresource in the wrong order, causing conflicts with the fake client used in tests.

**Solution**: 
- Reordered operations to always update the status subresource first, then update object metadata/labels
- This follows Kubernetes controller best practices for handling status subresources

**Code Changes**:
```go
// Before: Update object first, then status
if err := r.Update(ctx, action); err != nil { ... }
if err := r.Status().Update(ctx, action); err != nil { ... }

// After: Update status first, then object
if err := r.Status().Update(ctx, action); err != nil { ... }
if err := r.Update(ctx, action); err != nil { ... }
```

### 2. Test Reconciliation Expectations

**Problem**: Tests expected multiple phase transitions to happen in a single reconciliation call.

**Root Cause**: This expectation doesn't match real Kubernetes controller behavior, where each reconciliation typically handles one state transition.

**Solution**:
- Created a `reconcileUntilPhase` helper function that simulates multiple reconciliation loops
- Updated tests to accept that transitions like `Pending -> Approved -> InProgress -> Succeeded` require multiple reconciliations

**Best Practice**: Controllers should be idempotent and handle one state transition per reconciliation.

### 3. Prometheus Mock Server

**Problem**: Prometheus query tests were failing with "query returned no data" errors.

**Root Cause**: The mock server expected GET requests with query parameters, but the prometheus client-go library uses POST requests with form-encoded data.

**Solution**:
```go
// Parse POST form data instead of URL query parameters
err := r.ParseForm()
if err != nil {
    http.Error(w, "Bad request", http.StatusBadRequest)
    return
}
query := r.FormValue("query")
```

### 4. Prometheus Health Check

**Problem**: IsHealthy() tests were failing.

**Root Cause**: The mock server was using the wrong endpoint path for the config API.

**Solution**: Changed endpoint from `/api/v1/config` to `/api/v1/status/config` to match the actual Prometheus API.

### 5. Nil Pointer Prevention

**Problem**: Tests were panicking with nil pointer dereferences when accessing action.Labels.

**Root Cause**: The Labels map wasn't initialized before use.

**Solution**: Added nil checks and initialization:
```go
if action.Labels == nil {
    action.Labels = make(map[string]string)
}
```

## Testing Best Practices Applied

1. **Status Subresource Handling**: Always update status before updating the main object
2. **Idempotent Reconciliation**: Each reconciliation should handle one logical operation
3. **Proper Mock Implementation**: Mock servers should match the actual API behavior
4. **Defensive Programming**: Always check for nil maps/slices before use
5. **Test Realism**: Tests should simulate real Kubernetes behavior with multiple reconciliation loops

## Verification

All tests now pass:
- ✅ HealingAction controller tests
- ✅ Prometheus metrics tests  
- ✅ No nil pointer panics
- ✅ Proper status update handling