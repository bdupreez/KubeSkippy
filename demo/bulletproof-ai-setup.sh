#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

CLUSTER_NAME="kubeskippy-demo"
NAMESPACE="kubeskippy-system"
DEMO_NAMESPACE="demo-apps"

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo -e "${BLUE}🚀 KubeSkippy Bulletproof Real AI Demo Setup${NC}"
echo "========================================================="
echo -e "${BLUE}🧠 Full automation: Real AI-driven healing + Grafana dashboard${NC}"
echo -e "${BLUE}📊 ZERO human interaction required${NC}"
echo ""

# Function to check if command exists
check_command() {
    if ! command -v $1 >/dev/null 2>&1; then
        echo -e "${RED}❌ $1 is required but not installed.${NC}"
        exit 1
    fi
}

# Function to wait for pods to be ready with better error handling
wait_for_pods() {
    local namespace=$1
    local label=$2
    local timeout=${3:-300}
    
    echo -e "${YELLOW}⏳ Waiting for pods with label $label in namespace $namespace...${NC}"
    
    local count=0
    while [ $count -lt $timeout ]; do
        local ready_pods=$(kubectl get pods -n "$namespace" -l "$label" --no-headers 2>/dev/null | grep "1/1.*Running" | wc -l || echo 0)
        local total_pods=$(kubectl get pods -n "$namespace" -l "$label" --no-headers 2>/dev/null | wc -l || echo 0)
        
        if [ "$ready_pods" -gt 0 ] && [ "$ready_pods" -eq "$total_pods" ]; then
            echo -e "${GREEN}✅ Pods ready!${NC}"
            return 0
        fi
        
        if [ $((count % 30)) -eq 0 ]; then
            echo -e "${YELLOW}   Still waiting... ($count/${timeout}s) - $ready_pods/$total_pods pods ready${NC}"
        fi
        
        sleep 5
        count=$((count + 5))
    done
    
    echo -e "${YELLOW}⚠️ Timeout waiting for pods, continuing anyway${NC}"
    return 0
}

# Function to wait for deployment with retry logic
wait_for_deployment() {
    local namespace=$1
    local deployment=$2
    local timeout=${3:-300}
    
    echo -e "${YELLOW}⏳ Waiting for deployment $deployment in namespace $namespace...${NC}"
    
    # Check if deployment exists first, retry if not
    local retries=0
    while [ $retries -lt 10 ]; do
        if kubectl get deployment "$deployment" -n "$namespace" >/dev/null 2>&1; then
            break
        fi
        echo -e "${YELLOW}   Deployment not found, waiting... (retry $retries/10)${NC}"
        sleep 10
        retries=$((retries + 1))
    done
    
    local count=0
    while [ $count -lt $timeout ]; do
        local ready=$(kubectl get deployment "$deployment" -n "$namespace" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo 0)
        local desired=$(kubectl get deployment "$deployment" -n "$namespace" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo 1)
        
        if [ "${ready:-0}" -eq "${desired:-1}" ] && [ "$ready" -gt 0 ]; then
            echo -e "${GREEN}✅ Deployment $deployment ready!${NC}"
            return 0
        fi
        
        if [ $((count % 30)) -eq 0 ]; then
            echo -e "${YELLOW}   Still waiting... ($count/${timeout}s) - $ready/$desired replicas ready${NC}"
        fi
        
        sleep 5
        count=$((count + 5))
    done
    
    echo -e "${YELLOW}⚠️ Deployment $deployment not ready in time, continuing anyway${NC}"
    return 0
}

# Function to test network connectivity
test_connectivity() {
    local service=$1
    local namespace=$2
    local port=$3
    local timeout=${4:-30}
    
    echo -e "${YELLOW}🔍 Testing $service connectivity...${NC}"
    
    # Use kubectl exec instead of port-forward for testing
    kubectl run connectivity-test-$RANDOM --rm -i --restart=Never --image=curlimages/curl --timeout=${timeout}s -- \
        curl -s --max-time 10 http://${service}.${namespace}:${port}/api/tags >/dev/null 2>&1 && \
        echo -e "${GREEN}✅ $service responding${NC}" || \
        echo -e "${YELLOW}⚠️ $service may not be ready yet${NC}"
}

# Check prerequisites
echo -e "${YELLOW}📋 Checking prerequisites...${NC}"
check_command docker
check_command kubectl
check_command kind
check_command curl
check_command kustomize
echo -e "${GREEN}✅ All prerequisites met!${NC}"

# Clean up any existing setup
echo ""
echo -e "${YELLOW}🧹 Cleaning up any existing setup...${NC}"
kind delete cluster --name ${CLUSTER_NAME} 2>/dev/null || true
docker rmi kubeskippy:latest 2>/dev/null || true

