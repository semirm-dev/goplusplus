# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.DEFAULT_GOAL := help

# Tools are taken from PATH and never installed by this Makefile.
# A missing one is reported with where to get it. Override any of them, e.g. make ENVTEST=/opt/setup-envtest test.
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
	for t in "$(CONTROLLER_GEN)" "$(ENVTEST)"; do \
		if command -v "$$t" >/dev/null 2>&1; then echo "ok:      $$t"; else echo "missing: $$t"; missing=1; fi; \
	done; exit $$missing

##@ Development

.PHONY: manifests
manifests: ## Generate the CustomResourceDefinition.
	$(call require,$(CONTROLLER_GEN),go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.22.0)
	"$(CONTROLLER_GEN)" crd paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: ## Generate DeepCopy, DeepCopyInto and DeepCopyObject methods.
	$(call require,$(CONTROLLER_GEN),go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.22.0)
	"$(CONTROLLER_GEN)" object paths="./..."

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
