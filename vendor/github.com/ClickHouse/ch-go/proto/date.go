package proto

import "time"

// Date represents Date value.
//
// https://clickhouse.com/docs/en/sql-reference/data-types/date/
type Date uint16

// DateLayout is default time format for Date.
const DateLayout = "2006-01-02"

// secInDay represents seconds in day.
//
// NB: works only on UTC, use time.Date, time.Time.AddDate.
const secInDay = 24 * 60 * 60

// Unix returns unix timestamp of Date.
func (d Date) Unix() int64 {
	return secInDay * int64(d)
}

// Time returns UTC starting time.Time of Date.
//
// You can use time.Unix(d.Unix(), 0) to get Time in time.Local location.
func (d Date) Time() time.Time {
	return time.Unix(d.Unix(), 0).UTC()
}

func (d Date) String() string {
	return d.Time().UTC().Format(DateLayout)
}

// ToDate returns Date of time.Time.
func ToDate(t time.Time) Date {
	if t.IsZero() {
		return 0
	}
	_, offset := t.Zone()
	return Date((t.Unix() + int64(offset)) / secInDay)
}

// NewDate returns the Date corresponding to year, month and day in UTC.
func NewDate(year int, month time.Month, day int) Date {
	return ToDate(time.Date(year, month, day, 0, 0, 0, 0, time.UTC))
}
