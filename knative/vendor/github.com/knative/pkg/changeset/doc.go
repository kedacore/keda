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

// Package changeset provides Knative utilities for fetching GitHub Commit ID
// from kodata directory. It requires GitHub HEAD file to be linked into
// Knative component source code via the following command:
//   ln -s -r .git/HEAD ./cmd/<knative-component-name>/kodata/
// Then ko will build this file into $KO_DATA_PATH when building the container
// for a Knative component.
package changeset
