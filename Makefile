##################################################
# Variables                                      #
##################################################
SHELL           = /bin/bash

# If E2E_IMAGE_TAG is defined, we are on pr e2e test and we have to use the new tag and append -test to the repository
ifeq '${E2E_IMAGE_TAG}' ''
VERSION ?= main
# SUFIX here is intentional empty to not append nothing to the repository
SUFFIX =
endif

ifneq '${E2E_IMAGE_TAG}' ''
VERSION = ${E2E_IMAGE_TAG}
SUFFIX = -test
endif

IMAGE_REGISTRY ?= ghcr.io
IMAGE_REPO     ?= kedacore

IMAGE_CONTROLLER = $(IMAGE_REGISTRY)/$(IMAGE_REPO)/keda$(SUFFIX):$(VERSION)
IMAGE_ADAPTER    = $(IMAGE_REGISTRY)/$(IMAGE_REPO)/keda-metrics-apiserver$(SUFFIX):$(VERSION)

BUILD_TOOLS_GO_VERSION = 1.18.8
IMAGE_BUILD_TOOLS = $(IMAGE_REGISTRY)/$(IMAGE_REPO)/build-tools:$(BUILD_TOOLS_GO_VERSION)

ARCH       ?=amd64
CGO        ?=0
TARGET_OS  ?=linux

BUILD_PLATFORMS ?= linux/amd64,linux/arm64
OUTPUT_TYPE     ?= registry

GIT_VERSION ?= $(shell git describe --always --abbrev=7)
GIT_COMMIT  ?= $(shell git rev-list -1 HEAD)
DATE        = $(shell date -u +"%Y.%m.%d.%H.%M.%S")

TEST_CLUSTER_NAME ?= keda-nightly-run-3

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

GO_BUILD_VARS= GO111MODULE=on CGO_ENABLED=$(CGO) GOOS=$(TARGET_OS) GOARCH=$(ARCH)
GO_LDFLAGS="-X=github.com/kedacore/keda/v2/version.GitCommit=$(GIT_COMMIT) -X=github.com/kedacore/keda/v2/version.Version=$(VERSION)"

COSIGN_FLAGS ?= -a GIT_HASH=${GIT_COMMIT} -a GIT_VERSION=${VERSION} -a BUILD_DATE=${DATE}

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.23

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

##################################################
# All                                            #
##################################################
.PHONY: all
all: build

##################################################
# Tests                                          #
##################################################

##@ Test
.PHONY: install-test-deps
install-test-deps:
	go install github.com/jstemmer/go-junit-report/v2@latest

.PHONY: test
test: manifests generate fmt vet envtest install-test-deps ## Run tests and export the result to junit format.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test -v 2>&1 ./... -coverprofile cover.out | go-junit-report -iocopy -set-exit-code -out report.xml

.PHONY: get-cluster-context
get-cluster-context: ## Get Azure cluster context.
	@az login --service-principal -u $(AZURE_SP_APP_ID) -p "$(AZURE_SP_KEY)" --tenant $(AZURE_SP_TENANT)
	@az aks get-credentials \
		--name $(TEST_CLUSTER_NAME) \
		--subscription $(AZURE_SUBSCRIPTION) \
		--resource-group $(AZURE_RESOURCE_GROUP)

.PHONY: e2e-test
e2e-test: get-cluster-context ## Run e2e tests against Azure cluster.
	TERMINFO=/etc/terminfo
	TERM=linux
	./tests/run-all.sh

.PHONY: e2e-test-local
e2e-test-local: ## Run e2e tests against Kubernetes cluster configured in ~/.kube/config.
	./tests/run-all.sh

.PHONY: e2e-test-clean-crds
e2e-test-clean-crds: ## Delete all scaled objects and jobs across all namespaces
	./tests/clean-crds.sh

.PHONY: e2e-test-clean
e2e-test-clean: get-cluster-context ## Delete all namespaces labeled with type=e2e
	kubectl delete ns -l type=e2e

.PHONY: smoke-test
smoke-test: ## Run e2e tests against Kubernetes cluster configured in ~/.kube/config.
	./tests/run-smoke-tests.sh

##################################################
# Development                                    #
##################################################

##@ Development

manifests: controller-gen ## Generate ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) crd:crdVersions=v1 rbac:roleName=keda-operator paths="./..." output:crd:artifacts:config=config/crd/bases
	# withTriggers is only used for duck typing so we only need the deepcopy methods
	# However operator-sdk generate doesn't appear to have an option for that
	# until this issue is fixed: https://github.com/kubernetes-sigs/controller-tools/issues/398
	rm config/crd/bases/keda.sh_withtriggers.yaml

generate: controller-gen mockgen-gen ## Generate code containing DeepCopy, DeepCopyInto, DeepCopyObject method implementations (API) and mocks.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

