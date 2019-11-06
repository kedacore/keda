##################################################
# Variables                                      #
##################################################
IMAGE_TAG      ?= 0.0.4
IMAGE_REGISTRY ?= docker.io
IMAGE_REPO     ?= kedacore

IMAGE_CONTROLLER = $(IMAGE_REGISTRY)/$(IMAGE_REPO)/keda:$(IMAGE_TAG)
IMAGE_ADAPTER    = $(IMAGE_REGISTRY)/$(IMAGE_REPO)/keda-metrics-adapter:$(IMAGE_TAG)


ARCH       ?=amd64
CGO        ?=0
TARGET_OS  ?=linux

GIT_VERSION = $(shell git describe --always --abbrev=7)
GIT_COMMIT  = $(shell git rev-list -1 HEAD)
DATE        = $(shell date -u +"%Y.%m.%d.%H.%M.%S")

##################################################
# All                                            #
##################################################
.PHONY: All
all: test build

##################################################
# Tests                                          #
##################################################
.PHONY: test
test:
	go test ./...

.PHONY: e2e-test
e2e-test:
	TERMINFO=/etc/terminfo
	TERM=linux
	@az login --service-principal -u $(AZURE_SP_ID) -p "$(AZURE_SP_KEY)" --tenant $(AZURE_SP_TENANT)
	@az aks get-credentials \
		--name keda-nightly-run \
		--subscription $(AZURE_SUBSCRIPTION) \
		--resource-group $(AZURE_RESOURCE_GROUP)
	npm install --prefix tests
	npm test --verbose --prefix tests

##################################################
# PUBLISH                                        #
##################################################
.PHONY: publish
publish: build
	docker push $(IMAGE_ADAPTER)
	docker push $(IMAGE_CONTROLLER)

##################################################
# Build                                          #
##################################################
GO_BUILD_VARS= GO111MODULE=on CGO_ENABLED=$(CGO) GOOS=$(TARGET_OS) GOARCH=$(ARCH)

.PHONY: build
build: build-adapter build-controller

.PHONY: build-controller
build-controller: generate-api pkg/scalers/liiklus/LiiklusService.pb.go
	$(GO_BUILD_VARS) operator-sdk build $(IMAGE_CONTROLLER) \
		--go-build-args "-ldflags -X=main.GitCommit=$(GIT_COMMIT)"

.PHONY: build-adapter
build-adapter: generate-api pkg/scalers/liiklus/LiiklusService.pb.go
	$(GO_BUILD_VARS) go build \
		-ldflags "-X=main.GitCommit=$(GIT_COMMIT)" \
		-o build/_output/bin/keda-adapter \
		cmd/adapter/main.go
	docker build -f build/Dockerfile.adapter -t $(IMAGE_ADAPTER) .

.PHONY: generate-api
generate-api:
	$(GO_BUILD_VARS) operator-sdk generate k8s
	$(GO_BUILD_VARS) operator-sdk generate openapi

pkg/scalers/liiklus/LiiklusService.pb.go: hack/LiiklusService.proto
	protoc -I hack/ hack/LiiklusService.proto --go_out=plugins=grpc:pkg/scalers/liiklus

pkg/scalers/liiklus/mocks/mock_liiklus.go: pkg/scalers/liiklus/LiiklusService.pb.go
	mockgen github.com/kedacore/keda/pkg/scalers/liiklus LiiklusServiceClient > pkg/scalers/liiklus/mocks/mock_liiklus.go

##################################################
# Helm Chart tasks                               #
##################################################
.PHONY: build-chart-edge
build-chart-edge:
	rm -rf /tmp/keda-edge
	cp -r -L chart/keda /tmp/keda-edge
	sed -i "s/^name:.*/name: keda-edge/g" /tmp/keda-edge/Chart.yaml
	sed -i "s/^version:.*/version: $(IMAGE_TAG)-$(DATE)-$(GIT_VERSION)/g" /tmp/keda-edge/Chart.yaml
	sed -i "s/^appVersion:.*/appVersion: $(GIT_VERSION)/g" /tmp/keda-edge/Chart.yaml

	helm lint /tmp/keda-edge/
	helm package /tmp/keda-edge/

.PHONY: publish-edge-chart
publish-edge-chart: build-chart-edge
	$(eval CHART := $(shell find . -maxdepth 1 -type f -iname 'keda-edge-$(IMAGE_TAG)-*' -print -quit))
	@az storage blob upload \
		--container-name helm \
		--name $(CHART) \
		--file $(CHART) \
		--account-name kedacore \
		--sas-token "$(STORAGE_HELM_SAS_TOKEN)"

	@az storage blob download \
		--container-name helm \
		--name index.yaml \
		--file old_index.yaml \
		--account-name kedacore \
		--sas-token "$(STORAGE_HELM_SAS_TOKEN)" 2>/dev/null | true

	[ -s ./old_index.yaml ] && helm repo index . --url https://kedacore.azureedge.net/helm --merge old_index.yaml || true
	[ ! -s ./old_index.yaml ] && helm repo index . --url https://kedacore.azureedge.net/helm || true

	@az storage blob upload \
		--container-name helm \
		--name index.yaml \
		--file index.yaml \
		--account-name kedacore \
		--sas-token "$(STORAGE_HELM_SAS_TOKEN)"
