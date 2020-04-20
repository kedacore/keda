##################################################
# Variables                                      #
##################################################
VERSION		   ?= master
IMAGE_REGISTRY ?= docker.io
IMAGE_REPO     ?= kedacore

IMAGE_CONTROLLER = $(IMAGE_REGISTRY)/$(IMAGE_REPO)/keda:$(VERSION)
IMAGE_ADAPTER    = $(IMAGE_REGISTRY)/$(IMAGE_REPO)/keda-metrics-adapter:$(VERSION)

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
	IMAGE_CONTROLLER=$(IMAGE_CONTROLLER) IMAGE_ADAPTER=$(IMAGE_ADAPTER) npm test --verbose --prefix tests

##################################################
# PUBLISH                                        #
##################################################
.PHONY: publish
publish: build
	docker push $(IMAGE_ADAPTER)
	docker push $(IMAGE_CONTROLLER)

##################################################
# Release                                        #
##################################################
K8S_DEPLOY_FILES = $(shell find ./deploy -name '*.yaml')

.PHONY: release
release: release_prepare release_file release_pkg

.PHONY: release_file
release_file:
	@sed -i 's@Version   =.*@Version   = "$(VERSION)"@g' ./version/version.go;
	@for file in $(K8S_DEPLOY_FILES); do \
	sed -i 's@app.kubernetes.io/version:.*@app.kubernetes.io/version: "$(VERSION)"@g' $$file; \
	sed -i 's@image: docker.io/kedacore/keda:.*@image: docker.io/kedacore/keda:$(VERSION)@g' $$file; \
	sed -i 's@image: docker.io/kedacore/keda-metrics-adapter:.*@image: docker.io/kedacore/keda-metrics-adapter:$(VERSION)@g' $$file; \
	done

.PHONY: release_prepare
release_prepare:
	rm -rf ./keda-$(VERSION)
	rm -f ./keda-$(VERSION).tar.gz
	rm -f ./keda-$(VERSION).zip
	mkdir -p ./keda-$(VERSION)/crds

.PHONY: release_pkg
release_pkg:
	cp -r ./deploy/*.yaml ./keda-$(VERSION)
	cp ./deploy/crds/*_crd.yaml ./keda-$(VERSION)/crds
	tar -z -cf ./keda-$(VERSION).tar.gz keda-$(VERSION)/
	zip -r ./keda-$(VERSION).zip keda-$(VERSION)/
	rm -rf ./keda-$(VERSION)

##################################################
# Build                                          #
##################################################
GO_BUILD_VARS= GO111MODULE=on CGO_ENABLED=$(CGO) GOOS=$(TARGET_OS) GOARCH=$(ARCH)

.PHONY: checkenv
checkenv:
ifndef GOROOT
	@echo "WARNING: GOROOT is not defined"
endif

.PHONY: build
build: checkenv build-adapter build-controller

.PHONY: build-controller
build-controller: generate-api pkg/scalers/liiklus/LiiklusService.pb.go
	$(GO_BUILD_VARS) operator-sdk build $(IMAGE_CONTROLLER) \
		--go-build-args "-ldflags -X=github.com/kedacore/keda/version.Version=$(VERSION) -o build/_output/bin/keda"

.PHONY: build-adapter
build-adapter: generate-api pkg/scalers/liiklus/LiiklusService.pb.go
	$(GO_BUILD_VARS) go build \
		-ldflags "-X=github.com/kedacore/keda/version.GitCommit=$(GIT_COMMIT) -X=github.com/kedacore/keda/version.Version=$(VERSION)" \
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
