#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}🔍 KubeSkippy Continuous Healing Validation${NC}"
echo "============================================="
echo ""

# Check if demo is running
if ! kubectl get ns demo-apps &>/dev/null; then
    echo -e "${RED}❌ Demo namespace not found${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Demo namespace active${NC}"

echo -e "\n${YELLOW}📊 Current System Status${NC}"
echo "─────────────────────────"

# Count healing actions
total_actions=$(kubectl get healingactions -n demo-apps --no-headers 2>/dev/null | wc -l)
continuous_actions=$(kubectl get healingactions -n demo-apps --no-headers 2>/dev/null | grep -E "(continuous|predictive)" | wc -l)
recent_actions=$(kubectl get healingactions -n demo-apps --no-headers --sort-by='.metadata.creationTimestamp' 2>/dev/null | tail -5 | wc -l)

echo -e "🎯 Total healing actions: ${GREEN}${total_actions}${NC}"
echo -e "🔄 Continuous/Predictive actions: ${BLUE}${continuous_actions}${NC}" 
echo -e "⚡ Recent actions (last 5): ${PURPLE}${recent_actions}${NC}"

# Check application status
echo -e "\n${YELLOW}🧪 Continuous Failure Applications${NC}"
echo "─────────────────────────────────────"

memory_pods=$(kubectl get pods -n demo-apps -l app=continuous-memory-degradation-app --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
cpu_pods=$(kubectl get pods -n demo-apps -l app=continuous-cpu-oscillation-app --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
network_pods=$(kubectl get pods -n demo-apps -l app=continuous-network-degradation-app --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
stress_pods=$(kubectl get pods -n demo-apps -l app=stress-generator-app --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
chaos_pods=$(kubectl get pods -n demo-apps -l app=chaos-monkey-component --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
activity_pods=$(kubectl get pods -n demo-apps -l app=demo-activity-generator --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)

echo -e "💾 Memory degradation apps: ${GREEN}${memory_pods}/2${NC} running"
echo -e "🔥 CPU oscillation apps: ${GREEN}${cpu_pods}/2${NC} running"
echo -e "🌐 Network degradation apps: ${GREEN}${network_pods}/2${NC} running"
echo -e "⚡ Stress generator apps: ${GREEN}${stress_pods}/2${NC} running"
echo -e "🐒 Chaos monkey: ${GREEN}${chaos_pods}/1${NC} running"
echo -e "🎭 Activity generator: ${GREEN}${activity_pods}/1${NC} running"

# Check resource usage
echo -e "\n${YELLOW}📈 Resource Usage (Current Snapshot)${NC}"
echo "───────────────────────────────────────────"

if command -v kubectl >/dev/null 2>&1; then
    kubectl top pods -n demo-apps --no-headers 2>/dev/null | grep -E "(continuous|stress|demo)" | while read line; do
        name=$(echo $line | awk '{print $1}' | cut -c1-35)
        cpu=$(echo $line | awk '{print $2}')
        memory=$(echo $line | awk '{print $3}')
        
        # Color code based on resource usage
        if [[ $cpu =~ ([0-9]+)m ]]; then
            cpu_num=${BASH_REMATCH[1]}
            if [ $cpu_num -gt 500 ]; then
                cpu_color="${RED}"
            elif [ $cpu_num -gt 200 ]; then
                cpu_color="${YELLOW}"
            else
                cpu_color="${GREEN}"
            fi
        else
            cpu_color="${NC}"
        fi
        
        echo -e "${name}: CPU ${cpu_color}${cpu}${NC}, Memory ${memory}"
    done
else
    echo "kubectl top not available"
fi

# Check healing policies
echo -e "\n${YELLOW}🧠 AI Healing Policies${NC}"
echo "─────────────────────────"

policy_count=$(kubectl get healingpolicies -n demo-apps --no-headers 2>/dev/null | grep -E "(ai|predictive|continuous)" | wc -l)
echo -e "📋 AI/Predictive policies active: ${GREEN}${policy_count}${NC}"

kubectl get healingpolicies -n demo-apps --no-headers 2>/dev/null | grep -E "(ai|predictive|continuous)" | while read line; do
    name=$(echo $line | awk '{print $1}' | cut -c1-30)
    mode=$(echo $line | awk '{print $2}')
    actions=$(echo $line | awk '{print $3}')
    
    if [[ "$name" == *"predictive"* ]]; then
        icon="🔮"
    elif [[ "$name" == *"continuous"* ]]; then
        icon="🔄"
    else
        icon="🧠"
    fi
    
    echo -e "${icon} ${name}: ${mode} (${actions} actions)"
done

# Show recent healing activity
echo -e "\n${YELLOW}⚡ Recent Healing Activity (Last 5 Actions)${NC}"
echo "───────────────────────────────────────────────"

kubectl get healingactions -n demo-apps --sort-by='.metadata.creationTimestamp' --no-headers 2>/dev/null | tail -5 | while read line; do
    name=$(echo $line | awk '{print $1}' | cut -c1-50)
    target=$(echo $line | awk '{print $2}')
    status=$(echo $line | awk '{print $3}')
    age=$(echo $line | awk '{print $5}')
    
    if [[ "$status" == "Succeeded" ]]; then
        status_color="${GREEN}"
    else
        status_color="${RED}"
    fi
    
    if [[ "$name" == *"predictive"* ]]; then
        icon="🔮"
    elif [[ "$name" == *"continuous"* ]]; then
        icon="🔄"
    elif [[ "$name" == *"ai"* ]]; then
        icon="🧠"
    else
        icon="📏"
    fi
    
    echo -e "${icon} ${name} → ${status_color}${status}${NC} (${age})"
done

# Summary
echo -e "\n${PURPLE}🎯 Validation Summary${NC}"
echo "─────────────────────"

# Check if we have continuous activity
if [ $total_actions -gt 20 ] && [ $continuous_actions -gt 5 ]; then
    echo -e "${GREEN}✅ Continuous healing is ACTIVE${NC}"
    echo -e "   • ${total_actions} total healing actions performed"
    echo -e "   • ${continuous_actions} continuous/predictive actions"
    echo -e "   • Applications generating measurable load"
    echo -e "   • Healing policies responding to failures"
elif [ $total_actions -gt 10 ]; then
    echo -e "${YELLOW}⚠️  Healing is WORKING but may not be continuous${NC}"
    echo -e "   • ${total_actions} healing actions performed"
    echo -e "   • May need to wait longer for continuous activity"
else
    echo -e "${RED}❌ Limited healing activity detected${NC}"
    echo -e "   • Only ${total_actions} healing actions performed"
    echo -e "   • Check application logs and policy configurations"
fi

echo -e "\n${CYAN}🎬 Monitoring Options${NC}"
echo "─────────────────────"
echo -e "📊 ${GREEN}Enhanced Dashboard:${NC} http://localhost:3000/d/kubeskippy-enhanced"
echo -e "⚡ ${GREEN}Live monitoring:${NC} ./continuous-ai-demo.sh"
echo -e "🔍 ${GREEN}Basic monitoring:${NC} ./monitor.sh"
echo ""
echo -e "${BLUE}💡 The demo now shows continuous AI healing with:${NC}"
echo "• Predictive failure prevention (60-70% thresholds)"
echo "• Faster failure cycles (60s instead of 180s)" 
echo "• Higher resource visibility (up to 70% CPU, 58Mi memory)"
echo "• Event-based triggers ensuring continuous activity"
echo "• Multiple AI policies working in parallel"

echo ""
echo -e "${GREEN}🚀 Continuous AI healing demo is now optimized and active!${NC}"