//
// DISCLAIMER
//
// Copyright 2017-2025 ArangoDB GmbH, Cologne, Germany
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

package http

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"

	velocypack "github.com/arangodb/go-velocypack"

	driver "github.com/arangodb/go-driver"
)

type velocyPackBody struct {
	body []byte
}

func NewVelocyPackBodyBuilder() *velocyPackBody {
	return &velocyPackBody{}
}

// SetBody sets the content of the request.
// The protocol of the connection determines what kinds of marshalling is taking place.
func (b *velocyPackBody) SetBody(body ...interface{}) error {

	switch len(body) {
	case 0:
		return driver.WithStack(errors.New("Must provide at least 1 body"))
	case 1:
		if data, err := velocypack.Marshal(body[0]); err != nil {
			return driver.WithStack(err)
		} else {
			b.body = data
		}
		return nil
	case 2:
		mo := mergeObject{Object: body[1], Merge: body[0]}
		if data, err := velocypack.Marshal(mo); err != nil {
			return driver.WithStack(err)
		} else {
			b.body = data
		}
		return nil
	default:
		return driver.WithStack(errors.New("Must provide at most 2 bodies"))
	}

	return nil
}

// SetBodyArray sets the content of the request as an array.
// If the given mergeArray is not nil, its elements are merged with the elements in the body array (mergeArray data overrides bodyArray data).
// The protocol of the connection determines what kinds of marshalling is taking place.
func (b *velocyPackBody) SetBodyArray(bodyArray interface{}, mergeArray []map[string]interface{}) error {

	bodyArrayVal := reflect.ValueOf(bodyArray)
	switch bodyArrayVal.Kind() {
	case reflect.Array, reflect.Slice:
		// OK
	default:
		return driver.WithStack(driver.InvalidArgumentError{Message: fmt.Sprintf("bodyArray must be slice, got %s", bodyArrayVal.Kind())})
	}
	if mergeArray == nil {
		// Simple case; just marshal bodyArray directly.
		if data, err := velocypack.Marshal(bodyArray); err != nil {
			return driver.WithStack(err)
		} else {
			b.body = data
		}
		return nil
	}
	// Complex case, mergeArray is not nil
	elementCount := bodyArrayVal.Len()
	mergeObjects := make([]mergeObject, elementCount)
	for i := 0; i < elementCount; i++ {
		mergeObjects[i] = mergeObject{
			Object: bodyArrayVal.Index(i).Interface(),
			Merge:  mergeArray[i],
		}
	}
	// Now marshal merged array
	if data, err := velocypack.Marshal(mergeObjects); err != nil {
		return driver.WithStack(err)
	} else {
		b.body = data
	}
	return nil
}

// SetBodyImportArray sets the content of the request as an array formatted for importing documents.
// The protocol of the connection determines what kinds of marshalling is taking place.
func (b *velocyPackBody) SetBodyImportArray(bodyArray interface{}) error {
	bodyArrayVal := reflect.ValueOf(bodyArray)

	switch bodyArrayVal.Kind() {
	case reflect.Array, reflect.Slice:
		// OK
	default:
		return driver.WithStack(driver.InvalidArgumentError{Message: fmt.Sprintf("bodyArray must be slice, got %s", bodyArrayVal.Kind())})
	}
	// Render elements
	buf := &bytes.Buffer{}
	encoder := velocypack.NewEncoder(buf)
	if err := encoder.Encode(bodyArray); err != nil {
		return driver.WithStack(err)
	}
	b.body = buf.Bytes()
	return nil
}

func (b *velocyPackBody) GetBody() []byte {
	return b.body
}

func (b *velocyPackBody) GetContentType() string {
	return "application/x-velocypack"
}

func (b *velocyPackBody) Clone() driver.BodyBuilder {
	return &velocyPackBody{
		body: b.GetBody(),
	}
}
