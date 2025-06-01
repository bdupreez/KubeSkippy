#!/bin/bash
set -e

echo "🤖 Starting Continuous AI Demo - Generating Resource Pressure"
echo "============================================================="
echo "🎯 This script creates continuous resource pressure to trigger AI analysis"
echo "📊 Watch the Grafana dashboard populate with real AI reasoning data"
echo ""

# Check if cluster is ready
if ! kubectl cluster-info >/dev/null 2>&1; then
    echo "❌ Kubernetes cluster not accessible. Run ./bulletproof-ai-setup.sh first"
    exit 1
fi

# Check if KubeSkippy is deployed
if ! kubectl get deployment kubeskippy-controller-manager -n kubeskippy-system >/dev/null 2>&1; then
    echo "❌ KubeSkippy operator not found. Run ./bulletproof-ai-setup.sh first"
    exit 1
fi

echo "✅ Prerequisites met - starting continuous demo"
echo ""

# Deploy continuous pressure applications
echo "🚀 Deploying continuous pressure applications..."

cat > continuous-pressure-apps.yaml << 'EOF'
# Continuous Memory Degradation App
apiVersion: apps/v1
kind: Deployment
metadata:
  name: continuous-memory-degradation
  namespace: demo-apps
  labels:
    app: continuous-memory-degradation
    demo-type: continuous-pressure
spec:
  replicas: 1
  selector:
    matchLabels:
      app: continuous-memory-degradation
  template:
    metadata:
      labels:
        app: continuous-memory-degradation
    spec:
      containers:
      - name: memory-degrader
        image: busybox
        command: 
        - /bin/sh
        - -c
        - |
          echo "Starting continuous memory degradation..."
          # Start with small allocation, gradually increase
          i=1
          while true; do
            echo "Memory allocation cycle $i - allocating memory..."
            
            # Create memory pressure by allocating and holding memory
            dd if=/dev/zero of=/tmp/memfile_$i bs=10M count=1 2>/dev/null || true
            
            # Hold for 30 seconds to trigger policy evaluation
            sleep 30
            
            # Occasionally clean up to create cycles
            if [ $((i % 4)) -eq 0 ]; then
              echo "Cleaning up memory files..."
              rm -f /tmp/memfile_* 2>/dev/null || true
              sleep 10
            fi
            
            i=$((i + 1))
            
            # Reset counter to prevent infinite growth
            if [ $i -gt 20 ]; then
              i=1
              rm -f /tmp/memfile_* 2>/dev/null || true
            fi
          done
        resources:
          requests:
            memory: "100Mi"
            cpu: "50m"
          limits:
            memory: "500Mi"
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
    demo-type: continuous-pressure
spec:
  replicas: 1
  selector:
    matchLabels:
      app: continuous-cpu-oscillation
  template:
    metadata:
      labels:
        app: continuous-cpu-oscillation
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
            echo "CPU cycle $cycle - creating CPU pressure..."
            
            # High CPU phase - stress the CPU for 45 seconds
            echo "High CPU phase..."
            timeout 45s sh -c 'while true; do :; done' &
            cpu_pid=$!
            
            # Wait for pressure to build and trigger policy evaluation
            sleep 50
            
            # Kill CPU stress
            kill $cpu_pid 2>/dev/null || true
            
            # Low CPU phase - rest for 30 seconds  
            echo "Low CPU phase..."
            sleep 30
            
            # Medium CPU phase
            echo "Medium CPU phase..."
            timeout 20s sh -c 'for i in $(seq 1 1000000); do echo $i > /dev/null; done' &
            sleep 25
            
            echo "Completed cycle $cycle"
          done
        resources:
          requests:
            memory: "50Mi"
            cpu: "100m"
          limits:
            memory: "100Mi"
            cpu: "800m"
---
# Flaky Network App
apiVersion: apps/v1
kind: Deployment
metadata:
  name: flaky-network-app
  namespace: demo-apps
  labels:
    app: flaky-network-app
    demo-type: continuous-pressure
spec:
  replicas: 2
  selector:
    matchLabels:
      app: flaky-network-app
  template:
    metadata:
      labels:
        app: flaky-network-app
    spec:
      containers:
      - name: flaky-service
        image: nginx:alpine
        ports:
        - containerPort: 80
        command:
        - /bin/sh
        - -c
        - |
          echo "Starting flaky network service..."
          # Start nginx in background
          nginx -g 'daemon off;' &
          nginx_pid=$!
          
          cycle=0
          while true; do
            cycle=$((cycle + 1))
            echo "Network flaky cycle $cycle"
            
            # Normal operation for 60 seconds
            echo "Normal operation phase..."
            sleep 60
            
            # Simulate network issues by stopping nginx
            echo "Simulating network failure..."
            kill $nginx_pid 2>/dev/null || true
            sleep 20
            
            # Restart nginx
            echo "Recovering from network failure..."
            nginx -g 'daemon off;' &
            nginx_pid=$!
            sleep 10
            
            echo "Completed flaky cycle $cycle"
          done
        resources:
          requests:
            memory: "32Mi"
            cpu: "10m"
          limits:
            memory: "64Mi"
            cpu: "100m"
