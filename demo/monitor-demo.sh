#!/bin/bash
echo "👀 KubeSkippy Demo Monitoring Dashboard"
echo "====================================="

# Check cluster status
echo "🏗️ Cluster Status:"
kubectl cluster-info --context kind-kubeskippy-demo | head -2

echo ""
echo "🤖 Ollama AI Status:"
kubectl get pods -n kubeskippy-system -l app=ollama --no-headers

echo ""
echo "📊 Monitoring Stack:"
kubectl get pods -n monitoring --no-headers

echo ""
echo "🔧 KubeSkippy Operator:"
kubectl get pods -n kubeskippy-system -l control-plane=controller-manager --no-headers

echo ""
echo "🎯 Demo Applications:"
kubectl get pods -n demo-apps --no-headers

echo ""
echo "🏥 Healing Policies:"
kubectl get healingpolicies -n demo-apps --no-headers 2>/dev/null || echo "No policies found"

echo ""
echo "⚡ Recent Healing Actions:"
kubectl get healingactions -n demo-apps --no-headers --sort-by=.metadata.creationTimestamp 2>/dev/null | tail -5 || echo "No actions yet"

echo ""
echo "📡 Port Forward Status:"
if pgrep -f "kubectl port-forward.*grafana" >/dev/null; then
    echo "✅ Grafana port forward running"
else
    echo "❌ Grafana port forward not running"
fi

if pgrep -f "kubectl port-forward.*prometheus" >/dev/null; then
    echo "✅ Prometheus port forward running"
else
    echo "❌ Prometheus port forward not running"
fi

echo ""
echo "🌐 Access URLs:"
echo "📊 Grafana: http://localhost:3000 (admin/admin)"
echo "📈 Prometheus: http://localhost:9090"
echo ""
echo "📝 Commands:"
echo "  Watch healing actions: kubectl get healingactions -n demo-apps -w"
echo "  Operator logs: kubectl logs -f -n kubeskippy-system deployment/kubeskippy-controller-manager"
echo "  Restart port forwards: ./start-port-forwards.sh"