// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 The Kubernetes Authors

package store

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/go-logr/logr"
	"github.com/spf13/afero"

	"sigs.k8s.io/controller-runtime/tools/setup-envtest/versions"
)

// TODO(directxman12): error messages don't show full path, which is gonna make
// things hard to debug

// Item is a version-platform pair.
type Item struct {
	Version  versions.Concrete
	Platform versions.Platform
}

// dirName returns the directory name in the store for this item.
func (i Item) dirName() string {
	return i.Platform.BaseName(i.Version)
}
func (i Item) String() string {
	return fmt.Sprintf("%s (%s)", i.Version, i.Platform)
}

// Filter is a version spec & platform selector (i.e. platform
// potentially with wilcards) to filter store items.
type Filter struct {
	Version  versions.Spec
	Platform versions.Platform
}

// Matches checks if this filter matches the given item.
func (f Filter) Matches(item Item) bool {
	return f.Version.Matches(item.Version) && f.Platform.Matches(item.Platform)
}

// Store knows how to list, load, store, and delete envtest tools.
type Store struct {
	// Root is the root FS that the store stores in.  You'll probably
	// want to use a BasePathFS to scope it down to a particular directory.
	//
	// Note that if for some reason there are nested BasePathFSes, and they're
	// interrupted by a non-BasePathFS, Path won't work properly.
	Root afero.Fs
}

// NewAt creates a new store on disk at the given path.
func NewAt(path string) *Store {
	return &Store{
		Root: afero.NewBasePathFs(afero.NewOsFs(), path),
	}
}

// Initialize ensures that the store is all set up on disk, etc.
func (s *Store) Initialize(ctx context.Context) error {
	log, err := logr.FromContext(ctx)
	if err != nil {
		return err
	}

	log.V(1).Info("ensuring base binaries dir exists")
	if err := s.unpackedBase().MkdirAll("", 0755); err != nil {
		return fmt.Errorf("unable to make sure base binaries dir exists: %w", err)
	}
	return nil
}

// Has checks if an item exists in the store.
func (s *Store) Has(item Item) (bool, error) {
	path := s.unpackedPath(item.dirName())
	_, err := path.Stat("")
	if err != nil && !errors.Is(err, afero.ErrFileNotFound) {
		return false, fmt.Errorf("unable to check if version-platform dir exists: %w", err)
	}
	return err == nil, nil
}

// List lists all items matching the given filter.
//
// Results are stored by version (newest first), and OS/arch (consistently,
// but no guaranteed ordering).
func (s *Store) List(ctx context.Context, matching Filter) ([]Item, error) {
	var res []Item
	if err := s.eachItem(ctx, matching, func(_ string, item Item) {
		res = append(res, item)
	}); err != nil {
		return nil, fmt.Errorf("unable to list version-platform pairs in store: %w", err)
	}

	sort.Slice(res, func(i, j int) bool {
		if !res[i].Version.Matches(res[j].Version) {
			return res[i].Version.NewerThan(res[j].Version)
		}
		return orderPlatforms(res[i].Platform, res[j].Platform)
	})

	return res, nil
}

// Add adds this item to the store, with the given contents (a .tar.gz file).
func (s *Store) Add(ctx context.Context, item Item, contents io.Reader) (resErr error) {
	log, err := logr.FromContext(ctx)
	if err != nil {
		return err
	}

	itemName := item.dirName()
	log = log.WithValues("version-platform", itemName)
	itemPath := s.unpackedPath(itemName)

	// make sure to clean up if we hit an error
	defer func() {
		if resErr != nil {
			// intentially ignore this because we can't really do anything
			err := s.removeItem(itemPath)
			if err != nil {
				log.Error(err, "unable to clean up partially added version-platform pair after error")
			}
		}
	}()

	log.V(1).Info("ensuring version-platform binaries dir exists and is empty & writable")
	_, err = itemPath.Stat("")
	if err != nil && !errors.Is(err, afero.ErrFileNotFound) {
		return fmt.Errorf("unable to ensure version-platform binaries dir %s exists", itemName)
	}
	if err == nil { // exists
		log.V(1).Info("cleaning up old version-platform binaries dir")
		if err := s.removeItem(itemPath); err != nil {
			return fmt.Errorf("unable to clean up existing version-platform binaries dir %s", itemName)
		}
	}
	if err := itemPath.MkdirAll("", 0755); err != nil {
		return fmt.Errorf("unable to make sure entry dir %s exists", itemName)
	}

	log.V(1).Info("extracting archive")
	gzStream, err := gzip.NewReader(contents)
	if err != nil {
		return fmt.Errorf("unable to start un-gz-ing entry archive")
	}
	tarReader := tar.NewReader(gzStream)

	var header *tar.Header
	for header, err = tarReader.Next(); err == nil; header, err = tarReader.Next() {
		if header.Typeflag != tar.TypeReg { // TODO(directxman12): support symlinks, etc?
			log.V(1).Info("skipping non-regular-file entry in archive", "entry", header.Name)
			continue
		}
		// just dump all files to the main path, ignoring the prefixed directory
		// paths -- they're redundant.  We also ignore bits for the most part (except for X),
		// preferfing our own scheme.
		targetPath := filepath.Base(header.Name)
		log.V(1).Info("writing archive file to disk", "archive file", header.Name, "on-disk file", targetPath)
		perms := 0555 & header.Mode // make sure we're at most r+x
		binOut, err := itemPath.OpenFile(targetPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(perms))
		if err != nil {
			return fmt.Errorf("unable to create file %s from archive to disk for version-platform pair %s", targetPath, itemName)
		}
		if err := func() error { // IIFE to get the defer properly in a loop
			defer binOut.Close()
			if _, err := io.Copy(binOut, tarReader); err != nil { //nolint:gosec
				return fmt.Errorf("unable to write file %s from archive to disk for version-platform pair %s", targetPath, itemName)
			}
			return nil
		}(); err != nil {
			return err
		}
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("unable to finish un-tar-ing the downloaded archive: %w", err)
	}
	log.V(1).Info("unpacked archive")

	log.V(1).Info("switching version-platform directory to read-only")
	if err := itemPath.Chmod("", 0555); err != nil {
		// don't bail, this isn't fatal
		log.Error(err, "unable to make version-platform directory read-only")
	}
	return nil
}

