package utils

import (
	"strconv"
	"strings"
	"time"

	"github.com/newrelic/newrelic-client-go/v2/pkg/nrtime"
)

// IntArrayToString converts an array of integers
// to a comma-separated list string -
// e.g [1, 2, 3] will be converted to "1,2,3".
func IntArrayToString(integers []int) string {
	sArray := make([]string, len(integers))

	for i, n := range integers {
		sArray[i] = strconv.Itoa(n)
	}

	return strings.Join(sArray, ",")
}

// a helper function that is used to populate a timestamp with non-zero milliseconds
// if the millisecond count has been found zero, usually when generated with time.Time()
// in Golang which does not have a nanoseconds field; which helps mutations such as those
// in changetracking, the API of which requires a timestamp with non-zero milliseconds.
func GetSafeTimestampWithMilliseconds(inputTimestamp nrtime.EpochMilliseconds) nrtime.EpochMilliseconds {
	timestamp := time.Time(inputTimestamp)

	// since time.Time in Go does not have a milliseconds field, which is why the implementation
	// of unmarshaling time.Time into a Unix timestamp in the serialization package relies on
	// nanoseconds to produce a value of milliseconds, we try employing a similar logic below

	if timestamp.Nanosecond() < 100000000 {
		timestamp = timestamp.Add(time.Nanosecond * 100000000)
	}

	return nrtime.EpochMilliseconds(timestamp)
}
