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

package column

import "time"

// getTimeWithDifferentLocation returns the same time but with different location, e.g.
// "2024-08-15 13:22:34 -03:00" will become "2024-08-15 13:22:34 +04:00".
func getTimeWithDifferentLocation(t time.Time, loc *time.Location) time.Time {
	year, month, day := t.Date()
	hour, minute, sec := t.Clock()

	return time.Date(year, month, day, hour, minute, sec, t.Nanosecond(), loc)
}
