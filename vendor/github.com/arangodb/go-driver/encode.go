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

//go:build !go1.8
// +build !go1.8

package driver

import "net/url"

// Escape the given value for use in a URL path.
func pathEscape(s string) string {
	return url.QueryEscape(s)
}

// pathUnescape unescapes the given value for use in a URL path.
func pathUnescape(s string) string {
	r, _ := url.QueryUnescape(s)
	return r
}
