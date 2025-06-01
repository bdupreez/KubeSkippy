#!/bin/bash
echo "🚀 Starting KubeSkippy Demo Port Forwards..."

# Kill any existing port forwards
pkill -f "kubectl port-forward.*grafana" 2>/dev/null || true
pkill -f "kubectl port-forward.*prometheus" 2>/dev/null || true
sleep 2

# Start Grafana port forward
echo "📊 Starting Grafana port forward (localhost:3000)..."
kubectl port-forward -n monitoring service/grafana 3000:3000 >/dev/null 2>&1 &
GRAFANA_PID=$!

# Start Prometheus port forward
echo "📈 Starting Prometheus port forward (localhost:9090)..."
kubectl port-forward -n monitoring service/prometheus 9090:9090 >/dev/null 2>&1 &
PROMETHEUS_PID=$!

# Wait and test
sleep 5

# Test connections
if curl -s http://localhost:3000 >/dev/null 2>&1; then
    echo "✅ Grafana accessible at http://localhost:3000 (admin/admin)"
else
    echo "⚠️ Grafana may not be ready yet"
fi

if curl -s http://localhost:9090 >/dev/null 2>&1; then
    echo "✅ Prometheus accessible at http://localhost:9090"
else
    echo "⚠️ Prometheus may not be ready yet"
fi

# Save PIDs
echo $GRAFANA_PID > /tmp/kubeskippy-grafana.pid
echo $PROMETHEUS_PID > /tmp/kubeskippy-prometheus.pid

echo ""
echo "🎯 Port forwards are running in background"
echo "📝 To stop: ./stop-port-forwards.sh"
echo "📝 To restart: ./start-port-forwards.sh"