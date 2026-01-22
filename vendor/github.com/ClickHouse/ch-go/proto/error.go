package proto

import "fmt"

// Error on server side.
type Error int

func (e Error) Error() string {
	if e.IsAError() {
		return fmt.Sprintf("%s (%d)", e.String(), e)
	}
	return fmt.Sprintf("UNKNOWN (%d)", e)
}

//go:generate go run github.com/dmarkham/enumer -transform snake_upper -type Error -trimprefix Err -output error_enum.go
