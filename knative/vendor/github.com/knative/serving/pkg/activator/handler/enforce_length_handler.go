/*
Copyright 2018 The Knative Authors
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
)

// EnforceMaxContentLengthHandler prevents uploads larger than `MaxContentLengthBytes`
type EnforceMaxContentLengthHandler struct {
	NextHandler           http.Handler
	MaxContentLengthBytes int64
}

func (h *EnforceMaxContentLengthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// If we have a ContentLength, we can fail early
	if r.ContentLength > h.MaxContentLengthBytes {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}

	// Enforce MaxContentLengthBytes
	r.Body = http.MaxBytesReader(w, r.Body, h.MaxContentLengthBytes)

	h.NextHandler.ServeHTTP(w, r)
}
