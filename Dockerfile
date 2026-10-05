# Build the manager binary
FROM --platform=$BUILDPLATFORM ghcr.io/kedacore/keda-tools:1.26.8@sha256:43316d99880b57ee18ad12f119acb37bbea7be99c73219166a529bbfa7b9340c AS builder

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
FROM gcr.io/distroless/static:nonroot@sha256:e2e927ec666bae08560abb3c55d0659eceabb657f56b6782ab500a9fc7f555e3
WORKDIR /
COPY --from=builder /workspace/bin/keda .
# 65532 is numeric for nonroot
USER 65532:65532

ENTRYPOINT ["/keda", "--zap-log-level=info", "--zap-encoder=console"]
