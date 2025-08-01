// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Deep equality test via reflection on only public methods and members.
// This is deeply hacky as we're not inside of the reflect package; this will not
// be as performant as reflect.DeepEqual but will handle proto objects at any point
// in an object hierarchy
package temporalproto

import (
	"reflect"
	"regexp"
	"unsafe"

	"google.golang.org/protobuf/proto"
)

// During deepValueEqual, must keep track of checks that are
// in progress. The comparison algorithm assumes that all
// checks in progress are true when it reencounters them.
// Visited comparisons are stored in a map indexed by visit.
type visit struct {
	a1  unsafe.Pointer
	a2  unsafe.Pointer
	typ reflect.Type
}

var publicMethodRgx = regexp.MustCompile("^[A-Z]")

func pointerTo(v reflect.Value) unsafe.Pointer {
	if v.CanAddr() {
		return v.Addr().UnsafePointer()
	}
	return v.UnsafePointer()
}

// Tests for deep equality using reflected types. The map argument tracks
// comparisons that have already been seen, which allows short circuiting on
// recursive types.
func deepValueEqual(v1, v2 reflect.Value, visited map[visit]bool) bool {
	if !v1.IsValid() || !v2.IsValid() {
		return v1.IsValid() == v2.IsValid()
	}
	if v1.Type() != v2.Type() {
		return false
	}

	// We want to avoid putting more in the visited map than we need to.
	// For any possible reference cycle that might be encountered,
	// hard(v1, v2) needs to return true for at least one of the types in the cycle,
	// and it's safe and valid to get Value's internal pointer.
	hard := func(v1, v2 reflect.Value) bool {
		switch v1.Kind() {
		case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface:
			// Nil pointers cannot be cyclic. Avoid putting them in the visited map.
			return !v1.IsNil() && !v2.IsNil()
		}
		return false
	}

	if hard(v1, v2) {
		addr1 := pointerTo(v1)
		addr2 := pointerTo(v2)
		if uintptr(addr1) > uintptr(addr2) {
			// Canonicalize order to reduce number of entries in visited.
			// Assumes non-moving garbage collector.
			addr1, addr2 = addr2, addr1
		}

		// Short circuit if references are already seen.
		typ := v1.Type()
		v := visit{addr1, addr2, typ}
		if visited[v] {
			return true
		}

		// Remember for later.
		visited[v] = true
	}

	switch v1.Kind() {
	case reflect.Array:
		for i := 0; i < v1.Len(); i++ {
			if !deepValueEqual(v1.Index(i), v2.Index(i), visited) {
				return false
			}
		}
		return true
	case reflect.Slice:
		if v1.IsNil() != v2.IsNil() {
			return false
		}
		if v1.Len() != v2.Len() {
			return false
		}
		if v1.UnsafePointer() == v2.UnsafePointer() {
			return true
		}
		for i := 0; i < v1.Len(); i++ {
			if !deepValueEqual(v1.Index(i), v2.Index(i), visited) {
				return false
			}
		}
		return true
	case reflect.Interface:
		if v1.IsNil() || v2.IsNil() {
			return v1.IsNil() == v2.IsNil()
		}
		return deepValueEqual(v1.Elem(), v2.Elem(), visited)
	case reflect.Pointer:
		if v1.UnsafePointer() == v2.UnsafePointer() {
			return true
		}

		v1v, ok := v1.Interface().(proto.Message)
		v2v, ok2 := v2.Interface().(proto.Message)
		if ok && ok2 {
			return proto.Equal(v1v, v2v)
		}

		return deepValueEqual(v1.Elem(), v2.Elem(), visited)
	case reflect.Struct:
		for i, n := 0, v1.NumField(); i < n; i++ {
			if !publicMethodRgx.MatchString(v1.Field(i).String()) {
				continue
			}
			if !deepValueEqual(v1.Field(i), v2.Field(i), visited) {
				return false
			}
		}
		return true
	case reflect.Map:
		if v1.IsNil() != v2.IsNil() {
			return false
		}
		if v1.Len() != v2.Len() {
			return false
		}
		if v1.UnsafePointer() == v2.UnsafePointer() {
			return true
		}
		for _, k := range v1.MapKeys() {
			val1 := v1.MapIndex(k)
			val2 := v2.MapIndex(k)
			if !val1.IsValid() || !val2.IsValid() || !deepValueEqual(val1, val2, visited) {
				return false
			}
		}
		return true
	case reflect.Func:
		if v1.IsNil() && v2.IsNil() {
			return true
		}
		// Can't do better than this:
		return false
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v1.Int() == v2.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v1.Uint() == v2.Uint()
	case reflect.String:
		return v1.String() == v2.String()
	case reflect.Bool:
		return v1.Bool() == v2.Bool()
	case reflect.Float32, reflect.Float64:
		return v1.Float() == v2.Float()
	case reflect.Complex64, reflect.Complex128:
		return v1.Complex() == v2.Complex()
	default:
		// Normal equality suffices
		return v1.Elem().Interface() == v2.Elem().Interface()
	}
}

// DeepEqual behaves as reflect.DeepEqual except:
// 1. Proto structs will be compared using proto.Equal when encountered
// 2. Only public member variables will be compared
//
// DeepEqual should _only_ be used when proto.Equal or reflect.DeepEqual
// aren't useable, such as when comparing normal Go structs that have
// proto structs as members
func DeepEqual(x, y any) bool {
	if x == nil || y == nil {
		return x == y
	}
	v1 := reflect.ValueOf(x)
	v2 := reflect.ValueOf(y)
	if v1.Type() != v2.Type() {
		return false
	}
	return deepValueEqual(v1, v2, make(map[visit]bool))
}
