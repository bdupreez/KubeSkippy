#!/bin/bash
set -e

CLUSTER_NAME="kubeskippy-demo"
NAMESPACE="kubeskippy-system"
DEMO_NAMESPACE="demo-apps"

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

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

kind create cluster --name ${CLUSTER_NAME} --config "$SCRIPT_DIR/../tests/kind-config.yaml"
echo "✅ Kind cluster created!"

# Install CRDs
echo ""
echo "📦 Installing CRDs..."
kubectl apply -f "$SCRIPT_DIR/../config/crd/bases/"
echo "✅ CRDs installed!"

# Install metrics-server
echo ""
echo "📊 Installing metrics-server..."
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
# Patch metrics-server for kind (disable TLS verification)
kubectl patch deployment metrics-server -n kube-system --type='json' -p='[
  {
    "op": "add",
    "path": "/spec/template/spec/containers/0/args/-",
    "value": "--kubelet-insecure-tls"
  },
  {
    "op": "add", 
    "path": "/spec/template/spec/containers/0/args/-",
    "value": "--kubelet-preferred-address-types=InternalIP,ExternalIP,Hostname"
  }
]'
echo "⏳ Waiting for metrics-server to be ready..."
kubectl wait --for=condition=ready pod -l k8s-app=metrics-server -n kube-system --timeout=60s || {
    echo "⚠️  Metrics-server taking longer than expected, continuing..."
}
echo "✅ Metrics-server installed!"

# Deploy Ollama
echo ""
echo "🤖 Deploying Ollama..."
kubectl apply -f "$SCRIPT_DIR/../tests/ollama-deployment.yaml"
echo "⏳ Waiting for Ollama to be ready..."
# Note: Ollama deploys to ai-nanny-system namespace
kubectl wait --for=condition=ready pod -l app=ollama -n ai-nanny-system --timeout=60s || {
    echo "⚠️  Ollama taking longer than expected, continuing..."
}
echo "✅ Ollama deployed!"

# Build and load operator image
echo ""
echo "🔨 Building operator image..."
docker build -t kubeskippy:latest "$SCRIPT_DIR/.."
echo "  - Loading image to Kind cluster..."
kind load docker-image kubeskippy:latest --name ${CLUSTER_NAME}
# Wait a moment for the image to be available
sleep 5
echo "✅ Operator image built and loaded!"

# Deploy operator
echo ""
echo "🚀 Deploying KubeSkippy operator..."
kubectl create namespace ${NAMESPACE} || true

# Build the kustomized operator manifests and deploy
echo "  - Building operator manifests..."
# Go to project root
cd "$SCRIPT_DIR/.."
cd config/manager && kustomize edit set image controller=kubeskippy:latest
# Set imagePullPolicy to Never for local images
cd ../..
# Apply the manifests and then patch the deployment
kustomize build config/default | kubectl apply -f -
# Patch the deployment to use local image
kubectl patch deployment kubeskippy-controller-manager -n ${NAMESPACE} --type='json' -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/imagePullPolicy", "value": "Never"}]' || true

echo "⏳ Waiting for operator to be ready..."
kubectl wait --for=condition=ready pod -l control-plane=controller-manager -n ${NAMESPACE} --timeout=180s || {
    echo "⚠️  Operator deployment issue, checking status..."
    kubectl get pods -n ${NAMESPACE}
    kubectl describe pods -n ${NAMESPACE} | grep -A 5 "Events:"
}

# Apply RBAC fix for leader election
echo "  - Applying RBAC fixes..."
kubectl apply -f monitoring/fix-rbac.yaml

# Create metrics service for operator
echo "  - Creating metrics service..."
kubectl apply -f monitoring/operator-metrics-service.yaml

# Restart operator to pick up RBAC changes
echo "  - Restarting operator..."
kubectl rollout restart deployment kubeskippy-controller-manager -n ${NAMESPACE}
kubectl wait --for=condition=ready pod -l control-plane=controller-manager -n ${NAMESPACE} --timeout=120s || {
    echo "⚠️  Operator restart taking longer than expected, continuing..."
}

echo "✅ Operator deployed!"

# Create demo namespace
echo ""
echo "📁 Creating demo namespace..."
kubectl create namespace ${DEMO_NAMESPACE} || true
echo "✅ Demo namespace created!"