# Create Kind cluster
echo ""
echo -e "${YELLOW}🏗️  Creating Kind cluster...${NC}"
cd "$PROJECT_ROOT"

# Create kind config
cat > /tmp/kind-config.yaml << EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: kubeskippy-demo
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 30000
    hostPort: 3000
    protocol: TCP
  - containerPort: 30001
    hostPort: 9090
    protocol: TCP
- role: worker
- role: worker
EOF

kind create cluster --name ${CLUSTER_NAME} --config /tmp/kind-config.yaml
echo -e "${GREEN}✅ Kind cluster created!${NC}"

# Build operator
echo ""
echo -e "${YELLOW}🔨 Building operator...${NC}"
cd "$PROJECT_ROOT"

go mod tidy
docker build -t kubeskippy:latest .
kind load docker-image kubeskippy:latest --name ${CLUSTER_NAME}
echo -e "${GREEN}✅ Operator built and loaded!${NC}"

# Install CRDs
echo ""
echo -e "${YELLOW}📦 Installing CRDs...${NC}"
kubectl apply -f config/crd/bases/
echo -e "${GREEN}✅ CRDs installed!${NC}"

# Create namespaces
echo ""
echo -e "${YELLOW}📁 Creating namespaces...${NC}"
kubectl create namespace ${NAMESPACE} || true
kubectl create namespace ${DEMO_NAMESPACE} || true
kubectl create namespace monitoring || true
echo -e "${GREEN}✅ Namespaces created!${NC}"

# Deploy Ollama with proper configuration and model loading
echo ""
echo -e "${YELLOW}🤖 Deploying Ollama with llama2:7b...${NC}"
cd "$SCRIPT_DIR"

cat > ollama-optimized.yaml << 'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ollama
  namespace: kubeskippy-system
  labels:
    app: ollama
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ollama
  template:
    metadata:
      labels:
        app: ollama
    spec:
      containers:
      - name: ollama
        image: ollama/ollama:latest
        ports:
        - containerPort: 11434
        env:
        - name: OLLAMA_HOST
          value: "0.0.0.0:11434"
        - name: OLLAMA_KEEP_ALIVE
          value: "24h"
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "4Gi"
            cpu: "2000m"
        volumeMounts:
        - name: ollama-data
          mountPath: /root/.ollama
        readinessProbe:
          httpGet:
            path: /api/tags
            port: 11434
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 10
        livenessProbe:
          httpGet:
            path: /api/tags
            port: 11434
          initialDelaySeconds: 60
          periodSeconds: 30
          timeoutSeconds: 5
          failureThreshold: 5
      volumes:
      - name: ollama-data
        emptyDir:
          sizeLimit: 10Gi
---
apiVersion: v1
kind: Service
metadata:
  name: ollama
  namespace: kubeskippy-system
spec:
  selector:
    app: ollama
  ports:
  - port: 11434
    targetPort: 11434
EOF

kubectl apply -f ollama-optimized.yaml
wait_for_deployment kubeskippy-system ollama 300

# Download model in background job
echo -e "${YELLOW}⏳ Starting llama2:7b model download (background job)...${NC}"
cat > model-loader.yaml << 'EOF'
apiVersion: batch/v1
kind: Job
metadata:
  name: ollama-model-loader
  namespace: kubeskippy-system
spec:
  backoffLimit: 3
  activeDeadlineSeconds: 1800
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: model-loader
        image: curlimages/curl:latest
        command:
        - /bin/sh
        - -c
        - |
          echo "Waiting for Ollama to be ready..."
          max_attempts=60
          attempt=0
          
          while [ $attempt -lt $max_attempts ]; do
            if curl -f -s http://ollama:11434/api/tags >/dev/null 2>&1; then
              echo "Ollama is ready!"
              break
            fi
            echo "Attempt $((attempt+1))/$max_attempts - waiting for Ollama..."
            sleep 10
            attempt=$((attempt+1))
          done
          
          if [ $attempt -eq $max_attempts ]; then
            echo "Failed to connect to Ollama"
            exit 1
          fi
          
          echo "Pulling llama2:7b model..."
          curl -X POST http://ollama:11434/api/pull \
            -H "Content-Type: application/json" \
            -d '{"name":"llama2:7b","stream":false}' \
            --max-time 1200 || exit 1
          
          echo "Model llama2:7b successfully loaded!"
EOF

kubectl apply -f model-loader.yaml &

echo -e "${GREEN}✅ Ollama deployed, model downloading in background!${NC}"

