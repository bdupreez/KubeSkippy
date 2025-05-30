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

# AI-Driven Healing Status
echo -e "\n${BLUE}🤖 AI-Driven Healing Status${NC}"
echo "───────────────────────────"

current_mode=$(kubectl get healingpolicy ai-driven-healing -n demo-apps -o jsonpath='{.spec.mode}' 2>/dev/null)
if [[ "$current_mode" == "automatic" ]]; then
    echo -e "AI-driven healing: ${GREEN}✓ ENABLED${NC} (automatic mode)"
    echo -e "\n${YELLOW}Recent AI-driven healing actions:${NC}"
    ai_actions=$(kubectl get healingactions -n demo-apps 2>/dev/null | grep -E "ai-driven|ai-" | tail -5)
    if [[ -z "$ai_actions" ]]; then
        echo "No AI actions yet (wait 1-2 minutes for AI analysis)"
    else
        echo "$ai_actions" | while read line; do
            name=$(echo $line | awk '{print $1}')
            phase=$(echo $line | awk '{print $4}')
            if [[ "$phase" == "Completed" ]]; then
                echo -e "• ${GREEN}✓${NC} $name"
            elif [[ "$phase" == "Failed" ]]; then
                echo -e "• ${RED}✗${NC} $name"
            else
                echo -e "• ${YELLOW}⟳${NC} $name ($phase)"
            fi
        done
    fi
else
    echo -e "AI-driven healing: ${YELLOW}○ DISABLED${NC} ($current_mode mode)"
    echo -e "\nTo enable AI-driven healing, run:"
    echo -e "  ${GREEN}kubectl patch healingpolicy ai-driven-healing -n demo-apps --type merge -p '{\"spec\":{\"mode\":\"automatic\"}}'${NC}"
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
if [[ "$current_mode" != "automatic" ]]; then
    echo "• Enable AI healing: kubectl patch healingpolicy ai-driven-healing -n demo-apps --type merge -p '{\"spec\":{\"mode\":\"automatic\"}}'"
fi
echo "• Clean up: ./cleanup.sh"