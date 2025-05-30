#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Deploying Continuous AI Healing Demo${NC}"
echo "=========================================="
echo ""

# Check if demo namespace exists
if ! kubectl get ns demo-apps &>/dev/null; then
    echo -e "${RED}❌ Demo namespace not found. Please run ./setup.sh first${NC}"
    exit 1
fi

echo -e "${YELLOW}📦 Deploying Continuous Failure Applications${NC}"
echo "──────────────────────────────────────────────"

# Deploy continuous failure applications
echo "Deploying continuous memory degradation app..."
kubectl apply -f apps/continuous-memory-degradation-app.yaml

echo "Deploying continuous CPU oscillation app..."
kubectl apply -f apps/continuous-cpu-oscillation-app.yaml

echo "Deploying continuous network degradation app..."
kubectl apply -f apps/continuous-network-degradation-app.yaml

echo "Deploying chaos monkey component..."
kubectl apply -f apps/chaos-monkey-component.yaml

echo -e "${GREEN}✅ Continuous failure applications deployed!${NC}"

echo -e "\n${YELLOW}🧠 Deploying Predictive AI Healing Policies${NC}"
echo "────────────────────────────────────────────────"

# Deploy predictive AI policies
echo "Deploying predictive AI healing policy..."
kubectl apply -f policies/predictive-ai-healing.yaml

echo "Updating enhanced AI-driven healing policy..."
kubectl apply -f policies/ai-driven-healing.yaml

echo -e "${GREEN}✅ Predictive AI policies deployed!${NC}"

echo -e "\n${PURPLE}⏳ Waiting for applications to start...${NC}"
echo "───────────────────────────────────────────"

# Wait for deployments to be ready
kubectl wait --for=condition=available --timeout=120s deployment/continuous-memory-degradation-app -n demo-apps
kubectl wait --for=condition=available --timeout=120s deployment/continuous-cpu-oscillation-app -n demo-apps
kubectl wait --for=condition=available --timeout=120s deployment/continuous-network-degradation-app -n demo-apps
kubectl wait --for=condition=available --timeout=120s deployment/chaos-monkey-component -n demo-apps

echo -e "${GREEN}✅ All applications are running!${NC}"

echo -e "\n${BLUE}📊 Current Demo Status${NC}"
echo "─────────────────────"

# Show current status
echo -e "\n${YELLOW}Applications:${NC}"
kubectl get deployments -n demo-apps -o custom-columns="NAME:.metadata.name,READY:.status.readyReplicas,TYPE:.metadata.labels.failure-type"

echo -e "\n${YELLOW}Healing Policies:${NC}"
kubectl get healingpolicies -n demo-apps -o custom-columns="NAME:.metadata.name,MODE:.spec.mode,ACTIONS:.status.totalActions"

echo -e "\n${PURPLE}🎯 Continuous AI Demo Features${NC}"
echo "──────────────────────────────────"
echo ""
echo -e "🧠 ${BLUE}Predictive AI Capabilities:${NC}"
echo "• Memory degradation trend analysis"
echo "• CPU oscillation pattern detection"
echo "• Network degradation early warning"
echo "• Multi-metric correlation analysis"
echo "• Predictive failure prevention"
echo ""
echo -e "🔄 ${BLUE}Continuous Failure Scenarios:${NC}"
echo "• Gradual memory degradation (3-minute cycles)"
echo "• CPU oscillation patterns (4-minute waves)"
echo "• Network degradation simulation (4-minute cycles)"
echo "• Random chaos injection (30-120 second intervals)"
echo ""
echo -e "⚡ ${BLUE}Enhanced Healing Features:${NC}"
echo "• Early intervention (70% degradation threshold)"
echo "• Trend-based prediction (5-minute horizon)"
echo "• Confidence-scored actions (70% threshold)"
echo "• Coordinated system-wide healing"
echo "• Adaptive throttling for continuous scenarios"

echo -e "\n${GREEN}🎬 Monitor the Continuous AI Demo:${NC}"
echo "──────────────────────────────────────"
echo ""
echo -e "📊 ${GREEN}Watch Live AI Activity:${NC}"
echo "   ./monitor.sh"
echo ""
echo -e "🧠 ${GREEN}Enhanced Grafana Dashboard:${NC}"
echo "   http://localhost:3000/d/kubeskippy-enhanced"
echo ""
echo -e "🔍 ${GREEN}Monitor Specific Components:${NC}"
echo "   kubectl logs -f deployment/continuous-memory-degradation-app -n demo-apps -c memory-degradation-app"
echo "   kubectl logs -f deployment/chaos-monkey-component -n demo-apps"
echo "   kubectl get healingactions -n demo-apps -w"
echo ""
echo -e "🎯 ${GREEN}Check Predictive Actions:${NC}"
echo "   kubectl get healingactions -n demo-apps | grep predictive"
echo "   kubectl get healingpolicies -n demo-apps -o yaml | grep -A5 -B5 prediction"

echo -e "\n${BLUE}💡 Demo Highlights to Watch:${NC}"
echo "─────────────────────────────"
echo ""
echo "1. 🧠 AI predicts memory failures before they occur"
echo "2. 🔄 Continuous healing prevents cascade failures"
echo "3. 📈 Trend analysis detects degradation patterns"
echo "4. ⚡ Early intervention at 70% vs 90% traditional thresholds"
echo "5. 🎯 Multi-metric correlation improves prediction accuracy"
echo "6. 🤖 Chaos engineering creates realistic failure scenarios"

echo -e "\n${PURPLE}🚀 Continuous AI Demo is Ready!${NC}"
echo ""
echo "The demo now features predictive AI that intervenes before complete failure,"
echo "continuous healing scenarios, and chaos engineering for realistic testing."
echo ""
echo -e "Press ${YELLOW}Ctrl+C${NC} to stop monitoring when ready."

# Start live monitoring
echo -e "\n${PURPLE}Live Continuous AI Monitor:${NC}"
echo "───────────────────────────────"

# Simple monitoring loop
while true; do
    # Count different types of actions
    total_actions=$(kubectl get healingactions -n demo-apps --no-headers 2>/dev/null | wc -l)
    ai_actions=$(kubectl get healingactions -n demo-apps --no-headers 2>/dev/null | grep -i ai | wc -l)
    predictive_actions=$(kubectl get healingactions -n demo-apps --no-headers 2>/dev/null | grep predictive | wc -l)
    
    # Count running applications
    memory_pods=$(kubectl get pods -n demo-apps -l app=continuous-memory-degradation-app --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
    cpu_pods=$(kubectl get pods -n demo-apps -l app=continuous-cpu-oscillation-app --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
    network_pods=$(kubectl get pods -n demo-apps -l app=continuous-network-degradation-app --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
    chaos_pods=$(kubectl get pods -n demo-apps -l app=chaos-monkey-component --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
    
    printf "\r${GREEN}🎯 Total: %d${NC} | ${BLUE}🧠 AI: %d${NC} | ${PURPLE}🔮 Predictive: %d${NC} | ${YELLOW}🧪 Apps: %d/%d/%d/%d${NC}" \
           "$total_actions" "$ai_actions" "$predictive_actions" "$memory_pods" "$cpu_pods" "$network_pods" "$chaos_pods"
    
    sleep 5
done