// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// straight from 'time.go'
const MaxTimeDuration = time.Duration(1<<63 - 1)

// DurationTo8601Seconds takes a duration and returns a string period of whole seconds (int cast of float)
func DurationTo8601Seconds(duration time.Duration) string {
	const dotnetTimeSpanMax = "P10675199DT2H48M5.4775807S"
	// we'll do this mapping just to make things simpler for interop - if other users were expecting the
	// .net max value then we'll give it to them.
	if duration == MaxTimeDuration {
		return dotnetTimeSpanMax
	}

	// if we shove all of this data into 'seconds' the service doesn't appear to be able to handle it.
	minutes := duration / time.Minute
	seconds := (duration % time.Minute) / time.Second

	return fmt.Sprintf("PT%dM%dS", minutes, seconds)
}

// DurationToStringPtr converts a time.Duration to an ISO8601 duration as a string pointer.
func DurationToStringPtr(duration *time.Duration) *string {
	if duration != nil && *duration > 0 {
		val := DurationTo8601Seconds(*duration)
		return &val
	}

	return nil
}

var iso8601Regex = regexp.MustCompile(
	`P` +
		`(?:(?P<years>[\d.,]+)Y)?` +
		`(?:(?P<months>[\d.,]+)M)?` +
		`(?:(?P<weeks>[\d.,]+)W)?` +
		`(?:(?P<days>[\d.,]+)D)?` +
		`(?:T` +
		`(?:(?P<hours>[\d.,]+)H)?` +
		`(?:(?P<minutes>[\d.,]+)M)?` +
		`(?:(?P<seconds>[\d.,]+)S)?` +
		`)?`,
)

// ISO8601StringToDuration converts an ISO8601 string to a Go time.Duration
func ISO8601StringToDuration(durationStr *string) (*time.Duration, error) {
	if durationStr == nil {
		return nil, nil
	}

	matches := iso8601Regex.FindAllStringSubmatch(*durationStr, -1)

	if matches == nil || len(matches) != 1 {
		return nil, fmt.Errorf("duration (%s) didn't match the regexp", *durationStr)
	}

	var sum float64
	names := iso8601Regex.SubexpNames()

	for i := 0; i < len(matches[0]); i++ {
		if names[i] == "" {
			continue
		}

		name := names[i]
		value := matches[0][i]

		if value == "" {
			continue
		}

		nanoSeconds, err := strconv.ParseFloat(value, 64)

		if err != nil {
			return nil, err
		}

		switch name {
		case "years":
			nanoSeconds *= float64(365 * 24 * time.Hour)
		case "months":
			nanoSeconds *= float64(30 * 24 * time.Hour)
		case "weeks":
			nanoSeconds *= float64(7 * 24 * time.Hour)
		case "days":
			nanoSeconds *= float64(24 * time.Hour)
		case "hours":
			nanoSeconds *= float64(time.Hour)
		case "minutes":
			nanoSeconds *= float64(time.Minute)
		case "seconds":
			nanoSeconds *= float64(time.Second)
		}

		sum += nanoSeconds
	}

	// if they exceed our native time.Duration type (can happen since .NET's TimeSpan.Max
	// can exceed 292 years) we'll just map it to our 'max'.
	if sum >= float64(MaxTimeDuration) {
		duration := time.Duration(MaxTimeDuration)
		return &duration, nil
	}

	duration, err := time.ParseDuration(fmt.Sprintf("%fns", sum))

	if err != nil {
		return nil, err
	}

	return &duration, nil
}

func Int32ToPtr(val *int32) *int32 {
	if val != nil && *val > 0 {
		return val
	}

	return nil
}
