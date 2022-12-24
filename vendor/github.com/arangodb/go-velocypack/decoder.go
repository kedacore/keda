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

// This code is heavily inspired by the Go sources.
// See https://golang.org/src/encoding/json/

package velocypack

import (
	"bytes"
	"encoding"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"strconv"
)

// A Decoder decodes velocypack values into Go structures.
type Decoder struct {
	r io.Reader
}

// Unmarshaler is implemented by types that can convert themselves from Velocypack.
type Unmarshaler interface {
	UnmarshalVPack(Slice) error
}

// NewDecoder creates a new Decoder that reads data from the given reader.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		r: r,
	}
}

// Unmarshal reads v from the given Velocypack encoded data slice.
//
// Unmarshal uses the inverse of the encodings that
// Marshal uses, allocating maps, slices, and pointers as necessary,
// with the following additional rules:
//
// To unmarshal VelocyPack into a pointer, Unmarshal first handles the case of
// the VelocyPack being the VelocyPack literal Null. In that case, Unmarshal sets
// the pointer to nil. Otherwise, Unmarshal unmarshals the VelocyPack into
// the value pointed at by the pointer. If the pointer is nil, Unmarshal
// allocates a new value for it to point to.
//
// To unmarshal VelocyPack into a value implementing the Unmarshaler interface,
// Unmarshal calls that value's UnmarshalVPack method, including
// when the input is a VelocyPack Null.
// Otherwise, if the value implements encoding.TextUnmarshaler
// and the input is a VelocyPack quoted string, Unmarshal calls that value's
// UnmarshalText method with the unquoted form of the string.
//
// To unmarshal VelocyPack into a struct, Unmarshal matches incoming object
// keys to the keys used by Marshal (either the struct field name or its tag),
// preferring an exact match but also accepting a case-insensitive match.
// Unmarshal will only set exported fields of the struct.
//
// To unmarshal VelocyPack into an interface value,
// Unmarshal stores one of these in the interface value:
//
//	bool, for VelocyPack Bool's
//	float64 for VelocyPack Double's
//	uint64 for VelocyPack UInt's
//	int64 for VelocyPack Int's
//	string, for VelocyPack String's
//	[]interface{}, for VelocyPack Array's
//	map[string]interface{}, for VelocyPack Object's
//	nil for VelocyPack Null.
//	[]byte for VelocyPack Binary.
//
// To unmarshal a VelocyPack array into a slice, Unmarshal resets the slice length
// to zero and then appends each element to the slice.
// As a special case, to unmarshal an empty VelocyPack array into a slice,
// Unmarshal replaces the slice with a new empty slice.
//
// To unmarshal a VelocyPack array into a Go array, Unmarshal decodes
// VelocyPack array elements into corresponding Go array elements.
// If the Go array is smaller than the VelocyPack array,
// the additional VelocyPack array elements are discarded.
// If the VelocyPack array is smaller than the Go array,
// the additional Go array elements are set to zero values.
//
// To unmarshal a VelocyPack object into a map, Unmarshal first establishes a map to
// use. If the map is nil, Unmarshal allocates a new map. Otherwise Unmarshal
// reuses the existing map, keeping existing entries. Unmarshal then stores
// key-value pairs from the VelocyPack object into the map. The map's key type must
// either be a string, an integer, or implement encoding.TextUnmarshaler.
//
// If a VelocyPack value is not appropriate for a given target type,
// or if a VelocyPack number overflows the target type, Unmarshal
// skips that field and completes the unmarshaling as best it can.
// If no more serious errors are encountered, Unmarshal returns
// an UnmarshalTypeError describing the earliest such error.
//
// The VelocyPack Null value unmarshals into an interface, map, pointer, or slice
// by setting that Go value to nil. Because null is often used in VelocyPack to mean
// ``not present,'' unmarshaling a VelocyPack Null into any other Go type has no effect
// on the value and produces no error.
//
func Unmarshal(data Slice, v interface{}) error {
	if err := unmarshalSlice(data, v); err != nil {
		return WithStack(err)
	}
	return nil
}

