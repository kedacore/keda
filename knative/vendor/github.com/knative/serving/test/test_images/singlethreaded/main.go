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

// The singlethreaded program
package main

import (
	"flag"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/knative/serving/test"
)

var (
	lockedFlag uint32
)

func handler(w http.ResponseWriter, r *http.Request) {
	// Use an atomic int as a simple boolean flag we can check safely
	if !atomic.CompareAndSwapUint32(&lockedFlag, 0, 1) {
		// Return HTTP 500 if more than 1 request at a time gets in
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer atomic.StoreUint32(&lockedFlag, 0)

	time.Sleep(500 * time.Millisecond)
	fmt.Fprintf(w, "One at a time")
}

func main() {
	flag.Parse()

	test.ListenAndServeGracefully(":8080", handler)
}
