##################################################
# Variables                                      #
##################################################
ARCH?=amd64
CGO?=0
TARGET_OS?=linux

##################################################
# Docker variables                               #
##################################################

BASE_IMAGE_NAME := kore
IMAGE_NAMESPACE := project-kore
IMAGE_TAG       := $(CIRCLE_BRANCH)

IMAGE_NAME      := $(ACR_REGISTRY)/$(IMAGE_NAMESPACE)/$(BASE_IMAGE_NAME):$(IMAGE_TAG)

##################################################
# Tests                                          #
##################################################
.PHONY: test
test:
	# Add actual test script
	go test ./...

##################################################
# Build                                          #
##################################################
.PHONY: ci-build-all
ci-build-all: build build-container push-container

.PHONY: build
build:
	CGO_ENABLED=$(CGO) GOOS=$(TARGET_OS) GOARCH=$(ARCH) go build -o dist/kore cmd/main.go

.PHONY: build-container
build-container: build
	docker build -t $(IMAGE_NAME) .

.PHONY: push-container
push-container: build-container
	docker push $(IMAGE_NAME)

