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

package http

import (
	"encoding/json"

	driver "github.com/arangodb/go-driver"
)

// mergeObject is a helper used to merge 2 objects into JSON.
type mergeObject struct {
	Object interface{}
	Merge  interface{}
}

func (m mergeObject) MarshalJSON() ([]byte, error) {
	m1, err := toMap(m.Object)
	if err != nil {
		return nil, driver.WithStack(err)
	}
	m2, err := toMap(m.Merge)
	if err != nil {
		return nil, driver.WithStack(err)
	}
	var merged map[string]interface{}
	// If m1 an empty object?
	if len(m1) == 0 {
		merged = m2
	} else if len(m2) == 0 {
		merged = m1
	} else {
		// Merge
		merged = make(map[string]interface{})
		for k, v := range m1 {
			merged[k] = v
		}
		for k, v := range m2 {
			merged[k] = v
		}
	}
	// Marshal merged map
	data, err := json.Marshal(merged)
	if err != nil {
		return nil, driver.WithStack(err)
	}
	return data, nil
}

// toMap converts the given object to a map (using JSON marshal/unmarshal when needed)
func toMap(object interface{}) (map[string]interface{}, error) {
	if m, ok := object.(map[string]interface{}); ok {
		return m, nil
	}
	data, err := json.Marshal(object)
	if err != nil {
		return nil, driver.WithStack(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, driver.WithStack(err)
	}
	return m, nil
}
