#!/bin/bash

echo "🔧 Fixing Grafana Dashboard AI Queries"

# Backup the file
cp /Users/bdp/Engineering/KubeSkippy/demo/grafana/grafana-demo.yaml /Users/bdp/Engineering/KubeSkippy/demo/grafana/grafana-demo.yaml.bak

# Replace all remaining AI patterns with actual trigger patterns
perl -i -pe 's/trigger_type=~"\\.*ai\\.*"/trigger_type=~"predictive.*|continuous.*"/g' /Users/bdp/Engineering/KubeSkippy/demo/grafana/grafana-demo.yaml

echo "✅ Updated AI trigger patterns"

# Apply the updated dashboard
kubectl apply -f /Users/bdp/Engineering/KubeSkippy/demo/grafana/grafana-demo.yaml

echo "✅ Applied updated dashboard"

# Restart Grafana to pick up changes
kubectl rollout restart deployment/grafana -n monitoring

echo "✅ Restarted Grafana"

echo "🎯 Dashboard will be available shortly at: http://localhost:3000/d/kubeskippy-enhanced"