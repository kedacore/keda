/*
Copyright 2021 The Knative Authors

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

package kmap

// Copy makes a copy of the map.
func Copy(a map[string]string) map[string]string {
	ret := make(map[string]string, len(a))
	for k, v := range a {
		ret[k] = v
	}
	return ret
}

// Union returns a map constructed from the union of input maps.
// where values from latter maps win.
func Union(maps ...map[string]string) map[string]string {
	if len(maps) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(maps[0]))

	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// Filter creates a copy of the provided map, filtering out the elements
// that match `filter`.
// nil `filter` is accepted.
func Filter(in map[string]string, filter func(string) bool) map[string]string {
	ret := make(map[string]string, len(in))
	for k, v := range in {
		if filter != nil && filter(k) {
			continue
		}
		ret[k] = v
	}
	return ret
}

// ExcludeKeys creates a copy of the provided map filtering out the excluded `keys`
func ExcludeKeys(in map[string]string, keys ...string) map[string]string {
	return ExcludeKeyList(in, keys)
}

// ExcludeKeyList creates a copy of the provided map filtering out excluded `keys`
func ExcludeKeyList(in map[string]string, keys []string) map[string]string {
	ret := make(map[string]string, len(in))

outer:
	for k, v := range in {
		// opted to skip memory allocation (creating a set) in favour of
		// looping since the places Knative will use this we typically
		// exclude one or two keys
		for _, excluded := range keys {
			if k == excluded {
				continue outer
			}
		}
		ret[k] = v
	}
	return ret
}
