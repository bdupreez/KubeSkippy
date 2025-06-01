#!/bin/bash
set -e

CLUSTER_NAME="kubeskippy-demo"
NAMESPACE="kubeskippy-system"
DEMO_NAMESPACE="demo-apps"

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

echo "🚀 KubeSkippy AI & Continuous Healing Demo Setup"
echo "=================================================="
echo "🧠 Includes: AI-driven healing, predictive analysis, continuous failures"
echo "📊 Enhanced: Grafana dashboard with AI intelligence metrics"
echo ""

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

# Create namespace first
echo ""
echo "📁 Creating namespace..."
kubectl create namespace ${NAMESPACE} || true

# Start parallel deployment of independent components
echo ""
echo "🚀 Starting parallel deployment of components..."

# Start Docker build in background (this is slow)
echo "🔨 Building operator image (background)..."
docker build -t kubeskippy:latest "$SCRIPT_DIR/.." > /tmp/docker-build.log 2>&1 &
DOCKER_BUILD_PID=$!

# Deploy metrics-server (independent)
echo "📊 Installing metrics-server..."
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml > /dev/null
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
]' > /dev/null

# Deploy Ollama (independent, slow)
echo "🤖 Deploying Ollama..."
kubectl apply -f "$SCRIPT_DIR/../tests/ollama-deployment.yaml" > /dev/null || {
    echo "  ❌ Failed to deploy Ollama"
    exit 1
}

echo ""
echo "⏳ Waiting for components to be ready..."
echo "  - Docker build, metrics-server, and Ollama are running in parallel"
echo "  - This may take 3-5 minutes total"

# Wait for Docker build to complete first (we need this for operator)
echo "  - Waiting for Docker build to complete..."
wait $DOCKER_BUILD_PID
if [ $? -eq 0 ]; then
    echo "  ✅ Docker build completed"
    echo "  - Loading image to Kind cluster..."
    kind load docker-image kubeskippy:latest --name ${CLUSTER_NAME}
    echo "  ✅ Operator image loaded"
else
    echo "  ❌ Docker build failed, check /tmp/docker-build.log"
    exit 1
fi

# Check metrics-server (non-blocking, it usually works)
kubectl wait --for=condition=ready pod -l k8s-app=metrics-server -n kube-system --timeout=10s > /dev/null 2>&1 && echo "  ✅ Metrics-server ready" || echo "  ⏳ Metrics-server still starting..."

# Start Ollama readiness check in background (we don't need to block on it)
echo "  - Starting Ollama readiness check (background)..."
kubectl wait --for=condition=ready pod -l app=ollama -n ${NAMESPACE} --timeout=300s > /dev/null 2>&1 && echo "  ✅ Ollama ready" || echo "  ⚠️  Ollama timeout" &
OLLAMA_WAIT_PID=$!

# Deploy operator
echo ""
echo "🚀 Deploying KubeSkippy operator..."

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
kubectl apply -f "$SCRIPT_DIR/monitoring/fix-rbac.yaml"

# Create metrics service for operator
echo "  - Creating metrics service..."
kubectl apply -f "$SCRIPT_DIR/monitoring/operator-metrics-service.yaml"

# Restart operator to pick up RBAC changes
echo "  - Restarting operator..."
kubectl rollout restart deployment kubeskippy-controller-manager -n ${NAMESPACE}
kubectl wait --for=condition=ready pod -l control-plane=controller-manager -n ${NAMESPACE} --timeout=120s || {
    echo "⚠️  Operator restart taking longer than expected, continuing..."
}

echo "✅ Operator fully deployed with RBAC!"

# Create demo namespace
echo ""
echo "📁 Creating demo namespace..."
kubectl create namespace ${DEMO_NAMESPACE} || true
echo "✅ Demo namespace created!"

