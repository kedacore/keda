// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 The Kubernetes Authors

package env

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/go-logr/logr"
	"github.com/spf13/afero" // too bad fs.FS isn't writable :-/

	"sigs.k8s.io/controller-runtime/tools/setup-envtest/remote"
	"sigs.k8s.io/controller-runtime/tools/setup-envtest/store"
	"sigs.k8s.io/controller-runtime/tools/setup-envtest/versions"
)

// Env represents an environment for downloading and otherwise manipulating
// envtest binaries.
//
// In general, the methods will use the Exit{,Cause} functions from this package
// to indicate errors. Catch them with a `defer HandleExitWithCode()`.
type Env struct {
	// the following *must* be set on input

	// Platform is our current platform
	Platform versions.PlatformItem

	// VerifySum indicates whether we should run checksums.
	VerifySum bool
	// NoDownload forces us to not contact remote services,
	// looking only at local files instead.
	NoDownload bool
	// ForceDownload forces us to ignore local files and always
	// contact remote services & re-download.
	ForceDownload bool

	// Client is our remote client for contacting remote services.
	Client remote.Client

	// Log allows us to log.
	Log logr.Logger

	// the following *may* be set on input, or may be discovered

	// Version is the version(s) that we want to download
	// (may be automatically retrieved later on).
	Version versions.Spec

	// Store is used to load/store entries to/from disk.
	Store *store.Store

	// FS is the file system to read from/write to for provisioning temp files
	// for storing the archives temporarily.
	FS afero.Afero

	// Out is the place to write output text to
	Out io.Writer

	// manualPath is the manually discovered path from PathMatches, if
	// a non-store path was used.  It'll be printed by PrintInfo if present.
	manualPath string
}

// CheckCoherence checks that this environment has filled-out, coherent settings
// (e.g. NoDownload & ForceDownload aren't both set).
func (e *Env) CheckCoherence() {
	if e.NoDownload && e.ForceDownload {
		Exit(2, "cannot both skip downloading *and* force re-downloading")
	}

	if e.Platform.OS == "" || e.Platform.Arch == "" {
		Exit(2, "must specify non-empty OS and arch (did you specify bad --os or --arch values?)")
	}
}

func (e *Env) filter() store.Filter {
	return store.Filter{Version: e.Version, Platform: e.Platform.Platform}
}

func (e *Env) item() store.Item {
	concreteVer := e.Version.AsConcrete()
	if concreteVer == nil || e.Platform.IsWildcard() {
		panic("no platform/version set") // unexpected, print stack trace
	}
	return store.Item{Version: *concreteVer, Platform: e.Platform.Platform}
}

// ListVersions prints out all available versions matching this Env's
// platform & version selector (respecting NoDownload to figure
// out whether or not to match remote versions).
func (e *Env) ListVersions(ctx context.Context) {
	out := tabwriter.NewWriter(e.Out, 4, 4, 2, ' ', 0)
	defer out.Flush()
	localVersions, err := e.Store.List(ctx, e.filter())
	if err != nil {
		ExitCause(2, err, "unable to list installed versions")
	}
	for _, item := range localVersions {
		// already filtered by onDiskVersions
		fmt.Fprintf(out, "(installed)\tv%s\t%s\n", item.Version, item.Platform)
	}

	if e.NoDownload {
		return
	}

	remoteVersions, err := e.Client.ListVersions(ctx)
	if err != nil {
		ExitCause(2, err, "unable list to available versions")
	}

	for _, set := range remoteVersions {
		if !e.Version.Matches(set.Version) {
			continue
		}
		sort.Slice(set.Platforms, func(i, j int) bool {
			return orderPlatforms(set.Platforms[i].Platform, set.Platforms[j].Platform)
		})
		for _, plat := range set.Platforms {
			if e.Platform.Matches(plat.Platform) {
				fmt.Fprintf(out, "(available)\tv%s\t%s\n", set.Version, plat)
			}
		}
	}
}

