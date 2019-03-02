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

package http

import "net/http"

// LastHeaderValue gets the last value associated with the given key.
// It is case insensitive; textproto.CanonicalMIMEHeaderKey is used
// to canonicalize the provided key.
// If there are no values associated with the key, Get returns "".
func LastHeaderValue(header http.Header, key string) string {
	if header == nil {
		return ""
	}

	v := header[http.CanonicalHeaderKey(key)]

	if len(v) == 0 {
		return ""
	}

	return v[len(v)-1]
}