# Deploy demo applications
echo ""
echo "🎯 Deploying demo applications..."
cd "$SCRIPT_DIR"
if [ -d "apps" ]; then
    echo "  - Deploying standard demo apps..."
    for app in apps/crashloop-app.yaml apps/memory-leak-app.yaml apps/cpu-spike-app.yaml apps/flaky-web-app.yaml; do
        kubectl apply -f "$app" || echo "    ⚠️ Failed to deploy $(basename $app)"
    done
    
    echo "  - Deploying AI & continuous failure apps..."
    for app in apps/continuous-*.yaml apps/stress-generator-app.yaml apps/demo-activity-generator.yaml apps/chaos-monkey-component.yaml; do
        if [ -f "$app" ]; then
            kubectl apply -f "$app" || echo "    ⚠️ Failed to deploy $(basename $app)"
        fi
    done
    
    echo "  - Deploying pattern failure app..."
    kubectl apply -f apps/pattern-failure-app.yaml || echo "    ⚠️ Failed to deploy pattern-failure-app"
else
    echo "  ❌ Apps directory not found!"
    exit 1
fi
echo "✅ Demo applications deployed! (includes AI & continuous failure scenarios)"

# Apply healing policies  
echo ""
echo "🏥 Applying healing policies..."
cd "$SCRIPT_DIR"
POLICY_SUCCESS=0
POLICY_FAILED=0

# Apply core healing policies first
for policy in policies/crashloop-healing.yaml policies/memory-healing.yaml policies/cpu-spike-healing.yaml policies/service-degradation-healing.yaml; do
    if [ -f "$policy" ]; then
        echo "  - Applying $(basename $policy)..."
        if kubectl apply -f $policy; then
            ((POLICY_SUCCESS++))
        else
            echo "    ⚠️ Failed to apply $(basename $policy)"
            ((POLICY_FAILED++))
        fi
    fi
done

# Apply AI and predictive policies
echo "  - Applying AI & predictive healing policies..."
for policy in policies/ai-driven-healing.yaml policies/ai-intelligent-healing-simple.yaml policies/predictive-ai-healing-simple.yaml policies/ai-strategic-healing.yaml policies/continuous-*.yaml; do
    if [ -f "$policy" ]; then
        echo "  - Applying $(basename $policy)..."
        if kubectl apply -f $policy; then
            ((POLICY_SUCCESS++))
        else
            echo "    ⚠️ Failed to apply $(basename $policy)"
            ((POLICY_FAILED++))
        fi
    fi
done

# Apply Prometheus-based policies if monitoring is enabled
if [[ "$WITH_PROMETHEUS" == "true" ]] || [[ "$WITH_MONITORING" == "true" ]] || [[ "$1" == "--with-prometheus" ]] || [[ "$1" == "--with-monitoring" ]]; then
    if [ -f "policies/prometheus-based-healing.yaml" ]; then
        echo "  - Applying prometheus-based-healing.yaml..."
        if kubectl apply -f policies/prometheus-based-healing.yaml; then
            ((POLICY_SUCCESS++))
        else
            echo "    ⚠️ Failed to apply prometheus-based-healing.yaml"
            ((POLICY_FAILED++))
        fi
    fi
fi

echo "✅ Healing policies applied! (${POLICY_SUCCESS} successful, ${POLICY_FAILED} failed)"
echo "🧠 AI-driven healing enabled by default with predictive capabilities"

# Deploy monitoring stack by default (can be disabled with --no-monitoring)
if [[ "$1" != "--no-monitoring" ]] && [[ "$1" != "--basic" ]]; then
    echo ""
    echo "📊 Deploying monitoring stack..."
    
    # Deploy monitoring components in parallel
    echo "  - Deploying monitoring stack (parallel)..."
    kubectl apply -f monitoring/kube-state-metrics.yaml > /dev/null
    kubectl apply -f prometheus/prometheus-demo.yaml > /dev/null
    
    # Deploy Grafana by default (unless specifically basic mode)
    if [[ "$1" != "--basic" ]] && [[ "$1" != "--prometheus-only" ]]; then
        kubectl apply -f grafana/grafana-demo.yaml > /dev/null
        echo "  - Deployed: kube-state-metrics, Prometheus, Grafana"
    else
        echo "  - Deployed: kube-state-metrics, Prometheus"
    fi
    
    # Start readiness checks in background
    echo "  - Starting readiness checks..."
    kubectl wait --for=condition=ready pod -l app=kube-state-metrics -n kube-system --timeout=120s > /dev/null 2>&1 && echo "  ✅ kube-state-metrics ready" || echo "  ⚠️  kube-state-metrics timeout" &
    kubectl wait --for=condition=ready pod -l app=prometheus -n monitoring --timeout=120s > /dev/null 2>&1 && echo "  ✅ Prometheus ready" || echo "  ⚠️  Prometheus timeout" &
    
    if [[ "$1" == "--with-monitoring" ]] || [[ "$WITH_MONITORING" == "true" ]]; then
        kubectl wait --for=condition=ready pod -l app=grafana -n monitoring --timeout=120s > /dev/null 2>&1 && echo "  ✅ Grafana ready" || echo "  ⚠️  Grafana timeout" &
    fi
    
    # Update operator config to use Prometheus (non-blocking)
    kubectl patch configmap kubeskippy-config -n ${NAMESPACE} --type merge -p '
    {
      "data": {
        "config.yaml": "metrics:\n  prometheusURL: \"http://prometheus.monitoring:9090\"\n"
      }
    }' > /dev/null 2>&1 || echo "  - Config will use Prometheus when operator restarts"
    
    echo "  - Monitoring stack deployment initiated"
