# Build the manager binary
FROM --platform=$BUILDPLATFORM ghcr.io/kedacore/keda-tools:1.26.6@sha256:7db93d7d2720c0ddbc67d35291cf8ba8f34a28c3125d08b0f93fe4c7f2e6debc AS builder

ARG BUILD_VERSION=main
ARG GIT_COMMIT=HEAD
ARG GIT_VERSION=main

WORKDIR /workspace

COPY go.mod go.sum ./
ARG GOMODCACHE=/go/pkg/mod
RUN --mount=type=cache,target=${GOMODCACHE} \
  go mod download

COPY Makefile Makefile

# Copy the go source
COPY hack/ hack/
COPY version/ version/
COPY cmd/ cmd/
COPY apis/ apis/
COPY controllers/ controllers/
COPY pkg/ pkg/
COPY schema/ schema/

# Build
# https://www.docker.com/blog/faster-multi-platform-builds-dockerfile-cross-compilation-guide/
ARG TARGETOS
ARG TARGETARCH
ARG GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target=${GOCACHE} --mount=type=cache,target=${GOMODCACHE} \
  VERSION=${BUILD_VERSION} GIT_COMMIT=${GIT_COMMIT} GIT_VERSION=${GIT_VERSION} TARGET_OS=$TARGETOS ARCH=$TARGETARCH \
  make manager

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot@sha256:f7f8f729987ad0fdf6b05eeeae94b26e6a0f613bdf46feea7fc40f7bd72953e6
WORKDIR /
COPY --from=builder /workspace/bin/keda .
# 65532 is numeric for nonroot
USER 65532:65532

ENTRYPOINT ["/keda", "--zap-log-level=info", "--zap-encoder=console"]