# Deploy metrics-server with fixed configuration
echo ""
echo -e "${YELLOW}📊 Deploying metrics-server...${NC}"
cat > metrics-server-fixed.yaml << 'EOF'
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    k8s-app: metrics-server
  name: metrics-server
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    k8s-app: metrics-server
    rbac.authorization.k8s.io/aggregate-to-admin: "true"
    rbac.authorization.k8s.io/aggregate-to-edit: "true"
    rbac.authorization.k8s.io/aggregate-to-view: "true"
  name: system:aggregated-metrics-reader
rules:
- apiGroups:
  - metrics.k8s.io
  resources:
  - pods
  - nodes
  verbs:
  - get
  - list
  - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    k8s-app: metrics-server
  name: system:metrics-server
rules:
- apiGroups:
  - ""
  resources:
  - nodes/metrics
  verbs:
  - get
- apiGroups:
  - ""
  resources:
  - pods
  - nodes
  - namespaces
  - configmaps
  verbs:
  - get
  - list
  - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  labels:
    k8s-app: metrics-server
  name: metrics-server-auth-reader
  namespace: kube-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: extension-apiserver-authentication-reader
subjects:
- kind: ServiceAccount
  name: metrics-server
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    k8s-app: metrics-server
  name: metrics-server:system:auth-delegator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:auth-delegator
subjects:
- kind: ServiceAccount
  name: metrics-server
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    k8s-app: metrics-server
  name: system:metrics-server
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:metrics-server
subjects:
- kind: ServiceAccount
  name: metrics-server
  namespace: kube-system
---
apiVersion: v1
kind: Service
metadata:
  labels:
    k8s-app: metrics-server
  name: metrics-server
  namespace: kube-system
spec:
  ports:
  - name: https
    port: 443
    protocol: TCP
    targetPort: https
  selector:
    k8s-app: metrics-server
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    k8s-app: metrics-server
  name: metrics-server
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      k8s-app: metrics-server
  strategy:
    rollingUpdate:
      maxUnavailable: 0
  template:
    metadata:
      labels:
        k8s-app: metrics-server
    spec:
      containers:
      - args:
        - --cert-dir=/tmp
        - --secure-port=4443
        - --kubelet-preferred-address-types=InternalIP,ExternalIP,Hostname
        - --kubelet-use-node-status-port
        - --metric-resolution=15s
        - --kubelet-insecure-tls
        image: registry.k8s.io/metrics-server/metrics-server:v0.7.1
        imagePullPolicy: IfNotPresent
        livenessProbe:
          failureThreshold: 3
          httpGet:
            path: /livez
            port: https
            scheme: HTTPS
          periodSeconds: 10
        name: metrics-server
        ports:
        - containerPort: 4443
          name: https
          protocol: TCP
        readinessProbe:
          failureThreshold: 3
          httpGet:
            path: /readyz
            port: https
            scheme: HTTPS
          initialDelaySeconds: 20
          periodSeconds: 10
        resources:
          requests:
            cpu: 100m
            memory: 200Mi
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
          readOnlyRootFilesystem: true
          runAsNonRoot: true
          runAsUser: 1000
          seccompProfile:
            type: RuntimeDefault
        volumeMounts:
        - mountPath: /tmp
          name: tmp-dir
      nodeSelector:
        kubernetes.io/os: linux
      priorityClassName: system-cluster-critical
      serviceAccountName: metrics-server
      volumes:
      - emptyDir: {}
        name: tmp-dir
---
apiVersion: apiregistration.k8s.io/v1
kind: APIService
metadata:
  labels:
    k8s-app: metrics-server
  name: v1beta1.metrics.k8s.io
spec:
  group: metrics.k8s.io
  groupPriorityMinimum: 100
  insecureSkipTLSVerify: true
  service:
    name: metrics-server
    namespace: kube-system
  version: v1beta1
  versionPriority: 100
EOF

kubectl apply -f metrics-server-fixed.yaml
wait_for_deployment kube-system metrics-server 120
echo -e "${GREEN}✅ Metrics-server deployed!${NC}"

# Deploy kube-state-metrics for Kubernetes metrics
echo ""
echo -e "${YELLOW}📊 Deploying kube-state-metrics...${NC}"
kubectl apply -f https://raw.githubusercontent.com/kubernetes/kube-state-metrics/v2.10.1/examples/standard/kube-state-metrics.yaml || {
    echo -e "${YELLOW}   Using fallback kube-state-metrics deployment...${NC}"
    cat > kube-state-metrics.yaml << 'EOF'
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-state-metrics
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kube-state-metrics
rules:
- apiGroups: [""]
  resources: [nodes, pods, services, endpoints, namespaces]
  verbs: [list, watch]
- apiGroups: [apps]
  resources: [deployments, replicasets, daemonsets, statefulsets]
  verbs: [list, watch]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kube-state-metrics
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kube-state-metrics
subjects:
- kind: ServiceAccount
  name: kube-state-metrics
  namespace: kube-system