// Remove removes all items matching the given filter.
//
// It returns a list of the successfully removed items (even in the case
// of an error).
func (s *Store) Remove(ctx context.Context, matching Filter) ([]Item, error) {
	log, err := logr.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	var removed []Item
	var savedErr error
	if err := s.eachItem(ctx, matching, func(name string, item Item) {
		log.V(1).Info("Removing version-platform pair at path", "version-platform", item, "path", name)

		if err := s.removeItem(s.unpackedPath(name)); err != nil {
			log.Error(err, "unable to make existing version-platform dir writable to clean it up", "path", name)
			savedErr = fmt.Errorf("unable to remove version-platform pair %s (dir %s): %w", item, name, err)
			return // don't mark this as removed in the report
		}
		removed = append(removed, item)
	}); err != nil {
		return removed, fmt.Errorf("unable to list version-platform pairs to figure out what to delete: %w", err)
	}
	if savedErr != nil {
		return removed, savedErr
	}
	return removed, nil
}

// Path returns an actual path that case be used to access this item.
func (s *Store) Path(item Item) (string, error) {
	path := s.unpackedPath(item.dirName())
	// NB(directxman12): we need root's realpath because RealPath only
	// looks at its own path, and so thus doesn't prepend the underlying
	// root's base path.
	//
	// Technically, if we're fed something that's double wrapped as root,
	// this'll be wrong, but this is basically as much as we can do
	return afero.FullBaseFsPath(path.(*afero.BasePathFs), ""), nil
}

// unpackedBase returns the directory in which item dirs lives.
func (s *Store) unpackedBase() afero.Fs {
	return afero.NewBasePathFs(s.Root, "k8s")
}

// unpackedPath returns the item dir with this name.
func (s *Store) unpackedPath(name string) afero.Fs {
	return afero.NewBasePathFs(s.unpackedBase(), name)
}

// eachItem iterates through the on-disk versions that match our version & platform selector,
// calling the callback for each.
func (s *Store) eachItem(ctx context.Context, filter Filter, cb func(name string, item Item)) error {
	log, err := logr.FromContext(ctx)
	if err != nil {
		return err
	}

	entries, err := afero.ReadDir(s.unpackedBase(), "")
	if err != nil {
		return fmt.Errorf("unable to list folders in store's unpacked directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			log.V(1).Info("skipping dir entry, not a folder", "entry", entry.Name())
			continue
		}
		ver, pl := versions.ExtractWithPlatform(versions.VersionPlatformRE, entry.Name())
		if ver == nil {
			log.V(1).Info("skipping dir entry, not a version", "entry", entry.Name())
			continue
		}
		item := Item{Version: *ver, Platform: pl}

		if !filter.Matches(item) {
			log.V(1).Info("skipping on disk version, does not match version and platform selectors", "platform", pl, "version", ver, "entry", entry.Name())
			continue
		}

		cb(entry.Name(), item)
	}

	return nil
}

// removeItem removes the given item directory from disk.
func (s *Store) removeItem(itemDir afero.Fs) error {
	if err := itemDir.Chmod("", 0755); err != nil {
		// no point in trying to remove if we can't fix the permissions, bail here
		return fmt.Errorf("unable to make version-platform dir writable: %w", err)
	}
	if err := itemDir.RemoveAll(""); err != nil && !errors.Is(err, afero.ErrFileNotFound) {
		return fmt.Errorf("unable to remove version-platform dir: %w", err)
	}
	return nil
}

// orderPlatforms orders platforms by OS then arch.
func orderPlatforms(first, second versions.Platform) bool {
	// sort by OS, then arch
	if first.OS != second.OS {
		return first.OS < second.OS
	}
	return first.Arch < second.Arch
}
