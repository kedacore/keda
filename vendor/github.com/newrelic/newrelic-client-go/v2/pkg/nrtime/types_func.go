package nrtime

import (
	"github.com/newrelic/newrelic-client-go/v2/internal/serialization"
)

//
// Functions created on auto-generated types
//

// MarshalJSON wrapper for EpochSeconds
func (t EpochSeconds) MarshalJSON() ([]byte, error) {
	return serialization.EpochTime(t).MarshalJSON()
}

// UnmarshalJSON wrapper for EpochSeconds
func (t *EpochSeconds) UnmarshalJSON(s []byte) error {
	return (*serialization.EpochTime)(t).UnmarshalJSON(s)
}

// String returns the time formatted using the format string
func (t EpochSeconds) String() string {
	return serialization.EpochTime(t).String()
}

// MarshalJSON wrapper for EpochMilliseconds
func (t EpochMilliseconds) MarshalJSON() ([]byte, error) {
	return serialization.EpochTime(t).MarshalJSON()
}

// UnmarshalJSON wrapper for EpochMilliseconds
func (t *EpochMilliseconds) UnmarshalJSON(s []byte) error {
	return (*serialization.EpochTime)(t).UnmarshalJSON(s)
}

// String returns the time formatted using the format string
func (t EpochMilliseconds) String() string {
	return serialization.EpochTime(t).String()
}

func (t *Seconds) UnmarshalJSON(s []byte) error {
	// Handle null values
	if string(s) == "null" {
		return nil
	}

	*t = Seconds(string(s))

	return nil
}
