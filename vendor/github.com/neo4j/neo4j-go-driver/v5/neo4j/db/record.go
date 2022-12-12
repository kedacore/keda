/*
 * Copyright (c) "Neo4j"
 * Neo4j Sweden AB [https://neo4j.com]
 *
 * This file is part of Neo4j.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package db

type Record struct {
	// Values contains all the values in the record.
	Values []any
	// Keys contains names of the values in the record.
	// Should not be modified. Same instance is used for all records within the same result.
	Keys []string
}

// Get returns the value corresponding to the given key along with a boolean that is true if
// a value was found and false if there were no key with the given name.
//
// If there are a lot of keys in combination with a lot of records to iterate, consider to retrieve
// values from Values slice directly or make a key -> index map before iterating. This implementation
// does not make or use a key -> index map since the overhead of making the map might not be beneficial
// for small and few records.
func (r Record) Get(key string) (any, bool) {
	for i, ckey := range r.Keys {
		if key == ckey {
			return r.Values[i], true
		}
	}
	return nil, false
}