adapter/generated/openapi/zz_generated.openapi.go: go.mod go.sum ## Generate OpenAPI for KEDA Metrics Adapter.
	@OPENAPI_PATH=`go list -mod=readonly -m -f '{{.Dir}}' k8s.io/kube-openapi`; \
	go run $${OPENAPI_PATH}/cmd/openapi-gen/openapi-gen.go --logtostderr \
	    -i k8s.io/metrics/pkg/apis/custom_metrics,k8s.io/metrics/pkg/apis/custom_metrics/v1beta1,k8s.io/metrics/pkg/apis/custom_metrics/v1beta2,k8s.io/metrics/pkg/apis/external_metrics,k8s.io/metrics/pkg/apis/external_metrics/v1beta1,k8s.io/metrics/pkg/apis/metrics,k8s.io/metrics/pkg/apis/metrics/v1beta1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/api/resource,k8s.io/apimachinery/pkg/version,k8s.io/api/core/v1 \
	    --build-tag autogenerated \
	    -h ./hack/boilerplate.go.txt \
	    -p ./adapter/generated/openapi \
	    -O zz_generated.openapi \
	    -o ./ \
	    -r /dev/null

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

golangci: ## Run golangci against code.
	golangci-lint run

clientset-verify: ## Verify that generated client-go clientset, listers and informers are up to date.
	go mod vendor
	./hack/verify-codegen.sh
	rm -rf vendor

clientset-generate: ## Generate client-go clientset, listers and informers.
	go mod vendor
	./hack/update-codegen.sh
	rm -rf vendor

# Generate Liiklus proto
pkg/scalers/liiklus/LiiklusService.pb.go: protoc-gen-go
	protoc --proto_path=hack LiiklusService.proto --go_out=pkg/scalers/liiklus --go-grpc_out=pkg/scalers/liiklus

# Generate ExternalScaler proto
pkg/scalers/externalscaler/externalscaler.pb.go: protoc-gen-go
	protoc --proto_path=pkg/scalers/externalscaler externalscaler.proto --go-grpc_out=pkg/scalers/externalscaler

protoc-gen-go: ## Download protoc-gen-go
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.1
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2.0

.PHONY: mockgen-gen
mockgen-gen: mockgen pkg/mock/mock_scaling/mock_interface.go pkg/mock/mock_scaler/mock_scaler.go pkg/mock/mock_scale/mock_interfaces.go pkg/mock/mock_client/mock_interfaces.go pkg/scalers/liiklus/mocks/mock_liiklus.go

pkg/mock/mock_scaling/mock_interface.go: pkg/scaling/scale_handler.go
	$(MOCKGEN) -destination=$@ -package=mock_scaling -source=$^
pkg/mock/mock_scaler/mock_scaler.go: pkg/scalers/scaler.go
	$(MOCKGEN) -destination=$@ -package=mock_scalers -source=$^
pkg/mock/mock_scale/mock_interfaces.go: $(shell go list -mod=readonly -f '{{ .Dir }}' -m k8s.io/client-go)/scale/interfaces.go
	$(MOCKGEN) -destination=$@ -package=mock_scale -source=$^
pkg/mock/mock_client/mock_interfaces.go: $(shell go list -mod=readonly -f '{{ .Dir }}' -m sigs.k8s.io/controller-runtime)/pkg/client/interfaces.go
	$(MOCKGEN) -destination=$@ -package=mock_client -source=$^
pkg/scalers/liiklus/mocks/mock_liiklus.go:
	$(MOCKGEN) -destination=$@ github.com/kedacore/keda/v2/pkg/scalers/liiklus LiiklusServiceClient

##################################################
# Build                                          #
##################################################

##@ Build

build: generate fmt vet manager adapter ## Build Operator (manager) and Metrics Server (adapter) binaries.

manager: generate
	${GO_BUILD_VARS} go build -ldflags $(GO_LDFLAGS) -o bin/keda main.go

adapter: generate adapter/generated/openapi/zz_generated.openapi.go
	${GO_BUILD_VARS} go build -ldflags $(GO_LDFLAGS) -o bin/keda-adapter adapter/main.go

run: manifests generate ## Run a controller from your host.
	WATCH_NAMESPACE="" go run -ldflags $(GO_LDFLAGS) ./main.go $(ARGS)

docker-build: ## Build docker images with the KEDA Operator and Metrics Server.
	DOCKER_BUILDKIT=1 docker build . -t ${IMAGE_CONTROLLER} --build-arg BUILD_VERSION=${VERSION} --build-arg GIT_VERSION=${GIT_VERSION} --build-arg GIT_COMMIT=${GIT_COMMIT}
	DOCKER_BUILDKIT=1 docker build -f Dockerfile.adapter -t ${IMAGE_ADAPTER} . --build-arg BUILD_VERSION=${VERSION} --build-arg GIT_VERSION=${GIT_VERSION} --build-arg GIT_COMMIT=${GIT_COMMIT}

publish: docker-build ## Push images on to Container Registry (default: ghcr.io).
	docker push $(IMAGE_CONTROLLER)
	docker push $(IMAGE_ADAPTER)

