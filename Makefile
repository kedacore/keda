##################################################
# Variables                                      #
##################################################
VERSION		   ?= v2
IMAGE_REGISTRY ?= docker.io
IMAGE_REPO     ?= kedacore

IMAGE_CONTROLLER = $(IMAGE_REGISTRY)/$(IMAGE_REPO)/keda:$(VERSION)
IMAGE_ADAPTER    = $(IMAGE_REGISTRY)/$(IMAGE_REPO)/keda-metrics-adapter:$(VERSION)

IMAGE_BUILD_TOOLS = $(IMAGE_REGISTRY)/$(IMAGE_REPO)/build-tools:v2

ARCH       ?=amd64
CGO        ?=0
TARGET_OS  ?=linux

GIT_VERSION = $(shell git describe --always --abbrev=7)
GIT_COMMIT  = $(shell git rev-list -1 HEAD)
DATE        = $(shell date -u +"%Y.%m.%d.%H.%M.%S")

TEST_CLUSTER_NAME ?= keda-nightly-run-2

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

GO_BUILD_VARS= GO111MODULE=on CGO_ENABLED=$(CGO) GOOS=$(TARGET_OS) GOARCH=$(ARCH)

##################################################
# All                                            #
##################################################
.PHONY: All
all: test build

##################################################
# Tests                                          #
##################################################
.PHONY: test
test: generate gofmt govet
	go test ./... -covermode=atomic -coverprofile cover.out

.PHONY: e2e-test
e2e-test:
	TERMINFO=/etc/terminfo
	TERM=linux
	@az login --service-principal -u $(AZURE_SP_ID) -p "$(AZURE_SP_KEY)" --tenant $(AZURE_SP_TENANT)
	@az aks get-credentials \
		--name $(TEST_CLUSTER_NAME) \
		--subscription $(AZURE_SUBSCRIPTION) \
		--resource-group $(AZURE_RESOURCE_GROUP)
	npm install --prefix tests

	./tests/run-all.sh

##################################################
# PUBLISH                                        #
##################################################
.PHONY: publish
publish: docker-build
	docker push $(IMAGE_CONTROLLER)
	docker push $(IMAGE_ADAPTER)

##################################################
# Release                                        #
##################################################
.PHONY: release
release: manifests kustomize
	cd config/manager && \
	$(KUSTOMIZE) edit set image docker.io/kedacore/keda=${IMAGE_CONTROLLER}
	cd config/metrics-server && \
    $(KUSTOMIZE) edit set image docker.io/kedacore/keda-metrics-adapter=${IMAGE_ADAPTER}
	cd config/default && \
    $(KUSTOMIZE) edit add label -f app.kubernetes.io/version:${VERSION}
	$(KUSTOMIZE) build config/default > keda-$(VERSION).yaml

.PHONY: set-version
set-version:
	@sed -i 's@Version   =.*@Version   = "$(VERSION)"@g' ./version/version.go;

##################################################
# RUN / (UN)INSTALL / DEPLOY                     #
##################################################
# Run against the configured Kubernetes cluster in ~/.kube/config
.PHONY: run
run: generate gofmt govet manifests
	go run \
	-ldflags "-X=github.com/kedacore/keda/version.GitCommit=$(GIT_COMMIT) -X=github.com/kedacore/keda/version.Version=$(VERSION)" \
	./main.go $(ARGS)

# Install CRDs into a cluster
.PHONY: install
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
.PHONY: uninstall
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
.PHONY: deploy
deploy: manifests kustomize
	cd config/manager && \
	$(KUSTOMIZE) edit set image docker.io/kedacore/keda=${IMAGE_CONTROLLER}
	cd config/metrics-server && \
    $(KUSTOMIZE) edit set image docker.io/kedacore/keda-metrics-adapter=${IMAGE_ADAPTER}
	cd config/default && \
    $(KUSTOMIZE) edit add label -f app.kubernetes.io/version:${VERSION}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

# Undeploy controller
.PHONY: undeploy
undeploy:
	$(KUSTOMIZE) build config/default | kubectl delete -f -

