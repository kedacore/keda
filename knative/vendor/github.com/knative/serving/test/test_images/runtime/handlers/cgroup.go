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
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/knative/serving/test/types"
)

// cgroupPaths is the set of cgroups probed and returned to the
// client as Cgroups.
var cgroupPaths = []string{
	"/sys/fs/cgroup/memory/memory.limit_in_bytes",
	"/sys/fs/cgroup/cpu/cpu.cfs_period_us",
	"/sys/fs/cgroup/cpu/cpu.cfs_quota_us",
	"/sys/fs/cgroup/cpu/cpu.shares"}

var (
	yes = true
	no  = false
)

func cgroups(paths ...string) []*types.Cgroup {
	var cgroups []*types.Cgroup
	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			cgroups = append(cgroups, &types.Cgroup{Name: path, Error: err.Error()})
			continue
		}

		bc, err := ioutil.ReadFile(path)
		if err != nil {
			cgroups = append(cgroups, &types.Cgroup{Name: path, Error: err.Error()})
			continue
		}
		cs := strings.Trim(string(bc), "\n")
		ic, err := strconv.Atoi(cs)
		if err != nil {
			cgroups = append(cgroups, &types.Cgroup{Name: path, Error: err.Error()})
			continue
		}

		// Try to write to the Cgroup. We expect this to fail as a cheap
		// method for read-only validation
		newValue := []byte{'9'}
		err = ioutil.WriteFile(path, newValue, 420)
		if err != nil {
			cgroups = append(cgroups, &types.Cgroup{Name: path, Value: &ic, ReadOnly: &yes})
		} else {
			cgroups = append(cgroups, &types.Cgroup{Name: path, Value: &ic, ReadOnly: &no})
		}
	}
	return cgroups
}