fi

# Wait for all background processes and provide final status
echo ""
echo "⏳ Finalizing setup..."
echo "  - Waiting for all background processes to complete..."

# Wait for Ollama if it's still running (with timeout)
if ps -p $OLLAMA_WAIT_PID > /dev/null 2>&1; then
    # Wait up to 30 seconds for Ollama
    for i in {1..30}; do
        if ! ps -p $OLLAMA_WAIT_PID > /dev/null 2>&1; then
            break
        fi
        sleep 1
    done
    # Kill if still running
    if ps -p $OLLAMA_WAIT_PID > /dev/null 2>&1; then
        kill $OLLAMA_WAIT_PID 2>/dev/null || true
    fi
fi

# Give background monitoring waits a moment to complete
sleep 2

# Final status check
echo ""
echo "🎯 Final Status Check:"
echo "  - Cluster: $(kubectl config current-context)"
echo "  - Operator: $(kubectl get pods -n ${NAMESPACE} -l control-plane=controller-manager --no-headers | wc -l | tr -d ' ') pod(s)"
echo "  - Demo apps: $(kubectl get deployments -n ${DEMO_NAMESPACE} --no-headers 2>/dev/null | wc -l | tr -d ' ') deployment(s)"
echo "  - Policies: $(kubectl get healingpolicies -n ${DEMO_NAMESPACE} --no-headers 2>/dev/null | wc -l | tr -d ' ') policy(ies)"

# Show monitoring access if deployed (default)
if [[ "$1" != "--no-monitoring" ]] && [[ "$1" != "--basic" ]]; then
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
    
    # Wait for Grafana to be fully initialized
    echo "  - Waiting for Grafana to be fully ready..."
    for i in {1..30}; do
        if curl -s -u admin:admin http://localhost:3000/api/health | grep -q '"status":"success"'; then
            echo "  ✅ Grafana is ready"
            break
        fi
        sleep 2
    done
    
    # Check if dashboards are already loaded via ConfigMap provisioning
    echo "  - Checking dashboard status..."
    DASHBOARD_COUNT=$(curl -s -u admin:admin 'http://localhost:3000/api/search?type=dash-db' | jq '. | length' 2>/dev/null || echo 0)
    
    if [ "${DASHBOARD_COUNT:-0}" -ge 2 ]; then
        echo "  ✅ Dashboards already provisioned via ConfigMap"
        echo "  - KubeSkippy Enhanced Dashboard: http://localhost:3000/d/kubeskippy-enhanced"
        echo "  - KubeSkippy Demo Overview: http://localhost:3000/d/kubeskippy-demo"
    else
        echo "  ⚠️ Dashboards not found, manual import may be needed"
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
echo "🎯 KubeSkippy AI & Continuous Healing Demo Ready!"
echo "Enhanced with predictive AI, continuous failures, and real-time monitoring."
echo ""
echo "To clean up: ./cleanup.sh"

# Function to fix Grafana dashboard (only used if monitoring is deployed)
fix_grafana_dashboard() {
  if kubectl get ns monitoring >/dev/null 2>&1; then
    echo "🔄 Fixing Grafana dashboard provisioning..."
    kubectl -n monitoring apply -f grafana/grafana-demo.yaml
    kubectl -n monitoring delete pod -l app.kubernetes.io/name=grafana
    echo "✅ Grafana dashboard fix applied!"
  fi
}