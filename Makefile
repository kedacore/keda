##################################################
# Variables                                      #
##################################################
SHELL           = /bin/bash

VERSION ?= main
SUFFIX ?=

IMAGE_REGISTRY ?= ghcr.io
IMAGE_REPO     ?= kedacore

IMAGE_CONTROLLER = $(IMAGE_REGISTRY)/$(IMAGE_REPO)/keda$(SUFFIX):$(VERSION)
IMAGE_ADAPTER    = $(IMAGE_REGISTRY)/$(IMAGE_REPO)/keda-metrics-apiserver$(SUFFIX):$(VERSION)
IMAGE_WEBHOOKS   = $(IMAGE_REGISTRY)/$(IMAGE_REPO)/keda-admission-webhooks$(SUFFIX):$(VERSION)

ARCH       ?=amd64
CGO        ?=0
TARGET_OS  ?=linux

BUILD_PLATFORMS ?= linux/amd64,linux/arm64,linux/s390x
OUTPUT_TYPE     ?= registry

GIT_VERSION ?= $(shell git describe --always --abbrev=7)
GIT_COMMIT  ?= $(shell git rev-list -1 HEAD)
DATE        = $(shell date -u +"%Y.%m.%d.%H.%M.%S")

TEST_CLUSTER_NAME ?= keda-e2e-cluster-nightly
NODE_POOL_SIZE ?= 1
NON_ROOT_USER_ID ?= 1000

GCP_WI_PROVIDER ?= projects/${TF_GCP_PROJECT_NUMBER}/locations/global/workloadIdentityPools/${TEST_CLUSTER_NAME}/providers/${TEST_CLUSTER_NAME}

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

GOPATH:=$(shell go env GOPATH)

GO_BUILD_VARS= GO111MODULE=on CGO_ENABLED=$(CGO) GOOS=$(TARGET_OS) GOARCH=$(ARCH)
GO_LDFLAGS="-X=github.com/kedacore/keda/v2/version.GitCommit=$(GIT_COMMIT) -X=github.com/kedacore/keda/v2/version.Version=$(VERSION)"

COSIGN_FLAGS ?= -y -a GIT_HASH=${GIT_COMMIT} -a GIT_VERSION=${VERSION} -a BUILD_DATE=${DATE}

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.33

GOLANGCI_VERSION:=2.5.0

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Scaler schema generation parameters
SCALERS_SCHEMA_SCALERS_BUILDER_FILE ?= pkg/scaling/scalers_builder.go
SCALERS_SCHEMA_SCALERS_FILES_DIR ?= pkg/scalers
SCALERS_SCHEMA_OUTPUT_FILE_PATH ?= schema/generated/
SCALERS_SCHEMA_OUTPUT_FILE_NAME ?= scalers-schema

ifneq '${VERSION}' 'main'
  OUTPUT_FILE_NAME :="${OUTPUT_FILE_NAME}-${VERSION}"
  SCALERS_SCHEMA_OUTPUT_FILE_NAME:="${SCALERS_SCHEMA_OUTPUT_FILE_NAME}-${VERSION}"
endif

##################################################
# All                                            #
##################################################
.PHONY: all
all: build

##################################################
# Tests                                          #
##################################################

##@ Test
.PHONY: test
test: manifests generate fmt vet envtest gotestsum ## Run tests and export the result to junit format.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" $(GOTESTSUM) --format standard-quiet --rerun-fails --junitfile report.xml

.PHONY: test-race
test-race: manifests generate fmt vet envtest gotestsum ## Run tests and export the result to junit format.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" $(GOTESTSUM) --format standard-quiet --rerun-fails --junitfile report-race.xml --packages=./... -- -race

.PHONY:
az-login:
	@az login --service-principal -u $(TF_AZURE_SP_APP_ID) -p "$(AZURE_SP_KEY)" --tenant $(TF_AZURE_SP_TENANT)

.PHONY: get-cluster-context
get-cluster-context: az-login ## Get Azure cluster context.
	@az aks get-credentials \
		--name $(TEST_CLUSTER_NAME) \
		--subscription $(TF_AZURE_SUBSCRIPTION) \
		--resource-group $(TF_AZURE_RESOURCE_GROUP)

.PHONY: scale-node-pool
scale-node-pool: az-login ## Scale nodepool.
	@az aks scale \
		--name $(TEST_CLUSTER_NAME) \
		--subscription $(TF_AZURE_SUBSCRIPTION) \
		--resource-group $(TF_AZURE_RESOURCE_GROUP) \
		--node-count $(NODE_POOL_SIZE)