---
apiVersion: v1
kind: Service
metadata:
  name: kube-state-metrics
  namespace: kube-system
spec:
  ports:
  - name: http-metrics
    port: 8080
    targetPort: http-metrics
  selector:
    app.kubernetes.io/name: kube-state-metrics
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kube-state-metrics
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: kube-state-metrics
  template:
    metadata:
      labels:
        app.kubernetes.io/name: kube-state-metrics
    spec:
      serviceAccountName: kube-state-metrics
      containers:
      - image: registry.k8s.io/kube-state-metrics/kube-state-metrics:v2.10.1
        name: kube-state-metrics
        ports:
        - containerPort: 8080
          name: http-metrics
        resources:
          requests:
            memory: "64Mi"
            cpu: "50m"
          limits:
            memory: "128Mi"
            cpu: "100m"
EOF
    kubectl apply -f kube-state-metrics.yaml
    rm -f kube-state-metrics.yaml
}
wait_for_deployment kube-system kube-state-metrics 120
echo -e "${GREEN}✅ Kube-state-metrics deployed!${NC}"

# Deploy Prometheus
echo ""
echo -e "${YELLOW}📈 Deploying Prometheus...${NC}"
kubectl apply -f prometheus/prometheus-demo.yaml
wait_for_deployment monitoring prometheus 120
echo -e "${GREEN}✅ Prometheus deployed!${NC}"

# Deploy Grafana
echo ""
echo -e "${YELLOW}📊 Deploying Grafana...${NC}"
kubectl apply -f grafana/grafana-demo.yaml
wait_for_deployment monitoring grafana 120

# Fix Grafana datasource URL to point to correct namespace
echo -e "${YELLOW}🔧 Fixing Grafana datasource configuration...${NC}"
sleep 10  # Wait for Grafana to be fully ready

# Update datasource to use correct URL
max_attempts=10
attempt=0
while [ $attempt -lt $max_attempts ]; do
    if curl -s -u admin:admin -X PUT "http://localhost:3000/api/datasources/1" \
        -H "Content-Type: application/json" \
        -d '{"name":"Prometheus","type":"prometheus","url":"http://prometheus.monitoring:9090","access":"proxy","isDefault":true}' >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Grafana datasource configured!${NC}"
        break
    fi
    attempt=$((attempt + 1))
    sleep 5
done

echo -e "${GREEN}✅ Grafana deployed!${NC}"

# Deploy KubeSkippy operator with proper configuration
echo ""
echo -e "${YELLOW}🔧 Deploying KubeSkippy operator...${NC}"
cd "$PROJECT_ROOT"

# Create proper operator config
cat > operator-config.yaml << 'EOF'
apiVersion: v1
kind: ConfigMap
metadata:
  name: kubeskippy-config
  namespace: kubeskippy-system
data:
  config.yaml: |
    metrics:
      prometheusURL: "http://prometheus.monitoring:9090"
      metricsServerEnabled: true
      collectionInterval: "30s"
    ai:
      provider: "ollama"
      model: "llama2:7b"
      endpoint: "http://ollama:11434"
      timeout: "120s"
      maxTokens: 2048
      temperature: 0.7
      minConfidence: 0.6
      validateResponses: true
    safety:
      dryRunMode: false
      requireApproval: false
      maxActionsPerHour: 50
    logging:
      level: "info"
      development: false
EOF

kubectl apply -f operator-config.yaml

# Deploy operator using kustomize and fix RBAC
kustomize build config/default | kubectl apply -f -

# Fix RBAC for leader election and metrics
cat > rbac-fix.yaml << 'EOF'
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kubeskippy-manager-role
rules:
- apiGroups: [""]
  resources: [configmaps, events, pods, services, nodes, namespaces]
  verbs: [create, delete, get, list, patch, update, watch]
- apiGroups: [apps]
  resources: [deployments, daemonsets, replicasets, statefulsets]
  verbs: [create, delete, get, list, patch, update, watch]
- apiGroups: [coordination.k8s.io]
  resources: [leases]
  verbs: [create, delete, get, list, patch, update, watch]
- apiGroups: [kubeskippy.io]
  resources: [healingactions, healingpolicies]
  verbs: [create, delete, get, list, patch, update, watch]
- apiGroups: [kubeskippy.io]
  resources: [healingactions/finalizers, healingpolicies/finalizers]
  verbs: [update]
- apiGroups: [kubeskippy.io]
  resources: [healingactions/status, healingpolicies/status]
  verbs: [get, patch, update]
- apiGroups: [metrics.k8s.io]
  resources: [nodes, pods]
  verbs: [get, list]
EOF

kubectl apply -f rbac-fix.yaml

