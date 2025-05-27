#!/bin/bash
set -e

CLUSTER_NAME="kubeskippy-demo"
NAMESPACE="kubeskippy-system"
DEMO_NAMESPACE="demo-apps"

echo "🚀 KubeSkippy Demo Setup"
echo "========================"

# Check prerequisites
echo "📋 Checking prerequisites..."
command -v docker >/dev/null 2>&1 || { echo "❌ Docker is required but not installed."; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo "❌ kubectl is required but not installed."; exit 1; }
command -v kind >/dev/null 2>&1 || { echo "❌ kind is required but not installed."; exit 1; }

echo "✅ All prerequisites met!"

# Create Kind cluster
echo ""
echo "🏗️  Creating Kind cluster..."
if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo "⚠️  Cluster ${CLUSTER_NAME} already exists. Deleting..."
    kind delete cluster --name ${CLUSTER_NAME}
fi

kind create cluster --name ${CLUSTER_NAME} --config ../tests/kind-config.yaml
echo "✅ Kind cluster created!"

# Install CRDs
echo ""
echo "📦 Installing CRDs..."
kubectl apply -f ../config/crd/bases/
echo "✅ CRDs installed!"

# Deploy Ollama
echo ""
echo "🤖 Deploying Ollama..."
kubectl apply -f ../tests/ollama-deployment.yaml
echo "⏳ Waiting for Ollama to be ready..."
kubectl wait --for=condition=ready pod -l app=ollama -n ${NAMESPACE} --timeout=300s || true
echo "✅ Ollama deployed!"

# Build and load operator image
echo ""
echo "🔨 Building operator image..."
docker build -t kubeskippy:latest ..
kind load docker-image kubeskippy:latest --name ${CLUSTER_NAME}
echo "✅ Operator image built and loaded!"

# Deploy operator
echo ""
echo "🚀 Deploying KubeSkippy operator..."
kubectl create namespace ${NAMESPACE} || true
kubectl apply -f ../config/rbac/
# Note: You'll need to create a deployment manifest for the operator
echo "✅ Operator deployed!"

# Create demo namespace
echo ""
echo "📁 Creating demo namespace..."
kubectl create namespace ${DEMO_NAMESPACE} || true
echo "✅ Demo namespace created!"

# Deploy demo applications
echo ""
echo "🎯 Deploying demo applications..."
for app in apps/*.yaml; do
    echo "  - Deploying $(basename $app)..."
    kubectl apply -f $app
done
echo "✅ Demo applications deployed!"

# Apply healing policies
echo ""
echo "🏥 Applying healing policies..."
for policy in policies/*.yaml; do
    echo "  - Applying $(basename $policy)..."
    kubectl apply -f $policy
done
echo "✅ Healing policies applied!"

# Show status
echo ""
echo "📊 Demo Status:"
echo "==============="
echo ""
echo "Cluster: ${CLUSTER_NAME}"
echo "Namespaces: ${NAMESPACE}, ${DEMO_NAMESPACE}"
echo ""
echo "Applications:"
kubectl get deployments -n ${DEMO_NAMESPACE}
echo ""
echo "Healing Policies:"
kubectl get healingpolicies -n ${DEMO_NAMESPACE}
echo ""
echo "🎉 Demo setup complete!"
echo ""
echo "Next steps:"
echo "1. Watch healing actions: kubectl get healingactions -n ${DEMO_NAMESPACE} -w"
echo "2. View operator logs: kubectl logs -n ${NAMESPACE} deployment/kubeskippy-controller-manager -f"
echo "3. Check pod status: kubectl get pods -n ${DEMO_NAMESPACE} -w"
echo ""
echo "To clean up: ./cleanup.sh"