.PHONY: e2e-regex-check
e2e-regex-check:
	go run -tags e2e ./tests/run-all.go regex-check

.PHONY: e2e-test
e2e-test: get-cluster-context ## Run e2e tests against Azure cluster.
	TERMINFO=/etc/terminfo
	TERM=linux
	go run -tags e2e ./tests/run-all.go

.PHONY: e2e-test-local
e2e-test-local: ## Run e2e tests against Kubernetes cluster configured in ~/.kube/config.
	go run -tags e2e ./tests/run-all.go

.PHONY: e2e-test-clean-crds
e2e-test-clean-crds: ## Delete all scaled objects and jobs across all namespaces
	./tests/clean-crds.sh

.PHONY: e2e-test-clean
e2e-test-clean: get-cluster-context ## Delete all namespaces labeled with type=e2e
	kubectl delete ns -l type=e2e
	# Clean up the strimzi CRDs, helm will not update them on Strimzi install if they already exist
	# and we get stranded on old versions when we try to upgrade
	kubectl get crd -o name | grep kafka.strimzi.io | xargs -r kubectl delete --ignore-not-found=true --timeout=60s

.PHONY: smoke-test
smoke-test: ## Run e2e tests against Kubernetes cluster configured in ~/.kube/config.
	./tests/run-smoke-tests.sh

##################################################
# Development                                    #
##################################################

##@ Development

manifests: controller-gen ## Generate ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) crd:crdVersions=v1,generateEmbeddedObjectMeta=true rbac:roleName=keda-operator paths="./..." output:crd:artifacts:config=config/crd/bases
	# withTriggers is only used for duck typing so we only need the deepcopy methods
	# However operator-sdk generate doesn't appear to have an option for that
	# until this issue is fixed: https://github.com/kubernetes-sigs/controller-tools/issues/398
	rm config/crd/bases/keda.sh_withtriggers.yaml

generate: controller-gen mockgen-gen proto-gen ## Generate code containing DeepCopy, DeepCopyInto, DeepCopyObject method implementations (API), mocks and proto.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

HAS_GOLANGCI_VERSION:=$(shell $(GOPATH)/bin/golangci-lint version --short)
.PHONY: golangci
golangci: ## Run golangci against code.
ifneq ($(HAS_GOLANGCI_VERSION), $(GOLANGCI_VERSION))
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v$(GOLANGCI_VERSION)
endif
	golangci-lint run

verify-manifests: ## Verify manifests are up to date.
	./hack/verify-manifests.sh

clientset-verify: ## Verify that generated client-go clientset, listers and informers are up to date.
	./hack/verify-codegen.sh

clientset-generate: ## Generate client-go clientset, listers and informers.
	./hack/update-codegen.sh

proto-gen: protoc-gen ## Generate Liiklus, ExternalScaler and MetricsService proto
	PATH="$(LOCALBIN):$(PATH)" protoc -I vendor --proto_path=hack LiiklusService.proto --go_out=pkg/scalers/liiklus --go-grpc_out=pkg/scalers/liiklus
	PATH="$(LOCALBIN):$(PATH)" protoc -I vendor --proto_path=pkg/scalers/externalscaler externalscaler.proto --go_out=pkg/scalers/externalscaler --go-grpc_out=pkg/scalers/externalscaler
	PATH="$(LOCALBIN):$(PATH)" protoc -I vendor --proto_path=pkg/metricsservice/api metrics.proto --go_out=pkg/metricsservice/api --go-grpc_out=pkg/metricsservice/api

.PHONY: mockgen-gen
mockgen-gen: mockgen pkg/mock/mock_scaling/mock_interface.go pkg/mock/mock_scaling/mock_executor/mock_interface.go pkg/mock/mock_scaler/mock_scaler.go pkg/mock/mock_scale/mock_interfaces.go pkg/mock/mock_client/mock_interfaces.go pkg/scalers/liiklus/mocks/mock_liiklus.go pkg/mock/mock_secretlister/mock_interfaces.go pkg/mock/mock_eventemitter/mock_interface.go

pkg/mock/mock_scaling/mock_interface.go: pkg/scaling/scale_handler.go
	$(MOCKGEN) -destination=$@ -package=mock_scaling -source=$^
pkg/mock/mock_scaling/mock_executor/mock_interface.go: pkg/scaling/executor/scale_executor.go
	$(MOCKGEN) -destination=$@ -package=mock_executor -source=$^
