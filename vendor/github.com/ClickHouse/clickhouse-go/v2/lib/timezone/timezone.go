// Licensed to ClickHouse, Inc. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. ClickHouse, Inc. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package timezone

import (
	"sync"
	"time"
)

var cache = struct {
	mutex sync.Mutex
	items map[string]*time.Location
}{
	items: make(map[string]*time.Location),
}

func Load(name string) (*time.Location, error) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	if tz, found := cache.items[name]; found {
		return tz, nil
	}
	tz, err := time.LoadLocation(name)
	if err != nil {
		return nil, err
	}
	cache.items[name] = tz
	return tz, nil
}
