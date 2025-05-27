#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🎮 KubeSkippy Interactive Demo${NC}"
echo "================================"
echo ""

# Function to wait for user
wait_for_user() {
    echo ""
    echo -e "${YELLOW}Press Enter to continue...${NC}"
    read
}

# Function to run command with output
run_command() {
    echo -e "${GREEN}$ $1${NC}"
    eval $1
}

# Introduction
echo -e "${BLUE}Welcome to the KubeSkippy demo!${NC}"
echo "This demo will show you how KubeSkippy automatically heals problematic applications."
echo ""
echo "Prerequisites:"
echo "- Kind cluster is running (make demo-up)"
echo "- KubeSkippy operator is deployed"
echo "- Demo apps and policies are deployed"
wait_for_user

# Show deployed applications
echo -e "${BLUE}📦 Demo Applications${NC}"
echo "Let's look at our problematic applications:"
echo ""
run_command "kubectl get deployments -n demo-apps"
echo ""
echo "These apps have various issues:"
echo "- crashloop-app: Crashes every 30 seconds"
echo "- memory-leak-app: Gradually consumes more memory"
echo "- cpu-spike-app: Has periodic CPU spikes"
echo "- flaky-web-app: Returns random errors"
wait_for_user

# Show healing policies
echo -e "${BLUE}🏥 Healing Policies${NC}"
echo "Now let's see the healing policies that will fix these issues:"
echo ""
run_command "kubectl get healingpolicies -n demo-apps"
wait_for_user

# Demo 1: CrashLoop Recovery
echo -e "${BLUE}Demo 1: Automatic CrashLoop Recovery${NC}"
echo "Watch the crashloop-app pods:"
echo ""
run_command "kubectl get pods -n demo-apps -l app=crashloop-app"
echo ""
echo "One of these pods will crash soon. Let's watch for healing actions:"
echo "(This may take 1-2 minutes for the crash pattern to be detected)"
echo ""
echo -e "${GREEN}$ kubectl get healingactions -n demo-apps -w | grep crashloop${NC}"
kubectl get healingactions -n demo-apps -w | grep crashloop &
WATCH_PID=$!

echo ""
echo "Waiting for healing action..."
sleep 30

kill $WATCH_PID 2>/dev/null || true
echo ""
echo "Let's check if a healing action was created:"
run_command "kubectl get healingactions -n demo-apps | grep crashloop || echo 'No actions yet, policy may need more time'"
wait_for_user

# Demo 2: Memory Leak Mitigation
echo -e "${BLUE}Demo 2: Memory Leak Mitigation${NC}"
echo "Let's check the memory usage of our leaky app:"
echo ""
run_command "kubectl top pods -n demo-apps | grep memory-leak || echo 'Metrics not available yet'"
echo ""
echo "The healing policy will restart pods when memory exceeds 85%"
echo "You can simulate high memory by scaling the deployment:"
echo ""
run_command "kubectl scale deployment memory-leak-app -n demo-apps --replicas=1"
wait_for_user

# Demo 3: AI-Driven Analysis
echo -e "${BLUE}Demo 3: AI-Driven Analysis${NC}"
echo "The AI healing policy analyzes complex issues:"
echo ""
run_command "kubectl describe healingpolicy ai-driven-healing -n demo-apps | grep -A 10 'Spec:'"
echo ""
echo "AI recommendations start in dry-run mode for safety"
echo "Let's check if any AI analysis has been triggered:"
echo ""
run_command "kubectl get healingactions -n demo-apps -o wide | grep -E 'ai|NAME' || echo 'No AI actions yet'"
wait_for_user

# Show operator logs
echo -e "${BLUE}📊 Operator Activity${NC}"
echo "Let's see what the operator is doing:"
echo ""
echo -e "${GREEN}$ kubectl logs -n kubeskippy-system deployment/kubeskippy-controller-manager --tail=20${NC}"
kubectl logs -n kubeskippy-system deployment/kubeskippy-controller-manager --tail=20 2>/dev/null || echo "Operator logs not available yet"
wait_for_user

# Manual trigger demo
echo -e "${BLUE}Demo 4: Manual Healing Action${NC}"
echo "You can also trigger healing actions manually:"
echo ""
cat << 'EOF'
kubectl create -f - <<YAML
apiVersion: kubeskippy.io/v1alpha1
kind: HealingAction
metadata:
  name: manual-restart-demo
  namespace: demo-apps
spec:
  policyName: manual
  target:
    apiVersion: v1
    kind: Pod
    labelSelector:
      matchLabels:
        app: flaky-web-app
  action:
    type: restart
    restartAction:
      strategy: rolling
YAML
EOF
echo ""
echo "Would you like to create this manual healing action? (y/N)"
read -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    kubectl create -f - <<EOF
apiVersion: kubeskippy.io/v1alpha1
kind: HealingAction
metadata:
  name: manual-restart-demo-$(date +%s)
  namespace: demo-apps
spec:
  policyName: manual
  target:
    apiVersion: v1
    kind: Pod
    labelSelector:
      matchLabels:
        app: flaky-web-app
  action:
    type: restart
    restartAction:
      strategy: rolling
EOF
    echo "✅ Manual healing action created!"
fi
wait_for_user

# Summary
echo -e "${BLUE}🎉 Demo Complete!${NC}"
echo ""
echo "What we've seen:"
echo "✅ Automatic detection of pod crashes"
echo "✅ Memory leak mitigation"
echo "✅ AI-driven analysis capabilities"
echo "✅ Manual healing triggers"
echo ""
echo "Useful commands to explore further:"
echo "- Watch all healing actions: kubectl get healingactions -n demo-apps -w"
echo "- View operator logs: kubectl logs -n kubeskippy-system deployment/kubeskippy-controller-manager -f"
echo "- Describe a policy: kubectl describe healingpolicy <name> -n demo-apps"
echo "- Check metrics: kubectl top pods -n demo-apps"
echo ""
echo "To reset the demo: make demo-reset"
echo "To clean up: make demo-down"
echo ""
echo -e "${GREEN}Thank you for trying KubeSkippy!${NC}"