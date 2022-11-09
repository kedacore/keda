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

// KeyPriority is a utility struct for getting values from a map
// given a list of ordered keys
//
// This is to help the migration/renaming of annotations & labels
type KeyPriority []string

// Key returns the default key that should be used for
// accessing the map
func (p KeyPriority) Key() string {
	// this intentionally panics rather than returning an empty string
	return p[0]
}

// Value iterates looks up the ordered keys in the map and returns
// a string value. An empty string will be returned if the keys
// are not present in the map
func (p KeyPriority) Value(m map[string]string) string {
	_, v, _ := p.Get(m)
	return v
}

// Get iterates over the ordered keys and looks up the corresponding
// values in the map
//
// It returns the key, value, and true|false signaling whether the
// key was present in the map
//
// If no key is present the default key (lowest ordinal) is returned
// with an empty string as the value
func (p KeyPriority) Get(m map[string]string) (string, string, bool) {
	var k, v string
	var ok bool
	for _, k = range p {
		v, ok = m[k]
		if ok {
			return k, v, ok
		}
	}

	return p.Key(), "", false
}

// UpdateKey will update the map with the KeyPriority's default
// key iff any of the other synonym keys are present
func (p KeyPriority) UpdateKey(m map[string]string) {
	if k, v, ok := p.Get(m); ok && k != p.Key() {
		delete(m, k)
		m[p.Key()] = v
	}
}

// UpdateKeys iterates over the lookups and updates entries in the map
// to use the default key
func UpdateKeys(m map[string]string, keys ...KeyPriority) map[string]string {
	for _, key := range keys {
		key.UpdateKey(m)
	}
	return m
}
