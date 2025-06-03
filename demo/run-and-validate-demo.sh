#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo -e "${BLUE}=== KubeSkippy Demo Runner with Auto-Fix ===${NC}"
echo -e "${BLUE}This script will set up, validate, and fix any issues automatically${NC}"
echo ""

# Function to apply fixes based on validation failures
apply_fixes() {
    local fix_applied=false
    
    echo -e "${YELLOW}▶ Analyzing issues and applying fixes...${NC}"
    
    # Fix 1: Check if metrics are not being emitted
    if ! curl -s 'http://localhost:9090/api/v1/query?query=kubeskippy_healing_actions_total' 2>/dev/null | grep -q '"result":\[{'; then
        echo "  • Fixing: Metrics not being emitted - restarting operator"
        kubectl rollout restart deployment/kubeskippy-controller-manager -n kubeskippy-system
        fix_applied=true
        sleep 30
    fi
    
    # Fix 2: Check if AI policies are not in automatic mode
    if ! kubectl get healingpolicies -n demo-apps -o json | jq -r '.items[] | select(.metadata.name | contains("ai-")) | .spec.mode' | grep -q automatic; then
        echo "  • Fixing: AI policies not in automatic mode"
        for policy in $(kubectl get healingpolicies -n demo-apps -o name | grep ai-); do
            kubectl patch $policy -n demo-apps --type merge -p '{"spec":{"mode":"automatic"}}'
        done
        fix_applied=true
    fi
    
    # Fix 3: Check if demo apps don't have correct labels
    if [ $(kubectl get pods -n demo-apps -l demo=kubeskippy --no-headers | wc -l) -lt 4 ]; then
        echo "  • Fixing: Demo app labels"
        kubectl patch deployment continuous-memory-degradation -n demo-apps --type='merge' -p='{"metadata":{"labels":{"demo":"kubeskippy"}},"spec":{"template":{"metadata":{"labels":{"demo":"kubeskippy"}}}}}'
        kubectl patch deployment continuous-cpu-oscillation -n demo-apps --type='merge' -p='{"metadata":{"labels":{"demo":"kubeskippy"}},"spec":{"template":{"metadata":{"labels":{"demo":"kubeskippy"}}}}}'
        kubectl patch deployment chaos-monkey-component -n demo-apps --type='merge' -p='{"metadata":{"labels":{"demo":"kubeskippy"}},"spec":{"template":{"metadata":{"labels":{"demo":"kubeskippy"}}}}}'
        fix_applied=true
        sleep 20
    fi
    
    # Fix 4: Ensure port forwards are running
    if ! pgrep -f "kubectl port-forward.*grafana" >/dev/null || ! pgrep -f "kubectl port-forward.*prometheus" >/dev/null; then
        echo "  • Fixing: Port forwards not active"
        ./start-port-forwards.sh
        fix_applied=true
        sleep 10
    fi
    
    # Fix 5: Force trigger healing actions if none exist after 5 minutes
    local action_count=$(kubectl get healingactions -n demo-apps --no-headers 2>/dev/null | wc -l || echo 0)
    if [ "$action_count" -eq 0 ] && [ "$VALIDATION_ATTEMPT" -gt 1 ]; then
        echo "  • Fixing: No healing actions - forcing trigger evaluation"
        
        # Patch a deployment to force a change that triggers policies
        kubectl patch deployment continuous-memory-degradation -n demo-apps --type='merge' \
            -p='{"spec":{"template":{"metadata":{"annotations":{"kubeskippy.io/force-trigger":"'$(date +%s)'"}}}}}'
        
        # Scale a deployment to trigger events
        kubectl scale deployment continuous-cpu-oscillation -n demo-apps --replicas=2
        sleep 5
        kubectl scale deployment continuous-cpu-oscillation -n demo-apps --replicas=1
        
        fix_applied=true
        sleep 30
    fi
    
    # Fix 6: Restart Ollama if AI is not responding
    if ! kubectl exec -n kubeskippy-system deployment/ollama -- curl -s localhost:11434/api/tags 2>/dev/null | grep -q llama2; then
        echo "  • Fixing: Ollama not responding - checking model loader"
        
        # Check if model loader job exists and is complete
        if kubectl get job ollama-model-loader -n kubeskippy-system >/dev/null 2>&1; then
            job_status=$(kubectl get job ollama-model-loader -n kubeskippy-system -o jsonpath='{.status.conditions[?(@.type=="Complete")].status}' 2>/dev/null || echo "Unknown")
            if [ "$job_status" != "True" ]; then
                echo "    Model still loading - this is normal, continuing..."
            fi
        else
            echo "    Model loader job missing - this may affect AI features"
        fi
    fi
    
    # Fix 7: Update operator environment if AI is not configured
    if ! kubectl get deployment kubeskippy-controller-manager -n kubeskippy-system -o json | jq '.spec.template.spec.containers[0].env[] | select(.name=="AI_PROVIDER")' | grep -q ollama; then
        echo "  • Fixing: AI provider not configured in operator"
        kubectl patch deployment kubeskippy-controller-manager -n kubeskippy-system --type='json' -p='[
            {"op": "add", "path": "/spec/template/spec/containers/0/env/-", "value": {"name": "AI_PROVIDER", "value": "ollama"}},
            {"op": "add", "path": "/spec/template/spec/containers/0/env/-", "value": {"name": "AI_MODEL", "value": "llama2:7b"}},
            {"op": "add", "path": "/spec/template/spec/containers/0/env/-", "value": {"name": "AI_ENDPOINT", "value": "http://ollama:11434"}}
        ]' 2>/dev/null || echo "    Environment already configured"
        fix_applied=true
        sleep 30
    fi
    
    if $fix_applied; then
        echo -e "${GREEN}  ✓ Fixes applied${NC}"
        return 0
    else
        echo -e "${YELLOW}  No fixes needed${NC}"
        return 1
    fi
}

