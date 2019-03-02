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

package test

//image_constants.go defines constants that are shared between test-images and conformance tests

//EnvImageServerPort is the port on which the environment test-image server starts.
// TODO: Modify this port number after https://github.com/knative/serving/issues/2258 is fixed for a stricter verification.
const EnvImageServerPort = 8080

//EnvImageEnvVarsPath path exposed by environment test-image to fetch environment variables.
const EnvImageEnvVarsPath = "/envvars"

//EnvImageFilePathInfoPath path exposed by environment test-image to fetch information for filepaths
const EnvImageFilePathInfoPath = "/filepath"

//EnvImageFilePathQueryParam query param to be used with EnvImageFilePathInfoPath to specify filepath
const EnvImageFilePathQueryParam = "path"
