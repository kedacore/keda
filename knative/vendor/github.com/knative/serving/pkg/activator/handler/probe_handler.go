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

package handler

import (
	"net/http"

	"github.com/knative/serving/pkg/network"
)

// ProbeHandler handles responding to Knative internal network probes.
type ProbeHandler struct {
	NextHandler http.Handler
}

func (h *ProbeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// If this header is set the request was sent by a Knative component
	// probing the network, respond with a 200 and our component name.
	if r.Header.Get(network.ProbeHeaderName) != "" {
		w.Write([]byte("activator"))
		return
	}

	h.NextHandler.ServeHTTP(w, r)
}
