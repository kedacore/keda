package protocol

import (
	"errors"
	"strings"

	protocolpb "go.temporal.io/api/protocol/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var ErrProtoNameNotFound = errors.New("protocol name not found")

// NameFromMessage extracts the name of the protocol to which the supplied
// message belongs.
func NameFromMessage(msg *protocolpb.Message) (string, error) {
	bodyType := string(msg.GetBody().MessageName())
	if bodyType == "" {
		return "", ErrProtoNameNotFound
	}

	if lastDot := strings.LastIndex(bodyType, "."); lastDot > -1 {
		bodyType = bodyType[0:lastDot]
	}
	return bodyType, nil
}

// MustMarshalAny serializes a protobuf message into an Any or panics.
func MustMarshalAny(msg proto.Message) *anypb.Any {
	result, err := anypb.New(msg)
	if err != nil {
		panic(err)
	}
	return result
}