// Decode reads v from the decoder stream.
func (e *Decoder) Decode(v interface{}) error {
	s, err := SliceFromReader(e.r)
	if err != nil {
		return WithStack(err)
	}
	if err := unmarshalSlice(s, v); err != nil {
		return WithStack(err)
	}
	return nil
}

// unmarshalSlice reads v from the given slice.
func unmarshalSlice(data Slice, v interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return &InvalidUnmarshalError{reflect.TypeOf(v)}
	}

	d := &decodeState{}
	// We decode rv not rv.Elem because the Unmarshaler interface
	// test must be applied at the top level of the value.
	d.unmarshalValue(data, rv)
	return d.savedError
}

var (
	textUnmarshalerType = reflect.TypeOf(new(encoding.TextUnmarshaler)).Elem()
	numberType          = reflect.TypeOf(json.Number(""))
)

type decodeState struct {
	useNumber    bool
	errorContext struct { // provides context for type errors
		Struct string
		Field  string
	}
	savedError error
}

// error aborts the decoding by panicking with err.
func (d *decodeState) error(err error) {
	panic(d.addErrorContext(err))
}

// saveError saves the first err it is called with,
// for reporting at the end of the unmarshal.
func (d *decodeState) saveError(err error) {
	if d.savedError == nil {
		d.savedError = d.addErrorContext(err)
	}
}

// addErrorContext returns a new error enhanced with information from d.errorContext
func (d *decodeState) addErrorContext(err error) error {
	if d.errorContext.Struct != "" || d.errorContext.Field != "" {
		switch err := err.(type) {
		case *UnmarshalTypeError:
			err.Struct = d.errorContext.Struct
			err.Field = d.errorContext.Field
			return err
		}
	}
	return err
}

// unmarshalValue unmarshals any slice into given v.
func (d *decodeState) unmarshalValue(data Slice, v reflect.Value) {
	if !v.IsValid() {
		return
	}

	switch data.Type() {
	case Array:
		d.unmarshalArray(data, v)
	case Object:
		d.unmarshalObject(data, v)
	case Bool, Int, SmallInt, UInt, Double, Binary, BCD, String:
		d.unmarshalLiteral(data, v)
	}
}

