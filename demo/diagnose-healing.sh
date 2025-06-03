#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== KubeSkippy Healing Diagnostics ===${NC}"
echo ""

# Function to check and report
check_status() {
    local description="$1"
    local command="$2"
    local output=$(eval "$command" 2>&1)
    echo -e "${YELLOW}▶ $description${NC}"
    echo "$output"
    echo ""
}

# 1. Check healing policies
echo -e "${BLUE}1. Healing Policies Status${NC}"
check_status "All healing policies" "kubectl get healingpolicies -n demo-apps"
check_status "AI policies in automatic mode" "kubectl get healingpolicies -n demo-apps -o json | jq -r '.items[] | \"\(.metadata.name): mode=\(.spec.mode)\"'"

# 2. Check operator logs for errors
echo -e "${BLUE}2. Recent Operator Logs${NC}"
check_status "Last 20 operator logs" "kubectl logs -n kubeskippy-system deployment/kubeskippy-controller-manager --tail=20"
check_status "Operator errors" "kubectl logs -n kubeskippy-system deployment/kubeskippy-controller-manager --tail=100 | grep -i error | tail -5 || echo 'No errors found'"

# 3. Check metrics collection
echo -e "${BLUE}3. Metrics Collection${NC}"
check_status "Prometheus targets" "curl -s localhost:9090/api/v1/targets | jq '.data.activeTargets[] | {job: .labels.job, health: .health}' 2>/dev/null || echo 'Prometheus not accessible'"
check_status "KubeSkippy metrics" "curl -s localhost:8080/metrics 2>/dev/null | grep -E '^kubeskippy_' | head -10 || echo 'Metrics endpoint not accessible'"

# 4. Check demo app status
echo -e "${BLUE}4. Demo Applications${NC}"
check_status "Pod status" "kubectl get pods -n demo-apps"
check_status "Recent pod events" "kubectl get events -n demo-apps --sort-by='.lastTimestamp' | tail -10"

# 5. Check resource usage
echo -e "${BLUE}5. Resource Usage${NC}"
check_status "Node resources" "kubectl top nodes 2>/dev/null || echo 'Metrics not available yet'"
check_status "Pod resources" "kubectl top pods -n demo-apps 2>/dev/null || echo 'Metrics not available yet'"

# 6. Check healing actions
echo -e "${BLUE}6. Healing Actions${NC}"
action_count=$(kubectl get healingactions -n demo-apps --no-headers 2>/dev/null | wc -l || echo 0)
echo -e "Total healing actions: ${GREEN}$action_count${NC}"
if [ "$action_count" -gt 0 ]; then
    check_status "Recent healing actions" "kubectl get healingactions -n demo-apps --sort-by='.metadata.creationTimestamp' | tail -5"
    check_status "Action types" "kubectl get healingactions -n demo-apps -o json | jq -r '.items[].spec.type' | sort | uniq -c"
else
    echo -e "${RED}No healing actions created yet${NC}"
fi
echo ""

# 7. Check AI backend
echo -e "${BLUE}7. AI Backend (Ollama)${NC}"
check_status "Ollama pod status" "kubectl get pod -n kubeskippy-system -l app=ollama"
check_status "Ollama API" "kubectl exec -n kubeskippy-system deployment/ollama -- curl -s localhost:11434/api/tags 2>/dev/null | jq '.models[].name' 2>/dev/null || echo 'Ollama not ready or model not loaded'"

# 8. Check policy triggers
echo -e "${BLUE}8. Policy Trigger Evaluation${NC}"
echo "Checking if policies should be triggering..."

# Check CPU usage against policy thresholds
cpu_threshold=50
echo -n "Checking CPU usage > ${cpu_threshold}%: "
high_cpu_pods=$(kubectl top pods -n demo-apps 2>/dev/null | awk -v threshold=$cpu_threshold 'NR>1 && $3>threshold {print $1}' | wc -l || echo 0)
if [ "$high_cpu_pods" -gt 0 ]; then
    echo -e "${GREEN}$high_cpu_pods pods exceed threshold${NC}"
else
    echo -e "${YELLOW}No pods exceed threshold${NC}"
fi

# Check memory usage against policy thresholds
mem_threshold=25
echo -n "Checking Memory usage > ${mem_threshold}%: "
high_mem_pods=$(kubectl top pods -n demo-apps 2>/dev/null | awk -v threshold=$mem_threshold 'NR>1 && $5>threshold {print $1}' | wc -l || echo 0)
if [ "$high_mem_pods" -gt 0 ]; then
    echo -e "${GREEN}$high_mem_pods pods exceed threshold${NC}"
else
    echo -e "${YELLOW}No pods exceed threshold${NC}"
fi

# 9. Recommendations
echo ""
echo -e "${BLUE}9. Recommendations${NC}"

if [ "$action_count" -eq 0 ]; then
    echo "No healing actions detected. Try these steps:"
    echo "1. Wait 2-3 more minutes for metrics to stabilize"
    echo "2. Force a trigger: kubectl delete pod -n demo-apps \$(kubectl get pod -n demo-apps -o name | head -1)"
    echo "3. Check operator connection to Prometheus:"
    echo "   kubectl logs -n kubeskippy-system deployment/kubeskippy-controller-manager | grep -i prometheus"
    echo "4. Verify operator has correct RBAC permissions:"
    echo "   kubectl auth can-i create healingactions --as=system:serviceaccount:kubeskippy-system:kubeskippy-controller-manager -n demo-apps"
fi

if [ "$high_cpu_pods" -eq 0 ] && [ "$high_mem_pods" -eq 0 ]; then
    echo ""
    echo "Demo apps may not be generating enough load. Check if they're running correctly:"
    echo "  kubectl logs -n demo-apps deployment/continuous-cpu-oscillation --tail=10"
    echo "  kubectl logs -n demo-apps deployment/continuous-memory-degradation --tail=10"
fi

echo ""
echo -e "${GREEN}Diagnostics complete!${NC}"