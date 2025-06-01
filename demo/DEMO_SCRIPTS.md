# KubeSkippy Demo Scripts

This directory contains scripts to run a complete KubeSkippy AI-driven healing demo.

## Quick Start (Zero Interaction)

```bash
# Clone the repo and run the bulletproof setup
git clone <repo-url>
cd KubeSkippy/demo
./bulletproof-ai-setup.sh
```

## Scripts Overview

### Primary Setup
- **`bulletproof-ai-setup.sh`** - Main setup script with zero human interaction required
  - Sets up Kind cluster with real Ollama AI (llama2:7b)
  - Deploys Prometheus + Grafana monitoring with correct data sources
  - Deploys KubeSkippy operator with AI reasoning panel
  - Deploys kube-state-metrics for Kubernetes metrics
  - Creates continuous pressure applications that trigger healing actions
  - Includes all RBAC fixes and service configurations
  - Handles all errors and edge cases automatically

### Port Forward Management
- **`start-port-forwards.sh`** - Start port forwarding for Grafana and Prometheus
- **`stop-port-forwards.sh`** - Stop all port forwarding

### Monitoring & Status
- **`monitor-demo.sh`** - Show comprehensive demo status dashboard
- **`continuous-ai-demo.sh`** - Optional: Deploy additional continuous pressure apps (already included in bulletproof script)
- **`setup.sh`** - Legacy setup script (use bulletproof-ai-setup.sh instead)

### Cleanup
- **`cleanup-demo.sh`** - Complete cleanup of demo environment

## Access URLs

After running the setup:
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090

## Key Features

✅ **Real AI**: Uses genuine llama2:7b model for healing decisions  
✅ **Zero Interaction**: No manual steps or human intervention required  
✅ **AI Reasoning Panel**: Grafana dashboard shows AI decision-making process  
✅ **Auto Port Forwarding**: Persistent access to monitoring dashboards  
✅ **Complete Automation**: Handles errors, retries, and edge cases  

## Troubleshooting

If port forwards stop working:
```bash
./start-port-forwards.sh
```

To check demo status:
```bash
./monitor-demo.sh
```

To completely start over:
```bash
./cleanup-demo.sh
./bulletproof-ai-setup.sh
```