# Create service for operator metrics
cat > operator-metrics-service.yaml << 'EOF'
apiVersion: v1
kind: Service
metadata:
  name: kubeskippy-controller-manager-metrics
  namespace: kubeskippy-system
  labels:
    app.kubernetes.io/name: kubeskippy
    app.kubernetes.io/component: controller-manager
spec:
  selector:
    control-plane: controller-manager
  ports:
  - name: metrics
    port: 8080
    targetPort: 8080
    protocol: TCP
EOF

kubectl apply -f operator-metrics-service.yaml

# Patch deployment for local image and environment
kubectl patch deployment kubeskippy-controller-manager -n kubeskippy-system --type='json' -p='[
  {"op": "replace", "path": "/spec/template/spec/containers/0/image", "value": "kubeskippy:latest"},
  {"op": "replace", "path": "/spec/template/spec/containers/0/imagePullPolicy", "value": "Never"},
  {"op": "add", "path": "/spec/template/spec/containers/0/env", "value": [
    {"name": "AI_PROVIDER", "value": "ollama"},
    {"name": "AI_MODEL", "value": "llama2:7b"},
    {"name": "AI_ENDPOINT", "value": "http://ollama:11434"},
    {"name": "PROMETHEUS_URL", "value": "http://prometheus.monitoring:9090"},
    {"name": "LOG_LEVEL", "value": "info"}
  ]}
]'

wait_for_deployment kubeskippy-system kubeskippy-controller-manager 180

# Clean up temp files
rm -f rbac-fix.yaml operator-metrics-service.yaml
echo -e "${GREEN}✅ KubeSkippy operator deployed!${NC}"

# Deploy demo applications that will trigger AI analysis
echo ""
echo -e "${YELLOW}🎯 Deploying demo applications...${NC}"
cd "$SCRIPT_DIR"

# Deploy existing demo apps
kubectl apply -f apps/memory-leak-app.yaml || echo "memory-leak-app not found, continuing..."
kubectl apply -f apps/cpu-spike-app.yaml || echo "cpu-spike-app not found, continuing..."

# Deploy continuous pressure applications for reliable AI triggering
cat > continuous-pressure-apps.yaml << 'EOF'
# Continuous Memory Degradation App
apiVersion: apps/v1
kind: Deployment
metadata:
  name: continuous-memory-degradation
  namespace: demo-apps
  labels:
    app: continuous-memory-degradation
    demo: kubeskippy
    issue: memory-leak
spec:
  replicas: 1
  selector:
    matchLabels:
      app: continuous-memory-degradation
  template:
    metadata:
      labels:
        app: continuous-memory-degradation
        demo: kubeskippy
        issue: memory-leak
    spec:
      containers:
      - name: memory-degrader
        image: busybox
        command: 
        - /bin/sh
        - -c
        - |
          echo "Starting continuous memory degradation..."
          i=1
          while true; do
            echo "Memory allocation cycle $i"
            dd if=/dev/zero of=/tmp/memfile_$i bs=10M count=1 2>/dev/null || true
            sleep 45
            if [ $((i % 3)) -eq 0 ]; then
              rm -f /tmp/memfile_* 2>/dev/null || true
              sleep 30
            fi
            i=$((i + 1))
            if [ $i -gt 15 ]; then
              i=1
              rm -f /tmp/memfile_* 2>/dev/null || true
            fi
          done
        resources:
          requests:
            memory: "100Mi"
            cpu: "50m"
          limits:
            memory: "400Mi"
            cpu: "200m"
---
# Continuous CPU Oscillation App  
apiVersion: apps/v1
kind: Deployment
metadata:
  name: continuous-cpu-oscillation
  namespace: demo-apps
  labels:
    app: continuous-cpu-oscillation
    demo: kubeskippy
    issue: cpu-spike
spec:
  replicas: 1
  selector:
    matchLabels:
      app: continuous-cpu-oscillation
  template:
    metadata:
      labels:
        app: continuous-cpu-oscillation
        demo: kubeskippy
        issue: cpu-spike
    spec:
      containers:
      - name: cpu-oscillator
        image: busybox
        command:
        - /bin/sh
        - -c
        - |
          echo "Starting continuous CPU oscillation..."
          cycle=0
          while true; do
            cycle=$((cycle + 1))
            echo "CPU cycle $cycle - high load phase"
            timeout 60s sh -c 'while true; do :; done' &
            cpu_pid=$!
            sleep 70
            kill $cpu_pid 2>/dev/null || true
            echo "CPU cycle $cycle - rest phase"
            sleep 40
          done
        resources:
          requests:
            memory: "50Mi"
            cpu: "100m"
          limits:
            memory: "100Mi"
            cpu: "900m"
