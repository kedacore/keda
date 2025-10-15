package serviceerror

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.temporal.io/api/errordetails/v1"
)

type (
	// ServerVersionNotSupported represents client version is not supported error.
	ServerVersionNotSupported struct {
		Message                       string
		ServerVersion                 string
		ClientSupportedServerVersions string
		st                            *status.Status
	}
)

// NewServerVersionNotSupported returns new ServerVersionNotSupported error.
func NewServerVersionNotSupported(serverVersion, supportedVersions string) error {
	return &ServerVersionNotSupported{
		Message:                       fmt.Sprintf("Server version %s is not supported. Client supports server versions: %s", serverVersion, supportedVersions),
		ServerVersion:                 serverVersion,
		ClientSupportedServerVersions: supportedVersions,
	}
}

// Error returns string message.
func (e *ServerVersionNotSupported) Error() string {
	return e.Message
}

func (e *ServerVersionNotSupported) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	st := status.New(codes.FailedPrecondition, e.Message)
	st, _ = st.WithDetails(
		&errordetails.ServerVersionNotSupportedFailure{
			ServerVersion:                 e.ServerVersion,
			ClientSupportedServerVersions: e.ClientSupportedServerVersions,
		},
	)
	return st
}

func newServerVersionNotSupported(st *status.Status, errDetails *errordetails.ServerVersionNotSupportedFailure) error {
	return &ServerVersionNotSupported{
		Message:                       st.Message(),
		ServerVersion:                 errDetails.GetServerVersion(),
		ClientSupportedServerVersions: errDetails.GetClientSupportedServerVersions(),
		st:                            st,
	}
}
