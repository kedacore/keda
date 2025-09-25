//go:build tools
// +build tools

// Package tools imports things required by build scripts, to force `go mod` to see them as dependencies
package tools

import (
	// Import code-generator to use in build tools
	_ "github.com/jstemmer/go-junit-report/v2"
	_ "go.uber.org/mock/mockgen"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
	_ "gotest.tools/gotestsum"
	_ "k8s.io/code-generator"
	_ "k8s.io/code-generator/cmd/validation-gen"
	_ "k8s.io/kube-openapi/cmd/openapi-gen"
	_ "sigs.k8s.io/controller-runtime/tools/setup-envtest"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
	_ "sigs.k8s.io/kustomize/kustomize/v5"
)
