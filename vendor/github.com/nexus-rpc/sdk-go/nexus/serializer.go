package nexus

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
)

// A Reader is a container for a [Header] and an [io.Reader].
// It is used to stream inputs and outputs in the various client and server APIs.
type Reader struct {
	// ReaderCloser contains request or response data. May be nil for empty data.
	io.ReadCloser
	// Header that should include information on how to deserialize this content.
	// Headers constructed by the framework always have lower case keys.
	// User provided keys are considered case-insensitive by the framework.
	Header Header
}

// A Content is a container for a [Header] and a byte slice.
// It is used by the SDK's [Serializer] interface implementations.
type Content struct {
	// Header that should include information on how to deserialize this content.
	// Headers constructed by the framework always have lower case keys.
	// User provided keys are considered case-insensitive by the framework.
	Header Header
	// Data contains request or response data. May be nil for empty data.
	Data []byte
}

// A LazyValue holds a value encoded in an underlying [Reader].
//
// ⚠️ When a LazyValue is returned from a client - if directly accessing the [Reader] - it must be read it in its
// entirety and closed to free up the associated HTTP connection. Otherwise the [LazyValue.Consume] method must be
// called.
//
// ⚠️ When a LazyValue is passed to a server handler, it must not be used after the returning from the handler method.
type LazyValue struct {
	serializer Serializer
	Reader     *Reader
}

// Create a new [LazyValue] from a given serializer and reader.
func NewLazyValue(serializer Serializer, reader *Reader) *LazyValue {
	return &LazyValue{
		serializer: serializer,
		Reader:     reader,
	}
}

// Consume consumes the lazy value, decodes it from the underlying [Reader], and stores the result in the value pointed
// to by v.
//
//	var v int
//	err := lazyValue.Consume(&v)
func (l *LazyValue) Consume(v any) error {
	defer l.Reader.Close()
	data, err := io.ReadAll(l.Reader)
	if err != nil {
		return err
	}
	return l.serializer.Deserialize(&Content{
		Header: l.Reader.Header,
		Data:   data,
	}, v)
}

// Serializer is used by the framework to serialize/deserialize input and output.
// To customize serialization logic, implement this interface and provide your implementation to framework methods such
// as [NewClient] and [NewHTTPHandler].
// By default, the SDK supports serialization of JSONables, byte slices, and nils.
type Serializer interface {
	// Serialize encodes a value into a [Content].
	Serialize(any) (*Content, error)
	// Deserialize decodes a [Content] into a given reference.
	Deserialize(*Content, any) error
}

var anyType = reflect.TypeOf((*any)(nil)).Elem()

var errSerializerIncompatible = errors.New("incompatible serializer")

type serializerChain []Serializer

func (c serializerChain) Serialize(v any) (*Content, error) {
	for _, l := range c {
		p, err := l.Serialize(v)
		if err != nil {
			if errors.Is(err, errSerializerIncompatible) {
				continue
			}
			return nil, err
		}
		return p, nil
	}
	return nil, errSerializerIncompatible
}

func (c serializerChain) Deserialize(content *Content, v any) error {
	lenc := len(c)
	for i := range c {
		l := c[lenc-i-1]
		if err := l.Deserialize(content, v); err != nil {
			if errors.Is(err, errSerializerIncompatible) {
				continue
			}
			return err
		}
		return nil
	}
	return errSerializerIncompatible
}

var _ Serializer = serializerChain{}

type jsonSerializer struct{}

func (jsonSerializer) Deserialize(c *Content, v any) error {
	if !isMediaTypeJSON(c.Header["type"]) {
		return errSerializerIncompatible
	}
	return json.Unmarshal(c.Data, &v)
}

func (jsonSerializer) Serialize(v any) (*Content, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return &Content{
		Header: Header{
			"type": "application/json",
		},
		Data: data,
	}, nil
}

var _ Serializer = jsonSerializer{}

type nilSerializer struct{}

func (nilSerializer) Deserialize(c *Content, v any) error {
	if len(c.Data) > 0 {
		return errSerializerIncompatible
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer {
		return fmt.Errorf("cannot deserialize into non pointer: %v", v)
	}
	if rv.IsNil() {
		return fmt.Errorf("cannot deserialize into nil pointer: %v", v)
	}
	re := rv.Elem()
	if !re.CanSet() {
		return fmt.Errorf("non settable type: %v", v)
	}
	// Set the zero value for the given type.
	re.Set(reflect.Zero(re.Type()))

	return nil
}

func (nilSerializer) Serialize(v any) (*Content, error) {
	if v != nil {
		rv := reflect.ValueOf(v)
		if !(rv.Kind() == reflect.Pointer && rv.IsNil()) {
			return nil, errSerializerIncompatible
		}
	}
	return &Content{
		Header: Header{},
		Data:   nil,
	}, nil
}

var _ Serializer = nilSerializer{}

type byteSliceSerializer struct{}

func (byteSliceSerializer) Deserialize(c *Content, v any) error {
	if !isMediaTypeOctetStream(c.Header["type"]) {
		return errSerializerIncompatible
	}
	if bPtr, ok := v.(*[]byte); ok {
		if bPtr == nil {
			return fmt.Errorf("cannot deserialize into nil pointer: %v", v)
		}
		*bPtr = c.Data
		return nil
	}
	// v is *any
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer {
		return fmt.Errorf("cannot deserialize into non pointer: %v", v)
	}
	if rv.IsNil() {
		return fmt.Errorf("cannot deserialize into nil pointer: %v", v)
	}
	if rv.Elem().Type() != anyType {
		return fmt.Errorf("unsupported value type for content: %v", v)
	}
	rv.Elem().Set(reflect.ValueOf(c.Data))
	return nil
}

func (byteSliceSerializer) Serialize(v any) (*Content, error) {
	if b, ok := v.([]byte); ok {
		return &Content{
			Header: Header{
				"type": "application/octet-stream",
			},
			Data: b,
		}, nil
	}
	return nil, errSerializerIncompatible
}

var _ Serializer = byteSliceSerializer{}

type compositeSerializer struct {
	serializerChain
}

var defaultSerializer Serializer = compositeSerializer{
	serializerChain([]Serializer{nilSerializer{}, byteSliceSerializer{}, jsonSerializer{}}),
}