publish-controller-multiarch: ## Build and push multi-arch Docker image for KEDA Operator.
	docker buildx build --output=type=${OUTPUT_TYPE} --platform=${BUILD_PLATFORMS} . -t ${IMAGE_CONTROLLER} --build-arg BUILD_VERSION=${VERSION} --build-arg GIT_VERSION=${GIT_VERSION} --build-arg GIT_COMMIT=${GIT_COMMIT}

publish-adapter-multiarch: ## Build and push multi-arch Docker image for KEDA Metrics Server.
	docker buildx build --output=type=${OUTPUT_TYPE} --platform=${BUILD_PLATFORMS} -f Dockerfile.adapter -t ${IMAGE_ADAPTER} . --build-arg BUILD_VERSION=${VERSION} --build-arg GIT_VERSION=${GIT_VERSION} --build-arg GIT_COMMIT=${GIT_COMMIT}

publish-multiarch: publish-controller-multiarch publish-adapter-multiarch ## Push multi-arch Docker images on to Container Registry (default: ghcr.io).

release: manifests kustomize set-version ## Produce new KEDA release in keda-$(VERSION).yaml file.
	cd config/manager && \
	$(KUSTOMIZE) edit set image ghcr.io/kedacore/keda=${IMAGE_CONTROLLER}
	cd config/metrics-server && \
    $(KUSTOMIZE) edit set image ghcr.io/kedacore/keda-metrics-apiserver=${IMAGE_ADAPTER}
	# Need this workaround to mitigate a problem with inserting labels into selectors,
	# until this issue is solved: https://github.com/kubernetes-sigs/kustomize/issues/1009
	@sed -i".out" -e 's@version:[ ].*@version: $(VERSION)@g' config/default/kustomize-config/metadataLabelTransformer.yaml
	rm -rf config/default/kustomize-config/metadataLabelTransformer.yaml.out
	$(KUSTOMIZE) build config/default > keda-$(VERSION).yaml

sign-images: ## Sign KEDA images published on GitHub Container Registry
	COSIGN_EXPERIMENTAL=1 cosign sign ${COSIGN_FLAGS} $(IMAGE_CONTROLLER)
	COSIGN_EXPERIMENTAL=1 cosign sign ${COSIGN_FLAGS} $(IMAGE_ADAPTER)

.PHONY: set-version
set-version:
	@sed -i".out" -e 's@Version[ ]*=.*@Version = "$(VERSION)"@g' ./version/version.go;
	rm -rf ./version/version.go.out

##################################################
# Deployment                                     #
##################################################

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && \
	$(KUSTOMIZE) edit set image ghcr.io/kedacore/keda=${IMAGE_CONTROLLER}
	cd config/metrics-server && \
    $(KUSTOMIZE) edit set image ghcr.io/kedacore/keda-metrics-apiserver=${IMAGE_ADAPTER}
	if [ "$(AZURE_RUN_WORKLOAD_IDENTITY_TESTS)" = true ]; then \
		cd config/service_account && \
		$(KUSTOMIZE) edit add label --force azure.workload.identity/use:true; \
		$(KUSTOMIZE) edit add annotation --force azure.workload.identity/client-id:${AZURE_SP_APP_ID} azure.workload.identity/tenant-id:${AZURE_SP_TENANT}; \
	fi
	if [ "$(AWS_RUN_IDENTITY_TESTS)" = true ]; then \
		cd config/service_account && \
		$(KUSTOMIZE) edit add annotation --force eks.amazonaws.com/role-arn:arn:aws:iam::${AWS_ACCOUNT_ID}:role/${TEST_CLUSTER_NAME}; \
	fi
	# Need this workaround to mitigate a problem with inserting labels into selectors,
	# until this issue is solved: https://github.com/kubernetes-sigs/kustomize/issues/1009
	@sed -i".out" -e 's@version:[ ].*@version: $(VERSION)@g' config/default/kustomize-config/metadataLabelTransformer.yaml
	rm -rf config/default/kustomize-config/metadataLabelTransformer.yaml.out
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: e2e-test-clean-crds ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -


CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.0)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.5.5)

ENVTEST = $(shell pwd)/bin/setup-envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

MOCKGEN = $(shell pwd)/bin/mockgen
mockgen: ## Download mockgen locally if necessary.
	$(call go-get-tool,$(MOCKGEN),github.com/golang/mock/mockgen@v1.6.0)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

##################################################
# General                                        #
##################################################

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: docker-build-tools
docker-build-tools: ## Build build-tools image
	docker build -f tools/build-tools.Dockerfile -t $(IMAGE_BUILD_TOOLS) --build-arg GO_VERSION=$(BUILD_TOOLS_GO_VERSION) .

.PHONY: publish-build-tools
publish-build-tools: ## Build and push multi-arch Docker image for build-tools.
	docker buildx build --push --platform=${BUILD_PLATFORMS} -f tools/build-tools.Dockerfile -t ${IMAGE_BUILD_TOOLS} --build-arg GO_VERSION=$(BUILD_TOOLS_GO_VERSION) .

.PHONY: docker-build-dev-containers
docker-build-dev-containers: ## Build dev-containers image
	docker build -f .devcontainer/Dockerfile .
