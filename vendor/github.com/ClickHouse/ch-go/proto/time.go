package proto

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Time32 represents duration in seconds.
type Time32 int32

// Time64 represents duration up until nanoseconds.
type Time64 int64

// String implements formatting Time32 to string of form Hour:Minutes:Seconds.
func (t Time32) String() string {
	d := time.Duration(t) * time.Second

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

// String implements formatting Time64 to string of form Hour:Minutes:Seconds.NanoSeconds.
func (t Time64) String() string {
	d := time.Duration(t)

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	secs := int(d.Seconds()) % 60
	nanos := d.Nanoseconds() % 1e9

	// NOTE(kavi): do we need multiple formatting depending on precision (3, 6, 9) instead of
	// always 9?
	return fmt.Sprintf("%02d:%02d:%02d.%09d", hours, minutes, secs, nanos)
}

// ParseTime32 parses string of form "12:34:56" to valid Time32 type.
func ParseTime32(s string) (Time32, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid time format: %s", s)
	}

	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}

	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}

	seconds, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, err
	}

	totalSeconds := int32(hours*3600 + minutes*60 + seconds)
	return Time32(totalSeconds), nil
}

// ParseTime64 parses string of form "12:34:56.789" to valid Time64 type.
func ParseTime64(s string) (Time64, error) {
	timePart, fractionalStr, ok := strings.Cut(s, ".")
	if !ok {
		fractionalStr = ""
	}

	parts := strings.Split(timePart, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid time format: %s", s)
	}

	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}

	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}

	seconds, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, err
	}

	// Calculate total seconds since midnight
	totalSeconds := int64(hours*3600 + minutes*60 + seconds)

	// Parse fractional part (default to nanoseconds scale)
	var fractional int64
	if fractionalStr != "" {
		// Pad or truncate to 9 digits (nanoseconds)
		for len(fractionalStr) < 9 {
			fractionalStr += "0"
		}
		if len(fractionalStr) > 9 {
			fractionalStr = fractionalStr[:9]
		}
		fractional, err = strconv.ParseInt(fractionalStr, 10, 64)
		if err != nil {
			return 0, err
		}
	}

	// Store as decimal with nanosecond scale
	return Time64(totalSeconds*1e9 + fractional), nil
}

// IntoTime32 converts time.Druation into Time32 up to seconds precision.
func IntoTime32(t time.Duration) Time32 {
	return Time32(int(t.Seconds()))
}

// IntoTime64 converts time.Duration to Time64 up to nanoseconds precision
func IntoTime64(t time.Duration) Time64 {
	return IntoTime64WithPrecision(t, PrecisionMax)
}

// IntoTime64WithPrecision converts time.Duration to Time64 with specified precision
func IntoTime64WithPrecision(d time.Duration, precision Precision) Time64 {
	res := truncateDuration(d, precision)
	// When the column type is say Time64(6) we store Time64 as microseconds. Basically scale to
	// it's precission
	return Time64(res.Nanoseconds() / precision.Scale())
}

// Duration converts Time32 into time.Duration up to seconds precision
func (t Time32) Duration() time.Duration {
	seconds := int32(t)
	return time.Second * time.Duration(seconds)
}

// Duration converts Time64 into time.Duration up to nanoseconds precision
func (t Time64) Duration() time.Duration {
	return t.ToDurationWithPrecision(PrecisionMax)
}

// ToDurationWithPrecision converts Time64 to time.Duration with specified precision
// up until PrecisionMax (nanoseconds)
func (t Time64) ToDurationWithPrecision(precision Precision) time.Duration {
	// `t` is stored with precision `precision`. Scale it to nanoseconds before converting it into
	// time.Duration. Because time.Duration is always nanoseconds.
	res := time.Duration(int64(t) * precision.Scale())
	return truncateDuration(res, precision)
}

func truncateDuration(d time.Duration, precision Precision) time.Duration {
	var res time.Duration
	switch precision {
	case PrecisionSecond:
		res = d.Truncate(time.Second)
	case PrecisionMilli:
		res = d.Truncate(time.Millisecond)
	case PrecisionMicro:
		res = d.Truncate(time.Microsecond)
	// NOTE: NO additional case needed for PrecisionMax, given it's type alias for PrecisionNano
	case PrecisionNano:
		res = d
	default:
		// if wrong precision, treat it as Millisecond.
		res = d.Truncate(time.Millisecond)
	}

	return res
}