// LatestVersion returns the latest version matching our version selector and
// platform from the remote server, with the corresponding checksum for later
// use as well.
func (e *Env) LatestVersion(ctx context.Context) (versions.Concrete, versions.PlatformItem) {
	vers, err := e.Client.ListVersions(ctx)
	if err != nil {
		ExitCause(2, err, "unable to list versions to find latest one")
	}
	for _, set := range vers {
		if !e.Version.Matches(set.Version) {
			e.Log.V(1).Info("skipping non-matching version", "version", set.Version)
			continue
		}
		// double-check that our platform is supported
		for _, plat := range set.Platforms {
			// NB(directxman12): we're already iterating in order, so no
			// need to check if the wildcard is latest vs any
			if e.Platform.Matches(plat.Platform) && e.Version.Matches(set.Version) {
				return set.Version, plat
			}
		}
		e.Log.Info("latest version not supported for your platform, checking older ones", "version", set.Version, "platform", e.Platform)
	}

	Exit(2, "unable to find a version that was supported for platform %s", e.Platform)
	return versions.Concrete{}, versions.PlatformItem{} // unreachable, but Go's type system can't express the "never" type
}

// ExistsAndValid checks if our current (concrete) version & platform
// exist on disk (unless ForceDownload is set, in which cause it always
// returns false).
//
// Must be called after EnsureVersionIsSet so that we have a concrete
// Version selected.  Must have a concrete platform, or ForceDownload
// must be set.
func (e *Env) ExistsAndValid() bool {
	if e.ForceDownload {
		// we always want to download, so don't check here
		return false
	}

	if e.Platform.IsWildcard() {
		Exit(2, "you must have a concrete platform with this command -- you cannot use wildcard platforms with fetch or switch")
	}

	exists, err := e.Store.Has(e.item())
	if err != nil {
		ExitCause(2, err, "unable to check if existing version exists")
	}

	if exists {
		e.Log.Info("applicable version found on disk", "version", e.Version)
	}
	return exists
}

// EnsureVersionIsSet ensures that we have a non-wildcard version
// configured.
//
// If necessary, it will enumerate on-disk and remote versions to accomplish
// this, finding a version that matches our version selector and platform.
// It will always yield a concrete version, it *may* yield a concrete platform
// as well.
func (e *Env) EnsureVersionIsSet(ctx context.Context) {
	if e.Version.AsConcrete() != nil {
		return
	}
	var localVer *versions.Concrete
	var localPlat versions.Platform

	items, err := e.Store.List(ctx, e.filter())
	if err != nil {
		ExitCause(2, err, "unable to determine installed versions")
	}

	for _, item := range items {
		if !e.Version.Matches(item.Version) || !e.Platform.Matches(item.Platform) {
			e.Log.V(1).Info("skipping version, doesn't match", "version", item.Version, "platform", item.Platform)
			continue
		}
		// NB(directxman12): we're already iterating in order, so no
		// need to check if the wildcard is latest vs any
		ver := item.Version // copy to avoid referencing iteration variable
		localVer = &ver
		localPlat = item.Platform
		break
	}

	if e.NoDownload || !e.Version.CheckLatest {
		// no version specified, but we either
		//
		// a) shouldn't contact remote
		// b) don't care to find the absolute latest
		//
		// so just find the latest local version
		if localVer != nil {
			e.Version.MakeConcrete(*localVer)
			e.Platform.Platform = localPlat
			return
		}
		if e.NoDownload {
			Exit(2, "no applicable on-disk versions for %s found, you'll have to download one, or run list -i to see what you do have", e.Platform)
		}
		// if we didn't ask for the latest version, but don't have anything
		// available, try the internet ;-)
	}

	// no version specified and we need the latest in some capacity, so find latest from remote
	// so find the latest local first, then compare it to the latest remote, and use whichever
	// of the two is more recent.
	e.Log.Info("no version specified, finding latest")
	serverVer, platform := e.LatestVersion(ctx)

	// if we're not forcing a download, and we have a newer local version, just use that
	if !e.ForceDownload && localVer != nil && localVer.NewerThan(serverVer) {
		e.Platform.Platform = localPlat // update our data with hash
		e.Version.MakeConcrete(*localVer)
		return
	}

	// otherwise, use the new version from the server
	e.Platform = platform // update our data with hash
	e.Version.MakeConcrete(serverVer)
}

