# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# YEAR defines the year value used for substituting the YEAR placeholder in the boilerplate header.
YEAR ?= $(shell date +%Y)

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.DEFAULT_GOAL := help

# Tools are taken from PATH and never installed by this Makefile.
# A missing one is reported with where to get it. Override any of them, e.g. make KIND=/opt/kind kind-up.
CONTAINER_TOOL ?= docker
KUBECTL ?= kubectl
KIND ?= kind
KUSTOMIZE ?= kustomize
CONTROLLER_GEN ?= controller-gen
ENVTEST ?= setup-envtest

# The Kubernetes version envtest runs, from k8s.io/api in go.mod (v0.37.0 -> 1.37).
ENVTEST_K8S_VERSION ?= $(shell go list -m -f '{{.Version}}' k8s.io/api | sed -E 's/^v?[0-9]+\.([0-9]+).*/1.\1/')
ENVTEST_DIR ?= $(CURDIR)/bin

# $(call require,<tool>,<where to get it>)
define require
@command -v $(1) >/dev/null 2>&1 || { echo "missing: $(1) -> $(2)"; exit 1; }
endef

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: tools
tools: ## Check that every tool is installed.
	@missing=0; \
	for t in "$(CONTAINER_TOOL)" "$(KUBECTL)" "$(KIND)" "$(KUSTOMIZE)" "$(CONTROLLER_GEN)" "$(ENVTEST)"; do \
		if command -v "$$t" >/dev/null 2>&1; then echo "ok:      $$t"; else echo "missing: $$t"; missing=1; fi; \
	done; exit $$missing

##@ Development

.PHONY: manifests
manifests: ## Generate the ClusterRole and CustomResourceDefinition objects.
	$(call require,$(CONTROLLER_GEN),go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.22.0)
	"$(CONTROLLER_GEN)" rbac:roleName=manager-role crd paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: ## Generate DeepCopy, DeepCopyInto and DeepCopyObject methods.
	$(call require,$(CONTROLLER_GEN),go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.22.0)
	"$(CONTROLLER_GEN)" object:headerFile="hack/boilerplate.go.txt",year=$(YEAR) paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet ## Run tests (envtest: a real kube-apiserver and etcd).
	$(call require,$(ENVTEST),go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)
	@assets="$$("$(ENVTEST)" use -i $(ENVTEST_K8S_VERSION) --bin-dir "$(ENVTEST_DIR)" -p path 2>/dev/null)" || { \
		echo "missing: envtest binaries for Kubernetes $(ENVTEST_K8S_VERSION) -> $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(ENVTEST_DIR)"; \
		exit 1; \
	}; \
	KUBEBUILDER_ASSETS="$$assets" go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	$(call require,$(CONTAINER_TOOL),https://docs.docker.com/get-docker/)
	$(CONTAINER_TOOL) build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(call require,$(CONTAINER_TOOL),https://docs.docker.com/get-docker/)
	$(CONTAINER_TOOL) push ${IMG}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(call require,$(KUSTOMIZE),https://kubectl.docs.kubernetes.io/installation/kustomize/)
	$(call require,$(KUBECTL),https://kubernetes.io/docs/tasks/tools/)
	"$(KUSTOMIZE)" build config/crd | "$(KUBECTL)" apply -f -

.PHONY: uninstall
uninstall: manifests ## Uninstall CRDs. Call with ignore-not-found=true to ignore missing resources.
	$(call require,$(KUSTOMIZE),https://kubectl.docs.kubernetes.io/installation/kustomize/)
	$(call require,$(KUBECTL),https://kubernetes.io/docs/tasks/tools/)
	"$(KUSTOMIZE)" build config/crd | "$(KUBECTL)" delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	$(call require,$(KUSTOMIZE),https://kubectl.docs.kubernetes.io/installation/kustomize/)
	$(call require,$(KUBECTL),https://kubernetes.io/docs/tasks/tools/)
	cd config/manager && "$(KUSTOMIZE)" edit set image controller=${IMG}
	"$(KUSTOMIZE)" build config/default | "$(KUBECTL)" apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller. Call with ignore-not-found=true to ignore missing resources.
	$(call require,$(KUSTOMIZE),https://kubectl.docs.kubernetes.io/installation/kustomize/)
	$(call require,$(KUBECTL),https://kubernetes.io/docs/tasks/tools/)
	"$(KUSTOMIZE)" build config/default | "$(KUBECTL)" delete --ignore-not-found=$(ignore-not-found) -f -

##@ Local cluster (kind)

KIND_LOCAL ?= widget-operator
KIND_IMG ?= widget-operator:dev

.PHONY: kind-up
kind-up: ## Create a local kind cluster from deploy/kind.yaml (control plane + one worker).
	$(call require,$(KIND),https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
	"$(KIND)" create cluster --name $(KIND_LOCAL) --config deploy/kind.yaml

.PHONY: kind-deploy
kind-deploy: ## Build the image, load it into kind, deploy the operator and the sample Widget.
	$(call require,$(KIND),https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
	$(call require,$(KUBECTL),https://kubernetes.io/docs/tasks/tools/)
	$(MAKE) docker-build IMG=$(KIND_IMG)
	"$(KIND)" load docker-image $(KIND_IMG) --name $(KIND_LOCAL)
	$(MAKE) install deploy IMG=$(KIND_IMG)
	"$(KUBECTL)" -n widget-operator-system rollout status deployment/widget-operator-controller-manager
	"$(KUBECTL)" apply -k config/samples

.PHONY: kind-down
kind-down: ## Delete the local kind cluster.
	$(call require,$(KIND),https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
	"$(KIND)" delete cluster --name $(KIND_LOCAL)