// indirect walks down v allocating pointers as needed,
// until it gets to a non-pointer.
// if it encounters an Unmarshaler, indirect stops and returns that.
// if decodingNull is true, indirect stops at the last pointer so it can be set to nil.
func (d *decodeState) indirect(v reflect.Value, decodingNull bool) (Unmarshaler, json.Unmarshaler, encoding.TextUnmarshaler, reflect.Value) {
	// If v is a named type and is addressable,
	// start with its address, so that if the type has pointer methods,
	// we find them.
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		v = v.Addr()
	}
	for {
		// Load value from interface, but only if the result will be
		// usefully addressable.
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()
			if e.Kind() == reflect.Ptr && !e.IsNil() && (!decodingNull || e.Elem().Kind() == reflect.Ptr) {
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Ptr {
			break
		}

		if v.Elem().Kind() != reflect.Ptr && decodingNull && v.CanSet() {
			break
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		if v.Type().NumMethod() > 0 {
			if u, ok := v.Interface().(Unmarshaler); ok {
				return u, nil, nil, reflect.Value{}
			}
			if u, ok := v.Interface().(json.Unmarshaler); ok {
				return nil, u, nil, reflect.Value{}
			}
			if !decodingNull {
				if u, ok := v.Interface().(encoding.TextUnmarshaler); ok {
					return nil, nil, u, reflect.Value{}
				}
			}
		}
		v = v.Elem()
	}
	return nil, nil, nil, v
}

// unmarshalArray unmarshals an array slice into given v.
func (d *decodeState) unmarshalArray(data Slice, v reflect.Value) {
	// Check for unmarshaler.
	u, ju, ut, pv := d.indirect(v, false)
	if u != nil {
		if err := u.UnmarshalVPack(data); err != nil {
			d.error(err)
		}
		return
	}
	if ju != nil {
		json, err := data.JSONString()
		if err != nil {
			d.error(err)
		} else {
			if err := ju.UnmarshalJSON([]byte(json)); err != nil {
				d.error(err)
			}
		}
		return
	}
	if ut != nil {
		d.saveError(&UnmarshalTypeError{Value: "array", Type: v.Type()})
		return
	}

	v = pv

	// Check type of target.
	switch v.Kind() {
	case reflect.Interface:
		if v.NumMethod() == 0 {
			// Decoding into nil interface?  Switch to non-reflect code.
			v.Set(reflect.ValueOf(d.arrayInterface(data)))
			return
		}
		// Otherwise it's invalid.
		fallthrough
	default:
		d.saveError(&UnmarshalTypeError{Value: "array", Type: v.Type()})
		return
	case reflect.Array:
	case reflect.Slice:
		break
	}

	i := 0
	it, err := NewArrayIterator(data)
	if err != nil {
		d.error(err)
	}
	for it.IsValid() {
		value, err := it.Value()
		if err != nil {
			d.error(err)
		}

		// Get element of array, growing if necessary.
		if v.Kind() == reflect.Slice {
			// Grow slice if necessary
			if i >= v.Cap() {
				newcap := v.Cap() + v.Cap()/2
				if newcap < 4 {
					newcap = 4
				}
				newv := reflect.MakeSlice(v.Type(), v.Len(), newcap)
				reflect.Copy(newv, v)
				v.Set(newv)
			}
			if i >= v.Len() {
				v.SetLen(i + 1)
			}
		}

		if i < v.Len() {
			// Decode into element.
			d.unmarshalValue(value, v.Index(i))
		}
		i++
		if err := it.Next(); err != nil {
			d.error(err)
		}
	}

	if i < v.Len() {
		if v.Kind() == reflect.Array {
			// Array. Zero the rest.
			z := reflect.Zero(v.Type().Elem())
			for ; i < v.Len(); i++ {
				v.Index(i).Set(z)
			}
		} else {
			v.SetLen(i)
		}
	}
	if i == 0 && v.Kind() == reflect.Slice {
		v.Set(reflect.MakeSlice(v.Type(), 0, 0))
	}
}

// unmarshalObject unmarshals an object slice into given v.
func (d *decodeState) unmarshalObject(data Slice, v reflect.Value) {
	// Check for unmarshaler.
	u, ju, ut, pv := d.indirect(v, false)
	if u != nil {
		if err := u.UnmarshalVPack(data); err != nil {
			d.error(err)
		}
		return
	}
	if ju != nil {
		json, err := data.JSONString()
		if err != nil {
			d.error(err)
		} else {
			if err := ju.UnmarshalJSON([]byte(json)); err != nil {
				d.error(err)
			}
		}
		return
	}
	if ut != nil {
		d.saveError(&UnmarshalTypeError{Value: "object", Type: v.Type()})
		return
	}
	v = pv

	// Decoding into nil interface?  Switch to non-reflect code.
	if v.Kind() == reflect.Interface && v.NumMethod() == 0 {
		v.Set(reflect.ValueOf(d.objectInterface(data)))
		return
	}

	// Check type of target:
	//   struct or
	//   map[T1]T2 where T1 is string, an integer type,
	//             or an encoding.TextUnmarshaler
	switch v.Kind() {
	case reflect.Map:
		// Map key must either have string kind, have an integer kind,
		// or be an encoding.TextUnmarshaler.
		t := v.Type()
		switch t.Key().Kind() {
		case reflect.String,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		default:
			if !reflect.PtrTo(t.Key()).Implements(textUnmarshalerType) {
				d.saveError(&UnmarshalTypeError{Value: "object", Type: v.Type()})
				return
			}
		}
		if v.IsNil() {
			v.Set(reflect.MakeMap(t))
		}
	case reflect.Struct:
		// ok
	default:
		d.saveError(&UnmarshalTypeError{Value: "object", Type: v.Type()})
		return
	}

	var mapElem reflect.Value

	it, err := NewObjectIterator(data)
	if err != nil {
		d.error(err)
	}
	for it.IsValid() {
		key, err := it.Key(true)
		if err != nil {
			d.error(err)
		}
		keyUTF8, err := key.GetStringUTF8()
		if err != nil {
			d.error(err)
		}
		value, err := it.Value()
		if err != nil {
			d.error(err)
		}

		// Figure out field corresponding to key.
		var subv reflect.Value
		destring := false // whether the value is wrapped in a string to be decoded first

		if v.Kind() == reflect.Map {
			elemType := v.Type().Elem()
			if !mapElem.IsValid() {
				mapElem = reflect.New(elemType).Elem()
			} else {
				mapElem.Set(reflect.Zero(elemType))
			}
			subv = mapElem
		} else {
			var f *field
			fields := cachedTypeFields(v.Type())
			for i := range fields {
				ff := &fields[i]
				if bytes.Equal(ff.nameBytes, key) {
					f = ff
					break
				}
				if f == nil && ff.equalFold(ff.nameBytes, keyUTF8) {
					f = ff
				}
			}
			if f != nil {
				subv = v
				destring = f.quoted
				for _, i := range f.index {
					if subv.Kind() == reflect.Ptr {
						if subv.IsNil() {
							subv.Set(reflect.New(subv.Type().Elem()))
						}
						subv = subv.Elem()
					}
					subv = subv.Field(i)
				}
				d.errorContext.Field = f.name
				d.errorContext.Struct = v.Type().Name()
			}
		}

		if destring {
			// Value should be a string that we'll decode as JSON
			valueUTF8, err := value.GetStringUTF8()
			if err != nil {
				d.saveError(fmt.Errorf("json: invalid use of ,string struct tag, expected string, got %s in %v (%v)", value.Type(), subv.Type(), err))
			}
			v, err := ParseJSONFromUTF8(valueUTF8)
			if err != nil {
				d.saveError(err)
			} else {
				d.unmarshalValue(v, subv)
			}
		} else {
			d.unmarshalValue(value, subv)
		}

		// Write value back to map;
		// if using struct, subv points into struct already.
		if v.Kind() == reflect.Map {
			kt := v.Type().Key()
			var kv reflect.Value
			switch {
			case kt.Kind() == reflect.String:
				kv = reflect.ValueOf(keyUTF8).Convert(kt)
			case reflect.PtrTo(kt).Implements(textUnmarshalerType):
				kv = reflect.New(v.Type().Key())
				d.literalStore(key, kv, true)
				kv = kv.Elem()
			default:
				keyStr := string(keyUTF8)
				switch kt.Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					n, err := strconv.ParseInt(keyStr, 10, 64)
					if err != nil || reflect.Zero(kt).OverflowInt(n) {
						d.saveError(&UnmarshalTypeError{Value: "number " + keyStr, Type: kt})
						return
					}
					kv = reflect.ValueOf(n).Convert(kt)
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
					n, err := strconv.ParseUint(keyStr, 10, 64)
					if err != nil || reflect.Zero(kt).OverflowUint(n) {
						d.saveError(&UnmarshalTypeError{Value: "number " + keyStr, Type: kt})
						return
					}
					kv = reflect.ValueOf(n).Convert(kt)
				default:
					panic("json: Unexpected key type") // should never occur
				}
			}
			v.SetMapIndex(kv, subv)
		}

		d.errorContext.Struct = ""
		d.errorContext.Field = ""

		if err := it.Next(); err != nil {
			d.error(err)
		}
	}
}