// Fetch ensures that the requested platform and version are on disk.
// You must call EnsureVersionIsSet before calling this method.
//
// If ForceDownload is set, we always download, otherwise we only download
// if we're missing the version on disk.
func (e *Env) Fetch(ctx context.Context) {
	log := e.Log.WithName("fetch")

	// if we didn't just fetch it, grab the sum to verify
	if e.VerifySum && e.Platform.Hash == nil {
		if err := e.Client.FetchSum(ctx, *e.Version.AsConcrete(), &e.Platform); err != nil {
			ExitCause(2, err, "unable to fetch hash for requested version")
		}
	}
	if !e.VerifySum {
		e.Platform.Hash = nil // skip verification
	}

	var packedPath string

	// cleanup on error (needs to be here so it will happen after the other defers)
	defer e.cleanupOnError(func() {
		if packedPath != "" {
			e.Log.V(1).Info("cleaning up downloaded archive", "path", packedPath)
			if err := e.FS.Remove(packedPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
				e.Log.Error(err, "unable to clean up archive path", "path", packedPath)
			}
		}
	})

	archiveOut, err := e.FS.TempFile("", "*-"+e.Platform.ArchiveName(*e.Version.AsConcrete()))
	if err != nil {
		ExitCause(2, err, "unable to open file to write downloaded archive to")
	}
	defer archiveOut.Close()
	packedPath = archiveOut.Name()
	log.V(1).Info("writing downloaded archive", "path", packedPath)

	if err := e.Client.GetVersion(ctx, *e.Version.AsConcrete(), e.Platform, archiveOut); err != nil {
		ExitCause(2, err, "unable to download requested version")
	}
	log.V(1).Info("downloaded archive", "path", packedPath)

	if err := archiveOut.Sync(); err != nil { // sync before reading back
		ExitCause(2, err, "unable to flush downloaded archive file")
	}
	if _, err := archiveOut.Seek(0, 0); err != nil {
		ExitCause(2, err, "unable to jump back to beginning of archive file to unzip")
	}

	if err := e.Store.Add(ctx, e.item(), archiveOut); err != nil {
		ExitCause(2, err, "unable to store version to disk")
	}

	log.V(1).Info("removing archive from disk", "path", packedPath)
	if err := e.FS.Remove(packedPath); err != nil {
		// don't bail, this isn't fatal
		log.Error(err, "unable to remove downloaded archive", "path", packedPath)
	}
}

// cleanup on error cleans up if we hit an exitCode error.
//
// Use it in a defer.
func (e *Env) cleanupOnError(extraCleanup func()) {
	cause := recover()
	if cause == nil {
		return
	}
	// don't panic in a panic handler
	var exit *exitCode
	if asExit(cause, &exit) && exit.code != 0 {
		e.Log.Info("cleaning up due to error")
		// we already log in the function, and don't want to panic, so
		// ignore the error
		extraCleanup()
	}
	panic(cause) // re-start the panic now that we're done
}

// Remove removes the data for our version selector & platform from disk.
func (e *Env) Remove(ctx context.Context) {
	items, err := e.Store.Remove(ctx, e.filter())
	for _, item := range items {
		fmt.Fprintf(e.Out, "removed %s\n", item)
	}
	if err != nil {
		ExitCause(2, err, "unable to remove all requested version(s)")
	}
}

