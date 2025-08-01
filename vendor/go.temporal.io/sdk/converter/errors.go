package converter

import (
	"errors"
)

var (
	// ErrMetadataIsNotSet is returned when metadata is not set.
	ErrMetadataIsNotSet = errors.New("metadata is not set")
	// ErrEncodingIsNotSet is returned when payload encoding metadata is not set.
	ErrEncodingIsNotSet = errors.New("payload encoding metadata is not set")
	// ErrEncodingIsNotSupported is returned when payload encoding is not supported.
	ErrEncodingIsNotSupported = errors.New("payload encoding is not supported")
	// ErrUnableToEncode is returned when unable to encode.
	ErrUnableToEncode = errors.New("unable to encode")
	// ErrUnableToDecode is returned when unable to decode.
	ErrUnableToDecode = errors.New("unable to decode")
	// ErrUnableToSetValue is returned when unable to set value.
	ErrUnableToSetValue = errors.New("unable to set value")
	// ErrUnableToFindConverter is returned when unable to find converter.
	ErrUnableToFindConverter = errors.New("unable to find converter")
	// ErrTypeNotImplementProtoMessage is returned when value doesn't implement proto.Message.
	ErrTypeNotImplementProtoMessage = errors.New("type doesn't implement proto.Message")
	// ErrValuePtrIsNotPointer is returned when proto value is not a pointer.
	ErrValuePtrIsNotPointer = errors.New("not a pointer type")
	// ErrValuePtrMustConcreteType is returned when proto value is of interface type.
	ErrValuePtrMustConcreteType = errors.New("must be a concrete type, not interface")
	// ErrTypeIsNotByteSlice is returned when value is not of *[]byte type.
	ErrTypeIsNotByteSlice = errors.New("type is not *[]byte")
)