// unmarshalLiteral unmarshals a literal slice into given v.
func (d *decodeState) unmarshalLiteral(data Slice, v reflect.Value) {
	d.literalStore(data, v, false)
}

// The xxxInterface routines build up a value to be stored
// in an empty interface. They are not strictly necessary,
// but they avoid the weight of reflection in this common case.

// valueInterface is like value but returns interface{}
func (d *decodeState) valueInterface(data Slice) interface{} {
	switch data.Type() {
	case Array:
		return d.arrayInterface(data)
	case Object:
		return d.objectInterface(data)
	default:
		return d.literalInterface(data)
	}
}

// arrayInterface is like array but returns []interface{}.
func (d *decodeState) arrayInterface(data Slice) []interface{} {
	l, err := data.Length()
	if err != nil {
		d.error(err)
	}
	v := make([]interface{}, 0, l)
	it, err := NewArrayIterator(data)
	if err != nil {
		d.error(err)
	}
	for it.IsValid() {
		value, err := it.Value()
		if err != nil {
			d.error(err)
		}

		v = append(v, d.valueInterface(value))

		// Move to next field
		if err := it.Next(); err != nil {
			d.error(err)
		}
	}
	return v
}

// objectInterface is like object but returns map[string]interface{}.
func (d *decodeState) objectInterface(data Slice) map[string]interface{} {
	m := make(map[string]interface{})
	it, err := NewObjectIterator(data)
	if err != nil {
		d.error(err)
	}
	for it.IsValid() {
		key, err := it.Key(true)
		if err != nil {
			d.error(err)
		}
		keyStr, err := key.GetString()
		if err != nil {
			d.error(err)
		}
		value, err := it.Value()
		if err != nil {
			d.error(err)
		}

		// Read value.
		m[keyStr] = d.valueInterface(value)

		// Move to next field
		if err := it.Next(); err != nil {
			d.error(err)
		}
	}
	return m
}

