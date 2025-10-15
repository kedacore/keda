package serializer

import (
	"encoding/json"
	"fmt"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/serviceerror"
	"google.golang.org/protobuf/proto"
)

type (

	// SerializationError is an error type for serialization
	SerializationError struct {
		msg string
	}

	// DeserializationError is an error type for deserialization
	DeserializationError struct {
		msg string
	}

	// UnknownEncodingTypeError is an error type for unknown or unsupported encoding type
	UnknownEncodingTypeError struct {
		encodingType enumspb.EncodingType
	}

	// Marshaler is implemented by objects that can marshal themselves
	Marshaler interface {
		Marshal() ([]byte, error)
	}
)

// SerializeBatchEvents serializes batch events into a datablob proto
func SerializeBatchEvents(events []*historypb.HistoryEvent, encodingType enumspb.EncodingType) (*commonpb.DataBlob, error) {
	return serialize(&historypb.History{Events: events}, encodingType)
}

func serializeProto(p Marshaler, encodingType enumspb.EncodingType) (*commonpb.DataBlob, error) {
	if p == nil {
		return nil, nil
	}

	var data []byte
	var err error

	switch encodingType {
	case enumspb.ENCODING_TYPE_PROTO3:
		data, err = p.Marshal()
	case enumspb.ENCODING_TYPE_JSON:
		encodingType = enumspb.ENCODING_TYPE_JSON
		pb, ok := p.(proto.Message)
		if !ok {
			return nil, NewSerializationError("could not cast protomarshal interface to proto.message")
		}
		data, err = NewJSONPBEncoder().Encode(pb)
	default:
		return nil, NewUnknownEncodingTypeError(encodingType)
	}

	if err != nil {
		return nil, NewSerializationError(err.Error())
	}

	// Shouldn't happen, but keeping
	if data == nil {
		return nil, nil
	}

	return NewDataBlob(data, encodingType), nil
}

// DeserializeBatchEvents deserializes batch events from a datablob proto
func DeserializeBatchEvents(data *commonpb.DataBlob) ([]*historypb.HistoryEvent, error) {
	if data == nil {
		return nil, nil
	}
	if len(data.Data) == 0 {
		return nil, nil
	}

	events := &historypb.History{}
	var err error
	switch data.EncodingType {
	case enumspb.ENCODING_TYPE_JSON:
		err = NewJSONPBEncoder().Decode(data.Data, events)
	case enumspb.ENCODING_TYPE_PROTO3:
		err = proto.Unmarshal(data.Data, events)
	default:
		return nil, NewDeserializationError("DeserializeBatchEvents invalid encoding")
	}
	if err != nil {
		return nil, err
	}
	return events.Events, nil
}

func serialize(input interface{}, encodingType enumspb.EncodingType) (*commonpb.DataBlob, error) {
	if input == nil {
		return nil, nil
	}

	if p, ok := input.(Marshaler); ok {
		return serializeProto(p, encodingType)
	}

	var data []byte
	var err error

	switch encodingType {
	case enumspb.ENCODING_TYPE_JSON: // For backward-compatibility
		data, err = json.Marshal(input)
	default:
		return nil, NewUnknownEncodingTypeError(encodingType)
	}

	if err != nil {
		return nil, NewSerializationError(err.Error())
	}

	return NewDataBlob(data, encodingType), nil
}

// NewUnknownEncodingTypeError returns a new instance of encoding type error
func NewUnknownEncodingTypeError(encodingType enumspb.EncodingType) error {
	return &UnknownEncodingTypeError{encodingType: encodingType}
}

func (e *UnknownEncodingTypeError) Error() string {
	return fmt.Sprintf("unknown or unsupported encoding type %v", e.encodingType)
}

// NewSerializationError returns a SerializationError
func NewSerializationError(msg string) error {
	return &SerializationError{msg: msg}
}

func (e *SerializationError) Error() string {
	return fmt.Sprintf("serialization error: %v", e.msg)
}

// NewDeserializationError returns a DeserializationError
func NewDeserializationError(msg string) error {
	return &DeserializationError{msg: msg}
}

func (e *DeserializationError) Error() string {
	return fmt.Sprintf("deserialization error: %v", e.msg)
}

// NewDataBlob creates new blob data
func NewDataBlob(data []byte, encodingType enumspb.EncodingType) *commonpb.DataBlob {
	if len(data) == 0 {
		return nil
	}

	return &commonpb.DataBlob{
		Data:         data,
		EncodingType: encodingType,
	}
}

// DeserializeBlobDataToHistoryEvents deserialize the blob data to history event data
func DeserializeBlobDataToHistoryEvents(
	dataBlobs []*commonpb.DataBlob, filterType enumspb.HistoryEventFilterType,
) (*historypb.History, error) {

	var historyEvents []*historypb.HistoryEvent

	for _, batch := range dataBlobs {
		events, err := DeserializeBatchEvents(batch)
		if err != nil {
			return nil, err
		}
		if len(events) == 0 {
			return nil, &serviceerror.Internal{
				Message: "corrupted history event batch, empty events",
			}
		}

		historyEvents = append(historyEvents, events...)
	}

	if filterType == enumspb.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT {
		historyEvents = []*historypb.HistoryEvent{historyEvents[len(historyEvents)-1]}
	}
	return &historypb.History{Events: historyEvents}, nil
}
