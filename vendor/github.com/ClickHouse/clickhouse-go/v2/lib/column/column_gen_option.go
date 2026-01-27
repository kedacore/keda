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

import "github.com/ClickHouse/ch-go/proto"

// ColStrProvider defines provider of proto.ColStr
type ColStrProvider func() proto.ColStr

// colStrProvider provide proto.ColStr for Column() when type is String
var colStrProvider ColStrProvider = defaultColStrProvider

// defaultColStrProvider defines sample provider for proto.ColStr
func defaultColStrProvider() proto.ColStr {
	return proto.ColStr{}
}

// issue: https://github.com/ClickHouse/clickhouse-go/issues/1164
// WithAllocBufferColStrProvider allow pre alloc buffer cap for proto.ColStr
//
//	It is more suitable for scenarios where a lot of data is written in batches
func WithAllocBufferColStrProvider(cap int) {
	colStrProvider = func() proto.ColStr {
		return proto.ColStr{Buf: make([]byte, 0, cap)}
	}
}

// WithColStrProvider more flexible than WithAllocBufferColStrProvider, such as use sync.Pool
func WithColStrProvider(provider ColStrProvider) {
	colStrProvider = provider
}
