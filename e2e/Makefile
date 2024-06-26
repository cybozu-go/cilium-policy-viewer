CILIUM_VERSION := 1.15.3

BIN_DIR := $(shell pwd)/../bin
TOOLS_DIR := $(BIN_DIR)/download
CACHE_DIR := $(shell pwd)/../cache

CILIUM_POLICY := $(BIN_DIR)/cilium-policy
HELM := $(TOOLS_DIR)/helm --repository-cache $(CACHE_DIR)/helm/repository --repository-config $(CACHE_DIR)/helm/repositories.yaml
KIND := $(TOOLS_DIR)/kind
KUBECTL := $(TOOLS_DIR)/kubectl
KUSTOMIZE := $(TOOLS_DIR)/kustomize
YQ := $(TOOLS_DIR)/yq

##@ Basic

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Test

.PHONY: wait-for-workloads
wait-for-workloads:
	$(KUBECTL) wait --for=condition=Available --all deployments --all-namespaces --timeout=1h
	$(KUBECTL) wait --for=condition=Ready --all pods --all-namespaces --timeout=1h

.PHONY: start
start:
	docker pull quay.io/cilium/cilium:v$(CILIUM_VERSION)
	$(KIND) create cluster --config testdata/cluster.yaml
	$(KIND) load docker-image quay.io/cilium/cilium:v$(CILIUM_VERSION)
	$(HELM) install cilium cilium/cilium --version $(CILIUM_VERSION) \
		--namespace kube-system \
		--set image.pullPolicy=IfNotPresent \
		--set ipam.mode=kubernetes
	$(KUSTOMIZE) build testdata | $(KUBECTL) apply -f -
	$(MAKE) --no-print-directory wait-for-workloads

run-test-pod-%:
	cat testdata/template/ubuntu.yaml | \
		$(YQ) '.metadata.name = "$*"' | \
		$(YQ) '.spec.selector.matchLabels = {"test": "$*"}' | \
		$(YQ) '.spec.template.metadata.labels = {"test": "$*"}' | \
		$(KUBECTL) apply -f -

.PHONY: install-test-pod
install-test-pod:
	$(MAKE) --no-print-directory run-test-pod-self
	$(MAKE) --no-print-directory run-test-pod-l3-ingress-explicit-allow
	$(MAKE) --no-print-directory run-test-pod-l3-ingress-no-rule
	$(MAKE) --no-print-directory run-test-pod-l3-ingress-explicit-deny
	$(MAKE) --no-print-directory run-test-pod-l3-egress-implicit-deny
	$(MAKE) --no-print-directory run-test-pod-l3-egress-explicit-deny
	$(MAKE) --no-print-directory wait-for-workloads
	$(KUBECTL) apply -f testdata/policy/l3.yaml

.PHONY: install-cilium-policy
install-cilium-policy:
	$(MAKE) -C ../ build
	PODNAME=$$($(KUBECTL) get po -l app=ubuntu -o name | cut -d'/' -f2); \
	$(KUBECTL) cp $(CILIUM_POLICY) $${PODNAME}:/tmp/; \
	$(KUBECTL) exec $${PODNAME} -- chmod +x /tmp/cilium-policy

.PHONY: test
test:
	go test -v -race . -ginkgo.v -ginkgo.fail-fast

.PHONY: stop
stop:
	$(KIND) delete cluster
