// +build tools

// Package tools imports things required by build scripts, to force `go mod` to see them as dependencies
package tools

import (
	// Import code-generator to use in build tools
	_ "k8s.io/code-generator"
)
