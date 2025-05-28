# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- Fixed HealingAction controller state transition logic to properly handle status updates
- Fixed Prometheus test mock server to handle client-go POST requests correctly
- Fixed nil pointer dereference in controller tests by ensuring proper map initialization
- Improved controller reconciliation logic following Kubernetes best practices
- Updated test expectations to match actual controller behavior (multiple reconciliations for state transitions)

### Changed
- Reordered controller update operations: status updates now happen before metadata/label updates
- Enhanced test helper functions to better simulate real Kubernetes reconciliation loops
- Improved Prometheus client health check endpoint configuration

### Added
- Better test diagnostics with detailed logging of state transitions
- Comprehensive test coverage documentation in README

## [0.1.0] - 2025-01-27

### Added
- Initial release with core operator functionality
- HealingPolicy and HealingAction CRDs
- Multiple trigger types (metric, event, condition)
- Various remediation actions (restart, scale, patch, delete)
- AI integration with Ollama and OpenAI
- Prometheus metrics integration
- Safety controls and rate limiting
- Comprehensive demo environment
- E2E test suite