# Deploy demo applications
echo ""
echo "🎯 Deploying demo applications..."
cd "$SCRIPT_DIR"
for app in apps/*.yaml; do
    if [[ -f "$app" ]]; then
        echo "  - Deploying $(basename $app)..."
        kubectl apply -f "$app"
    fi
done
echo "✅ Demo applications deployed!"

# Apply healing policies
echo ""
echo "🏥 Applying healing policies..."
cd "$SCRIPT_DIR"
for policy in policies/*.yaml; do
    if [[ ! "$policy" == *"prometheus-based"* ]] || [[ "$WITH_PROMETHEUS" == "true" ]] || [[ "$WITH_MONITORING" == "true" ]] || [[ "$1" == "--with-prometheus" ]] || [[ "$1" == "--with-monitoring" ]]; then
        echo "  - Applying $(basename $policy)..."
        kubectl apply -f $policy
    fi
done
echo "✅ Healing policies applied!"

# Optional: Deploy Prometheus
if [[ "$1" == "--with-monitoring" ]] || [[ "$1" == "--with-prometheus" ]] || [[ "$WITH_PROMETHEUS" == "true" ]] || [[ "$WITH_MONITORING" == "true" ]]; then
    echo ""
    echo "📊 Deploying monitoring stack..."
    
    # Deploy kube-state-metrics first
    echo "  - Deploying kube-state-metrics..."
    kubectl apply -f monitoring/kube-state-metrics.yaml
    kubectl wait --for=condition=ready pod -l app=kube-state-metrics -n kube-system --timeout=120s || {
        echo "⚠️  kube-state-metrics taking longer than expected, continuing..."
    }
    
    echo "  - Deploying Prometheus..."
    kubectl apply -f prometheus/prometheus-demo.yaml
    echo "⏳ Waiting for Prometheus to be ready..."
    kubectl wait --for=condition=ready pod -l app=prometheus -n monitoring --timeout=120s || {
        echo "⚠️  Prometheus taking longer than expected, continuing..."
    }
    
    # Update operator config to use Prometheus
    kubectl patch configmap kubeskippy-config -n ${NAMESPACE} --type merge -p '
    {
      "data": {
        "config.yaml": "metrics:\n  prometheusURL: \"http://prometheus.monitoring:9090\"\n"
      }
    }' 2>/dev/null || echo "Config will use Prometheus when operator restarts"
    
    echo "✅ Prometheus deployed!"
    
    # Deploy Grafana if monitoring stack is requested
    if [[ "$1" == "--with-monitoring" ]] || [[ "$WITH_MONITORING" == "true" ]]; then
        echo ""
        echo "📈 Deploying Grafana..."
        kubectl apply -f grafana/grafana-demo.yaml
        echo "⏳ Waiting for Grafana to be ready..."
        kubectl wait --for=condition=ready pod -l app=grafana -n monitoring --timeout=120s || {
            echo "⚠️  Grafana taking longer than expected, continuing..."
        }
        echo "✅ Grafana deployed!"
        
        # Show access information
        echo ""
        echo "📊 Monitoring Access:"
        echo "Starting port forwarding automatically..."
        
        # Kill any existing port-forward processes
        pkill -f "port-forward.*grafana" 2>/dev/null || true
        pkill -f "port-forward.*prometheus" 2>/dev/null || true
        
        # Start port forwarding in background
        kubectl port-forward -n monitoring svc/grafana 3000:3000 > /dev/null 2>&1 &
        kubectl port-forward -n monitoring svc/prometheus 9090:9090 > /dev/null 2>&1 &
        
        sleep 3
        echo "✅ Port forwarding started!"
        echo "Prometheus: http://localhost:9090"
        echo "Grafana: http://localhost:3000 (admin/admin)"
    fi
fi

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
if [[ "$1" == "--with-monitoring" ]] || [[ "$WITH_MONITORING" == "true" ]]; then
    echo "🎉 Demo with monitoring is ready!"
    echo ""
    echo "Access the monitoring:"
    echo "- Grafana Dashboard: http://localhost:3000 (admin/admin)"
    echo "- Prometheus: http://localhost:9090"
    echo ""
    echo "Monitor the demo:"
    echo "- Watch healing: ./monitor.sh"
    echo "- Quick status: ./check-demo.sh"
else
    echo "Optional monitoring:"
    echo "- Redeploy with monitoring: ./setup.sh --with-monitoring"
    echo "- Prometheus only: ./setup.sh --with-prometheus"
fi
echo ""
echo "To clean up: ./cleanup.sh"