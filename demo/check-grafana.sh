#!/bin/bash

echo "🔍 Checking Grafana Dashboard Status"
echo "===================================="

# Check if port-forward is active
if ! ps aux | grep -q "[p]ort-forward.*grafana"; then
    echo "⚠️  Grafana port-forward not active. Starting..."
    kubectl port-forward -n monitoring svc/grafana 3000:3000 > /dev/null 2>&1 &
    sleep 3
fi

# Check Grafana health
echo -n "Grafana Status: "
if curl -s -u admin:admin http://localhost:3000/api/health | grep -q "ok"; then
    echo "✅ Healthy"
else
    echo "❌ Not responding"
    exit 1
fi

# Check datasource
echo -n "Prometheus Datasource: "
if curl -s -u admin:admin http://localhost:3000/api/datasources | grep -q "Prometheus"; then
    echo "✅ Configured"
else
    echo "❌ Not found"
fi

# List dashboards
echo ""
echo "📊 Available Dashboards:"
curl -s -u admin:admin 'http://localhost:3000/api/search?type=dash-db' | jq -r '.[] | "  - \(.title): http://localhost:3000/d/\(.uid)"'

echo ""
echo "🎯 Direct Links:"
echo "  - Enhanced Dashboard (with AI metrics): http://localhost:3000/d/kubeskippy-enhanced"
echo "  - Original Dashboard: http://localhost:3000/d/kubeskippy-demo"
echo ""
echo "Login: admin/admin"