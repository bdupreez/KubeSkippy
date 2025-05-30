#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

echo -e "${BLUE}🧠 KubeSkippy AI Intelligence Showcase${NC}"
echo "===================================="
echo ""

# Check if demo is running
if ! kubectl get ns demo-apps &>/dev/null; then
    echo -e "${RED}❌ Demo not running. Please run ./setup.sh --with-monitoring first${NC}"
    exit 1
fi

echo -e "${YELLOW}📊 Current Healing Status${NC}"
echo "─────────────────────────"

# Show current healing actions
echo -e "\n${YELLOW}Active Healing Actions:${NC}"
kubectl get healingactions -n demo-apps --no-headers | head -5 | while read line; do
    name=$(echo $line | awk '{print $1}')
    target=$(echo $line | awk '{print $2}')
    phase=$(echo $line | awk '{print $3}')
    
    if [[ "$name" == *"ai"* ]]; then
        echo -e "🤖 ${GREEN}$name${NC} → $target ($phase)"
    else
        echo -e "📏 ${YELLOW}$name${NC} → $target ($phase)"
    fi
done

echo -e "\n${PURPLE}Phase 1: Deploy Complex Pattern Failure App${NC}"
echo "─────────────────────────────────────────────"

# Deploy the pattern failure app
echo "Deploying pattern-failure-app to showcase AI pattern recognition..."
kubectl apply -f apps/pattern-failure-app.yaml

echo "Deploying enhanced AI policy..."
kubectl apply -f policies/ai-intelligent-healing-simple.yaml

echo -e "${GREEN}✅ Complex scenario deployed!${NC}"

echo -e "\n${PURPLE}Phase 2: Monitor AI vs Rule-based Healing${NC}"
echo "──────────────────────────────────────────────"

echo "The demo now includes:"
echo "• 🤖 AI-driven pattern recognition"
echo "• 📊 AI vs Rule-based effectiveness comparison"
echo "• 🧠 AI confidence level tracking"
echo "• 🎯 Strategic vs reactive healing"

echo -e "\n${YELLOW}Dashboard Features:${NC}"
echo "• AI Confidence Level gauge"
echo "• AI vs Rule-based effectiveness comparison"
echo "• AI Action Type Distribution"
echo "• Pattern Recognition Results"

echo -e "\n${BLUE}📈 Access Enhanced Dashboard:${NC}"
echo "─────────────────────────────"

# Check if port-forward is running
if ! ps aux | grep -q "[p]ort-forward.*grafana"; then
    echo "Starting Grafana port-forward..."
    kubectl port-forward -n monitoring svc/grafana 3000:3000 > /dev/null 2>&1 &
    sleep 3
fi

echo -e "🎯 ${GREEN}Enhanced AI Dashboard:${NC} http://localhost:3000/d/kubeskippy-enhanced"
echo -e "🔍 ${GREEN}Original Dashboard:${NC} http://localhost:3000/d/kubeskippy-demo"
echo ""
echo "Login: admin/admin"

echo -e "\n${PURPLE}Phase 3: Real-time AI Demonstration${NC}"
echo "───────────────────────────────────"

echo "Watch for these AI capabilities:"
echo ""
echo -e "🧠 ${BLUE}Pattern Recognition:${NC}"
echo "   • Complex failure patterns that rules miss"
echo "   • Multi-dimensional correlations"
echo "   • Time-based failure prediction"
echo ""
echo -e "🎯 ${BLUE}Intelligent Decision Making:${NC}"
echo "   • Confidence-based action selection"
echo "   • Alternative strategy consideration"
echo "   • Resource optimization recommendations"
echo ""
echo -e "⚡ ${BLUE}Proactive vs Reactive:${NC}"
echo "   • AI detects problems before full failure"
echo "   • Strategic deletions vs mass restarts"
echo "   • Preventive scaling based on patterns"

echo -e "\n${YELLOW}🔬 Monitoring Commands:${NC}"
echo "─────────────────────"
echo "• Watch AI actions:     ${GREEN}kubectl get healingactions -n demo-apps -w | grep ai${NC}"
echo "• Monitor pattern app:  ${GREEN}kubectl logs -f deployment/pattern-failure-app -n demo-apps${NC}"
echo "• Check AI confidence:  ${GREEN}./check-grafana.sh${NC}"
echo "• Full monitoring:      ${GREEN}./monitor.sh${NC}"

echo -e "\n${BLUE}💡 Demo Script:${NC}"
echo "───────────────"
echo "1. Open the Enhanced Dashboard in your browser"
echo "2. Watch the 'AI Intelligence Dashboard' section"
echo "3. Observe AI Confidence Level changes"
echo "4. Compare AI vs Rule-based healing rates"
echo "5. Note the strategic action distribution"

echo -e "\n${GREEN}🚀 AI Showcase is ready!${NC}"
echo ""
echo "The pattern-failure-app will trigger complex failure scenarios"
echo "that showcase AI's superior pattern recognition and decision-making."
echo ""
echo -e "Press ${YELLOW}Ctrl+C${NC} to exit this script and start monitoring!"

# Live monitoring loop
echo -e "\n${PURPLE}Live AI Activity Monitor:${NC}"
echo "─────────────────────────"

while true; do
    # Show current AI actions
    ai_actions=$(kubectl get healingactions -n demo-apps --no-headers 2>/dev/null | grep -i ai | wc -l)
    rule_actions=$(kubectl get healingactions -n demo-apps --no-headers 2>/dev/null | grep -v -i ai | wc -l)
    
    # Show pattern app status
    pattern_pods=$(kubectl get pods -n demo-apps -l app=pattern-failure-app --no-headers 2>/dev/null | wc -l)
    pattern_restarts=$(kubectl get pods -n demo-apps -l app=pattern-failure-app -o jsonpath='{.items[*].status.containerStatuses[*].restartCount}' 2>/dev/null | awk '{sum += $1} END {print sum+0}')
    
    printf "\r${GREEN}🤖 AI Actions: %d${NC} | ${YELLOW}📏 Rule Actions: %d${NC} | ${BLUE}🔄 Pattern Restarts: %d${NC} | ${PURPLE}📱 Pattern Pods: %d${NC}" \
           "$ai_actions" "$rule_actions" "$pattern_restarts" "$pattern_pods"
    
    sleep 5
done