#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
PASS=0
FAIL=0
WARNINGS=0

# Validation functions
check() {
    local description="$1"
    local command="$2"
    local expected="${3:-0}"  # Default expected is success (0)
    
    echo -n "Checking: $description... "
    
    if eval "$command" >/dev/null 2>&1; then
        if [ "$expected" -eq 0 ]; then
            echo -e "${GREEN}✓${NC}"
            ((PASS++))
            return 0
        else
            echo -e "${RED}✗ (expected failure but succeeded)${NC}"
            ((FAIL++))
            return 1
        fi
    else
        if [ "$expected" -ne 0 ]; then
            echo -e "${GREEN}✓${NC}"
            ((PASS++))
            return 0
        else
            echo -e "${RED}✗${NC}"
            ((FAIL++))
            return 1
        fi
    fi
}

check_with_output() {
    local description="$1"
    local command="$2"
    local expected_output="$3"
    
    echo -n "Checking: $description... "
    
    local output=$(eval "$command" 2>/dev/null || echo "ERROR")
    if [[ "$output" == *"$expected_output"* ]]; then
        echo -e "${GREEN}✓${NC} ($output)"
        ((PASS++))
        return 0
    else
        echo -e "${RED}✗${NC} (got: $output, expected: $expected_output)"
        ((FAIL++))
        return 1
    fi
}

check_count() {
    local description="$1"
    local command="$2"
    local operator="$3"
    local expected="$4"
    
    echo -n "Checking: $description... "
    
    local count=$(eval "$command" 2>/dev/null || echo 0)
    local result=false
    
    case "$operator" in
        ">=") [ "$count" -ge "$expected" ] && result=true ;;
        ">")  [ "$count" -gt "$expected" ] && result=true ;;
        "==") [ "$count" -eq "$expected" ] && result=true ;;
        "<")  [ "$count" -lt "$expected" ] && result=true ;;
        "<=") [ "$count" -le "$expected" ] && result=true ;;
    esac
    
    if $result; then
        echo -e "${GREEN}✓${NC} (count: $count)"
        ((PASS++))
        return 0
    else
        echo -e "${RED}✗${NC} (got: $count, expected: $operator $expected)"
        ((FAIL++))
        return 1
    fi
}

wait_for_condition() {
    local description="$1"
    local command="$2"
    local timeout="${3:-60}"
    local interval="${4:-5}"
    
    echo -n "Waiting for: $description... "
    
    local elapsed=0
    while [ $elapsed -lt $timeout ]; do
        if eval "$command" >/dev/null 2>&1; then
            echo -e "${GREEN}✓${NC} (${elapsed}s)"
            ((PASS++))
            return 0
        fi
        sleep $interval
        elapsed=$((elapsed + interval))
        echo -n "."
    done
    
    echo -e "${YELLOW}⚠ Timeout after ${timeout}s${NC}"
    ((WARNINGS++))
    return 1
}

echo -e "${BLUE}=== KubeSkippy Demo Validation ===${NC}"
echo ""

# Pre-requisites
echo -e "${YELLOW}▶ Checking Prerequisites${NC}"
check "Kubernetes connection" "kubectl cluster-info --context kind-kubeskippy-demo"
check "Kind cluster exists" "kind get clusters | grep -q kubeskippy-demo"

# Namespaces
echo ""
echo -e "${YELLOW}▶ Checking Namespaces${NC}"
check "kubeskippy-system namespace" "kubectl get namespace kubeskippy-system"
check "demo-apps namespace" "kubectl get namespace demo-apps"
check "monitoring namespace" "kubectl get namespace monitoring"

# Infrastructure
echo ""
echo -e "${YELLOW}▶ Checking Infrastructure Components${NC}"
check "Metrics-server deployment" "kubectl get deployment metrics-server -n kube-system"
check "Kube-state-metrics deployment" "kubectl get deployment kube-state-metrics -n kube-system"
wait_for_condition "Metrics API availability" "kubectl top nodes" 120 10

