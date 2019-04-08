##################################################
# Variables                                      #
##################################################
ARCH?=amd64
CGO?=0
TARGET_OS?=linux

##################################################
# Variables                                      #
##################################################

BASE_IMAGE_NAME := kore
IMAGE_TAG       := $(CIRCLE_BRANCH)
IMAGE_NAME      := $(ACR_REGISTRY)/$(BASE_IMAGE_NAME):$(IMAGE_TAG)

GIT_VERSION = $(shell git describe --always --abbrev=7)
GIT_COMMIT  = $(shell git rev-list -1 HEAD)
DATE        = $(shell date -u +"%Y.%m.%d.%H.%M.%S")

##################################################
# Tests                                          #
##################################################
.PHONY: test
test:
	# Add actual test script
	go test ./...

.PHONY: e2e-test
e2e-test:
	./tests/run_tests.sh

##################################################
# Build                                          #
##################################################
.PHONY: ci-build-all
ci-build-all: build-container push-container

.PHONY: build
build:
	CGO_ENABLED=$(CGO) GOOS=$(TARGET_OS) GOARCH=$(ARCH) go build \
		-ldflags "-X main.GitCommit=$(GIT_COMMIT)" \
		-o dist/kore \
		cmd/main.go

.PHONY: build-container
build-container:
	docker build -t $(IMAGE_NAME) .

.PHONY: push-container
push-container: build-container
	docker push $(IMAGE_NAME)


##################################################
# Helm Chart tasks                               #
##################################################
.PHONY: build-chart-edge
build-chart-edge:
	rm -rf /tmp/kore-edge
	cp -r -L chart/kore /tmp/kore-edge
	sed -i "s/^name:.*/name: kore-edge/g" /tmp/kore-edge/Chart.yaml
	sed -i "s/^version:.*/version: 0.0.1-$(DATE)-$(GIT_VERSION)/g" /tmp/kore-edge/Chart.yaml
	sed -i "s/^appVersion:.*/appVersion: $(GIT_VERSION)/g" /tmp/kore-edge/Chart.yaml
	sed -i "s/^  tag:.*/  tag: master/g" /tmp/kore-edge/values.yaml

	helm lint /tmp/kore-edge/
	helm package /tmp/kore-edge/

.PHONY: publish-edge-chart
publish-edge-chart: build-chart-edge
	az acr helm push -n korecr $(shell find . -maxdepth 1 -type f -iname 'kore-edge-0.0.1-*' -print -quit)
