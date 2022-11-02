# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.18.8 AS builder

ARG BUILD_VERSION=main
ARG GIT_COMMIT=HEAD
ARG GIT_VERSION=main

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

COPY Makefile Makefile

# Copy the go source
COPY hack/ hack/
COPY version/ version/
COPY main.go main.go
COPY adapter/ adapter/
COPY apis/ apis/
COPY controllers/ controllers/
COPY pkg/ pkg/

# Build
# https://www.docker.com/blog/faster-multi-platform-builds-dockerfile-cross-compilation-guide/
ARG TARGETOS
ARG TARGETARCH
RUN VERSION=${BUILD_VERSION} GIT_COMMIT=${GIT_COMMIT} GIT_VERSION=${GIT_VERSION} TARGET_OS=$TARGETOS ARCH=$TARGETARCH make manager

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/bin/keda .
# 65532 is numeric for nonroot
USER 65532:65532

ENTRYPOINT ["/keda", "--zap-log-level=info", "--zap-encoder=console"]