# KubeSkippy Operator
echo ""
echo -e "${YELLOW}▶ Checking KubeSkippy Operator${NC}"
check "Operator deployment" "kubectl get deployment kubeskippy-controller-manager -n kubeskippy-system"
check "Operator running" "kubectl get pods -n kubeskippy-system -l control-plane=controller-manager -o jsonpath='{.items[0].status.phase}' | grep -q Running"
check "Metrics service" "kubectl get service kubeskippy-controller-manager-metrics -n kubeskippy-system"
check_count "Operator error logs" "kubectl logs -n kubeskippy-system deployment/kubeskippy-controller-manager --tail=100 2>/dev/null | grep -i error | grep -v 'TLS handshake error' | wc -l" "<=" 5

# AI Backend
echo ""
echo -e "${YELLOW}▶ Checking AI Backend (Ollama)${NC}"
check "Ollama deployment" "kubectl get deployment ollama -n kubeskippy-system"
check "Ollama service" "kubectl get service ollama -n kubeskippy-system"
wait_for_condition "Ollama pod ready" "kubectl get pods -n kubeskippy-system -l app=ollama -o jsonpath='{.items[0].status.conditions[?(@.type==\"Ready\")].status}' | grep -q True" 180 10

# Check if model is loaded (might still be loading)
if kubectl get job ollama-model-loader -n kubeskippy-system >/dev/null 2>&1; then
    job_status=$(kubectl get job ollama-model-loader -n kubeskippy-system -o jsonpath='{.status.conditions[?(@.type=="Complete")].status}' 2>/dev/null || echo "Unknown")
    if [ "$job_status" == "True" ]; then
        echo -e "Model loader job: ${GREEN}✓ Completed${NC}"
        ((PASS++))
    else
        echo -e "Model loader job: ${YELLOW}⚠ Still running (this is OK, will complete in background)${NC}"
        ((WARNINGS++))
    fi
fi

# Monitoring Stack
echo ""
echo -e "${YELLOW}▶ Checking Monitoring Stack${NC}"
check "Prometheus deployment" "kubectl get deployment prometheus -n monitoring"
check "Prometheus running" "kubectl get pods -n monitoring -l app=prometheus -o jsonpath='{.items[0].status.phase}' | grep -q Running"
check "Grafana deployment" "kubectl get deployment grafana -n monitoring"
check "Grafana running" "kubectl get pods -n monitoring -l app=grafana -o jsonpath='{.items[0].status.phase}' | grep -q Running"

# Demo Applications
echo ""
echo -e "${YELLOW}▶ Checking Demo Applications${NC}"
check_count "Demo app deployments" "kubectl get deployments -n demo-apps --no-headers | wc -l" ">=" 6
check "continuous-memory-degradation" "kubectl get deployment continuous-memory-degradation -n demo-apps"
check "continuous-cpu-oscillation" "kubectl get deployment continuous-cpu-oscillation -n demo-apps"
check "chaos-monkey-component" "kubectl get deployment chaos-monkey-component -n demo-apps"
check_count "Pods with demo label" "kubectl get pods -n demo-apps -l demo=kubeskippy --no-headers | wc -l" ">=" 4

# Healing Policies
echo ""
echo -e "${YELLOW}▶ Checking Healing Policies${NC}"
check_count "Total healing policies" "kubectl get healingpolicies -n demo-apps --no-headers | wc -l" ">=" 4
check_count "AI healing policies" "kubectl get healingpolicies -n demo-apps --no-headers | grep -c ai-" ">=" 3
check "AI strategic healing policy" "kubectl get healingpolicy ai-strategic-healing -n demo-apps"
check_with_output "AI policies in automatic mode" "kubectl get healingpolicy ai-strategic-healing -n demo-apps -o jsonpath='{.spec.mode}'" "automatic"

