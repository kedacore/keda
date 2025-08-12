package converter

import failurepb "go.temporal.io/api/failure/v1"

// FailureConverter is used by the sdk to serialize/deserialize errors
// that need to be sent over the wire.
// To use a custom FailureConverter, set FailureConverter in client, through client.Options.
type FailureConverter interface {
	// ErrorToFailure converts a error to a Failure proto message.
	ErrorToFailure(err error) *failurepb.Failure

	// FailureToError converts a Failure proto message to a Go Error.
	FailureToError(failure *failurepb.Failure) error
}

type encodedFailure struct {
	Message    string `json:"message"`
	StackTrace string `json:"stack_trace"`
}

// EncodeCommonFailureAttributes packs failure attributes to a payload so that they flow through a dataconverter.
func EncodeCommonFailureAttributes(dc DataConverter, failure *failurepb.Failure) error {
	var err error

	failure.EncodedAttributes, err = dc.ToPayload(encodedFailure{
		Message:    failure.Message,
		StackTrace: failure.StackTrace,
	})
	if err != nil {
		return err
	}
	failure.Message = "Encoded failure"
	failure.StackTrace = ""

	return nil
}

// DecodeCommonFailureAttributes unpacks failure attributes from a stored payload, if present.
func DecodeCommonFailureAttributes(dc DataConverter, failure *failurepb.Failure) {
	var ea encodedFailure
	if failure.GetEncodedAttributes() != nil && dc.FromPayload(failure.GetEncodedAttributes(), &ea) == nil {
		failure.Message = ea.Message
		failure.StackTrace = ea.StackTrace
	}
}
