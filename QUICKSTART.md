# Quick Start Guide

## Prerequisites
- Go 1.21+
- Docker
- kubectl
- Kind (for local development)
- Kubebuilder 3.14.0

## Initial Setup

### 1. Clone and Initialize
```bash
# Clone the repo (update with your repo URL)
git clone https://github.com/yourusername/k8s-ai-nanny
cd k8s-ai-nanny

# Install dependencies
go mod download

# Install development tools
make controller-gen
make kustomize
```

### 2. Create Local Development Cluster
```bash
# Create Kind cluster
make kind-create

# Install Ollama for local AI
make install-ollama

# Wait for Ollama to pull the model (this takes a few minutes)
kubectl logs -n ai-nanny-system -l job-name=ollama-pull-model -f
```

### 3. Initialize Kubebuilder Project
```bash
# Initialize the operator
kubebuilder init --domain ai-nanny.io --repo github.com/yourusername/k8s-ai-nanny

# Create the NannyConfig API
kubebuilder create api --group nanny --version v1alpha1 --kind NannyConfig --resource --controller

# Create webhook for validation
kubebuilder create webhook --group nanny --version v1alpha1 --kind NannyConfig --defaulting --validation
```

### 4. Test the Operator Locally
```bash
# Install CRDs
make install

# Run the operator locally
make run
```

### 5. Deploy to Cluster
```bash
# Build and load image to Kind
make docker-build IMG=k8s-ai-nanny:latest
make kind-load IMG=k8s-ai-nanny:latest

# Deploy the operator
make deploy IMG=k8s-ai-nanny:latest

# Check deployment
kubectl get pods -n ai-nanny-system
```

## Development Workflow

### Running Tests
```bash
# Unit tests
make test

# E2E tests (requires running cluster)
make test-e2e
```

### Making Changes
1. Edit code
2. Run `make generate` to update generated code
3. Run `make manifests` to update CRDs
4. Test locally with `make run`
5. Build and deploy with `make docker-build && make deploy`

## GitOps Setup

### 1. Fork/Create Repository
- Fork this repository or create a new one
- Update all references to `yourusername` in the code

### 2. Setup GitHub Secrets
Add these secrets to your GitHub repository:
- `DOCKER_USERNAME`: Your Docker Hub username
- `DOCKER_PASSWORD`: Your Docker Hub password

### 3. Install ArgoCD
```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Wait for ArgoCD to be ready
kubectl wait --for=condition=Ready pods --all -n argocd --timeout=300s

# Port forward to access UI
kubectl port-forward svc/argocd-server -n argocd 8080:443

# Get admin password
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
```

### 4. Deploy Application
```bash
# Apply ArgoCD application
kubectl apply -f argocd/application.yaml

# Check sync status
kubectl get applications -n argocd
```

## Next Steps

1. **Implement Core Features**:
   - Complete the NannyConfig CRD in `api/v1alpha1/nannyconfig_types.go`
   - Implement the controller logic in `controllers/nannyconfig_controller.go`
   - Add Prometheus metrics collection
   - Integrate with Ollama for AI analysis

2. **Add Safety Controls**:
   - Implement protected resources list
   - Add dry-run mode
   - Create audit logging

3. **Build Remediation Actions**:
   - Pod restart logic
   - Resource adjustment
   - Cleanup routines

4. **Testing**:
   - Write comprehensive unit tests
   - Create chaos test scenarios
   - Add integration tests

## Troubleshooting

### Ollama Connection Issues
```bash
# Check Ollama is running
kubectl get pods -n ai-nanny-system -l app=ollama

# Check logs
kubectl logs -n ai-nanny-system -l app=ollama

# Test connection
kubectl run test-ollama --rm -it --image=curlimages/curl -- curl http://ollama-service.ai-nanny-system:11434/api/tags
```

### Operator Issues
```bash
# Check operator logs
kubectl logs -n ai-nanny-system deployment/ai-nanny-controller-manager

# Check CRDs
kubectl get crds | grep nanny

# Check RBAC
kubectl describe clusterrole ai-nanny-manager-role
```