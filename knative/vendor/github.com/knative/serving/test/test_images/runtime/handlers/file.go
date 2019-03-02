/*
Copyright 2019 The Knative Authors
 Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
     http://www.apache.org/licenses/LICENSE-2.0
 Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package handlers

import (
	"os"

	"github.com/knative/serving/test/types"
)

// filePaths is the set of filepaths probed and returned to the
// client as FileInfo.
var filePaths = []string{"/tmp",
	"/var/log",
	"/dev/log",
	"/etc/hosts",
	"/etc/hostname",
	"/etc/resolv.conf"}

func fileInfo(paths ...string) []*types.FileInfo {
	var infoList []*types.FileInfo

	for _, path := range paths {
		file, err := os.Stat(path)
		if err != nil {
			infoList = append(infoList, &types.FileInfo{Name: path, Error: err.Error()})
			continue
		}
		size := file.Size()
		dir := file.IsDir()
		infoList = append(infoList, &types.FileInfo{Name: path,
			Size:    &size,
			Mode:    file.Mode().String(),
			ModTime: file.ModTime(),
			IsDir:   &dir})
	}
	return infoList
}
