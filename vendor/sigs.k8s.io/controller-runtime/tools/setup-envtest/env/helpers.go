// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 The Kubernetes Authors

package env

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/tools/setup-envtest/versions"
)

// orderPlatforms orders platforms by OS then arch.
func orderPlatforms(first, second versions.Platform) bool {
	// sort by OS, then arch
	if first.OS != second.OS {
		return first.OS < second.OS
	}
	return first.Arch < second.Arch
}

// PrintFormat indicates how to print out fetch and switch results.
// It's a valid pflag.Value so it can be used as a flag directly.
type PrintFormat int

const (
	// PrintOverview prints human-readable data,
	// including path, version, arch, and checksum (when available).
	PrintOverview PrintFormat = iota
	// PrintPath prints *only* the path, with no decoration.
	PrintPath
	// PrintEnv prints the path with the corresponding env variable, so that
	// you can source the output like
	// `source $(fetch-envtest switch -p env 1.20.x)`.
	PrintEnv
)

func (f PrintFormat) String() string {
	switch f {
	case PrintOverview:
		return "overview"
	case PrintPath:
		return "path"
	case PrintEnv:
		return "env"
	default:
		panic(fmt.Sprintf("unexpected print format %d", int(f)))
	}
}

// Set sets the value of this as a flag.
func (f *PrintFormat) Set(val string) error {
	switch val {
	case "overview":
		*f = PrintOverview
	case "path":
		*f = PrintPath
	case "env":
		*f = PrintEnv
	default:
		return fmt.Errorf("unknown print format %q, use one of overview|path|env", val)
	}
	return nil
}

// Type is the type of this value as a flag.
func (PrintFormat) Type() string {
	return "{overview|path|env}"
}
