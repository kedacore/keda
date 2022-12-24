//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//
// Author Ewout Prangsma
//

package util

import "strings"

var (
	urlFixer = strings.NewReplacer(
		"tcp://", "http://",
		"ssl://", "https://",
	)
)

// FixupEndpointURLScheme changes endpoint URL schemes used by arangod to ones used by go.
// E.g. "tcp://localhost:8529" -> "http://localhost:8529"
func FixupEndpointURLScheme(u string) string {
	return urlFixer.Replace(u)
}
