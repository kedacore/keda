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
	"flag"
	"fmt"
	"log"
	"os"

	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/knative/serving/test"
)

const (
	targetHostEnv = "TARGET_HOST"
)

var (
	httpProxy *httputil.ReverseProxy
)

func handler(w http.ResponseWriter, r *http.Request) {
	log.Print("Http proxy received a request.")
	// Reverse proxy does not automatically reset the Host header.
	// We need to manually reset it.
	r.Host = getTargetHostEnv()
	httpProxy.ServeHTTP(w, r)
	return
}

func getTargetHostEnv() string {
	value := os.Getenv(targetHostEnv)
	if value == "" {
		log.Fatalf("No env %v provided.", targetHostEnv)
	}
	return value
}

func initialHttpProxy(proxyUrl string) *httputil.ReverseProxy {
	target, err := url.Parse(proxyUrl)
	if err != nil {
		log.Fatalf("Failed to parse url %v", proxyUrl)
	}
	return httputil.NewSingleHostReverseProxy(target)
}

func main() {
	flag.Parse()
	log.Print("Http Proxy app started.")

	targetHost := getTargetHostEnv()
	targetUrl := fmt.Sprintf("http://%s", targetHost)
	httpProxy = initialHttpProxy(targetUrl)

	test.ListenAndServeGracefully(":8080", handler)
}