# Run setup if cluster doesn't exist
if ! kind get clusters 2>/dev/null | grep -q kubeskippy-demo; then
    echo -e "${YELLOW}▶ Running demo setup...${NC}"
    ./setup.sh
    echo -e "${GREEN}✓ Setup completed${NC}"
    
    # Wait for initial stabilization
    echo -e "${YELLOW}▶ Waiting for initial stabilization (2 minutes)...${NC}"
    sleep 120
else
    echo -e "${GREEN}✓ Demo cluster already exists${NC}"
fi

# Validation loop
MAX_ATTEMPTS=3
VALIDATION_ATTEMPT=0

while [ $VALIDATION_ATTEMPT -lt $MAX_ATTEMPTS ]; do
    VALIDATION_ATTEMPT=$((VALIDATION_ATTEMPT + 1))
    
    echo ""
    echo -e "${BLUE}=== Validation Attempt $VALIDATION_ATTEMPT/$MAX_ATTEMPTS ===${NC}"
    
    # Run validation
    if ./validate-demo.sh; then
        echo ""
        echo -e "${GREEN}✅ Demo validation PASSED!${NC}"
        break
    else
        echo ""
        echo -e "${YELLOW}⚠ Validation found issues${NC}"
        
        if [ $VALIDATION_ATTEMPT -lt $MAX_ATTEMPTS ]; then
            # Apply fixes
            apply_fixes
            
            # Wait before next validation
            echo -e "${YELLOW}▶ Waiting 30 seconds before next validation...${NC}"
            sleep 30
        else
            echo -e "${RED}❌ Validation failed after $MAX_ATTEMPTS attempts${NC}"
            echo "Run ./monitor-demo.sh to investigate"
            exit 1
        fi
    fi
done

# Final status check
echo ""
echo -e "${BLUE}=== Demo Status ===${NC}"
./monitor-demo.sh

echo ""
echo -e "${BLUE}=== Next Steps ===${NC}"
echo "1. Open Grafana: http://localhost:3000 (admin/admin)"
echo "2. Navigate to 'KubeSkippy Enhanced AI Healing Overview' dashboard"
echo "3. Watch healing actions: kubectl get healingactions -n demo-apps -w"
echo "4. Monitor AI logs: kubectl logs -f -n kubeskippy-system deployment/kubeskippy-controller-manager | grep -i 'ai\|confidence'"
echo ""
echo -e "${GREEN}✨ Demo is ready for presentation!${NC}"