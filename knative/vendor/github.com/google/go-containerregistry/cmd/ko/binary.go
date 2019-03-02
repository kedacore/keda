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
	"github.com/spf13/cobra"
)

// BinaryOptions represents options for the ko binary.
type BinaryOptions struct {
	// Path is the import path of the binary to publish.
	Path string
}

func addImageArg(cmd *cobra.Command, lo *BinaryOptions) {
	cmd.Flags().StringVarP(&lo.Path, "image", "i", lo.Path,
		"The import path of the binary to publish.")
}
