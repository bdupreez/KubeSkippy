#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}📊 KubeSkippy Dashboard Metrics Validation${NC}"
echo "============================================"
echo ""

# Test AI Confidence Level calculation
echo -e "${YELLOW}🧠 AI Confidence Level Calculation${NC}"
echo "─────────────────────────────────────"

ai_total=$(curl -s "http://localhost:9090/api/v1/query" -d 'query=sum(kubeskippy_healing_actions_total{trigger_type=~".*predictive.*|.*continuous.*"})' | jq -r '.data.result[0].value[1]' 2>/dev/null || echo "0")
total_actions=$(curl -s "http://localhost:9090/api/v1/query" -d 'query=sum(kubeskippy_healing_actions_total)' | jq -r '.data.result[0].value[1]' 2>/dev/null || echo "0")
confidence=$(curl -s "http://localhost:9090/api/v1/query" -d 'query=(sum(kubeskippy_healing_actions_total{trigger_type=~".*predictive.*|.*continuous.*"}) / sum(kubeskippy_healing_actions_total) * 3)' | jq -r '.data.result[0].value[1]' 2>/dev/null || echo "0")

echo -e "🎯 AI Actions: ${GREEN}${ai_total}${NC}"
echo -e "📊 Total Actions: ${BLUE}${total_actions}${NC}"
echo -e "🧠 AI Confidence Level: ${PURPLE}${confidence}${NC} (out of 3)"

if [ "${ai_total}" != "0" ] && [ "${ai_total}" != "null" ]; then
    percentage=$(echo "scale=1; ${ai_total} * 100 / ${total_actions}" | bc 2>/dev/null || echo "0")
    echo -e "📈 AI Percentage: ${GREEN}${percentage}%${NC}"
    echo -e "${GREEN}✅ AI Confidence panel should show data${NC}"
else
    echo -e "${RED}❌ No AI actions found - dashboard may show 'No Data'${NC}"
fi

# Test other key metrics
echo -e "\n${YELLOW}📈 Other Dashboard Metrics${NC}"
echo "─────────────────────────────"

# Test AI vs Rule-based comparison
ai_rate=$(curl -s "http://localhost:9090/api/v1/query" -d 'query=sum(rate(kubeskippy_healing_actions_total{trigger_type=~".*predictive.*|.*continuous.*"}[2m]))' | jq -r '.data.result[0].value[1]' 2>/dev/null || echo "0")
rule_rate=$(curl -s "http://localhost:9090/api/v1/query" -d 'query=sum(rate(kubeskippy_healing_actions_total{trigger_type!~".*predictive.*|.*continuous.*"}[2m]))' | jq -r '.data.result[0].value[1]' 2>/dev/null || echo "0")

echo -e "⚡ AI Activity Rate: ${GREEN}${ai_rate}${NC} actions/min"
echo -e "📏 Rule-based Activity Rate: ${YELLOW}${rule_rate}${NC} actions/min"

# Test trigger type breakdown
echo -e "\n${YELLOW}🔍 Trigger Type Breakdown${NC}"
echo "────────────────────────────"

curl -s "http://localhost:9090/api/v1/query" -d 'query=kubeskippy_healing_actions_total' | jq -r '.data.result[] | .metric.trigger_type' | sort | uniq -c | while read count type; do
    if [[ "$type" == *"predictive"* ]]; then
        icon="🔮"
        color="${PURPLE}"
    elif [[ "$type" == *"continuous"* ]]; then
        icon="🔄"
        color="${CYAN}"
    else
        icon="📏"
        color="${YELLOW}"
    fi
    printf "${icon} ${color}%-30s${NC}: %s actions\n" "$type" "$count"
done

echo -e "\n${GREEN}🎬 Dashboard Access${NC}"
echo "─────────────────────"
echo -e "🎯 Enhanced Dashboard: ${GREEN}http://localhost:3000/d/kubeskippy-enhanced${NC}"
echo -e "📊 Login: admin/admin"
echo ""
echo -e "${BLUE}💡 Key Panels to Check:${NC}"
echo "• AI Confidence Level (should show ~${confidence})"
echo "• AI vs Rule-based Effectiveness"
echo "• Healing Action Distribution"
echo "• Predictive AI & Continuous Healing section"

if [ "${ai_total}" -gt 10 ] && [ "${confidence}" != "0" ]; then
    echo -e "\n${GREEN}🚀 Dashboard metrics are working correctly!${NC}"
    echo -e "The AI Confidence Level panel should now display data."
else
    echo -e "\n${YELLOW}⚠️  Dashboard may need a few more minutes to populate data.${NC}"
    echo -e "Try refreshing the dashboard or wait for more healing activity."
fi