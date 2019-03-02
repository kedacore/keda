// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"crypto/md5"
	"encoding/hex"
	"path/filepath"

	"github.com/spf13/cobra"
)

// NameOptions represents options for the ko binary.
type NameOptions struct {
	// PreserveImportPaths preserves the full import path after KO_DOCKER_REPO.
	PreserveImportPaths bool
	// BaseImportPaths uses the base path without MD5 hash after KO_DOCKER_REPO.
	BaseImportPaths bool
}

func addNamingArgs(cmd *cobra.Command, no *NameOptions) {
	cmd.Flags().BoolVarP(&no.PreserveImportPaths, "preserve-import-paths", "P", no.PreserveImportPaths,
		"Whether to preserve the full import path after KO_DOCKER_REPO.")
	cmd.Flags().BoolVarP(&no.BaseImportPaths, "base-import-paths", "B", no.BaseImportPaths,
		"Whether to use the base path without MD5 hash after KO_DOCKER_REPO.")
}

func packageWithMD5(importpath string) string {
	hasher := md5.New()
	hasher.Write([]byte(importpath))
	return filepath.Base(importpath) + "-" + hex.EncodeToString(hasher.Sum(nil))
}

func preserveImportPath(importpath string) string {
	return importpath
}

func baseImportPaths(importpath string) string {
	return filepath.Base(importpath)
}
