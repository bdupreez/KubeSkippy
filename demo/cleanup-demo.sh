#!/bin/bash
echo "🧹 Cleaning up KubeSkippy Demo..."

# Stop port forwards
./stop-port-forwards.sh 2>/dev/null || true

# Delete Kind cluster
kind delete cluster --name kubeskippy-demo 2>/dev/null || true

# Clean up Docker images
docker rmi kubeskippy:latest 2>/dev/null || true

# Remove temp files
rm -f /tmp/kubeskippy-*.pid
rm -f /tmp/kind-config.yaml

echo "✅ Demo cleanup completed!"