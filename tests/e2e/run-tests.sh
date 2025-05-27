#!/bin/bash
set -e

echo "🧪 Running KubeSkippy E2E Tests"
echo "==============================="

# Configuration
CLUSTER_NAME="${E2E_CLUSTER_NAME:-kubeskippy-e2e}"
USE_EXISTING_CLUSTER="${USE_EXISTING_CLUSTER:-false}"
SKIP_CLUSTER_SETUP="${SKIP_CLUSTER_SETUP:-false}"

# Setup test cluster if needed
if [[ "$SKIP_CLUSTER_SETUP" != "true" ]]; then
    if [[ "$USE_EXISTING_CLUSTER" == "true" ]]; then
        echo "📌 Using existing cluster..."
    else
        echo "🏗️  Creating test cluster..."
        kind create cluster --name ${CLUSTER_NAME} --config ../kind-config.yaml
        
        echo "📦 Installing CRDs..."
        kubectl apply -f ../../config/crd/bases/
        
        echo "🤖 Deploying test dependencies..."
        kubectl apply -f ../ollama-deployment.yaml
        
        echo "⏳ Waiting for dependencies..."
        kubectl wait --for=condition=ready pod -l app=ollama -n kubeskippy-system --timeout=300s || true
    fi
fi

# Run tests
echo ""
echo "🚀 Running E2E tests..."
export USE_EXISTING_CLUSTER=true
export KUBECONFIG="${HOME}/.kube/config"

# Run ginkgo tests
if command -v ginkgo &> /dev/null; then
    ginkgo -v --race --trace --fail-fast
else
    go test -v -race ./...
fi

TEST_EXIT_CODE=$?

# Cleanup if not using existing cluster
if [[ "$USE_EXISTING_CLUSTER" != "true" ]] && [[ "$SKIP_CLUSTER_SETUP" != "true" ]]; then
    echo ""
    echo "🧹 Cleaning up test cluster..."
    kind delete cluster --name ${CLUSTER_NAME}
fi

echo ""
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo "✅ All E2E tests passed!"
else
    echo "❌ Some E2E tests failed!"
fi

exit $TEST_EXIT_CODE