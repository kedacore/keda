// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cache

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

// Default returns the default cache to use, or nil if no cache should be used.
func Default() *Cache {
	defaultOnce.Do(initDefaultCache)
	return defaultCache
}

var (
	defaultOnce  sync.Once
	defaultCache *Cache
)

// cacheREADME is a message stored in a README in the cache directory.
// Because the cache lives outside the normal Go trees, we leave the
// README as a courtesy to explain where it came from.
const cacheREADME = `This directory holds cached build artifacts from the Go build system.
Run "go clean -cache" if the directory is getting too large.
See golang.org to learn more about Go.
`

// initDefaultCache does the work of finding the default cache
// the first time Default is called.
func initDefaultCache() {
	dir, showWarnings := defaultDir()
	if dir == "off" {
		return
	}
	if err := os.MkdirAll(dir, 0777); err != nil {
		if showWarnings {
			fmt.Fprintf(os.Stderr, "go: disabling cache (%s) due to initialization failure: %s\n", dir, err)
		}
		return
	}
	if _, err := os.Stat(filepath.Join(dir, "README")); err != nil {
		// Best effort.
		ioutil.WriteFile(filepath.Join(dir, "README"), []byte(cacheREADME), 0666)
	}

	c, err := Open(dir)
	if err != nil {
		if showWarnings {
			fmt.Fprintf(os.Stderr, "go: disabling cache (%s) due to initialization failure: %s\n", dir, err)
		}
		return
	}
	defaultCache = c
}

// DefaultDir returns the effective GOCACHE setting.
// It returns "off" if the cache is disabled.
func DefaultDir() string {
	dir, _ := defaultDir()
	return dir
}

// defaultDir returns the effective GOCACHE setting.
// It returns "off" if the cache is disabled.
// The second return value reports whether warnings should
// be shown if the cache fails to initialize.
func defaultDir() (string, bool) {
	dir := os.Getenv("GOCACHE")
	if dir != "" {
		return dir, true
	}

	// Compute default location.
	dir, err := os.UserCacheDir()
	if err != nil {
		return "off", true
	}
	dir = filepath.Join(dir, "go-build")

	// Do this after filepath.Join, so that the path has been cleaned.
	showWarnings := true
	switch dir {
	case "/.cache/go-build":
		// probably docker run with -u flag
		// https://golang.org/issue/26280
		showWarnings = false
	}
	return dir, showWarnings
}
