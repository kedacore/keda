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

package velocypack

import "strconv"

var attributeTranslator attributeIDTranslator = &arangoAttributeIDTranslator{}

// attributeIDTranslator is used to translation integer style object keys to strings.
type attributeIDTranslator interface {
	IDToString(id uint64) string
}

type arangoAttributeIDTranslator struct{}

func (t *arangoAttributeIDTranslator) IDToString(id uint64) string {
	switch id {
	case 1:
		return "_key"
	case 2:
		return "_rev"
	case 3:
		return "_id"
	case 4:
		return "_from"
	case 5:
		return "_to"
	default:
		return strconv.FormatUint(id, 10)
	}
}
