// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 The Kubernetes Authors

package remote

import (
	"context"
	"io"

	"sigs.k8s.io/controller-runtime/tools/setup-envtest/versions"
)

// Client is an interface to get and list envtest binary archives.
type Client interface {
	ListVersions(ctx context.Context) ([]versions.Set, error)

	GetVersion(ctx context.Context, version versions.Concrete, platform versions.PlatformItem, out io.Writer) error

	FetchSum(ctx context.Context, ver versions.Concrete, pl *versions.PlatformItem) error
}
