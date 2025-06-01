#!/bin/bash
echo "🛑 Stopping KubeSkippy Demo Port Forwards..."

# Kill port forwards using PIDs if available
if [ -f /tmp/kubeskippy-grafana.pid ]; then
    GRAFANA_PID=$(cat /tmp/kubeskippy-grafana.pid)
    if [ -n "$GRAFANA_PID" ]; then
        kill $GRAFANA_PID 2>/dev/null && echo "✅ Stopped Grafana port forward"
    fi
    rm -f /tmp/kubeskippy-grafana.pid
fi

if [ -f /tmp/kubeskippy-prometheus.pid ]; then
    PROMETHEUS_PID=$(cat /tmp/kubeskippy-prometheus.pid)
    if [ -n "$PROMETHEUS_PID" ]; then
        kill $PROMETHEUS_PID 2>/dev/null && echo "✅ Stopped Prometheus port forward"
    fi
    rm -f /tmp/kubeskippy-prometheus.pid
fi

# Backup method: kill by process name
pkill -f "kubectl port-forward.*grafana" 2>/dev/null || true
pkill -f "kubectl port-forward.*prometheus" 2>/dev/null || true

echo "🎯 All port forwards stopped"