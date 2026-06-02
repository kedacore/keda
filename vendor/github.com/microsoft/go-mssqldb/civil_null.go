package mssql

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/golang-sql/civil"
)

type NullDate struct {
	Date  civil.Date
	Valid bool
}

func (n *NullDate) Scan(value interface{}) error {
	if value == nil {
		n.Date, n.Valid = civil.Date{}, false
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		n.Valid = true
		n.Date = civil.DateOf(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into NullDate", value)
	}
}

func (n NullDate) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Date, nil
}

func (n NullDate) String() string {
	if !n.Valid {
		return "NULL"
	}
	return n.Date.String()
}

func (n NullDate) MarshalText() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return n.Date.MarshalText()
}

func (n *NullDate) UnmarshalText(data []byte) error {
	if string(data) == "null" {
		n.Date, n.Valid = civil.Date{}, false
		return nil
	}
	n.Valid = true
	return n.Date.UnmarshalText(data)
}

type NullDateTime struct {
	DateTime civil.DateTime
	Valid    bool
}

func (n *NullDateTime) Scan(value interface{}) error {
	if value == nil {
		n.DateTime, n.Valid = civil.DateTime{}, false
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		n.Valid = true
		n.DateTime = civil.DateTimeOf(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into NullDateTime", value)
	}
}

func (n NullDateTime) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.DateTime, nil
}

func (n NullDateTime) String() string {
	if !n.Valid {
		return "NULL"
	}
	return n.DateTime.String()
}

func (n NullDateTime) MarshalText() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return n.DateTime.MarshalText()
}

func (n *NullDateTime) UnmarshalText(data []byte) error {
	if string(data) == "null" {
		n.DateTime, n.Valid = civil.DateTime{}, false
		return nil
	}
	n.Valid = true
	return n.DateTime.UnmarshalText(data)
}

type NullTime struct {
	Time  civil.Time
	Valid bool
}

func (n *NullTime) Scan(value interface{}) error {
	if value == nil {
		n.Time, n.Valid = civil.Time{}, false
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		n.Valid = true
		n.Time = civil.TimeOf(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into NullTime", value)
	}
}

func (n NullTime) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Time, nil
}

func (n NullTime) String() string {
	if !n.Valid {
		return "NULL"
	}
	return n.Time.String()
}

func (n NullTime) MarshalText() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return n.Time.MarshalText()
}

func (n *NullTime) UnmarshalText(data []byte) error {
	if string(data) == "null" {
		n.Time, n.Valid = civil.Time{}, false
		return nil
	}
	n.Valid = true
	return n.Time.UnmarshalText(data)
}
