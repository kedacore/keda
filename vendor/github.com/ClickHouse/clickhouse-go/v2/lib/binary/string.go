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

package binary

import (
	"reflect"
	"unsafe"
)

// Copied from https://github.com/m3db/m3/blob/master/src/x/unsafe/string.go#L62

func unsafeStr2Bytes(str string) []byte {
	if len(str) == 0 {
		return nil
	}
	var scratch []byte
	{
		slice := (*reflect.SliceHeader)(unsafe.Pointer(&scratch))
		slice.Len = len(str)
		slice.Cap = len(str)
		slice.Data = (*reflect.StringHeader)(unsafe.Pointer(&str)).Data
	}
	return scratch
}
