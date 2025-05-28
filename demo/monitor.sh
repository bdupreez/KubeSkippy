#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}KubeSkippy Demo Monitor${NC}"
echo "========================"
echo ""

while true; do
    clear
    echo -e "${BLUE}KubeSkippy Demo Monitor${NC} - $(date)"
    echo "================================================"
    
    # Pod Status
    echo -e "\n${YELLOW}📦 Pod Status:${NC}"
    kubectl get pods -n demo-apps --no-headers | while read line; do
        pod=$(echo $line | awk '{print $1}')
        ready=$(echo $line | awk '{print $2}')
        status=$(echo $line | awk '{print $3}')
        restarts=$(echo $line | awk '{print $4}')
        
        if [[ "$status" == "Running" ]]; then
            echo -e "${GREEN}✓${NC} $pod - $status (Restarts: $restarts)"
        else
            echo -e "${RED}✗${NC} $pod - $status (Restarts: $restarts)"
        fi
    done
    
    # Resource Usage
    echo -e "\n${YELLOW}📊 Resource Usage:${NC}"
    kubectl top pods -n demo-apps --no-headers 2>/dev/null | while read line; do
        pod=$(echo $line | awk '{print $1}')
        cpu=$(echo $line | awk '{print $2}')
        memory=$(echo $line | awk '{print $3}')
        
        # Highlight high CPU usage
        if [[ ${cpu%m} -gt 800 ]]; then
            echo -e "${RED}⚠️  $pod - CPU: $cpu, Memory: $memory${NC}"
        else
            echo -e "   $pod - CPU: $cpu, Memory: $memory"
        fi
    done
    
    # Healing Policies
    echo -e "\n${YELLOW}🏥 Healing Policies:${NC}"
    kubectl get healingpolicies -n demo-apps --no-headers | while read line; do
        name=$(echo $line | awk '{print $1}')
        mode=$(echo $line | awk '{print $2}')
        actions=$(echo $line | awk '{print $3}')
        
        if [[ -z "$actions" ]]; then
            actions="0"
        fi
        
        echo -e "   $name - Mode: $mode, Actions: $actions"
    done
    
    # Healing Actions
    echo -e "\n${YELLOW}⚡ Healing Actions:${NC}"
    actions=$(kubectl get healingactions -n demo-apps --no-headers 2>/dev/null)
    if [[ -z "$actions" ]]; then
        echo -e "   No healing actions yet..."
    else
        echo "$actions" | while read line; do
            name=$(echo $line | awk '{print $1}')
            status=$(echo $line | awk '{print $2}')
            target=$(echo $line | awk '{print $3}')
            
            if [[ "$status" == "Completed" ]]; then
                echo -e "${GREEN}✓${NC} $name - $status (Target: $target)"
            elif [[ "$status" == "Failed" ]]; then
                echo -e "${RED}✗${NC} $name - $status (Target: $target)"
            else
                echo -e "${YELLOW}⟳${NC} $name - $status (Target: $target)"
            fi
        done
    fi
    
    # Recent Events
    echo -e "\n${YELLOW}📝 Recent Events:${NC}"
    kubectl get events -n demo-apps --sort-by='.lastTimestamp' | tail -5 | grep -E "(BackOff|Failed|Error|Unhealthy|Restarted)" | while read line; do
        echo -e "   ${line:0:100}..."
    done
    
    # Operator Status
    echo -e "\n${YELLOW}🤖 Operator Status:${NC}"
    operator_pod=$(kubectl get pods -n kubeskippy-system -l control-plane=controller-manager --no-headers | awk '{print $1}')
    if [[ -n "$operator_pod" ]]; then
        echo -e "   Pod: $operator_pod"
        echo -e "   Recent logs:"
        kubectl logs -n kubeskippy-system pod/$operator_pod --tail=3 | sed 's/^/   /'
    fi
    
    # Monitoring Status
    monitoring_ns=$(kubectl get namespace monitoring 2>/dev/null)
    if [[ -n "$monitoring_ns" ]]; then
        echo -e "\n${YELLOW}📊 Monitoring Status:${NC}"
        
        # Prometheus status
        prom_pod=$(kubectl get pods -n monitoring -l app=prometheus --no-headers 2>/dev/null | awk '{print $1}')
        if [[ -n "$prom_pod" ]]; then
            prom_status=$(kubectl get pods -n monitoring -l app=prometheus --no-headers 2>/dev/null | awk '{print $3}')
            if [[ "$prom_status" == "Running" ]]; then
                echo -e "   ${GREEN}✓${NC} Prometheus: $prom_pod ($prom_status)"
                echo -e "     Access: kubectl port-forward -n monitoring svc/prometheus 9090:9090"
            else
                echo -e "   ${RED}✗${NC} Prometheus: $prom_pod ($prom_status)"
            fi
        fi
        
        # Grafana status
        grafana_pod=$(kubectl get pods -n monitoring -l app=grafana --no-headers 2>/dev/null | awk '{print $1}')
        if [[ -n "$grafana_pod" ]]; then
            grafana_status=$(kubectl get pods -n monitoring -l app=grafana --no-headers 2>/dev/null | awk '{print $3}')
            if [[ "$grafana_status" == "Running" ]]; then
                echo -e "   ${GREEN}✓${NC} Grafana: $grafana_pod ($grafana_status)"
                echo -e "     Access: kubectl port-forward -n monitoring svc/grafana 3000:3000"
                echo -e "     Login: admin/admin"
            else
                echo -e "   ${RED}✗${NC} Grafana: $grafana_pod ($grafana_status)"
            fi
        fi
    fi
    
    echo -e "\n${BLUE}Press Ctrl+C to exit${NC}"
    sleep 5
done