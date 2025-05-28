# KubeSkippy - Kubernetes Self-Healing Operator

An intelligent Kubernetes operator that automatically detects, diagnoses, and heals application issues using configurable policies and optional AI-powered analysis.

## 🚀 Current Status

**Version**: 0.1.0 (Alpha)  
**Status**: Functional with core features implemented  
**Last Updated**: January 2025

### ✅ Implemented Features

- **Core Operator Framework**
  - [x] Custom Resource Definitions (HealingPolicy, HealingAction)
  - [x] Policy-based healing with flexible triggers
  - [x] Multiple remediation actions (restart, scale, patch, delete)
  - [x] Safety controls and rate limiting
  - [x] Comprehensive event auditing

- **Healing Capabilities**
  - [x] **Metric-based triggers**: CPU, memory, restart counts, error rates
  - [x] **Event-based triggers**: Kubernetes events and conditions
  - [x] **Condition-based triggers**: Pod status like CrashLoopBackOff
  - [x] **Multiple action types**: Rolling restarts, horizontal scaling, configuration patches
  - [x] **Safety mechanisms**: Rate limiting, cooldown periods, protected resources

- **AI Integration**
  - [x] Ollama integration for local LLM inference
  - [x] OpenAI API support
  - [x] Intelligent root cause analysis
  - [x] Confidence-based recommendations
  - [x] Safety validation of AI suggestions

- **Production Features**
  - [x] Dry-run mode for testing policies
  - [x] Prometheus-compatible metrics with PromQL support
  - [x] Structured logging
  - [x] RBAC and security controls
  - [x] Helm chart for deployment
  - [x] Comprehensive test coverage with E2E tests

### 🎯 Demo & Testing

A comprehensive demo environment showcases all healing capabilities:

```bash
# Start the demo
cd demo
./setup.sh

# Watch healing in action
./monitor.sh

# Quick status check
./quick-demo.sh
```

The demo includes:
- **Crashloop healing**: Automatically fixes crashing pods
- **Memory leak mitigation**: Restarts pods with high memory usage
- **Service degradation handling**: Scales up based on error rates
- **AI-driven analysis**: Intelligent recommendations (optional)

## 📋 Key Features

### 1. **Policy-Based Healing**
Define what to monitor and how to respond:

```yaml
apiVersion: kubeskippy.io/v1alpha1
kind: HealingPolicy
metadata:
  name: memory-leak-healing
spec:
  selector:
    labelSelector:
      matchLabels:
        app: my-app
  triggers:
  - name: high-memory
    type: metric
    metricTrigger:
      query: "memory_usage_percent"
      threshold: 85
      operator: ">"
  actions:
  - name: restart-pod
    type: restart
    priority: 10
```

### 2. **Multiple Trigger Types**
- **Metrics**: CPU, memory, custom Prometheus queries
- **Events**: Kubernetes events (warnings, errors)
- **Conditions**: Pod/node conditions
- **AI-based**: Pattern recognition and anomaly detection

### 3. **Safe Remediation Actions**
- **Restart**: Rolling restart with configurable strategy
- **Scale**: Horizontal scaling up/down
- **Patch**: Apply configuration changes
- **Delete**: Remove and recreate resources

### 4. **AI-Powered Intelligence** (Optional)
- Local inference using Ollama (privacy-focused)
- Cloud inference using OpenAI API
- Root cause analysis and recommendations
- Learning from historical patterns

### 5. **Enterprise-Ready**
- Rate limiting prevents action storms
- Cooldown periods prevent flapping
- Audit trail for compliance
- Metrics for monitoring effectiveness

## 🛠️ Installation

### Prerequisites
- Kubernetes 1.26+
- Helm 3.0+
- Metrics Server installed
- (Optional) Ollama for AI features

### Quick Install

```bash
# Add the Helm repository
helm repo add kubeskippy https://kubeskippy.github.io/charts
helm repo update

# Install the operator
helm install kubeskippy kubeskippy/kubeskippy \
  --namespace kubeskippy-system \
  --create-namespace

# Verify installation
kubectl get pods -n kubeskippy-system
```

### Install with AI Support

```bash
# Deploy Ollama first
kubectl apply -f https://raw.githubusercontent.com/kubeskippy/kubeskippy/main/deploy/ollama.yaml

# Install operator with AI enabled
helm install kubeskippy kubeskippy/kubeskippy \
  --namespace kubeskippy-system \
  --create-namespace \
  --set ai.enabled=true \
  --set ai.provider=ollama
```

## 📖 Documentation

- [**Quick Start Guide**](QUICKSTART.md) - Get up and running in 5 minutes
- [**Demo Walkthrough**](demo/README.md) - See all features in action
- [**Architecture Overview**](docs/architecture-overview.md) - Technical deep dive
- [**AI-Driven Healing**](docs/ai-driven-healing-explained.md) - How AI integration works
- [**Prometheus Integration**](docs/prometheus-integration-guide.md) - Metrics and monitoring setup
- [**Grafana Dashboard Guide**](docs/grafana-dashboard-guide.md) - Visual monitoring with Grafana
- [**Operator Design**](OPERATOR_README.md) - Implementation details

## 🔄 How It Works

1. **Define** healing policies for your applications
2. **Deploy** policies alongside your workloads
3. **Monitor** as KubeSkippy watches for issues
4. **Heal** automatically when triggers fire
5. **Learn** from actions to improve over time

## 🚧 Roadmap

### Near Term (Q1 2024)
- [ ] Prometheus metrics integration
- [ ] Custom webhook actions
- [ ] Multi-cluster support
- [ ] Policy templates library

### Medium Term (Q2 2024)
- [ ] Predictive healing (fix before failure)
- [ ] Cost-aware actions
- [ ] Integration with PagerDuty/Slack
- [ ] Fine-tuned AI models

### Long Term
- [ ] Service mesh integration
- [ ] Chaos engineering integration
- [ ] Self-optimizing policies
- [ ] Kubernetes operator SDK v2

## 🤝 Contributing

We welcome contributions! See our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/yourusername/kubeskippy.git
cd kubeskippy

# Install dependencies
make install-deps

# Run tests
make test

# Run locally
make run
```

## 📊 Performance & Limitations

- **Reconciliation interval**: 30 seconds default
- **Metric collection overhead**: <1% CPU
- **AI inference time**: 5-30 seconds (with Ollama)
- **Supported resources**: Pods, Deployments, StatefulSets, DaemonSets
- **Max actions/hour**: Configurable per policy (default: 5)
- **Test coverage**: Unit tests for controllers, metrics, and remediation engine

## 🔒 Security

- RBAC controls for fine-grained permissions
- No secrets or sensitive data sent to AI
- Webhook admission control for policy validation
- Signed container images

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Kubernetes operator-sdk community
- Ollama team for local LLM inference
- Prometheus project for metrics
- All our contributors

---

**Ready to give your applications self-healing superpowers?** [Get Started →](QUICKSTART.md)