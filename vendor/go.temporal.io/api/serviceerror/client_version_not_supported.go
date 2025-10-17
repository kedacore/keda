package serviceerror

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.temporal.io/api/errordetails/v1"
)

type (
	// ClientVersionNotSupported represents client version is not supported error.
	ClientVersionNotSupported struct {
		Message           string
		ClientVersion     string
		ClientName        string
		SupportedVersions string
		st                *status.Status
	}
)

// NewClientVersionNotSupported returns new ClientVersionNotSupported error.
func NewClientVersionNotSupported(clientVersion, clientName, supportedVersions string) error {
	return &ClientVersionNotSupported{
		Message:           fmt.Sprintf("Client version %s is not supported. Server supports %s versions: %s", clientVersion, clientName, supportedVersions),
		ClientVersion:     clientVersion,
		ClientName:        clientName,
		SupportedVersions: supportedVersions,
	}
}

// NewClientVersionNotSupportedf returns new ClientVersionNotSupported error with formatted message.
func NewClientVersionNotSupportedf(clientVersion, clientName, supportedVersions, format string, args ...any) error {
	return &ClientVersionNotSupported{
		Message:           fmt.Sprintf(format, args...),
		ClientVersion:     clientVersion,
		ClientName:        clientName,
		SupportedVersions: supportedVersions,
	}
}

// Error returns string message.
func (e *ClientVersionNotSupported) Error() string {
	return e.Message
}

func (e *ClientVersionNotSupported) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	st := status.New(codes.FailedPrecondition, e.Message)
	st, _ = st.WithDetails(
		&errordetails.ClientVersionNotSupportedFailure{
			ClientVersion:     e.ClientVersion,
			ClientName:        e.ClientName,
			SupportedVersions: e.SupportedVersions,
		},
	)
	return st
}

func newClientVersionNotSupported(st *status.Status, errDetails *errordetails.ClientVersionNotSupportedFailure) error {
	return &ClientVersionNotSupported{
		Message:           st.Message(),
		ClientVersion:     errDetails.GetClientVersion(),
		ClientName:        errDetails.GetClientName(),
		SupportedVersions: errDetails.GetSupportedVersions(),
		st:                st,
	}
}
