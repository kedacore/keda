package serviceerror

import "google.golang.org/grpc/status"

type (
	ServiceError interface {
		error
		Status() *status.Status
	}
)
