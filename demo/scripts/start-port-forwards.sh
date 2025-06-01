#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Starting KubeSkippy Demo Port Forwards...${NC}"

# More thorough cleanup
echo -e "${YELLOW}🧹 Cleaning up existing port forwards...${NC}"
pkill -f "kubectl port-forward.*grafana" 2>/dev/null || true
pkill -f "kubectl port-forward.*prometheus" 2>/dev/null || true
pkill -f "kubectl port-forward.*monitoring" 2>/dev/null || true
sleep 3

# Verify cluster connectivity first
if ! kubectl cluster-info >/dev/null 2>&1; then
    echo -e "${RED}❌ Cannot connect to Kubernetes cluster${NC}"
    echo -e "${YELLOW}💡 Make sure your cluster is running: kind get clusters${NC}"
    exit 1
fi

# Verify services exist
if ! kubectl get service grafana -n monitoring >/dev/null 2>&1; then
    echo -e "${RED}❌ Grafana service not found in monitoring namespace${NC}"
    echo -e "${YELLOW}💡 Run: ./setup-clean.sh${NC}"
    exit 1
fi

if ! kubectl get service prometheus -n monitoring >/dev/null 2>&1; then
    echo -e "${RED}❌ Prometheus service not found in monitoring namespace${NC}"
    echo -e "${YELLOW}💡 Run: ./setup-clean.sh${NC}"
    exit 1
fi

# Start port forwards
echo -e "${YELLOW}🚀 Starting fresh port forwards...${NC}"

echo "📊 Starting Grafana port forward (localhost:3000)..."
kubectl port-forward -n monitoring service/grafana 3000:3000 >/dev/null 2>&1 &
GRAFANA_PID=$!

echo "📈 Starting Prometheus port forward (localhost:9090)..."
kubectl port-forward -n monitoring service/prometheus 9090:9090 >/dev/null 2>&1 &
PROMETHEUS_PID=$!

# Save PIDs immediately
echo $GRAFANA_PID > /tmp/kubeskippy-grafana.pid
echo $PROMETHEUS_PID > /tmp/kubeskippy-prometheus.pid

# Wait for port forwards to establish
echo "⏳ Waiting for port forwards to establish..."
sleep 8

# Test connections with retries
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

# Show results
echo ""
if [ "$grafana_success" = false ]; then
    echo -e "${RED}❌ Grafana not accessible after 30 seconds${NC}"
    echo -e "${YELLOW}💡 Check: kubectl get pods -n monitoring${NC}"
fi

if [ "$prometheus_success" = false ]; then
    echo -e "${RED}❌ Prometheus not accessible after 20 seconds${NC}"
    echo -e "${YELLOW}💡 Check: kubectl get pods -n monitoring${NC}"
fi

echo -e "${BLUE}📡 Port forward processes:${NC}"
echo "  Grafana PID: $GRAFANA_PID"
echo "  Prometheus PID: $PROMETHEUS_PID"

echo ""
echo -e "${GREEN}🎯 Port forwards are running in background${NC}"
echo -e "${YELLOW}📝 To stop: ./stop-port-forwards.sh${NC}"
echo -e "${YELLOW}📝 To restart: ./start-port-forwards.sh${NC}"
echo -e "${YELLOW}📝 To monitor: ./monitor-demo.sh${NC}"