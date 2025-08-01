package serviceerror

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	// DataLoss represents data loss error.
	DataLoss struct {
		Message string
		st      *status.Status
	}
)

// NewDataLoss returns new DataLoss error.
func NewDataLoss(message string) error {
	return &DataLoss{
		Message: message,
	}
}

// Error returns string message.
func (e *DataLoss) Error() string {
	return e.Message
}

func (e *DataLoss) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	return status.New(codes.DataLoss, e.Message)
}

func newDataLoss(st *status.Status) error {
	return &DataLoss{
		Message: st.Message(),
		st:      st,
	}
}
