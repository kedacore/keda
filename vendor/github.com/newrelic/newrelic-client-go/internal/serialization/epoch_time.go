package serialization

import (
	"fmt"
	"strconv"
	"time"
)

// EpochTime is a type used for unmarshaling timestamps represented in epoch time.
// Its underlying type is time.Time.
type EpochTime time.Time

// MarshalJSON is responsible for marshaling the EpochTime type.
func (e EpochTime) MarshalJSON() ([]byte, error) {
	ret := strconv.FormatInt(time.Time(e).UTC().Unix(), 10)
	milli := int64(time.Time(e).Nanosecond()) / int64(time.Millisecond)

	// Include milliseconds if there are some
	if milli > 0 {
		ret += fmt.Sprintf("%03d", milli)
	}

	return []byte(ret), nil
}

// UnmarshalJSON is responsible for unmarshaling the EpochTime type.
func (e *EpochTime) UnmarshalJSON(s []byte) error {
	var (
		err   error
		sec   int64
		milli int64
		nano  int64
	)

	// detect type of timestamp based on length
	switch l := len(s); {
	case l <= 10: // seconds
		sec, err = strconv.ParseInt(string(s), 10, 64)
	case l > 10 && l <= 16: // milliseconds
		milli, err = strconv.ParseInt(string(s[0:13]), 10, 64)
		if err != nil {
			return err
		}
		nano = milli * int64(time.Millisecond)
	case l > 16: // nanoseconds
		sec, err = strconv.ParseInt(string(s[0:10]), 10, 64)
		if err != nil {
			return err
		}
		nano, err = strconv.ParseInt(string(s[10:16]), 10, 64)
	default:
		return fmt.Errorf("unable to parse EpochTime: '%s'", s)
	}

	if err != nil {
		return err
	}

	// Convert and self store
	*(*time.Time)(e) = time.Unix(sec, nano).UTC()

	return nil
}

// Equal provides a comparator for the EpochTime type.
func (e EpochTime) Equal(u EpochTime) bool {
	return time.Time(e).Equal(time.Time(u))
}

// String returns the time formatted using the format string
func (e EpochTime) String() string {
	return time.Time(e).String()
}

// Unix returns the time formatted as seconds since Jan 1st, 1970
func (e EpochTime) Unix() int64 {
	return time.Time(e).Unix()
}
