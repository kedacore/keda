package proto

import "time"

// Date32 represents Date32 value.
//
// https://clickhouse.com/docs/en/sql-reference/data-types/date32/
type Date32 int32

// Unix returns unix timestamp of Date32.
//
// You can use time.Unix(d.Unix(), 0) to get Time in time.Local location.
func (d Date32) Unix() int64 {
	return secInDay * int64(d)
}

// Time returns UTC starting time.Time of Date32.
func (d Date32) Time() time.Time {
	return time.Unix(d.Unix(), 0).UTC()
}

func (d Date32) String() string {
	return d.Time().Format(DateLayout)
}

// ToDate32 returns Date32 of time.Time.
func ToDate32(t time.Time) Date32 {
	if t.IsZero() {
		return 0
	}
	_, offset := t.Zone()
	return Date32((t.Unix() + int64(offset)) / secInDay)
}

// NewDate32 returns the Date32 corresponding to year, month and day in UTC.
func NewDate32(year int, month time.Month, day int) Date32 {
	return ToDate32(time.Date(year, month, day, 0, 0, 0, 0, time.UTC))
}
