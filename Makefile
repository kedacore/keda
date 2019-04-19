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
IMAGE_TAG       := lecattar-dev
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
	$(eval CHART := $(shell find . -maxdepth 1 -type f -iname 'kore-edge-0.0.1-*' -print -quit))
	$(eval CS := $(shell az storage account show-connection-string --name projectkore --resource-group projectkore --subscription bfc7797c-d43a-4296-937f-93b8de26ba2b  --output json --query "connectionString"))
	@az storage blob upload \
		--container-name helm \
		--name $(CHART) \
		--file $(CHART) \
		--connection-string $(CS)

	@az storage blob download \
		--container-name helm \
		--name index.yaml \
		--file old_index.yaml \
		--connection-string $(CS) 2>/dev/null | true

	[ -s ./old_index.yaml ] && helm repo index . --url https://projectkore.blob.core.windows.net/helm --merge old_index.yaml || true
	[ ! -s ./old_index.yaml ] && helm repo index . --url https://projectkore.blob.core.windows.net/helm || true

	@az storage blob upload \
		--container-name helm \
		--name index.yaml \
		--file index.yaml \
		--connection-string $(CS)
