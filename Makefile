ARCH?=amd64

.PHONY: build
build: build-go

.PHONY: build-go
build-go:
	CGO_ENABLED=0 GOARCH=$(ARCH) go build -o dist/kore cmd/main.go