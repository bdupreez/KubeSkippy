#!/bin/bash

echo "🔍 KubeSkippy Demo Check"
echo "========================"

echo -e "\n💡 Useful Commands:"
echo "────────────────────"
echo "• Watch healing actions:    kubectl get healingactions -n demo-apps -w"
echo "• Monitor operator logs:    kubectl logs -n kubeskippy-system deployment/kubeskippy-controller-manager -f"
echo "• Run demo monitor:         ./monitor.sh"
echo "• Check resource usage:     kubectl top pods -n demo-apps"

echo -e "\n📊 Current Status:"
echo "=================="
echo -e "\nPods with issues:"
kubectl get pods -n demo-apps | grep -E "(CrashLoop|Error|0/1)"

echo -e "\nHigh resource usage:"
kubectl top pods -n demo-apps | awk 'NR==1 || $2 ~ /[0-9]+m/ && $2+0 > 500'

echo -e "\nHealing actions:"
kubectl get healingactions -n demo-apps 2>/dev/null || echo "No healing actions yet"

echo -e "\nOperator status:"
operator_pod=$(kubectl get pods -n kubeskippy-system -l control-plane=controller-manager --no-headers | awk '{print $3}')
echo "Operator is: $operator_pod"

echo -e "\n🎯 AI-Driven Healing Mode:"
mode=$(kubectl get healingpolicy ai-driven-healing -n demo-apps -o jsonpath='{.spec.mode}' 2>/dev/null || echo "Not found")
echo "Current mode: $mode"
if [[ "$mode" == "dryrun" ]]; then
    echo -e "\n💡 To enable AI-driven healing:"
    echo "   kubectl patch healingpolicy ai-driven-healing -n demo-apps --type merge -p '{\"spec\":{\"mode\":\"automatic\"}}'"
fi