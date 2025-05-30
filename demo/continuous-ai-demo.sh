#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 KubeSkippy Continuous AI & Predictive Healing Demo${NC}"
echo "=========================================================="
echo ""

# Check if demo is running
if ! kubectl get ns demo-apps &>/dev/null; then
    echo -e "${RED}❌ Demo not running. Please run:${NC}"
    echo "   ./setup.sh --with-monitoring"
    echo "   ./deploy-continuous-ai.sh"
    exit 1
fi

echo -e "${CYAN}✨ Enhanced AI Features Now Active:${NC}"
echo "─────────────────────────────────────"
echo ""

echo -e "${PURPLE}🔮 Predictive AI Capabilities:${NC}"
echo "• Early intervention at 60-70% thresholds (vs traditional 80-90%)"
echo "• Trend-based failure prediction with 5-minute horizon"
echo "• Multi-metric correlation analysis for better accuracy"
echo "• Proactive scaling before resource exhaustion"
echo "• Cascade failure prevention with early warning system"
echo ""

echo -e "${YELLOW}🔄 Continuous Failure Scenarios:${NC}"
echo "• Memory degradation cycles (3-minute cycles, 10-second steps)"
echo "• CPU oscillation patterns (3-minute waves, predictable spikes)"
echo "• Network degradation simulation (4-minute cycles)"
echo "• Chaos engineering (random failures every 30-120 seconds)"
echo ""

echo -e "${GREEN}🧠 AI vs Traditional Healing:${NC}"
echo "• AI intervenes at 70% degradation vs 90% traditional"
echo "• Predictive actions prevent failures vs reactive repairs"
echo "• Multi-dimensional pattern recognition vs single-metric rules"
echo "• Confidence-based decision making vs rigid thresholds"
echo ""

echo -e "${BLUE}📊 Current System Status:${NC}"
echo "─────────────────────────"

# Show current applications
echo -e "\n${YELLOW}Continuous Failure Applications:${NC}"
kubectl get deployments -n demo-apps | grep continuous | while read line; do
    name=$(echo $line | awk '{print $1}')
    ready=$(echo $line | awk '{print $2}')
    
    if [[ "$name" == *"memory"* ]]; then
        echo -e "💾 ${PURPLE}$name${NC}: $ready pods (memory degradation cycles)"
    elif [[ "$name" == *"cpu"* ]]; then
        echo -e "🔥 ${RED}$name${NC}: $ready pods (CPU oscillation patterns)"
    elif [[ "$name" == *"network"* ]]; then
        echo -e "🌐 ${BLUE}$name${NC}: $ready pods (network degradation)"
    fi
done

