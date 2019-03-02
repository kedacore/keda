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
	"bufio"
	"os"
	"strings"

	"github.com/knative/serving/test/types"
)

func mounts() []*types.Mount {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return []*types.Mount{{Error: err.Error()}}
	}
	defer file.Close()

	sc := bufio.NewScanner(file)
	var mounts []*types.Mount
	for sc.Scan() {
		ml := sc.Text()
		// Each line should be:
		// Device Path Type Options Unused Unused
		// sysfs /sys sysfs rw,nosuid,nodev,noexec,relatime 0 0
		ms := strings.Split(ml, " ")
		if len(ms) < 6 {
			return []*types.Mount{{Error: "unknown /proc/mounts format"}}
		}
		mounts = append(mounts, &types.Mount{
			Device:  ms[0],
			Path:    ms[1],
			Type:    ms[2],
			Options: strings.Split(ms[3], ",")})
	}

	if err := sc.Err(); err != nil {
		// Don't return partial list of mounts on error
		return []*types.Mount{{Error: err.Error()}}
	}

	return mounts
}
