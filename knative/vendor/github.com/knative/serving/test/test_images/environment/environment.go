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
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/knative/serving/test"
	"github.com/knative/serving/test/conformance"
)

func envvarsHandler(w http.ResponseWriter, r *http.Request) {
	envvars := make(map[string]string)
	for _, pair := range os.Environ() {
		tokens := strings.Split(pair, "=")
		envvars[tokens[0]] = tokens[1]
	}

	if resp, err := json.Marshal(envvars); err != nil {
		fmt.Fprintf(w, fmt.Sprintf("error building response : %v", err))
	} else {
		fmt.Fprintf(w, string(resp))
	}
}

func filepathInfoHandler(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get(test.EnvImageFilePathQueryParam)
	if len(filePath) == 0 {
		fmt.Fprintf(w, "no filepath found in queryparam")
		return
	}

	info, err := os.Stat(filePath)
	if err != nil {
		fmt.Fprintf(w, fmt.Sprintf("Error encountered retrieving info for path '%s': %v", filePath, err))
		return
	}

	resp, err := json.Marshal(conformance.FilePathInfo{
		FilePath:    filePath,
		IsDirectory: info.IsDir(),
		PermString:  info.Mode().Perm().String(),
	})
	if err != nil {
		fmt.Fprintf(w, fmt.Sprintf("error building response : %v", err))
	} else {
		fmt.Fprintf(w, string(resp))
	}
}

func main() {
	flag.Parse()
	log.Print("Environment test app started.")
	test.ListenAndServeGracefullyWithPattern(fmt.Sprintf(":%d", test.EnvImageServerPort), map[string]func(w http.ResponseWriter, r *http.Request){
		test.EnvImageEnvVarsPath:      envvarsHandler,
		test.EnvImageFilePathInfoPath: filepathInfoHandler,
	})
}