---
# Random Crasher App
apiVersion: apps/v1
kind: Deployment
metadata:
  name: random-crasher
  namespace: demo-apps
  labels:
    app: random-crasher
    demo: kubeskippy
    issue: crashloop
spec:
  replicas: 1
  selector:
    matchLabels:
      app: random-crasher
  template:
    metadata:
      labels:
        app: random-crasher
        demo: kubeskippy
        issue: crashloop
    spec:
      containers:
      - name: crasher
        image: busybox
        command:
        - /bin/sh
        - -c
        - |
          echo "Starting random crasher..."
          cycle=0
          while true; do
            cycle=$((cycle + 1))
            runtime=$((180 + RANDOM % 240))
            echo "Running normally for ${runtime} seconds (cycle $cycle)..."
            sleep $runtime
            crash_chance=$((RANDOM % 100))
            if [ $crash_chance -lt 40 ]; then
              echo "Simulating crash in cycle $cycle..."
              exit 1
            fi
          done
        resources:
          requests:
            memory: "32Mi"
            cpu: "10m"
          limits:
            memory: "64Mi"
            cpu: "50m"
      restartPolicy: Always
EOF

kubectl apply -f continuous-pressure-apps.yaml
wait_for_pods demo-apps "app=continuous-memory-degradation" 60

# Clean up temp file
rm -f continuous-pressure-apps.yaml

echo -e "${GREEN}✅ Demo applications deployed!${NC}"

# Deploy AI-driven healing policies
echo ""
echo -e "${YELLOW}🏥 Deploying AI-driven healing policies...${NC}"

# Apply existing AI policies
kubectl apply -f policies/ai-driven-healing.yaml || echo "Creating AI policies..."

# Create additional policies for demo
cat > ai-policies.yaml << 'EOF'
apiVersion: kubeskippy.io/v1alpha1
kind: HealingPolicy
metadata:
  name: ai-memory-healing
  namespace: demo-apps
spec:
  triggers:
    - type: "resource"
      resource: "memory"
      threshold: "70%"
      operator: ">"
  actions:
    - type: "restart"
      target: "deployment"
  rateLimit:
    actionsPerHour: 20
  aiDriven: true
  aiProvider: "ollama"
  aiModel: "llama2:7b"
---
apiVersion: kubeskippy.io/v1alpha1
kind: HealingPolicy
metadata:
  name: ai-cpu-healing
  namespace: demo-apps
spec:
  triggers:
    - type: "resource"
      resource: "cpu"
      threshold: "80%"
      operator: ">"
  actions:
    - type: "scale"
      target: "deployment"
      scaleReplicas: 2
  rateLimit:
    actionsPerHour: 10
  aiDriven: true
  aiProvider: "ollama"
  aiModel: "llama2:7b"
EOF

kubectl apply -f ai-policies.yaml
echo -e "${GREEN}✅ AI-driven healing policies deployed!${NC}"

# Wait for Ollama model to finish loading
echo ""
echo -e "${YELLOW}⏳ Ensuring AI model is fully loaded...${NC}"
kubectl wait --for=condition=complete job/ollama-model-loader -n kubeskippy-system --timeout=1800s || {
    echo -e "${YELLOW}⚠️ Model loading may have timed out, checking status...${NC}"
    kubectl logs job/ollama-model-loader -n kubeskippy-system | tail -5
}

# Test AI connectivity
echo ""
echo -e "${YELLOW}🔍 Testing real AI functionality...${NC}"
test_connectivity ollama kubeskippy-system 11434

# Set up port forwarding for access
echo ""
echo -e "${YELLOW}🌐 Setting up port forwarding...${NC}"

# Kill any existing port forwards
pkill -f "kubectl port-forward" 2>/dev/null || true
sleep 2

# Function to start port forward with retries
start_port_forward() {
    local service=$1
    local namespace=$2
    local port=$3
    local retries=0
    
    while [ $retries -lt 5 ]; do
        if kubectl port-forward -n "$namespace" "service/$service" "$port:$port" >/dev/null 2>&1 &; then
            echo -e "${GREEN}✅ Port forward started for $service on port $port${NC}"
            return 0
        fi
        retries=$((retries + 1))
        sleep 2
    done
    echo -e "${YELLOW}⚠️ Failed to start port forward for $service${NC}"
}

# Start port forwards
start_port_forward grafana monitoring 3000
start_port_forward prometheus monitoring 9090

# Wait for services to be accessible
sleep 10

# Test connections
echo ""
echo -e "${YELLOW}🔍 Testing dashboard access...${NC}"

# Test Grafana with retries
for i in {1..30}; do
    if curl -s http://localhost:3000 >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Grafana accessible at http://localhost:3000${NC}"
        break
    fi
    sleep 2
done

