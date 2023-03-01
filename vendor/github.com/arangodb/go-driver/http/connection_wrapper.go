//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
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

package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/arangodb/go-velocypack"

	"github.com/arangodb/go-driver"
)

type connectionDebugWrapper struct {
	driver.Connection
	ct driver.ContentType
}

func NewConnectionDebugWrapper(conn driver.Connection, ct driver.ContentType) driver.Connection {
	return &connectionDebugWrapper{conn, ct}
}

func (c *connectionDebugWrapper) Do(ctx context.Context, req driver.Request) (driver.Response, error) {
	if c.ct == driver.ContentTypeJSON {
		resp, err := c.Connection.Do(ctx, req)
		if err != nil {
			return resp, err
		}

		httpResponse, ok := resp.(*httpJSONResponse)
		if !ok {
			panic("can not cast response to the httpJSONResponse type!")
		}

		return &responseDebugWrapper{httpResponse}, err

	}
	return c.Connection.Do(ctx, req)
}

func (c *connectionDebugWrapper) Unmarshal(data driver.RawObject, result interface{}) error {
	ct := c.ct
	if ct == driver.ContentTypeVelocypack && len(data) >= 2 {
		// Poor mans auto detection of json
		l := len(data)
		if (data[0] == '{' && data[l-1] == '}') || (data[0] == '[' && data[l-1] == ']') {
			ct = driver.ContentTypeJSON
		}
	}
	switch ct {
	case driver.ContentTypeJSON:
		decoder := json.NewDecoder(strings.NewReader(string(data)))
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(result); err != nil {
			return driver.WithStack(err)
		}

		if err := json.Unmarshal(data, result); err != nil {
			fmt.Printf("Struct: %s \n", reflect.TypeOf(result).String())
			fmt.Printf("Response: %s \n\n", string(data))
			return driver.WithStack(errors.New(fmt.Sprintf("Struct: %s, Error: %s", reflect.TypeOf(result).String(), err.Error())))
		}
	case driver.ContentTypeVelocypack:
		if err := velocypack.Unmarshal(velocypack.Slice(data), result); err != nil {
			return driver.WithStack(err)
		}
	default:
		return driver.WithStack(fmt.Errorf("unsupported content type %d", int(c.ct)))
	}
	return nil
}

type responseDebugWrapper struct {
	*httpJSONResponse
}

func (r *responseDebugWrapper) ParseBody(field string, result interface{}) error {
	if field == "" {
		decoder := json.NewDecoder(strings.NewReader(string(r.httpJSONResponse.rawResponse)))
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(result); err != nil {
			fmt.Printf("Struct: %s \n", reflect.TypeOf(result).String())
			fmt.Printf("Response: %s \n\n", string(r.httpJSONResponse.rawResponse))
			return driver.WithStack(errors.New(fmt.Sprintf("Struct: %s, Error: %s", reflect.TypeOf(result).String(), err.Error())))
		}
	}
	return r.httpJSONResponse.ParseBody(field, result)
}