pkg/mock/mock_scaler/mock_scaler.go: pkg/scalers/scaler.go
	$(MOCKGEN) -destination=$@ -package=mock_scalers -source=$^
pkg/mock/mock_eventemitter/mock_interface.go: pkg/eventemitter/eventemitter.go
	$(MOCKGEN) -destination=$@ -package=mock_eventemitter -source=$^
pkg/mock/mock_secretlister/mock_interfaces.go: vendor/k8s.io/client-go/listers/core/v1/secret.go
	mkdir -p pkg/mock/mock_secretlister
	$(MOCKGEN) k8s.io/client-go/listers/core/v1 SecretLister,SecretNamespaceLister > $@
pkg/mock/mock_scale/mock_interfaces.go: vendor/k8s.io/client-go/scale/interfaces.go
	mkdir -p pkg/mock/mock_scale
	$(MOCKGEN) k8s.io/client-go/scale ScalesGetter,ScaleInterface > $@
pkg/mock/mock_client/mock_interfaces.go: vendor/sigs.k8s.io/controller-runtime/pkg/client/interfaces.go
	mkdir -p pkg/mock/mock_client
	$(MOCKGEN) sigs.k8s.io/controller-runtime/pkg/client Patch,Reader,Writer,StatusClient,StatusWriter,Client,WithWatch,FieldIndexer > $@
pkg/scalers/liiklus/mocks/mock_liiklus.go:
	$(MOCKGEN) -destination=$@ github.com/kedacore/keda/v2/pkg/scalers/liiklus LiiklusServiceClient

##################################################
# Build                                          #
##################################################

##@ Build

build: update-mod generate fmt vet manager adapter webhooks ## Build Operator (manager), Metrics Server (adapter) and Admision Web Hooks (webhooks) binaries.

update-mod:
	go mod tidy
	go mod vendor

manager: generate
	${GO_BUILD_VARS} go build -ldflags $(GO_LDFLAGS) -mod=vendor -o bin/keda cmd/operator/main.go

adapter: generate
	${GO_BUILD_VARS} go build -ldflags $(GO_LDFLAGS) -mod=vendor -o bin/keda-adapter cmd/adapter/main.go

webhooks: generate
	${GO_BUILD_VARS} go build -ldflags $(GO_LDFLAGS) -mod=vendor -o bin/keda-admission-webhooks cmd/webhooks/main.go

run: manifests generate ## Run a controller from your host.
	KEDA_CLUSTER_OBJECT_NAMESPACE=keda WATCH_NAMESPACE="" go run -ldflags $(GO_LDFLAGS) ./cmd/operator/main.go $(ARGS)

docker-build: ## Build docker images with the KEDA Operator and Metrics Server.
	DOCKER_BUILDKIT=1 docker build . -t ${IMAGE_CONTROLLER} --build-arg BUILD_VERSION=${VERSION} --build-arg GIT_VERSION=${GIT_VERSION} --build-arg GIT_COMMIT=${GIT_COMMIT}
	DOCKER_BUILDKIT=1 docker build -f Dockerfile.adapter -t ${IMAGE_ADAPTER} . --build-arg BUILD_VERSION=${VERSION} --build-arg GIT_VERSION=${GIT_VERSION} --build-arg GIT_COMMIT=${GIT_COMMIT}
	DOCKER_BUILDKIT=1 docker build -f Dockerfile.webhooks -t ${IMAGE_WEBHOOKS} . --build-arg BUILD_VERSION=${VERSION} --build-arg GIT_VERSION=${GIT_VERSION} --build-arg GIT_COMMIT=${GIT_COMMIT}

publish: docker-build ## Push images on to Container Registry (default: ghcr.io).
	docker push $(IMAGE_CONTROLLER)
	docker push $(IMAGE_ADAPTER)
	docker push $(IMAGE_WEBHOOKS)

publish-controller-multiarch: ## Build and push multi-arch Docker image for KEDA Operator.
	docker buildx build --output=type=${OUTPUT_TYPE} --platform=${BUILD_PLATFORMS} . -t ${IMAGE_CONTROLLER} --build-arg BUILD_VERSION=${VERSION} --build-arg GIT_VERSION=${GIT_VERSION} --build-arg GIT_COMMIT=${GIT_COMMIT}

