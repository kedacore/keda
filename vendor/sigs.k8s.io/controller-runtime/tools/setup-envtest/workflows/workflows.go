// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 The Kubernetes Authors

package workflows

import (
	"context"
	"io"

	"github.com/go-logr/logr"

	envp "sigs.k8s.io/controller-runtime/tools/setup-envtest/env"
)

// Use is a workflow that prints out information about stored
// version-platform pairs, downloading them if necessary & requested.
type Use struct {
	UseEnv      bool
	AssetsPath  string
	PrintFormat envp.PrintFormat
}

// Do executes this workflow.
func (f Use) Do(env *envp.Env) {
	ctx := logr.NewContext(context.TODO(), env.Log.WithName("use"))
	env.EnsureBaseDirs(ctx)
	if f.UseEnv {
		// the env var unconditionally
		if env.PathMatches(f.AssetsPath) {
			env.PrintInfo(f.PrintFormat)
			return
		}
	}
	env.EnsureVersionIsSet(ctx)
	if env.ExistsAndValid() {
		env.PrintInfo(f.PrintFormat)
		return
	}
	if env.NoDownload {
		envp.Exit(2, "no such version (%s) exists on disk for this architecture (%s) -- try running `list -i` to see what's on disk", env.Version, env.Platform)
	}
	env.Fetch(ctx)
	env.PrintInfo(f.PrintFormat)
}

// List is a workflow that lists version-platform pairs in the store
// and on the remote server that match the given filter.
type List struct{}

// Do executes this workflow.
func (List) Do(env *envp.Env) {
	ctx := logr.NewContext(context.TODO(), env.Log.WithName("list"))
	env.EnsureBaseDirs(ctx)
	env.ListVersions(ctx)
}

// Cleanup is a workflow that removes version-platform pairs from the store
// that match the given filter.
type Cleanup struct{}

// Do executes this workflow.
func (Cleanup) Do(env *envp.Env) {
	ctx := logr.NewContext(context.TODO(), env.Log.WithName("cleanup"))

	env.NoDownload = true
	env.ForceDownload = false

	env.EnsureBaseDirs(ctx)
	env.Remove(ctx)
}

// Sideload is a workflow that adds or replaces a version-platform pair in the
// store, using the given archive as the files.
type Sideload struct {
	Input       io.Reader
	PrintFormat envp.PrintFormat
}

// Do executes this workflow.
func (f Sideload) Do(env *envp.Env) {
	ctx := logr.NewContext(context.TODO(), env.Log.WithName("sideload"))

	env.EnsureBaseDirs(ctx)
	env.NoDownload = true
	env.Sideload(ctx, f.Input)
	env.PrintInfo(f.PrintFormat)
}