# Port Forwarding
echo ""
echo -e "${YELLOW}▶ Checking Port Forwarding${NC}"
if pgrep -f "kubectl port-forward.*grafana" >/dev/null && pgrep -f "kubectl port-forward.*prometheus" >/dev/null; then
    echo -e "Port forwards: ${GREEN}✓ Active${NC}"
    ((PASS++))
    
    # Test accessibility
    echo ""
    echo -e "${YELLOW}▶ Checking Service Accessibility${NC}"
    wait_for_condition "Grafana accessible" "curl -s -u admin:admin http://localhost:3000/api/health | grep -q ok" 30 5
    wait_for_condition "Prometheus accessible" "curl -s http://localhost:9090/-/ready | grep -q 'Prometheus is Ready'" 30 5
else
    echo -e "Port forwards: ${YELLOW}⚠ Not active - starting them${NC}"
    ((WARNINGS++))
    if [ -f "./start-port-forwards.sh" ]; then
        ./start-port-forwards.sh
        sleep 5
    fi
fi

# Healing Actions (give some time for actions to be created)
echo ""
echo -e "${YELLOW}▶ Checking Healing Actions (may need 2-3 minutes)${NC}"
current_actions=$(kubectl get healingactions -n demo-apps --no-headers 2>/dev/null | wc -l || echo 0)
if [ "$current_actions" -eq 0 ]; then
    echo -e "${YELLOW}No healing actions yet. Waiting for policies to trigger...${NC}"
    sleep 60
fi

check_count "Healing actions created" "kubectl get healingactions -n demo-apps --no-headers | wc -l" ">" 0
check_count "AI-driven healing actions" "kubectl get healingactions -n demo-apps | grep -c 'ai-' || echo 0" ">=" 0

# Metrics Validation
echo ""
echo -e "${YELLOW}▶ Checking Metrics Availability${NC}"
if [ "$current_actions" -gt 0 ]; then
    wait_for_condition "kubeskippy_healing_actions_total metric" "curl -s 'http://localhost:9090/api/v1/query?query=kubeskippy_healing_actions_total' | grep -q '\"result\":\\[{'" 30 5
fi
wait_for_condition "Container CPU metrics" "curl -s 'http://localhost:9090/api/v1/query?query=container_cpu_usage_seconds_total' | grep -q '\"result\":\\[{'" 30 5
wait_for_condition "Container memory metrics" "curl -s 'http://localhost:9090/api/v1/query?query=container_memory_usage_bytes' | grep -q '\"result\":\\[{'" 30 5

# Grafana Dashboard Check
echo ""
echo -e "${YELLOW}▶ Checking Grafana Dashboard${NC}"
dashboard_check=$(curl -s -u admin:admin "http://localhost:3000/api/search?query=KubeSkippy" 2>/dev/null | grep -c "KubeSkippy Enhanced" || echo 0)
if [ "$dashboard_check" -gt 0 ]; then
    echo -e "Enhanced dashboard: ${GREEN}✓ Found${NC}"
    ((PASS++))
else
    echo -e "Enhanced dashboard: ${RED}✗ Not found${NC}"
    ((FAIL++))
fi

# Summary
echo ""
echo -e "${BLUE}=== Validation Summary ===${NC}"
echo -e "Passed: ${GREEN}$PASS${NC}"
echo -e "Failed: ${RED}$FAIL${NC}"
echo -e "Warnings: ${YELLOW}$WARNINGS${NC}"

# Provide recommendations
if [ "$FAIL" -gt 0 ] || [ "$WARNINGS" -gt 0 ]; then
    echo ""
    echo -e "${YELLOW}▶ Recommendations:${NC}"
    
    if [ "$current_actions" -eq 0 ]; then
        echo "  • No healing actions yet - wait 2-3 minutes for policies to trigger"
        echo "  • Check operator logs: kubectl logs -n kubeskippy-system deployment/kubeskippy-controller-manager"
    fi
    
    if [ "$dashboard_check" -eq 0 ]; then
        echo "  • Dashboard may not be imported yet - check Grafana UI manually"
    fi
    
    echo "  • Run ./monitor-demo.sh to see real-time status"
    echo "  • Some warnings are expected during initial setup"
fi

# Exit code based on failures
if [ "$FAIL" -gt 0 ]; then
    exit 1
else
    exit 0
fi