publish-adapter-multiarch: ## Build and push multi-arch Docker image for KEDA Metrics Server.
	docker buildx build --output=type=${OUTPUT_TYPE} --platform=${BUILD_PLATFORMS} -f Dockerfile.adapter -t ${IMAGE_ADAPTER} . --build-arg BUILD_VERSION=${VERSION} --build-arg GIT_VERSION=${GIT_VERSION} --build-arg GIT_COMMIT=${GIT_COMMIT}

publish-webhooks-multiarch: ## Build and push multi-arch Docker image for KEDA Hooks.
	docker buildx build --output=type=${OUTPUT_TYPE} --platform=${BUILD_PLATFORMS} -f Dockerfile.webhooks -t ${IMAGE_WEBHOOKS} . --build-arg BUILD_VERSION=${VERSION} --build-arg GIT_VERSION=${GIT_VERSION} --build-arg GIT_COMMIT=${GIT_COMMIT}

publish-multiarch: publish-controller-multiarch publish-adapter-multiarch publish-webhooks-multiarch ## Push multi-arch Docker images on to Container Registry (default: ghcr.io).

release: manifests kustomize set-version generate-scalers-schema ## Produce new KEDA release in keda-$(VERSION).yaml file.
	cd config/manager && \
	$(KUSTOMIZE) edit set image ghcr.io/kedacore/keda=${IMAGE_CONTROLLER}
	cd config/metrics-server && \
    $(KUSTOMIZE) edit set image ghcr.io/kedacore/keda-metrics-apiserver=${IMAGE_ADAPTER}
	cd config/webhooks && \
    $(KUSTOMIZE) edit set image ghcr.io/kedacore/keda-admission-webhooks=${IMAGE_WEBHOOKS}
	# Need this workaround to mitigate a problem with inserting labels into selectors,
	# until this issue is solved: https://github.com/kubernetes-sigs/kustomize/issues/1009
	@sed -i".out" -e 's@version:[ ].*@version: $(VERSION)@g' config/default/kustomize-config/metadataLabelTransformer.yaml
	@sed -i".out" -e 's@version:[ ].*@version: $(VERSION)@g' config/minimal/kustomize-config/metadataLabelTransformer.yaml
	rm -rf config/default/kustomize-config/metadataLabelTransformer.yaml.out
	$(KUSTOMIZE) build config/default > keda-$(VERSION).yaml
	$(KUSTOMIZE) build config/minimal > keda-$(VERSION)-core.yaml
	$(KUSTOMIZE) build config/crd     > keda-$(VERSION)-crds.yaml

sign-images: ## Sign KEDA images published on GitHub Container Registry
	COSIGN_EXPERIMENTAL=1 cosign sign ${COSIGN_FLAGS} $(IMAGE_CONTROLLER)
	COSIGN_EXPERIMENTAL=1 cosign sign ${COSIGN_FLAGS} $(IMAGE_ADAPTER)
	COSIGN_EXPERIMENTAL=1 cosign sign ${COSIGN_FLAGS} $(IMAGE_WEBHOOKS)

.PHONY: set-version
set-version:
	@sed -i".out" -e 's@Version[ ]*=.*@Version = "$(VERSION)"@g' ./version/version.go;
	rm -rf ./version/version.go.out

.PHONY: generate-scalers-schema
generate-scalers-schema: ## Generate scalers schema
	GOBIN=$(LOCALBIN) go run ./schema/generate_scaler_schema.go --keda-version $(VERSION) --scalers-builder-file $(SCALERS_SCHEMA_SCALERS_BUILDER_FILE) --scalers-files-dir $(SCALERS_SCHEMA_SCALERS_FILES_DIR) --output-file-path $(SCALERS_SCHEMA_OUTPUT_FILE_PATH) --output-file-name $(SCALERS_SCHEMA_OUTPUT_FILE_NAME) --output-file-format both

.PHONY: verify-scalers-schema
verify-scalers-schema: ## Verify scalers schema
	./hack/verify-schema.sh

