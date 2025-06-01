#!/bin/bash
# Port forwarding functions

setup_port_forwards() {
    echo -e "${YELLOW}🧹 Cleaning up existing port forwards...${NC}"
    # Kill any existing port forwards more thoroughly
    pkill -f "kubectl port-forward.*grafana" 2>/dev/null || true
    pkill -f "kubectl port-forward.*prometheus" 2>/dev/null || true
    pkill -f "kubectl port-forward.*monitoring" 2>/dev/null || true
    sleep 3
    
    echo -e "${YELLOW}🚀 Starting fresh port forwards...${NC}"
    
    # Start Grafana port forward
    echo "📊 Starting Grafana port forward (localhost:3000)..."
    kubectl port-forward -n monitoring service/grafana 3000:3000 >/dev/null 2>&1 &
    GRAFANA_PID=$!
    
    # Start Prometheus port forward  
    echo "📈 Starting Prometheus port forward (localhost:9090)..."
    kubectl port-forward -n monitoring service/prometheus 9090:9090 >/dev/null 2>&1 &
    PROMETHEUS_PID=$!
    
    # Save PIDs immediately
    echo "$GRAFANA_PID" > /tmp/kubeskippy-grafana.pid
    echo "$PROMETHEUS_PID" > /tmp/kubeskippy-prometheus.pid
    
    # Wait for port forwards to establish
    echo "⏳ Waiting for port forwards to establish..."
    sleep 8
    
    # Test connections with better error handling
    echo ""
    echo -e "${YELLOW}🔍 Testing dashboard access...${NC}"
    
    # Test Grafana
    grafana_success=false
    for i in {1..15}; do
        if curl -s -m 5 http://localhost:3000 >/dev/null 2>&1; then
            echo -e "${GREEN}✅ Grafana accessible at http://localhost:3000 (admin/admin)${NC}"
            grafana_success=true
            break
        fi
        [ $((i % 5)) -eq 0 ] && echo -e "${YELLOW}   Still trying Grafana... ($i/15)${NC}"
        sleep 2
    done
    
    if [ "$grafana_success" = false ]; then
        echo -e "${RED}❌ Grafana not accessible after 30 seconds${NC}"
        echo -e "${YELLOW}💡 Try: ./start-port-forwards.sh${NC}"
    fi
    
    # Test Prometheus  
    prometheus_success=false
    for i in {1..10}; do
        if curl -s -m 5 http://localhost:9090 >/dev/null 2>&1; then
            echo -e "${GREEN}✅ Prometheus accessible at http://localhost:9090${NC}"
            prometheus_success=true
            break
        fi
        [ $((i % 5)) -eq 0 ] && echo -e "${YELLOW}   Still trying Prometheus... ($i/10)${NC}"
        sleep 2
    done
    
    if [ "$prometheus_success" = false ]; then
        echo -e "${RED}❌ Prometheus not accessible after 20 seconds${NC}"
        echo -e "${YELLOW}💡 Try: ./start-port-forwards.sh${NC}"
    fi
    
    # Show process status
    echo ""
    echo -e "${BLUE}📡 Port forward processes:${NC}"
    echo "  Grafana PID: $GRAFANA_PID"
    echo "  Prometheus PID: $PROMETHEUS_PID"
    
    # Verify processes are still running
    if ! kill -0 $GRAFANA_PID 2>/dev/null; then
        echo -e "${RED}⚠️ Grafana port forward died${NC}"
    fi
    if ! kill -0 $PROMETHEUS_PID 2>/dev/null; then
        echo -e "${RED}⚠️ Prometheus port forward died${NC}"
    fi
}