# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.28.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out

.PHONY: test-e2e
test-e2e: ## Run e2e tests
	cd tests/e2e && go test -v ./...

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Local Development

.PHONY: kind-create
kind-create: ## Create kind cluster
	kind create cluster --config=tests/kind-config.yaml --name ai-nanny

.PHONY: kind-delete
kind-delete: ## Delete kind cluster
	kind delete cluster --name ai-nanny

.PHONY: kind-load
kind-load: docker-build ## Load docker image into kind
	kind load docker-image ${IMG} --name ai-nanny

.PHONY: install-ollama
install-ollama: ## Install Ollama in kind cluster
	kubectl apply -f tests/ollama-deployment.yaml

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v5.2.1
CONTROLLER_TOOLS_VERSION ?= v0.13.0

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || GOBIN=$(LOCALBIN) go install sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

##@ Demo

DEMO_CLUSTER ?= kubeskippy-demo
DEMO_NAMESPACE ?= demo-apps

.PHONY: demo-up
demo-up: ## Start the demo environment
	cd demo && ./setup.sh

.PHONY: demo-down
demo-down: ## Tear down the demo environment
	cd demo && ./cleanup.sh

.PHONY: demo-deploy-operator
demo-deploy-operator: manifests docker-build ## Deploy operator to demo cluster
	kind load docker-image ${IMG} --name $(DEMO_CLUSTER)
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: demo-deploy-apps
demo-deploy-apps: ## Deploy demo applications
	kubectl apply -f demo/apps/

.PHONY: demo-apply-policies
demo-apply-policies: ## Apply healing policies
	kubectl apply -f demo/policies/

.PHONY: demo-watch
demo-watch: ## Watch demo healing actions
	@echo "Watching healing actions in $(DEMO_NAMESPACE)..."
	@echo "Press Ctrl+C to stop"
	@kubectl get healingactions -n $(DEMO_NAMESPACE) -w

.PHONY: demo-status
demo-status: ## Show demo status
	@echo "=== Deployments ==="
	@kubectl get deployments -n $(DEMO_NAMESPACE)
	@echo ""
	@echo "=== Pods ==="
	@kubectl get pods -n $(DEMO_NAMESPACE)
	@echo ""
	@echo "=== Healing Policies ==="
	@kubectl get healingpolicies -n $(DEMO_NAMESPACE)
	@echo ""
	@echo "=== Healing Actions ==="
	@kubectl get healingactions -n $(DEMO_NAMESPACE)

.PHONY: demo-logs
demo-logs: ## Show operator logs
	kubectl logs -n kubeskippy-system deployment/kubeskippy-controller-manager -f

.PHONY: demo-reset
demo-reset: ## Reset demo (delete and recreate apps)
	kubectl delete -f demo/apps/ --ignore-not-found=true
	kubectl delete healingactions -n $(DEMO_NAMESPACE) --all
	sleep 5
	kubectl apply -f demo/apps/