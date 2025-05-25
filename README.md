# K8s AI Auto-Healing Operator (KubeSkippy)

An intelligent Kubernetes operator that uses AI to automatically detect and heal cluster issues, ensuring high availability and optimal performance.

## Key Features

- **AI-Powered Analysis**: Uses local LLMs (via Ollama) to analyze cluster state and recommend healing actions
- **Safe Auto-Healing**: Implements multiple safety layers to prevent destructive actions
- **GitOps Ready**: Full CI/CD pipeline with ArgoCD integration
- **Prometheus Integration**: Leverages existing monitoring infrastructure
- **Dry-Run Mode**: Test healing actions before enabling automation
- **Audit Trail**: Complete logging of all decisions and actions

## Architecture

The operator follows a observe-analyze-act pattern:
1. **Observe**: Collect metrics from Prometheus and Kubernetes API
2. **Analyze**: Send cluster state to AI for intelligent analysis
3. **Act**: Execute approved healing actions with safety checks

## Quick Start

See [QUICKSTART.md](QUICKSTART.md) for detailed setup instructions.

```bash
# Create local cluster
make kind-create

# Install dependencies
make install-ollama

# Run operator locally
make run
```

## Safety Features

- **Protected Resources**: Never modify critical system components
- **Approval Levels**: Different actions require different approval
- **Circuit Breaker**: Stops remediation if too many failures
- **Rollback**: Can undo actions if they make things worse

## Project Status

This project is in active development. Current focus:
- [x] Architecture design
- [x] GitOps CI/CD pipeline
- [x] Local development environment
- [ ] Core operator implementation
- [ ] AI integration
- [ ] Remediation engine
- [ ] Production readiness

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `make test`
5. Submit a pull request

## License

MIT License - see LICENSE file for details