// literalInterface is like literal but returns an interface value.
func (d *decodeState) literalInterface(data Slice) interface{} {
	switch data.Type() {
	case Null:
		return nil

	case Bool:
		v, err := data.GetBool()
		if err != nil {
			d.error(err)
		}
		return v

	case String:
		v, err := data.GetString()
		if err != nil {
			d.error(err)
		}
		return v

	case Double:
		v, err := data.GetDouble()
		if err != nil {
			d.error(err)
		}
		return v

	case Int, SmallInt:
		v, err := data.GetInt()
		if err != nil {
			d.error(err)
		}
		intV := int(v)
		if int64(intV) == v {
			// Value fits in int
			return intV
		}
		return v

	case UInt:
		v, err := data.GetUInt()
		if err != nil {
			d.error(err)
		}
		return v

	case Binary:
		v, err := data.GetBinary()
		if err != nil {
			d.error(err)
		}
		return v

	default: // ??
		d.error(fmt.Errorf("unknown literal type: %s", data.Type()))
		return nil
	}
}

// literalStore decodes a literal stored in item into v.
//
// fromQuoted indicates whether this literal came from unwrapping a
// string from the ",string" struct tag option. this is used only to
// produce more helpful error messages.
func (d *decodeState) literalStore(item Slice, v reflect.Value, fromQuoted bool) {
	// Check for unmarshaler.
	if len(item) == 0 {
		//Empty string given
		d.saveError(fmt.Errorf("json: invalid use of ,string struct tag, trying to unmarshal empty slice into %v", v.Type()))
		return
	}
	isNull := item.IsNull() // null
	u, ju, ut, pv := d.indirect(v, isNull)
	if u != nil {
		if err := u.UnmarshalVPack(item); err != nil {
			d.error(err)
		}
		return
	}
	if ju != nil {
		json, err := item.JSONString()
		if err != nil {
			d.error(err)
		} else {
			if err := ju.UnmarshalJSON([]byte(json)); err != nil {
				d.error(err)
			}
		}
		return
	}
	if ut != nil {
		if !item.IsString() {
			//if item[0] != '"' {
			if fromQuoted {
				d.saveError(fmt.Errorf("json: invalid use of ,string struct tag, trying to unmarshal Slice of type %s into %v", item.Type(), v.Type()))
			} else {
				val := item.Type().String()
				d.saveError(&UnmarshalTypeError{Value: val, Type: v.Type()})
			}
			return
		}
		s, err := item.GetStringUTF8()
		if err != nil {
			if fromQuoted {
				d.error(fmt.Errorf("json: invalid use of ,string struct tag, trying to unmarshal slice of type %s into %v", item.Type(), v.Type()))
			} else {
				d.error(InternalError) // Out of sync
			}
		}
		if err := ut.UnmarshalText(s); err != nil {
			d.error(err)
		}
		return
	}

	v = pv

	switch item.Type() {
	case Null: // null
		// The main parser checks that only true and false can reach here,
		// but if this was a quoted string input, it could be anything.
		if fromQuoted /*&& string(item) != "null"*/ {
			d.saveError(fmt.Errorf("json: invalid use of ,string struct tag, trying to unmarshal %q into %v", item, v.Type()))
			break
		}
		switch v.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Map, reflect.Slice:
			v.Set(reflect.Zero(v.Type()))
			// otherwise, ignore null for primitives/string
		}
	case Bool: // true, false
		value, err := item.GetBool()
		if err != nil {
			d.error(err)
		}
		// The main parser checks that only true and false can reach here,
		// but if this was a quoted string input, it could be anything.
		if fromQuoted /*&& string(item) != "true" && string(item) != "false"*/ {
			d.saveError(fmt.Errorf("json: invalid use of ,string struct tag, trying to unmarshal %q into %v", item, v.Type()))
			break
		}
		switch v.Kind() {
		default:
			if fromQuoted {
				d.saveError(fmt.Errorf("json: invalid use of ,string struct tag, trying to unmarshal %q into %v", item, v.Type()))
			} else {
				d.saveError(&UnmarshalTypeError{Value: "bool", Type: v.Type()})
			}
		case reflect.Bool:
			v.SetBool(value)
		case reflect.Interface:
			if v.NumMethod() == 0 {
				v.Set(reflect.ValueOf(value))
			} else {
				d.saveError(&UnmarshalTypeError{Value: "bool", Type: v.Type()})
			}
		}

	case String: // string
		s, err := item.GetString()
		if err != nil {
			d.error(err)
		}
		switch v.Kind() {
		default:
			d.saveError(&UnmarshalTypeError{Value: "string", Type: v.Type()})
		case reflect.Slice:
			if v.Type().Elem().Kind() != reflect.Uint8 {
				d.saveError(&UnmarshalTypeError{Value: "string", Type: v.Type()})
				break
			}
			b, err := base64.StdEncoding.DecodeString(s)
			if err != nil {
				d.saveError(err)
				break
			}
			v.SetBytes(b)
		case reflect.String:
			v.SetString(string(s))
		case reflect.Interface:
			if v.NumMethod() == 0 {
				v.Set(reflect.ValueOf(string(s)))
			} else {
				d.saveError(&UnmarshalTypeError{Value: "string", Type: v.Type()})
			}
		}

	case Double:
		value, err := item.GetDouble()
		if err != nil {
			d.error(err)
		}
		switch v.Kind() {
		default:
			if v.Kind() == reflect.String && v.Type() == numberType {
				s, err := item.JSONString()
				if err != nil {
					d.error(err)
				}
				v.SetString(s)
				break
			}
			if fromQuoted {
				d.error(fmt.Errorf("json: invalid use of ,string struct tag, trying to unmarshal %q into %v", item, v.Type()))
			} else {
				d.error(&UnmarshalTypeError{Value: "number", Type: v.Type()})
			}
		case reflect.Interface:
			n, err := d.convertNumber(value)
			if err != nil {
				d.saveError(err)
				break
			}
			if v.NumMethod() != 0 {
				d.saveError(&UnmarshalTypeError{Value: "number", Type: v.Type()})
				break
			}
			v.Set(reflect.ValueOf(n))

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n := int64(value)
			if err != nil || v.OverflowInt(n) {
				d.saveError(&UnmarshalTypeError{Value: fmt.Sprintf("number %v", value), Type: v.Type()})
				break
			}
			v.SetInt(n)

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			n := uint64(value)
			if err != nil || v.OverflowUint(n) {
				d.saveError(&UnmarshalTypeError{Value: fmt.Sprintf("number %v", value), Type: v.Type()})
				break
			}
			v.SetUint(n)

		case reflect.Float32, reflect.Float64:
			n := value
			v.SetFloat(n)
		}

	case Int, SmallInt:
		value, err := item.GetInt()
		if err != nil {
			d.error(err)
		}
		switch v.Kind() {
		default:
			if v.Kind() == reflect.String && v.Type() == numberType {
				s, err := item.JSONString()
				if err != nil {
					d.error(err)
				}
				v.SetString(s)
				break
			}
			if fromQuoted {
				d.error(fmt.Errorf("json: invalid use of ,string struct tag, trying to unmarshal %q into %v", item, v.Type()))
			} else {
				d.error(&UnmarshalTypeError{Value: "number", Type: v.Type()})
			}
		case reflect.Interface:
			var n interface{}
			intValue := int(value)
			if int64(intValue) == value {
				// When the value fits in an int, use int type.
				n, err = d.convertNumber(intValue)
			} else {
				n, err = d.convertNumber(value)
			}
			if err != nil {
				d.saveError(err)
				break
			}
			if v.NumMethod() != 0 {
				d.saveError(&UnmarshalTypeError{Value: "number", Type: v.Type()})
				break
			}
			v.Set(reflect.ValueOf(n))

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n := value
			if err != nil || v.OverflowInt(n) {
				d.saveError(&UnmarshalTypeError{Value: fmt.Sprintf("number %v", value), Type: v.Type()})
				break
			}
			v.SetInt(n)

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			n := uint64(value)
			if err != nil || v.OverflowUint(n) {
				d.saveError(&UnmarshalTypeError{Value: fmt.Sprintf("number %v", value), Type: v.Type()})
				break
			}
			v.SetUint(n)

		case reflect.Float32, reflect.Float64:
			n := float64(value)
			if err != nil || v.OverflowFloat(n) {
				d.saveError(&UnmarshalTypeError{Value: fmt.Sprintf("number %v", value), Type: v.Type()})
				break
			}
			v.SetFloat(n)
		}

	case UInt:
		value, err := item.GetUInt()
		if err != nil {
			d.error(err)
		}
		switch v.Kind() {
		default:
			if v.Kind() == reflect.String && v.Type() == numberType {
				s, err := item.JSONString()
				if err != nil {
					d.error(err)
				}
				v.SetString(s)
				break
			}
			if fromQuoted {
				d.error(fmt.Errorf("json: invalid use of ,string struct tag, trying to unmarshal %q into %v", item, v.Type()))
			} else {
				d.error(&UnmarshalTypeError{Value: "number", Type: v.Type()})
			}
		case reflect.Interface:
			n, err := d.convertNumber(value)
			if err != nil {
				d.saveError(err)
				break
			}
			if v.NumMethod() != 0 {
				d.saveError(&UnmarshalTypeError{Value: "number", Type: v.Type()})
				break
			}
			v.Set(reflect.ValueOf(n))

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n := int64(value)
			if err != nil || v.OverflowInt(n) {
				d.saveError(&UnmarshalTypeError{Value: fmt.Sprintf("number %v", value), Type: v.Type()})
				break
			}
			v.SetInt(n)

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			n := value
			if err != nil || v.OverflowUint(n) {
				d.saveError(&UnmarshalTypeError{Value: fmt.Sprintf("number %v", value), Type: v.Type()})
				break
			}
			v.SetUint(n)

		case reflect.Float32, reflect.Float64:
			n := float64(value)
			if err != nil || v.OverflowFloat(n) {
				d.saveError(&UnmarshalTypeError{Value: fmt.Sprintf("number %v", value), Type: v.Type()})
				break
			}
			v.SetFloat(n)
		}

	case Binary:
		value, err := item.GetBinary()
		if err != nil {
			d.error(err)
		}
		switch v.Kind() {
		default:
			d.saveError(&UnmarshalTypeError{Value: "string", Type: v.Type()})
		case reflect.Slice:
			if v.Type().Elem().Kind() != reflect.Uint8 {
				d.saveError(&UnmarshalTypeError{Value: "binary", Type: v.Type()})
				break
			}
			v.SetBytes(value)
		case reflect.Interface:
			if v.NumMethod() == 0 {
				v.Set(reflect.ValueOf(value))
			} else {
				d.saveError(&UnmarshalTypeError{Value: "binary", Type: v.Type()})
			}
		}

	default: // number
		d.error(fmt.Errorf("Unknown type %s", item.Type()))
	}
}

// convertNumber converts the number literal s to a float64 or a Number
// depending on the setting of d.useNumber.
func (d *decodeState) convertNumber(s interface{}) (interface{}, error) {
	if d.useNumber {
		return json.Number(fmt.Sprintf("%v", s)), nil
	}
	return s, nil
}
