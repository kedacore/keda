package serviceerror

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.temporal.io/api/errordetails/v1"
)

type (
	// NewerBuildExists is returned to a poll request from a build that has been superceded by
	// a newer build in versioning metadata.
	NewerBuildExists struct {
		Message        string
		DefaultBuildID string
		st             *status.Status
	}
)

// NewNewerBuildExists returns new NewerBuildExists error.
func NewNewerBuildExists(defaultBuildID string) error {
	return &NewerBuildExists{
		Message:        fmt.Sprintf("Task queue has a newer compatible build: %q", defaultBuildID),
		DefaultBuildID: defaultBuildID,
	}
}

// Error returns string message.
func (e *NewerBuildExists) Error() string {
	return e.Message
}

func (e *NewerBuildExists) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	st := status.New(codes.OutOfRange, e.Message)
	st, _ = st.WithDetails(
		&errordetails.NewerBuildExistsFailure{
			DefaultBuildId: e.DefaultBuildID,
		},
	)
	return st
}

func newNewerBuildExists(st *status.Status, errDetails *errordetails.NewerBuildExistsFailure) error {
	return &NewerBuildExists{
		Message:        st.Message(),
		DefaultBuildID: errDetails.GetDefaultBuildId(),
		st:             st,
	}
}