---
apiVersion: v1
kind: Service
metadata:
  name: flaky-network-app
  namespace: demo-apps
spec:
  selector:
    app: flaky-network-app
  ports:
  - port: 80
    targetPort: 80
---
# Random Crasher App
apiVersion: apps/v1
kind: Deployment
metadata:
  name: random-crasher
  namespace: demo-apps
  labels:
    app: random-crasher
    demo-type: continuous-pressure
spec:
  replicas: 1
  selector:
    matchLabels:
      app: random-crasher
  template:
    metadata:
      labels:
        app: random-crasher
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
            
            # Run normally for a random time (2-8 minutes)
            runtime=$((120 + RANDOM % 360))
            echo "Running normally for ${runtime} seconds (cycle $cycle)..."
            sleep $runtime
            
            # Decide randomly whether to crash or continue
            crash_chance=$((RANDOM % 100))
            if [ $crash_chance -lt 30 ]; then
              echo "Simulating crash in cycle $cycle..."
              exit 1
            else
              echo "Continuing cycle $cycle..."
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

echo "✅ Continuous pressure applications deployed!"
echo ""

# Update continuous apps to have labels that match the existing AI policy
echo "🏷️ Updating continuous apps to match existing AI policy..."

# Update apps to have the correct labels for the existing ai-driven-healing policy
kubectl patch deployment continuous-memory-degradation -n demo-apps --type='merge' -p='
{
  "metadata": {
    "labels": {"demo": "kubeskippy", "issue": "memory-leak"}
  },
  "spec": {
    "template": {
      "metadata": {
        "labels": {"demo": "kubeskippy", "issue": "memory-leak"}
      }
    }
  }
}'

kubectl patch deployment continuous-cpu-oscillation -n demo-apps --type='merge' -p='
{
  "metadata": {
    "labels": {"demo": "kubeskippy", "issue": "cpu-spike"}
  },
  "spec": {
    "template": {
      "metadata": {
        "labels": {"demo": "kubeskippy", "issue": "cpu-spike"}
      }
    }
  }
}'

kubectl patch deployment random-crasher -n demo-apps --type='merge' -p='
{
  "metadata": {
    "labels": {"demo": "kubeskippy", "issue": "crashloop"}
  },
  "spec": {
    "template": {
      "metadata": {
        "labels": {"demo": "kubeskippy", "issue": "crashloop"}
      }
    }
  }
}'

kubectl patch deployment flaky-network-app -n demo-apps --type='merge' -p='
{
  "metadata": {
    "labels": {"demo": "kubeskippy", "issue": "service-degradation"}
  },
  "spec": {
    "template": {
      "metadata": {
        "labels": {"demo": "kubeskippy", "issue": "service-degradation"}
      }
    }
  }
}'

echo "✅ Continuous healing policies deployed!"
echo ""

# Wait for applications to start and create pressure
echo "⏳ Waiting for applications to start creating pressure..."
sleep 30

# Monitor and provide status
echo "📊 Continuous AI Demo Status:"
echo "=============================="
echo ""
echo "🎯 Applications generating pressure:"
kubectl get pods -n demo-apps -l demo-type=continuous-pressure --no-headers

echo ""
echo "🏥 Active healing policies:"
kubectl get healingpolicies -n demo-apps --no-headers

echo ""
echo "⚡ Recent healing actions (will increase over time):"
kubectl get healingactions -n demo-apps --no-headers 2>/dev/null | tail -5 || echo "No actions yet - will appear as pressure builds"

echo ""
echo "🤖 AI Analysis Activity:"
echo "========================"
echo "📈 Monitor AI reasoning metrics at: http://localhost:3000"
echo "📊 Check Prometheus metrics at: http://localhost:9090"
echo ""
echo "🔍 Live monitoring commands:"
echo "  Watch healing actions: kubectl get healingactions -n demo-apps -w"
echo "  Monitor operator logs:  kubectl logs -f -n kubeskippy-system deployment/kubeskippy-controller-manager"
echo "  Check app status:      kubectl get pods -n demo-apps -w"
echo ""
echo "📝 Expected behavior:"
echo "  • Memory degradation app will gradually consume memory"
echo "  • CPU oscillation app will create CPU pressure cycles"  
echo "  • Flaky network app will simulate service interruptions"
echo "  • Random crasher will occasionally crash and restart"
echo "  • KubeSkippy AI will analyze and create healing actions"
echo "  • Grafana dashboard will populate with AI reasoning data"
echo ""
echo "✅ Continuous AI demo is now running!"
echo "🎯 Check the Grafana dashboard in 2-3 minutes to see AI reasoning data"
echo ""
echo "🛑 To stop the demo: kubectl delete -f continuous-pressure-apps.yaml && kubectl delete -f continuous-healing-policies.yaml"

# Save cleanup files for later
mv continuous-pressure-apps.yaml /tmp/
mv continuous-healing-policies.yaml /tmp/

echo ""
echo "🔄 Demo will run continuously until stopped"