##################################################
# Deployment                                     #
##################################################

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply --server-side -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: install ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && \
	$(KUSTOMIZE) edit set image ghcr.io/kedacore/keda=${IMAGE_CONTROLLER} && \
	if [ "$(AZURE_RUN_WORKLOAD_IDENTITY_TESTS)" = true ]; then \
		$(KUSTOMIZE) edit add label --force azure.workload.identity/use:true; \
	fi
	cd config/metrics-server && \
    $(KUSTOMIZE) edit set image ghcr.io/kedacore/keda-metrics-apiserver=${IMAGE_ADAPTER}

	if [ "$(AZURE_RUN_WORKLOAD_IDENTITY_TESTS)" = true ]; then \
		cd config/service_account && \
		$(KUSTOMIZE) edit add label --force azure.workload.identity/use:true; \
		$(KUSTOMIZE) edit add annotation --force azure.workload.identity/client-id:${TF_AZURE_IDENTITY_1_APP_ID} azure.workload.identity/tenant-id:${TF_AZURE_SP_TENANT}; \
	fi
	if [ "$(AWS_RUN_IDENTITY_TESTS)" = true ]; then \
		cd config/service_account && \
		$(KUSTOMIZE) edit add annotation --force eks.amazonaws.com/role-arn:${TF_AWS_KEDA_ROLE}; \
	fi
	if [ "$(GCP_RUN_IDENTITY_TESTS)" = true ]; then \
		cd config/service_account && \
		$(KUSTOMIZE) edit add annotation --force cloud.google.com/workload-identity-provider:${GCP_WI_PROVIDER} cloud.google.com/service-account-email:${TF_GCP_SA_EMAIL} cloud.google.com/gcloud-run-as-user:${NON_ROOT_USER_ID} cloud.google.com/injection-mode:direct; \
	fi
	if [ "$(ENABLE_OPENTELEMETRY)" = true ]; then \
		cd config/e2e && \
		$(KUSTOMIZE) edit add patch --path opentelemetry/patch_operator.yml --group apps --kind Deployment --name keda-operator --version v1; \
	fi

	cd config/webhooks && \
	$(KUSTOMIZE) edit set image ghcr.io/kedacore/keda-admission-webhooks=${IMAGE_WEBHOOKS}

	# Need this workaround to mitigate a problem with inserting labels into selectors,
	# until this issue is solved: https://github.com/kubernetes-sigs/kustomize/issues/1009
	@sed -i".out" -e 's@version:[ ].*@version: $(VERSION)@g' config/default/kustomize-config/metadataLabelTransformer.yaml
	rm -rf config/default/kustomize-config/metadataLabelTransformer.yaml.out
	$(KUSTOMIZE) build config/e2e | kubectl apply -f -

undeploy: kustomize e2e-test-clean-crds ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/e2e | kubectl delete -f -
	make uninstall

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOTESTSUM ?= $(LOCALBIN)/gotestsum
MOCKGEN ?= $(LOCALBIN)/mockgen
PROTOCGEN ?= $(LOCALBIN)/protoc-gen-go
PROTOCGEN_GRPC ?= $(LOCALBIN)/protoc-gen-go-grpc
GO_JUNIT_REPORT ?= $(LOCALBIN)/go-junit-report

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Install controller-gen from vendor dir if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Install kustomize from vendor dir if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || GOBIN=$(LOCALBIN) go install sigs.k8s.io/kustomize/kustomize/v5

.PHONY: envtest
envtest: $(ENVTEST) ## Install envtest-setup from vendor dir if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest

.PHONY: gotestsum
gotestsum: $(GOTESTSUM) ## Install gotestsum from vendor dir if necessary.
$(GOTESTSUM): $(LOCALBIN)
	test -s $(LOCALBIN)/gotestsum || GOBIN=$(LOCALBIN) go install gotest.tools/gotestsum@latest

.PHONY: mockgen
mockgen: $(MOCKGEN) ## Install mockgen from vendor dir if necessary.
$(MOCKGEN): $(LOCALBIN)
	test -s $(LOCALBIN)/mockgen || GOBIN=$(LOCALBIN) go install go.uber.org/mock/mockgen

.PHONY: protoc-gen
protoc-gen: $(PROTOCGEN) $(PROTOCGEN_GRPC) ## Install protoc-gen from vendor dir if necessary.
$(PROTOCGEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install google.golang.org/protobuf/cmd/protoc-gen-go
$(PROTOCGEN_GRPC): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc

.PHONY: go-junit-report
go-junit-report: $(GO_JUNIT_REPORT) ## Install go-junit-report from vendor dir if necessary.
$(GO_JUNIT_REPORT): $(LOCALBIN)
	test -s $(LOCALBIN)/go-junit-report || GOBIN=$(LOCALBIN) go install github.com/jstemmer/go-junit-report/v2

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

.PHONY: docker-build-dev-containers
docker-build-dev-containers: ## Build dev-containers image
	docker build -f .devcontainer/Dockerfile .

.PHONY: validate-changelog
validate-changelog: ## Validate changelog
	./hack/validate-changelog.sh
