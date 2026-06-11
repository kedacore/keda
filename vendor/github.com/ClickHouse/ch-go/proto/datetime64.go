package proto

import (
	"time"
)

// Precision of DateTime64 and Time64.
//
// Tick size (precision): 10^(-precision) seconds.
// Valid range: [0:9].
type Precision byte

// Duration returns duration of single tick for precision.
func (p Precision) Duration() time.Duration {
	return time.Nanosecond * time.Duration(p.Scale())
}

// Valid reports whether precision is valid.
func (p Precision) Valid() bool {
	return p <= PrecisionMax
}

func (p Precision) Scale() int64 {
	d := int64(1)
	for i := PrecisionNano; i > p; i-- {
		d *= 10
	}
	return d
}

const (
	// PrecisionSecond is one second precision.
	PrecisionSecond Precision = 0
	// PrecisionMilli is millisecond precision.
	PrecisionMilli Precision = 3
	// PrecisionMicro is microsecond precision.
	PrecisionMicro Precision = 6
	// PrecisionNano is nanosecond precision.
	PrecisionNano Precision = 9

	// PrecisionMax is maximum precision (nanosecond).
	PrecisionMax = PrecisionNano
)

// DateTime64 represents DateTime64 type.
//
// See https://clickhouse.com/docs/en/sql-reference/data-types/datetime64/.
type DateTime64 int64

// ToDateTime64 converts time.Time to DateTime64.
func ToDateTime64(t time.Time, p Precision) DateTime64 {
	if t.IsZero() {
		return 0
	}
	return DateTime64(t.UnixNano() / p.Scale())
}

// Time returns DateTime64 as time.Time.
func (d DateTime64) Time(p Precision) time.Time {
	nsec := int64(d) * p.Scale()
	return time.Unix(nsec/1e9, nsec%1e9)
}