# Show chaos monkey
chaos_pods=$(kubectl get pods -n demo-apps -l app=chaos-monkey-component --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
echo -e "🐒 ${CYAN}chaos-monkey-component${NC}: $chaos_pods pod (random failure injection)"

# Show healing policies
echo -e "\n${YELLOW}AI Healing Policies:${NC}"
kubectl get healingpolicies -n demo-apps | grep -E "(ai|predictive|continuous)" | while read line; do
    name=$(echo $line | awk '{print $1}')
    mode=$(echo $line | awk '{print $2}')
    
    if [[ "$name" == *"predictive"* ]]; then
        echo -e "🔮 ${PURPLE}$name${NC}: $mode (early intervention)"
    elif [[ "$name" == *"continuous"* ]]; then
        echo -e "🔄 ${CYAN}$name${NC}: $mode (continuous monitoring)"
    elif [[ "$name" == *"ai"* ]]; then
        echo -e "🧠 ${GREEN}$name${NC}: $mode (AI-driven)"
    fi
done

# Show activity summary
echo -e "\n${YELLOW}Healing Activity Summary:${NC}"
total_actions=$(kubectl get healingactions -n demo-apps --no-headers 2>/dev/null | wc -l)
ai_actions=$(kubectl get healingactions -n demo-apps --no-headers 2>/dev/null | grep -i ai | wc -l)
recent_actions=$(kubectl get healingactions -n demo-apps --no-headers 2>/dev/null | head -5 | wc -l)

echo -e "📊 Total healing actions: ${GREEN}$total_actions${NC}"
echo -e "🧠 AI-driven actions: ${BLUE}$ai_actions${NC}"
echo -e "⚡ Recent actions: ${YELLOW}$recent_actions${NC}"

echo -e "\n${PURPLE}🎯 Demo Highlights to Observe:${NC}"
echo "─────────────────────────────────"
echo ""
echo "1. 🧠 Watch AI predict failures before they happen (70% vs 90% thresholds)"
echo "2. 🔄 Observe continuous healing preventing cascade failures"
echo "3. 📈 See trend analysis detecting degradation patterns early"
echo "4. ⚡ Notice faster intervention times with predictive AI"
echo "5. 🎯 Compare AI success rates vs traditional rule-based healing"
echo "6. 🤖 Experience realistic failure scenarios from chaos engineering"

echo -e "\n${GREEN}🎬 Monitoring Options:${NC}"
echo "─────────────────────"
echo ""
echo -e "📊 ${GREEN}Enhanced Grafana Dashboard:${NC}"
echo "   http://localhost:3000/d/kubeskippy-enhanced"
echo "   • 🧠 AI Intelligence Dashboard section"
echo "   • 🔮 Predictive AI & Continuous Healing section"
echo ""
echo -e "⚡ ${GREEN}Live Terminal Monitoring:${NC}"
echo "   ./monitor.sh"
echo ""
echo -e "🔍 ${GREEN}Specific Component Logs:${NC}"
echo "   kubectl logs -f deployment/continuous-memory-degradation-app -n demo-apps"
echo "   kubectl logs -f deployment/chaos-monkey-component -n demo-apps"
echo ""
echo -e "🎯 ${GREEN}Healing Action Tracking:${NC}"
echo "   kubectl get healingactions -n demo-apps -w"
echo "   kubectl get healingactions -n demo-apps | grep predictive"

echo -e "\n${BLUE}💡 Key Innovation: Predictive vs Reactive Healing${NC}"
echo "──────────────────────────────────────────────────"
echo ""
echo -e "${YELLOW}Traditional Reactive Healing:${NC}"
echo "• Waits for 80-90% resource utilization before acting"
echo "• Responds after failures occur"
echo "• Single-metric based decisions"
echo "• Risk of cascade failures"
echo ""
echo -e "${GREEN}AI Predictive Healing:${NC}"
echo "• Intervenes at 60-70% utilization (early warning)"
echo "• Predicts failures before they happen"
echo "• Multi-metric correlation analysis"
echo "• Prevents cascade failures proactively"
echo ""

echo -e "${CYAN}🚀 Enhanced Demo is Ready for Presentation!${NC}"
echo ""
echo "This demo now showcases:"
echo "• Clear AI superiority over traditional rule-based healing"
echo "• Predictive failure prevention vs reactive repair"
echo "• Continuous healing scenarios with realistic failure patterns"
echo "• Professional dashboard visualization of AI intelligence"
echo ""
echo -e "The demo answers: ${YELLOW}\"Why is AI-powered healing better?\"${NC} with"
echo "quantifiable evidence and real-time demonstration."
echo ""
echo -e "Press ${YELLOW}Enter${NC} to start live monitoring or ${YELLOW}Ctrl+C${NC} to exit..."
read -r

echo -e "\n${PURPLE}🔬 Live Continuous AI Monitor${NC}"
echo "──────────────────────────────"
echo ""
echo "Watch for predictive interventions, trend analysis, and early warnings..."
echo ""

# Live monitoring loop
while true; do
    # Get action counts
    total=$(kubectl get healingactions -n demo-apps --no-headers 2>/dev/null | wc -l)
    ai=$(kubectl get healingactions -n demo-apps --no-headers 2>/dev/null | grep -i ai | wc -l)
    predictive=$(kubectl get healingactions -n demo-apps --no-headers 2>/dev/null | grep predictive | wc -l)
    
    # Get app status
    memory_pods=$(kubectl get pods -n demo-apps -l app=continuous-memory-degradation-app --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
    cpu_pods=$(kubectl get pods -n demo-apps -l app=continuous-cpu-oscillation-app --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
    network_pods=$(kubectl get pods -n demo-apps -l app=continuous-network-degradation-app --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
    chaos_pods=$(kubectl get pods -n demo-apps -l app=chaos-monkey-component --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
    
    # Get policy status
    policies=$(kubectl get healingpolicies -n demo-apps --no-headers 2>/dev/null | grep -E "(ai|predictive|continuous)" | wc -l)
    
    # Show real-time status
    timestamp=$(date "+%H:%M:%S")
    printf "\r${CYAN}[$timestamp]${NC} ${GREEN}🎯 Total: %d${NC} | ${BLUE}🧠 AI: %d${NC} | ${PURPLE}🔮 Predictive: %d${NC} | ${YELLOW}📊 Policies: %d${NC} | ${CYAN}🧪 Apps: %d/%d/%d/%d${NC}" \
           "$total" "$ai" "$predictive" "$policies" "$memory_pods" "$cpu_pods" "$network_pods" "$chaos_pods"
    
    sleep 3
done