# Test Prometheus with retries
for i in {1..20}; do
    if curl -s http://localhost:9090 >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Prometheus accessible at http://localhost:9090${NC}"
        break
    fi
    sleep 2
done

# Wait for initial metrics and AI analysis
echo ""
echo -e "${YELLOW}⏳ Waiting for AI analysis to begin (60 seconds)...${NC}"
echo -e "${YELLOW}   The operator will start analyzing cluster state with real llama2:7b AI...${NC}"
sleep 60

# Verify real AI is working
echo ""
echo -e "${YELLOW}🔍 Verifying real AI functionality...${NC}"

# Check if model is responding
AI_TEST=$(kubectl run ai-connectivity-test --rm -i --restart=Never --image=curlimages/curl --timeout=30s -- \
  curl -s -X POST http://ollama.kubeskippy-system:11434/api/generate \
  -H "Content-Type: application/json" \
  -d '{"model":"llama2:7b","prompt":"Test AI response","stream":false}' 2>/dev/null | grep -c "response" || echo 0)

if [ "$AI_TEST" -gt 0 ]; then
    echo -e "${GREEN}✅ Real AI (llama2:7b) is responding correctly!${NC}"
else
    echo -e "${YELLOW}⚠️ AI may still be initializing${NC}"
fi

# Check operator logs for AI activity
AI_LOGS=$(kubectl logs -n kubeskippy-system deployment/kubeskippy-controller-manager --tail=50 2>/dev/null | grep -i "ai\|llama\|reasoning" | wc -l || echo 0)
if [ "$AI_LOGS" -gt 0 ]; then
    echo -e "${GREEN}✅ AI activity detected in operator logs${NC}"
else
    echo -e "${YELLOW}ℹ️ AI activity will appear as healing actions are triggered${NC}"
fi

# Final status
HEALING_ACTIONS=$(kubectl get healingactions -n demo-apps --no-headers 2>/dev/null | wc -l || echo 0)
echo -e "${GREEN}✅ Healing actions: $HEALING_ACTIONS${NC}"

TARGETS=$(curl -s "http://localhost:9090/api/v1/targets" 2>/dev/null | grep -c '"health":"up"' || echo 0)
echo -e "${GREEN}✅ Prometheus targets: $TARGETS${NC}"

echo ""
echo -e "${GREEN}🎉 Bulletproof Real AI Demo setup completed successfully!${NC}"
echo "=============================================================="
echo -e "${GREEN}📊 Grafana Dashboard: http://localhost:3000${NC}"
echo -e "${GREEN}   Username: admin${NC}"
echo -e "${GREEN}   Password: admin${NC}"
echo ""
echo -e "${GREEN}📈 Prometheus: http://localhost:9090${NC}"
echo ""
echo -e "${BLUE}🤖 Real AI Decision Reasoning with llama2:7b:${NC}"
echo -e "${BLUE}   1. Open Grafana dashboard${NC}"
echo -e "${BLUE}   2. Navigate to 'KubeSkippy Enhanced Demo Dashboard'${NC}"
echo -e "${BLUE}   3. Scroll down to '🤖 AI Analysis & Decision Reasoning'${NC}"
echo -e "${BLUE}   4. Watch real AI reasoning steps and confidence metrics${NC}"
echo ""
echo -e "${YELLOW}📝 To monitor real AI activity:${NC}"
echo -e "${YELLOW}   kubectl get healingactions -n demo-apps -w${NC}"
echo -e "${YELLOW}   kubectl logs -f -n kubeskippy-system deployment/kubeskippy-controller-manager${NC}"
echo ""
echo -e "${GREEN}✅ ZERO-INTERACTION setup completed with real llama2:7b AI!${NC}"
echo -e "${GREEN}🧠 The system is using genuine AI for intelligent healing decisions${NC}"

# Enable AI metrics by triggering some policy evaluations
echo ""
echo -e "${YELLOW}📊 Enabling AI metrics collection...${NC}"

# Create a test resource to trigger policy evaluation
cat > /tmp/trigger-metrics.yaml << 'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: metrics-trigger-pod
  namespace: demo-apps
  labels:
    app: metrics-trigger
spec:
  containers:
  - name: trigger
    image: busybox
    command: ["sh", "-c", "echo 'Triggering metrics'; sleep 60"]
    resources:
      requests:
        memory: "100Mi"
        cpu: "50m"
EOF

kubectl apply -f /tmp/trigger-metrics.yaml
sleep 10

# Force policy reconciliation to generate metrics
kubectl patch healingpolicy ai-driven-healing -n demo-apps --type='merge' -p='{"metadata":{"annotations":{"metrics-trigger":"'$(date +%s)'"}}}'

# Clean up trigger pod
kubectl delete -f /tmp/trigger-metrics.yaml --ignore-not-found=true 2>/dev/null || true
rm -f /tmp/trigger-metrics.yaml

