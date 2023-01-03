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
	"encoding/json"
	"io"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"sync"
)

// An Encoder encodes Go structures into velocypack values written to an output stream.
type Encoder struct {
	b Builder
	w io.Writer
}

// Marshaler is implemented by types that can convert themselves into Velocypack.
type Marshaler interface {
	MarshalVPack() (Slice, error)
}

// NewEncoder creates a new Encoder that writes output to the given writer.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w: w,
	}
}

// Marshal writes the Velocypack encoding of v to a buffer and returns that buffer.
//
// Marshal traverses the value v recursively.
// If an encountered value implements the Marshaler interface
// and is not a nil pointer, Marshal calls its MarshalVPack method
// to produce Velocypack.
// If an encountered value implements the json.Marshaler interface
// and is not a nil pointer, Marshal calls its MarshalJSON method
// to produce JSON and converts the resulting JSON to VelocyPack.
// If no MarshalVPack or MarshalJSON method is present but the
// value implements encoding.TextMarshaler instead, Marshal calls
// its MarshalText method and encodes the result as a Velocypack string.
// The nil pointer exception is not strictly necessary
// but mimics a similar, necessary exception in the behavior of
// UnmarshalVPack.
//
// Otherwise, Marshal uses the following type-dependent default encodings:
//
// Boolean values encode as Velocypack booleans.
//
// Floating point, integer, and Number values encode as Velocypack Int's, UInt's and Double's.
//
// String values encode as Velocypack strings.
//
// Array and slice values encode as Velocypack arrays, except that
// []byte encodes as Velocypack Binary data, and a nil slice
// encodes as the Null Velocypack value.
//
// Struct values encode as Velocypack objects.
// The encoding follows the same rules as specified for json.Marshal.
// This means that all `json` tags are fully supported.
//
// Map values encode as Velocypack objects.
// The encoding follows the same rules as specified for json.Marshal.
//
// Pointer values encode as the value pointed to.
// A nil pointer encodes as the Null Velocypack value.
//
// Interface values encode as the value contained in the interface.
// A nil interface value encodes as the Null Velocypack value.
//
// Channel, complex, and function values cannot be encoded in Velocypack.
// Attempting to encode such a value causes Marshal to return
// an UnsupportedTypeError.
//
// Velocypack cannot represent cyclic data structures and Marshal does not
// handle them. Passing cyclic structures to Marshal will result in
// an infinite recursion.
//
func Marshal(v interface{}) (result Slice, err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			if s, ok := r.(string); ok {
				panic(s)
			}
			err = r.(error)
		}
	}()
	var b Builder
	reflectValue(&b, reflect.ValueOf(v), encoderOptions{})
	return b.Slice()
}

// Encode writes the Velocypack encoding of v to the stream.
func (e *Encoder) Encode(v interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			if s, ok := r.(string); ok {
				panic(s)
			}
			err = r.(error)
		}
	}()
	e.b.Clear()
	reflectValue(&e.b, reflect.ValueOf(v), encoderOptions{})
	if _, err := e.b.WriteTo(e.w); err != nil {
		return WithStack(err)
	}
	return nil
}

