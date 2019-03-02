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
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/knative/serving/test"
)

func handler(w http.ResponseWriter, r *http.Request) {
	timeout, _ := strconv.Atoi(r.URL.Query().Get("timeout"))
	start := time.Now().UnixNano()
	time.Sleep(time.Duration(timeout) * time.Millisecond)
	end := time.Now().UnixNano()
	fmt.Fprintf(w, "%d,%d\n", start, end)
}

func main() {
	log.Print("Benchmark container for 'observed_concurrency' started.")
	log.Print("Requests against '/?timeout={TIME_IN_MILLISECONDS}' will sleep for the given time.")
	log.Print("Each request will return its serverside start and end-time in nanoseconds.")

	test.ListenAndServeGracefully(":8080", handler)
}
