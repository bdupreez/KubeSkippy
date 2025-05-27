#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 KubeSkippy Quick Demo${NC}"
echo "========================"
echo ""

# Check if demo is running
if ! kubectl get ns demo-apps &>/dev/null; then
    echo -e "${RED}❌ Demo not running. Please run ./setup.sh first${NC}"
    exit 1
fi

echo -e "${YELLOW}📊 Current Status${NC}"
echo "─────────────────"

# Show problematic pods
echo -e "\n${YELLOW}Problematic Pods:${NC}"
kubectl get pods -n demo-apps | grep -E "(CrashLoop|Error|0/1)" || echo "All pods healthy"

# Show resource usage
echo -e "\n${YELLOW}Resource Usage:${NC}"
kubectl top pods -n demo-apps 2>/dev/null | awk 'NR==1 || ($2 ~ /[0-9]+m/ && $2+0 > 100) || ($3 ~ /[0-9]+Mi/ && $3+0 > 300)' || echo "Metrics not available yet"

# Show healing policies
echo -e "\n${YELLOW}Active Healing Policies:${NC}"
kubectl get healingpolicies -n demo-apps --no-headers | while read line; do
    name=$(echo $line | awk '{print $1}')
    mode=$(echo $line | awk '{print $2}')
    actions=$(echo $line | awk '{print $3}')
    
    if [[ "$mode" == "automatic" ]]; then
        echo -e "${GREEN}✓${NC} $name - Mode: $mode, Actions: ${actions:-0}"
    else
        echo -e "${YELLOW}○${NC} $name - Mode: $mode, Actions: ${actions:-0}"
    fi
done

# Show recent healing actions
echo -e "\n${YELLOW}Recent Healing Actions:${NC}"
recent_actions=$(kubectl get healingactions -n demo-apps --no-headers 2>/dev/null | tail -5)
if [[ -z "$recent_actions" ]]; then
    echo "No healing actions yet (wait 1-2 minutes)"
else
    echo "$recent_actions" | while read line; do
        name=$(echo $line | awk '{print $1}')
        target=$(echo $line | awk '{print $2}')
        echo "• $name → $target"
    done
fi

# AI-Driven Healing Demo
echo -e "\n${BLUE}🤖 AI-Driven Healing Demo${NC}"
echo "──────────────────────────"

current_mode=$(kubectl get healingpolicy ai-driven-healing -n demo-apps -o jsonpath='{.spec.mode}' 2>/dev/null)
echo -e "Current mode: ${YELLOW}$current_mode${NC}"

if [[ "$1" == "--enable-ai" ]]; then
    echo -e "\n${YELLOW}Enabling AI-driven healing...${NC}"
    kubectl patch healingpolicy ai-driven-healing -n demo-apps \
        --type merge -p '{"spec":{"mode":"automatic"}}' >/dev/null
    
    echo "Waiting for AI analysis (30 seconds)..."
    sleep 30
    
    echo -e "\n${GREEN}AI-driven healing actions:${NC}"
    kubectl get healingactions -n demo-apps | grep ai-driven || echo "No AI actions yet"
    
    echo -e "\n${YELLOW}Reverting to dryrun mode...${NC}"
    kubectl patch healingpolicy ai-driven-healing -n demo-apps \
        --type merge -p '{"spec":{"mode":"dryrun"}}' >/dev/null
    echo -e "${GREEN}✓ AI-driven healing demo complete${NC}"
else
    echo -e "\nTo see AI-driven healing in action, run:"
    echo -e "  ${GREEN}./quick-demo.sh --enable-ai${NC}"
fi

# Summary
echo -e "\n${BLUE}📈 Summary${NC}"
echo "──────────"
total_actions=$(kubectl get healingactions -n demo-apps --no-headers 2>/dev/null | wc -l)
echo "• Total healing actions created: $total_actions"
echo "• Policies with actions: $(kubectl get healingpolicies -n demo-apps --no-headers | awk '$3 > 0' | wc -l)/5"

echo -e "\n${YELLOW}Next Steps:${NC}"
echo "• Watch live updates: ./monitor.sh"
echo "• Check detailed status: ./check-demo.sh"
echo "• Enable AI healing: ./quick-demo.sh --enable-ai"
echo "• Clean up: ./cleanup.sh"