##################################################
# Build                                          #
##################################################
.PHONY: build
build: manifests set-version manager adapter

# Build the docker image
docker-build: build
	docker build . -t ${IMAGE_CONTROLLER}
	docker build -f Dockerfile.adapter -t ${IMAGE_ADAPTER} .

# Build KEDA Operator binary
.PHONY: manager
manager: generate gofmt govet pkg/scalers/liiklus/LiiklusService.pb.go
	${GO_BUILD_VARS} go build \
	-ldflags "-X=github.com/kedacore/keda/version.GitCommit=$(GIT_COMMIT) -X=github.com/kedacore/keda/version.Version=$(VERSION)" \
	-o bin/keda main.go

# Build KEDA Metrics Server Adapter binary
.PHONY: adapter
adapter: generate gofmt govet pkg/scalers/liiklus/LiiklusService.pb.go
	${GO_BUILD_VARS} go build \
	-ldflags "-X=github.com/kedacore/keda/version.GitCommit=$(GIT_COMMIT) -X=github.com/kedacore/keda/version.Version=$(VERSION)" \
	-o bin/keda-adapter adapter/main.go

# Generate manifests e.g. CRD, RBAC etc.
.PHONY: manifests
manifests: controller-gen
	$(CONTROLLER_GEN) crd:crdVersions=v1beta1 rbac:roleName=keda-operator paths="./..." output:crd:artifacts:config=config/crd/bases
	# withTriggers is only used for duck typing so we only need the deepcopy methods
	# However operator-sdk generate doesn't appear to have an option for that
	# until this issue is fixed: https://github.com/kubernetes-sigs/controller-tools/issues/398
	rm config/crd/bases/keda.sh_withtriggers.yaml

# Generate code (API)
.PHONY: generate
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# find or download controller-gen
# download controller-gen if necessary
.PHONY: controller-gen
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	cd / ;\
	GO111MODULE=on go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

# find or download kustomize
.PHONY: kustomize
kustomize:
ifeq (, $(shell which kustomize))
	@{ \
	cd / ;\
	GO111MODULE=on go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

# Generate Liiklus proto
pkg/scalers/liiklus/LiiklusService.pb.go: hack/LiiklusService.proto
	protoc -I hack/ hack/LiiklusService.proto --go_out=plugins=grpc:pkg/scalers/liiklus

pkg/scalers/liiklus/mocks/mock_liiklus.go: pkg/scalers/liiklus/LiiklusService.pb.go
	mockgen github.com/kedacore/keda/pkg/scalers/liiklus LiiklusServiceClient > pkg/scalers/liiklus/mocks/mock_liiklus.go

# Run go fmt against code
.PHONY: gofmt
gofmt:
	go fmt ./...

# Run go vet against code
.PHONY: govet
govet:
	go vet ./...

##################################################
# Clientset                                      #
##################################################
# Kubebuilder project layout has API under 'api/v1alpha1'
# client-go codegen expects group name (keda) in the path ie. 'api/keda/v1alpha'
# Because there's no way how to modify any of these settings,
# we need to hack things a little bit (use tmp directory 'api/keda/v1alpha' and replace the name of package)
.PHONY: clientset-prepare
clientset-prepare:
	go mod vendor
	rm -rf api/keda
	mkdir api/keda
	cp -r api/v1alpha1 api/keda/v1alpha1

.PHONY: clientset-verify
clientset-verify: clientset-prepare
	./hack/verify-codegen.sh
	rm -rf api/keda

.PHONY: clientset-generate
clientset-generate: clientset-prepare
	./hack/update-codegen.sh
	find ./pkg/generated -type f -name "*.go" |\
	xargs sed -i "s#github.com/kedacore/keda/api/keda/v1alpha1#github.com/kedacore/keda/api/v1alpha1#g"
	rm -rf api/keda

##################################################
# Build Tools Image                              #
##################################################
.PHONY: publish-build-tools
publish-build-tools:
	docker build -f tools/build-tools.Dockerfile -t $(IMAGE_BUILD_TOOLS) .
	docker push $(IMAGE_BUILD_TOOLS)
