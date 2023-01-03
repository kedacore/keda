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

package driver

// DocumentMeta contains all meta data used to identifier a document.
type DocumentMeta struct {
	Key    string     `json:"_key,omitempty"`
	ID     DocumentID `json:"_id,omitempty"`
	Rev    string     `json:"_rev,omitempty"`
	OldRev string     `json:"_oldRev,omitempty"`
}

// validateKey returns an error if the given key is empty otherwise invalid.
func validateKey(key string) error {
	if key == "" {
		return WithStack(InvalidArgumentError{Message: "key is empty"})
	}
	return nil
}

// DocumentMetaSlice is a slice of DocumentMeta elements
type DocumentMetaSlice []DocumentMeta

// Keys returns the keys of all elements.
func (l DocumentMetaSlice) Keys() []string {
	keys := make([]string, len(l))
	for i, m := range l {
		keys[i] = m.Key
	}
	return keys
}

// Revs returns the revisions of all elements.
func (l DocumentMetaSlice) Revs() []string {
	revs := make([]string, len(l))
	for i, m := range l {
		revs[i] = m.Rev
	}
	return revs
}

// IDs returns the ID's of all elements.
func (l DocumentMetaSlice) IDs() []DocumentID {
	ids := make([]DocumentID, len(l))
	for i, m := range l {
		ids[i] = m.ID
	}
	return ids
}
