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
	"log"
	"net/http"

	"github.com/knative/serving/test/types"
)

func runtimeHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Retrieving Runtime Information")
	w.Header().Set("Content-Type", "application/json")

	k := &types.RuntimeInfo{
		Request: requestInfo(r),
		Host: &types.HostInfo{EnvVars: env(),
			Files:   fileInfo(filePaths...),
			Cgroups: cgroups(cgroupPaths...),
			Mounts:  mounts(),
		},
	}

	writeJSON(w, k)
}
