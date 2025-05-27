#!/bin/bash
set -e

CLUSTER_NAME="kubeskippy-demo"

echo "🧹 KubeSkippy Demo Cleanup"
echo "=========================="

# Delete Kind cluster
echo ""
echo "🗑️  Deleting Kind cluster..."
if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    kind delete cluster --name ${CLUSTER_NAME}
    echo "✅ Cluster deleted!"
else
    echo "⚠️  Cluster ${CLUSTER_NAME} not found."
fi

# Clean up Docker images (optional)
echo ""
read -p "Delete local Docker images? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "🗑️  Removing Docker images..."
    docker rmi kubeskippy:latest || true
    docker rmi ollama/ollama:latest || true
    echo "✅ Docker images removed!"
fi

echo ""
echo "✅ Cleanup complete!"