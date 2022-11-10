//go:build tools
// +build tools

// Package tools imports things required by build scripts, to force `go mod` to see them as dependencies
package tools

import (
	// Import code-generator to use in build tools
	_ "github.com/golang/mock/mockgen"
	_ "k8s.io/code-generator"
	_ "k8s.io/kube-openapi/cmd/openapi-gen"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
	_ "sigs.k8s.io/kustomize/kustomize/v4"
)
