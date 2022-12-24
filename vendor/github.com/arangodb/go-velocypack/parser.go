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

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"strings"
)

// ParserOptions controls how the Parser builds Velocypack.
type ParserOptions struct {
	// If set, all Array's will be unindexed.
	BuildUnindexedArrays bool
	// If set, all Objects's will be unindexed.
	BuildUnindexedObjects bool
}

// Parser is used to build VPack structures from JSON.
type Parser struct {
	options ParserOptions
	decoder *json.Decoder
	builder *Builder
}

// ParseJSON parses JSON from the given reader and returns the
// VPack equivalent.
func ParseJSON(r io.Reader, options ...ParserOptions) (Slice, error) {
	builder := &Builder{}
	p := NewParser(r, builder, options...)
	if err := p.Parse(); err != nil {
		return nil, WithStack(err)
	}
	slice, err := builder.Slice()
	if err != nil {
		return nil, WithStack(err)
	}
	return slice, nil
}

// ParseJSONFromString parses the given JSON string and returns the
// VPack equivalent.
func ParseJSONFromString(json string, options ...ParserOptions) (Slice, error) {
	return ParseJSON(strings.NewReader(json), options...)
}

// ParseJSONFromUTF8 parses the given JSON string and returns the
// VPack equivalent.
func ParseJSONFromUTF8(json []byte, options ...ParserOptions) (Slice, error) {
	return ParseJSON(bytes.NewReader(json), options...)
}

// NewParser initializes a new Parser with JSON from the given reader and
// it will store the parsers output in the given builder.
func NewParser(r io.Reader, builder *Builder, options ...ParserOptions) *Parser {
	d := json.NewDecoder(r)
	d.UseNumber()
	p := &Parser{
		decoder: d,
		builder: builder,
	}
	if len(options) > 0 {
		p.options = options[0]
	}
	return p
}

// Parse JSON from the parsers reader and build VPack structures in the
// parsers builder.
func (p *Parser) Parse() error {
	for {
		t, err := p.decoder.Token()
		if err == io.EOF {
			break
		} else if serr, ok := err.(*json.SyntaxError); ok {
			return WithStack(&ParseError{msg: err.Error(), Offset: serr.Offset})
		} else if err != nil {
			return WithStack(&ParseError{msg: err.Error()})
		}
		switch x := t.(type) {
		case nil:
			if err := p.builder.AddValue(NewNullValue()); err != nil {
				return WithStack(err)
			}
		case bool:
			if err := p.builder.AddValue(NewBoolValue(x)); err != nil {
				return WithStack(err)
			}
		case json.Number:
			if xu, err := strconv.ParseUint(string(x), 10, 64); err == nil {
				if err := p.builder.AddValue(NewUIntValue(xu)); err != nil {
					return WithStack(err)
				}
			} else if xi, err := x.Int64(); err == nil {
				if err := p.builder.AddValue(NewIntValue(xi)); err != nil {
					return WithStack(err)
				}
			} else {
				if xf, err := x.Float64(); err == nil {
					if err := p.builder.AddValue(NewDoubleValue(xf)); err != nil {
						return WithStack(err)
					}
				} else {
					return WithStack(&ParseError{msg: err.Error()})
				}
			}
		case string:
			if err := p.builder.AddValue(NewStringValue(x)); err != nil {
				return WithStack(err)
			}
		case json.Delim:
			switch x {
			case '[':
				if err := p.builder.OpenArray(p.options.BuildUnindexedArrays); err != nil {
					return WithStack(err)
				}
			case '{':
				if err := p.builder.OpenObject(p.options.BuildUnindexedObjects); err != nil {
					return WithStack(err)
				}
			case ']', '}':
				if err := p.builder.Close(); err != nil {
					return WithStack(err)
				}
			}
		}
	}
	return nil
}
