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

package conformance

//runtime_constract_types.go defines types that encapsulate run-time contract requirements as specified here: https://github.com/knative/serving/blob/master/docs/runtime-contract.md

// ShouldEnvvars defines the environment variables that "SHOULD" be set.
type ShouldEnvvars struct {
	Service       string `json:"K_SERVICE"`
	Configuration string `json:"K_CONFIGURATION"`
	Revision      string `json:"K_REVISION"`
}

// MustEnvvars defines environment variables that "MUST" be set.
type MustEnvvars struct {
	Port string `json:"PORT"`
}

// FilePathInfo data object returned by the environment test-image.
type FilePathInfo struct {
	FilePath    string `json:"FilePath"`
	IsDirectory bool   `json:"IsDirectory"`
	PermString  string `json:"PermString"`
}

// MustFilePathSpecs specifies the file-paths and expected permissions that MUST be set as specified in the runtime contract.
var MustFilePathSpecs = map[string]FilePathInfo{
	"/tmp": {
		IsDirectory: true,
		PermString:  "rw*rw*rw*", // * indicates no specification
	},
	"/var/log": {
		IsDirectory: true,
		PermString:  "rw*rw*rw*", // * indicates no specification
	},
	// TODO(#822): Add conformance tests for "/dev/log".
}

// ShouldFilePathSpecs specifies the file-paths and expected permissions that SHOULD be set as specified in the run-time contract.
var ShouldFilePathSpecs = map[string]FilePathInfo{
	"/etc/resolv.conf": {
		IsDirectory: false,
		PermString:  "rw*r**r**", // * indicates no specification
	},
}
