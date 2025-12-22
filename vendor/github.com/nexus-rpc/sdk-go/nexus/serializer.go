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
func (l *LazyValue) Consume(v any) (err error) {
	defer func() {
		closeErr := l.Reader.Close()
		err = errors.Join(err, closeErr)
	}()
	data, err := io.ReadAll(l.Reader)
	if err != nil {
		return err
	}
	err = l.serializer.Deserialize(&Content{
		Header: l.Reader.Header,
		Data:   data,
	}, v)
	return
}

// Serializer is used by the framework to serialize/deserialize input and output.
// To customize serialization logic, implement this interface and provide your implementation to framework methods such
// as [NewHTTPClient] and [NewHTTPHandler].
// By default, the SDK supports serialization of JSONables, byte slices, and nils.
//
// NOTE: Experimental
type Serializer interface {
	// Serialize encodes a value into a [Content].
	//
	// NOTE: Experimental
	Serialize(any) (*Content, error)
	// Deserialize decodes a [Content] into a given reference.
	//
	// NOTE: Experimental
	Deserialize(*Content, any) error
}

// FailureConverter is used by the framework to transform [error] instances to and from [Failure] instances.
// To customize conversion logic, implement this interface and provide your implementation to framework methods such as
// [NewClient] and [NewHTTPHandler].
// By default the SDK translates only error messages, losing type information and struct fields.
type FailureConverter interface {
	// ErrorToFailure converts an [error] to a [Failure].
	// Implementors should take a best-effort approach and never fail this method.
	// Note that the provided error may be nil.
	ErrorToFailure(error) Failure
	// ErrorToFailure converts a [Failure] to an [error].
	// Implementors should take a best-effort approach and never fail this method.
	FailureToError(Failure) error
}

var anyType = reflect.TypeOf((*any)(nil)).Elem()

// ErrSerializerIncompatible is a sentinel error emitted by [Serializer] implementations to signal that a serializer is
// incompatible with a given value or [Content].
var ErrSerializerIncompatible = errors.New("incompatible serializer")

// CompositeSerializer is a [Serializer] that composes multiple serializers together.
// During serialization, it tries each serializer in sequence until it finds a compatible serializer for the given value.
// During deserialization, it tries each serializer in reverse sequence until it finds a compatible serializer for the
// given content.
//
// NOTE: Experimental
type CompositeSerializer []Serializer

func (c CompositeSerializer) Serialize(v any) (*Content, error) {
	for _, l := range c {
		p, err := l.Serialize(v)
		if err != nil {
			if errors.Is(err, ErrSerializerIncompatible) {
				continue
			}
			return nil, err
		}
		return p, nil
	}
	return nil, ErrSerializerIncompatible
}

func (c CompositeSerializer) Deserialize(content *Content, v any) error {
	lenc := len(c)
	for i := range c {
		l := c[lenc-i-1]
		if err := l.Deserialize(content, v); err != nil {
			if errors.Is(err, ErrSerializerIncompatible) {
				continue
			}
			return err
		}
		return nil
	}
	return ErrSerializerIncompatible
}

var _ Serializer = CompositeSerializer{}

type jsonSerializer struct{}

func (jsonSerializer) Deserialize(c *Content, v any) error {
	if !isMediaTypeJSON(c.Header["type"]) {
		return ErrSerializerIncompatible
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

// NilSerializer is a [Serializer] that supports serialization of nil values.
//
// NOTE: Experimental
type NilSerializer struct{}

func (NilSerializer) Deserialize(c *Content, v any) error {
	if len(c.Data) > 0 {
		return ErrSerializerIncompatible
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

func (NilSerializer) Serialize(v any) (*Content, error) {
	if v != nil {
		rv := reflect.ValueOf(v)
		if rv.Kind() != reflect.Pointer || !rv.IsNil() {
			return nil, ErrSerializerIncompatible
		}
	}
	return &Content{
		Header: Header{},
		Data:   nil,
	}, nil
}

var _ Serializer = NilSerializer{}

type byteSliceSerializer struct{}

func (byteSliceSerializer) Deserialize(c *Content, v any) error {
	if !isMediaTypeOctetStream(c.Header["type"]) {
		return ErrSerializerIncompatible
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
	return nil, ErrSerializerIncompatible
}

var _ Serializer = byteSliceSerializer{}

var defaultSerializer Serializer = CompositeSerializer([]Serializer{NilSerializer{}, byteSliceSerializer{}, jsonSerializer{}})

// DefaultSerializer returns the SDK's default [Serializer] that handles serialization to and from JSONables, byte
// slices, and nil.
//
// NOTE: Experimental
func DefaultSerializer() Serializer {
	return defaultSerializer
}

type failureErrorFailureConverter struct{}

// ErrorToFailure implements FailureConverter.
func (e failureErrorFailureConverter) ErrorToFailure(err error) Failure {
	if err == nil {
		return Failure{}
	}
	if fe, ok := err.(*FailureError); ok {
		return fe.Failure
	}
	return Failure{
		Message: err.Error(),
	}
}

// FailureToError implements FailureConverter.
func (e failureErrorFailureConverter) FailureToError(f Failure) error {
	return &FailureError{f}
}

var defaultFailureConverter FailureConverter = failureErrorFailureConverter{}

// DefaultFailureConverter returns the SDK's default [FailureConverter] implementation. Arbitrary errors are converted
// to a simple [Failure] object with just the Message popluated and [FailureError] instances to their underlying
// [Failure] instance. [Failure] instances are converted to [FailureError] to allow access to the full failure metadata
// and details if available.
func DefaultFailureConverter() FailureConverter {
	return defaultFailureConverter
}
