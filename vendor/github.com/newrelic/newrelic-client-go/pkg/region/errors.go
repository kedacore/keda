package region

import (
	"fmt"
)

// InvalidError returns when the Region is not valid
type InvalidError struct {
	Message string
}

// Error string reported when an InvalidError happens
func (e InvalidError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("invalid region: %s", e.Message)
	}

	return "invalid region"
}

// ErrorNil returns an InvalidError message saying the value was nil
func ErrorNil() InvalidError {
	return InvalidError{
		Message: "value is nil",
	}
}

type UnknownError struct {
	Message string
}

func (e UnknownError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("unknown region: %s", e.Message)
	}

	return "unknown region"
}

// UnknownUsingDefaultError returns when the Region requested is not valid, but we want to give them something
type UnknownUsingDefaultError struct {
	Message string
}

// Error string reported when an InvalidError happens
func (e UnknownUsingDefaultError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("unknown region: %s, using default: %s", e.Message, Default.String())
	}

	return fmt.Sprintf("unknown region, using default: %s", Default.String())
}