// PrintInfo prints out information about a single, current version
// and platform, according to the given formatting info.
func (e *Env) PrintInfo(printFmt PrintFormat) {
	// use the manual path if it's set, otherwise use the standard path
	path := e.manualPath
	if e.manualPath == "" {
		item := e.item()
		var err error
		path, err = e.Store.Path(item)
		if err != nil {
			ExitCause(2, err, "unable to get path for version %s", item)
		}
	}
	switch printFmt {
	case PrintOverview:
		fmt.Fprintf(e.Out, "Version: %s\n", e.Version)
		fmt.Fprintf(e.Out, "OS/Arch: %s\n", e.Platform)
		if e.Platform.Hash != nil {
			fmt.Fprintf(e.Out, "%s: %s\n", e.Platform.Hash.Type, e.Platform.Hash.Value)
		}
		fmt.Fprintf(e.Out, "Path: %s\n", path)
	case PrintPath:
		fmt.Fprint(e.Out, path) // NB(directxman12): no newline -- want the bare path here
	case PrintEnv:
		// quote in case there are spaces, etc in the path
		// the weird string below works like this:
		// - you can't escape quotes in shell
		// - shell strings that are next to each other are concatenated (so "a""b""c" == "abc")
		// - you can intermix quote styles using the above
		// - so `'"'"'` --> CLOSE_QUOTE + "'" + OPEN_QUOTE
		shellQuoted := strings.ReplaceAll(path, "'", `'"'"'`)
		fmt.Fprintf(e.Out, "export KUBEBUILDER_ASSETS='%s'\n", shellQuoted)
	default:
		panic(fmt.Sprintf("unexpected print format %v", printFmt))
	}
}

// EnsureBaseDirs ensures that the base packed and unpacked directories
// exist.
//
// This should be the first thing called after CheckCoherence.
func (e *Env) EnsureBaseDirs(ctx context.Context) {
	if err := e.Store.Initialize(ctx); err != nil {
		ExitCause(2, err, "unable to make sure store is initialized")
	}
}

// Sideload takes an input stream, and loads it as if it had been a downloaded .tar.gz file
// for the current *concrete* version and platform.
func (e *Env) Sideload(ctx context.Context, input io.Reader) {
	log := e.Log.WithName("sideload")
	if e.Version.AsConcrete() == nil || e.Platform.IsWildcard() {
		Exit(2, "must specify a concrete version and platform to sideload.  Make sure you've passed a version, like 'sideload 1.21.0'")
	}
	log.V(1).Info("sideloading from input stream to version", "version", e.Version, "platform", e.Platform)
	if err := e.Store.Add(ctx, e.item(), input); err != nil {
		ExitCause(2, err, "unable to sideload item to disk")
	}
}

var (
	// expectedExecutables are the executables that are checked in PathMatches
	// for non-store paths.
	expectedExecutables = []string{
		"kube-apiserver",
		"etcd",
		"kubectl",
	}
)

// PathMatches checks if the path (e.g. from the environment variable)
// matches this version & platform selector, and if so, returns true.
func (e *Env) PathMatches(value string) bool {
	e.Log.V(1).Info("checking if (env var) path represents our desired version", "path", value)
	if value == "" {
		// if we're unset,
		return false
	}

	if e.versionFromPathName(value) {
		e.Log.V(1).Info("path appears to be in our store, using that info", "path", value)
		return true
	}

	e.Log.V(1).Info("path is not in our store, checking for binaries", "path", value)
	for _, expected := range expectedExecutables {
		_, err := e.FS.Stat(filepath.Join(value, expected))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				// one of our required binaries is missing, return false
				e.Log.V(1).Info("missing required binary in (env var) path", "binary", expected, "path", value)
				return false
			}
			ExitCause(2, err, "unable to check for existence of binary %s from existing (env var) path %s", value, expected)
		}
	}

	// success, all binaries present
	e.Log.V(1).Info("all required binaries present in (env var) path, using that", "path", value)

	// don't bother checking the version, the user explicitly asked us to use this
	// we don't know the version, so set it to wildcard
	e.Version = versions.AnyVersion
	e.Platform.OS = "*"
	e.Platform.Arch = "*"
	e.manualPath = value
	return true
}

// versionFromPathName checks if the given path's last component looks like one
// of our versions, and, if so, what version it represents.  If successful,
// it'll set version and platform, and return true.  Otherwise it returns
// false.
func (e *Env) versionFromPathName(value string) bool {
	baseName := filepath.Base(value)
	ver, pl := versions.ExtractWithPlatform(versions.VersionPlatformRE, baseName)
	if ver == nil {
		// not a version that we can tell
		return false
	}

	// yay we got a version!
	e.Version.MakeConcrete(*ver)
	e.Platform.Platform = pl
	e.manualPath = value // might be outside our store, set this just in case

	return true
}