// Builder returns a reference to the builder used in the given encoder.
func (e *Encoder) Builder() *Builder {
	return &e.b
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

func reflectValue(b *Builder, v reflect.Value, options encoderOptions) {
	valueEncoder(v)(b, v, options)
}

type encoderOptions struct {
	quoted bool
}

type encoderFunc func(b *Builder, v reflect.Value, options encoderOptions)

var encoderCache struct {
	sync.RWMutex
	m map[reflect.Type]encoderFunc
}

func valueEncoder(v reflect.Value) encoderFunc {
	if !v.IsValid() {
		return invalidValueEncoder
	}
	return typeEncoder(v.Type())
}

var (
	marshalerType     = reflect.TypeOf(new(Marshaler)).Elem()
	jsonMarshalerType = reflect.TypeOf(new(json.Marshaler)).Elem()
	textMarshalerType = reflect.TypeOf(new(encoding.TextMarshaler)).Elem()
	nullValue         = NewNullValue()
)

func typeEncoder(t reflect.Type) encoderFunc {
	encoderCache.RLock()
	f := encoderCache.m[t]
	encoderCache.RUnlock()
	if f != nil {
		return f
	}

	// To deal with recursive types, populate the map with an
	// indirect func before we build it. This type waits on the
	// real func (f) to be ready and then calls it. This indirect
	// func is only used for recursive types.
	encoderCache.Lock()
	if encoderCache.m == nil {
		encoderCache.m = make(map[reflect.Type]encoderFunc)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	encoderCache.m[t] = func(b *Builder, v reflect.Value, options encoderOptions) {
		wg.Wait()
		f(b, v, options)
	}
	encoderCache.Unlock()

	// Compute fields without lock.
	// Might duplicate effort but won't hold other computations back.
	f = newTypeEncoder(t, true)
	wg.Done()
	encoderCache.Lock()
	encoderCache.m[t] = f
	encoderCache.Unlock()
	return f
}

// newTypeEncoder constructs an encoderFunc for a type.
// The returned encoder only checks CanAddr when allowAddr is true.
func newTypeEncoder(t reflect.Type, allowAddr bool) encoderFunc {
	if t.Implements(marshalerType) {
		return marshalerEncoder
	}
	if t.Implements(jsonMarshalerType) {
		return jsonMarshalerEncoder
	}
	if t.Kind() != reflect.Ptr && allowAddr {
		if reflect.PtrTo(t).Implements(marshalerType) {
			return newCondAddrEncoder(addrMarshalerEncoder, newTypeEncoder(t, false))
		}
		if reflect.PtrTo(t).Implements(jsonMarshalerType) {
			return newCondAddrEncoder(addrJSONMarshalerEncoder, newTypeEncoder(t, false))
		}
	}

	if t.Implements(textMarshalerType) {
		return textMarshalerEncoder
	}
	if t.Kind() != reflect.Ptr && allowAddr {
		if reflect.PtrTo(t).Implements(textMarshalerType) {
			return newCondAddrEncoder(addrTextMarshalerEncoder, newTypeEncoder(t, false))
		}
	}

	switch t.Kind() {
	case reflect.Bool:
		return boolEncoder
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intEncoder
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uintEncoder
	case reflect.Float32, reflect.Float64:
		return doubleEncoder
	case reflect.String:
		return stringEncoder
	case reflect.Interface:
		return interfaceEncoder
	case reflect.Struct:
		return newStructEncoder(t)
	case reflect.Map:
		return newMapEncoder(t)
	case reflect.Slice:
		return newSliceEncoder(t)
	case reflect.Array:
		return newArrayEncoder(t)
	case reflect.Ptr:
		return newPtrEncoder(t)
	default:
		return unsupportedTypeEncoder
	}
}

func invalidValueEncoder(b *Builder, v reflect.Value, options encoderOptions) {
	b.addInternal(nullValue)
}

func marshalerEncoder(b *Builder, v reflect.Value, options encoderOptions) {
	if v.Kind() == reflect.Ptr && v.IsNil() {
		b.addInternal(nullValue)
		return
	}
	m, ok := v.Interface().(Marshaler)
	if !ok {
		b.addInternal(nullValue)
		return
	}
	if vpack, err := m.MarshalVPack(); err != nil {
		panic(&MarshalerError{v.Type(), err})
	} else {
		b.addInternal(NewSliceValue(vpack))
	}
}

func jsonMarshalerEncoder(b *Builder, v reflect.Value, options encoderOptions) {
	if v.Kind() == reflect.Ptr && v.IsNil() {
		b.addInternal(nullValue)
		return
	}
	m, ok := v.Interface().(json.Marshaler)
	if !ok {
		b.addInternal(nullValue)
		return
	}
	if json, err := m.MarshalJSON(); err != nil {
		panic(&MarshalerError{v.Type(), err})
	} else {
		// Convert JSON to vpack
		if slice, err := ParseJSON(bytes.NewReader(json)); err != nil {
			panic(&MarshalerError{v.Type(), err})
		} else {
			b.addInternal(NewSliceValue(slice))
		}
	}
}

func addrMarshalerEncoder(b *Builder, v reflect.Value, options encoderOptions) {
	va := v.Addr()
	if va.IsNil() {
		b.addInternal(nullValue)
		return
	}
	m := va.Interface().(Marshaler)
	if vpack, err := m.MarshalVPack(); err != nil {
		panic(&MarshalerError{Type: v.Type(), Err: err})
	} else {
		if err = b.AddValue(NewSliceValue(vpack)); err != nil {
			panic(&MarshalerError{Type: v.Type(), Err: err})
		}
	}
}

func addrJSONMarshalerEncoder(b *Builder, v reflect.Value, options encoderOptions) {
	va := v.Addr()
	if va.IsNil() {
		b.addInternal(nullValue)
		return
	}
	m := va.Interface().(json.Marshaler)
	if json, err := m.MarshalJSON(); err != nil {
		panic(&MarshalerError{Type: v.Type(), Err: err})
	} else {
		if slice, err := ParseJSON(bytes.NewReader(json)); err != nil {
			panic(&MarshalerError{v.Type(), err})
		} else {
			// copy VPack into buffer, checking validity.
			b.buf.Write(slice)
		}
	}
}

func textMarshalerEncoder(b *Builder, v reflect.Value, options encoderOptions) {
	if v.Kind() == reflect.Ptr && v.IsNil() {
		b.addInternal(nullValue)
		return
	}
	m := v.Interface().(encoding.TextMarshaler)
	text, err := m.MarshalText()
	if err != nil {
		panic(&MarshalerError{v.Type(), err})
	}
	b.addInternal(NewStringValue(string(text)))
}

func addrTextMarshalerEncoder(b *Builder, v reflect.Value, options encoderOptions) {
	va := v.Addr()
	if va.IsNil() {
		b.addInternal(nullValue)
		return
	}
	m := va.Interface().(encoding.TextMarshaler)
	text, err := m.MarshalText()
	if err != nil {
		panic(&MarshalerError{v.Type(), err})
	}
	b.addInternal(NewStringValue(string(text)))
}

func boolEncoder(b *Builder, v reflect.Value, options encoderOptions) {
	if options.quoted {
		b.addInternal(NewStringValue(strconv.FormatBool(v.Bool())))
	} else {
		b.addInternal(NewBoolValue(v.Bool()))
	}
}

func intEncoder(b *Builder, v reflect.Value, options encoderOptions) {
	if options.quoted {
		b.addInternal(NewStringValue(strconv.FormatInt(v.Int(), 10)))
	} else {
		b.addInternal(NewIntValue(v.Int()))
	}
}

func uintEncoder(b *Builder, v reflect.Value, options encoderOptions) {
	if options.quoted {
		b.addInternal(NewStringValue(strconv.FormatUint(v.Uint(), 10)))
	} else {
		b.addInternal(NewUIntValue(v.Uint()))
	}
}

func doubleEncoder(b *Builder, v reflect.Value, options encoderOptions) {
	if options.quoted {
		b.addInternal(NewStringValue(formatDouble(v.Float())))
	} else {
		b.addInternal(NewDoubleValue(v.Float()))
	}
}

func stringEncoder(b *Builder, v reflect.Value, options encoderOptions) {
	s := v.String()
	if options.quoted {
		raw, _ := json.Marshal(s)
		s = string(raw)
	}
	b.addInternal(NewStringValue(s))
}

func interfaceEncoder(b *Builder, v reflect.Value, options encoderOptions) {
	if v.IsNil() {
		b.addInternal(nullValue)
		return
	}
	vElem := v.Elem()
	valueEncoder(vElem)(b, vElem, options)
}

func unsupportedTypeEncoder(b *Builder, v reflect.Value, options encoderOptions) {
	panic(&UnsupportedTypeError{v.Type()})
}

type structEncoder struct {
	fields    []field
	fieldEncs []encoderFunc
}

func (se *structEncoder) encode(b *Builder, v reflect.Value, options encoderOptions) {
	if err := b.OpenObject(); err != nil {
		panic(err)
	}
	for i, f := range se.fields {
		fv := fieldByIndex(v, f.index)
		if !fv.IsValid() || f.omitEmpty && isEmptyValue(fv) {
			continue
		}
		// Key
		_, err := b.addInternalKey(f.name)
		if err != nil {
			panic(err)
		}
		// Value
		options.quoted = f.quoted
		se.fieldEncs[i](b, fv, options)
	}
	if err := b.Close(); err != nil {
		panic(err)
	}
}

func newStructEncoder(t reflect.Type) encoderFunc {
	fields := cachedTypeFields(t)
	se := &structEncoder{
		fields:    fields,
		fieldEncs: make([]encoderFunc, len(fields)),
	}
	for i, f := range fields {
		se.fieldEncs[i] = typeEncoder(typeByIndex(t, f.index))
	}
	return se.encode
}

type mapEncoder struct {
	elemEnc encoderFunc
}

func (e *mapEncoder) encode(b *Builder, v reflect.Value, options encoderOptions) {
	if v.IsNil() {
		b.addInternal(nullValue)
		return
	}
	if err := b.OpenObject(); err != nil {
		panic(err)
	}

	// Extract and sort the keys.
	keys := v.MapKeys()
	sv := make(reflectWithStringSlice, len(keys))
	for i, v := range keys {
		sv[i].v = v
		if err := sv[i].resolve(); err != nil {
			panic(&MarshalerError{v.Type(), err})
		}
	}
	sort.Sort(sv)

	for _, kv := range sv {
		// Key
		_, err := b.addInternalKey(kv.s)
		if err != nil {
			panic(err)
		}
		// Value
		e.elemEnc(b, v.MapIndex(kv.v), options)
	}
	if err := b.Close(); err != nil {
		panic(err)
	}
}

func newMapEncoder(t reflect.Type) encoderFunc {
	switch t.Key().Kind() {
	case reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
	default:
		if !t.Key().Implements(textMarshalerType) {
			return unsupportedTypeEncoder
		}
	}
	me := &mapEncoder{typeEncoder(t.Elem())}
	return me.encode
}

func encodeByteSlice(b *Builder, v reflect.Value, options encoderOptions) {
	if v.IsNil() {
		b.addInternal(nullValue)
		return
	}
	b.addInternal(NewBinaryValue(v.Bytes()))
}

// sliceEncoder just wraps an arrayEncoder, checking to make sure the value isn't nil.
type sliceEncoder struct {
	arrayEnc encoderFunc
}

func (se *sliceEncoder) encode(b *Builder, v reflect.Value, options encoderOptions) {
	if v.IsNil() {
		b.addInternal(nullValue)
		return
	}
	se.arrayEnc(b, v, options)
}

func newSliceEncoder(t reflect.Type) encoderFunc {
	// Byte slices get special treatment; arrays don't.
	if t.Elem().Kind() == reflect.Uint8 {
		p := reflect.PtrTo(t.Elem())
		if !p.Implements(marshalerType) && !p.Implements(jsonMarshalerType) && !p.Implements(textMarshalerType) {
			return encodeByteSlice
		}
	}
	enc := &sliceEncoder{newArrayEncoder(t)}
	return enc.encode
}

type arrayEncoder struct {
	elemEnc encoderFunc
}

func (ae *arrayEncoder) encode(b *Builder, v reflect.Value, options encoderOptions) {
	if err := b.OpenArray(); err != nil {
		panic(err)
	}
	n := v.Len()
	for i := 0; i < n; i++ {
		ae.elemEnc(b, v.Index(i), options)
	}
	if err := b.Close(); err != nil {
		panic(err)
	}
}

func newArrayEncoder(t reflect.Type) encoderFunc {
	enc := &arrayEncoder{typeEncoder(t.Elem())}
	return enc.encode
}

type ptrEncoder struct {
	elemEnc encoderFunc
}

func (pe *ptrEncoder) encode(b *Builder, v reflect.Value, options encoderOptions) {
	if v.IsNil() {
		b.addInternal(nullValue)
		return
	}
	pe.elemEnc(b, v.Elem(), options)
}

func newPtrEncoder(t reflect.Type) encoderFunc {
	enc := &ptrEncoder{typeEncoder(t.Elem())}
	return enc.encode
}

type condAddrEncoder struct {
	canAddrEnc, elseEnc encoderFunc
}

func (ce *condAddrEncoder) encode(b *Builder, v reflect.Value, options encoderOptions) {
	if v.CanAddr() {
		ce.canAddrEnc(b, v, options)
	} else {
		ce.elseEnc(b, v, options)
	}
}

// newCondAddrEncoder returns an encoder that checks whether its value
// CanAddr and delegates to canAddrEnc if so, else to elseEnc.
func newCondAddrEncoder(canAddrEnc, elseEnc encoderFunc) encoderFunc {
	enc := &condAddrEncoder{canAddrEnc: canAddrEnc, elseEnc: elseEnc}
	return enc.encode
}

type reflectWithString struct {
	v reflect.Value
	s string
}

func (w *reflectWithString) resolve() error {
	if w.v.Kind() == reflect.String {
		w.s = w.v.String()
		return nil
	}
	if tm, ok := w.v.Interface().(encoding.TextMarshaler); ok {
		buf, err := tm.MarshalText()
		w.s = string(buf)
		return err
	}
	switch w.v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		w.s = strconv.FormatInt(w.v.Int(), 10)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		w.s = strconv.FormatUint(w.v.Uint(), 10)
		return nil
	}
	panic("unexpected map key type")
}

type reflectWithStringSlice []reflectWithString

// Len is the number of elements in the collection.
func (l reflectWithStringSlice) Len() int {
	return len(l)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (l reflectWithStringSlice) Less(i, j int) bool {
	return l[i].s < l[j].s
}

// Swap swaps the elements with indexes i and j.
func (l reflectWithStringSlice) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}
