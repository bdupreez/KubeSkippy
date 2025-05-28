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

# Keep Docker images for faster subsequent runs
echo ""
echo "ℹ️  Keeping Docker images for faster subsequent runs."

echo ""
echo "✅ Cleanup complete!"