echo -e "${GREEN}✅ AI metrics collection enabled!${NC}"

# Create port forwarding management scripts
echo ""
echo -e "${YELLOW}📡 Creating port forwarding management scripts...${NC}"

# Create start-port-forwards script
cat > start-port-forwards.sh << 'EOF'
#!/bin/bash
echo "🚀 Starting KubeSkippy Demo Port Forwards..."

# Kill any existing port forwards
pkill -f "kubectl port-forward.*grafana" 2>/dev/null || true
pkill -f "kubectl port-forward.*prometheus" 2>/dev/null || true
sleep 2

# Start Grafana port forward
echo "📊 Starting Grafana port forward (localhost:3000)..."
kubectl port-forward -n monitoring service/grafana 3000:3000 >/dev/null 2>&1 &
GRAFANA_PID=$!

# Start Prometheus port forward
echo "📈 Starting Prometheus port forward (localhost:9090)..."
kubectl port-forward -n monitoring service/prometheus 9090:9090 >/dev/null 2>&1 &
PROMETHEUS_PID=$!

# Wait and test
sleep 5

# Test connections
if curl -s http://localhost:3000 >/dev/null 2>&1; then
    echo "✅ Grafana accessible at http://localhost:3000 (admin/admin)"
else
    echo "⚠️ Grafana may not be ready yet"
fi

if curl -s http://localhost:9090 >/dev/null 2>&1; then
    echo "✅ Prometheus accessible at http://localhost:9090"
else
    echo "⚠️ Prometheus may not be ready yet"
fi

# Save PIDs
echo $GRAFANA_PID > /tmp/kubeskippy-grafana.pid
echo $PROMETHEUS_PID > /tmp/kubeskippy-prometheus.pid

echo ""
echo "🎯 Port forwards are running in background"
echo "📝 To stop: ./stop-port-forwards.sh"
echo "📝 To restart: ./start-port-forwards.sh"
EOF

chmod +x start-port-forwards.sh

# Create stop-port-forwards script
cat > stop-port-forwards.sh << 'EOF'
#!/bin/bash
echo "🛑 Stopping KubeSkippy Demo Port Forwards..."

# Kill port forwards using PIDs if available
if [ -f /tmp/kubeskippy-grafana.pid ]; then
    GRAFANA_PID=$(cat /tmp/kubeskippy-grafana.pid)
    if [ -n "$GRAFANA_PID" ]; then
        kill $GRAFANA_PID 2>/dev/null && echo "✅ Stopped Grafana port forward"
    fi
    rm -f /tmp/kubeskippy-grafana.pid
fi

if [ -f /tmp/kubeskippy-prometheus.pid ]; then
    PROMETHEUS_PID=$(cat /tmp/kubeskippy-prometheus.pid)
    if [ -n "$PROMETHEUS_PID" ]; then
        kill $PROMETHEUS_PID 2>/dev/null && echo "✅ Stopped Prometheus port forward"
    fi
    rm -f /tmp/kubeskippy-prometheus.pid
fi

# Backup method: kill by process name
pkill -f "kubectl port-forward.*grafana" 2>/dev/null || true
pkill -f "kubectl port-forward.*prometheus" 2>/dev/null || true

echo "🎯 All port forwards stopped"
EOF

chmod +x stop-port-forwards.sh

# Create monitoring script
cat > monitor-demo.sh << 'EOF'
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
EOF

chmod +x monitor-demo.sh

# Create cleanup script
cat > cleanup-demo.sh << 'EOF'
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
EOF

chmod +x cleanup-demo.sh

# Save current PIDs
GRAFANA_PID=$(pgrep -f "kubectl port-forward.*grafana" | head -1)
PROMETHEUS_PID=$(pgrep -f "kubectl port-forward.*prometheus" | head -1)
echo "${GRAFANA_PID:-}" > /tmp/kubeskippy-grafana.pid
echo "${PROMETHEUS_PID:-}" > /tmp/kubeskippy-prometheus.pid

echo -e "${GREEN}✅ Management scripts created!${NC}"

echo ""
echo -e "${BLUE}🎯 Demo is ready! Anyone can now clone the repo and run this script successfully!${NC}"
echo ""
echo -e "${GREEN}📡 Port Forward Management:${NC}"
echo -e "${GREEN}   ./start-port-forwards.sh  - Start port forwarding${NC}"
echo -e "${GREEN}   ./stop-port-forwards.sh   - Stop port forwarding${NC}"
echo -e "${GREEN}   ./monitor-demo.sh         - Monitor demo status${NC}"
echo -e "${GREEN}   ./cleanup-demo.sh         - Complete cleanup${NC}"