/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/knative/serving/test"
)

func handler(w http.ResponseWriter, r *http.Request) {
	// Sleep for a set amount of time before sending headers
	if initialTimeout := r.URL.Query().Get("initialTimeout"); initialTimeout != "" {
		parsed, _ := strconv.Atoi(initialTimeout)
		time.Sleep(time.Duration(parsed) * time.Millisecond)
	}

	w.WriteHeader(http.StatusOK)

	// Explicitly flush the already written data to trigger (or not)
	// the time-to-first-byte timeout.
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Sleep for a set amount of time before sending response
	timeout, _ := strconv.Atoi(r.URL.Query().Get("timeout"))
	time.Sleep(time.Duration(timeout) * time.Millisecond)

	fmt.Fprintf(w, "Slept for %d milliseconds", timeout)
}

func main() {
	test.ListenAndServeGracefully(":8080", handler)
}
