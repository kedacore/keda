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

import "fmt"

// ContentType identifies the type of encoding to use for the data.
type ContentType int

const (
	// ContentTypeJSON encodes data as json
	ContentTypeJSON ContentType = iota
	// ContentTypeVelocypack encodes data as Velocypack
	ContentTypeVelocypack
)

func (ct ContentType) String() string {
	switch ct {
	case ContentTypeJSON:
		return "application/json"
	case ContentTypeVelocypack:
		return "application/x-velocypack"
	default:
		panic(fmt.Sprintf("Unknown content type %d", int(ct)))
	}
}
