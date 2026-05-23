package value

import (
	"fmt"
	"github.com/Azure/azure-kusto-go/azkustodata/types"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const tick = 100 * time.Nanosecond
const day = 24 * time.Hour

// Timespan represents a Kusto timespan type.  Timespan implements Kusto.
type Timespan struct {
	pointerValue[time.Duration]
}

func NewTimespan(v time.Duration) *Timespan {
	return &Timespan{newPointerValue[time.Duration](&v)}
}

func NewNullTimespan() *Timespan {
	return &Timespan{newPointerValue[time.Duration](nil)}
}

func TimespanFromString(s string) (*Timespan, error) {
	t := &Timespan{}
	err := t.Unmarshal(s)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// Marshal marshals the Timespan into a Kusto compatible string. The string is the constant invariant (c)
// format. See https://learn.microsoft.com/en-us/dotnet/standard/base-types/standard-timespan-format-strings#the-constant-c-format-specifier .
func (t *Timespan) Marshal() string {
	if t == nil || t.value == nil || *t.value/tick == 0 {
		return "00:00:00"
	}

	// val is used to track the duration value as we move our parts of our time into our string format.
	// For example, after we write to our string the number of days that value had, we remove those days
	// from the duration. We continue doing this until val only holds values < 10 millionth of a second (tick)
	// as that is the lowest precision in our string representation.
	val := *t.value

	var sb strings.Builder

	// Add a - sign if we have a negative value. Convert our value to positive for easier processing.
	if val < 0 {
		sb.WriteString("-")
		val = -val
	}

	// Only include the day if the duration is 1+ days.
	days := val / day
	if days > 0 {
		sb.WriteString(fmt.Sprintf("%d.", days))
	}

	hours := (val % day) / time.Hour
	minutes := (val % time.Hour) / time.Minute
	seconds := (val % time.Minute) / time.Second
	ticks := (val % time.Second) / tick

	sb.WriteString(fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds))

	if ticks > 0 {
		sb.WriteString(fmt.Sprintf(".%07d", ticks))
	}

	return sb.String()
}

// Unmarshal unmarshals i into Timespan. i must be a string representing a Values timespan or nil.
func (t *Timespan) Unmarshal(i interface{}) error {
	const (
		hoursIndex   = 0
		minutesIndex = 1
		secondsIndex = 2
	)

	if i == nil {
		t.value = nil
		return nil
	}

	v, ok := i.(string)
	if !ok {
		return convertError(t, i)
	}

	negative := false
	if len(v) > 1 {
		if string(v[0]) == "-" {
			negative = true
			v = v[1:]
		}
	}

	sp := strings.Split(v, ":")
	if len(sp) != 3 {
		return parseError(v, sp, fmt.Errorf("value to unmarshal into Timespan does not seem to fit format '00:00:00', where values are decimal(%s)", v))
	}

	var sum time.Duration

	d, err := t.unmarshalDaysHours(sp[hoursIndex])
	if err != nil {
		return parseError(v, sp, err)
	}
	sum += d

	d, err = t.unmarshalMinutes(sp[minutesIndex])
	if err != nil {
		return parseError(v, sp, err)
	}
	sum += d

	d, err = t.unmarshalSeconds(sp[secondsIndex])
	if err != nil {
		return parseError(v, sp, err)
	}

	sum += d

	if negative {
		sum = sum * time.Duration(-1)
	}

	t.value = &sum
	return nil
}

func (t *Timespan) unmarshalDaysHours(s string) (time.Duration, error) {
	sp := strings.Split(s, ".")
	switch len(sp) {
	case 1:
		hours, err := strconv.Atoi(s)
		if err != nil {
			return 0, fmt.Errorf("timespan's hours/day field was incorrect, was %s: %s", s, err)
		}
		return time.Duration(hours) * time.Hour, nil
	case 2:
		days, err := strconv.Atoi(sp[0])
		if err != nil {
			return 0, fmt.Errorf("timespan's hours/day field was incorrect, was %s", s)
		}
		hours, err := strconv.Atoi(sp[1])
		if err != nil {
			return 0, fmt.Errorf("timespan's hours/day field was incorrect, was %s", s)
		}
		return time.Duration(days)*day + time.Duration(hours)*time.Hour, nil
	}
	return 0, fmt.Errorf("timespan's hours/days field did not have the requisite '.'s, was %s", s)
}

func (t *Timespan) unmarshalMinutes(s string) (time.Duration, error) {
	s = strings.Split(s, ".")[0] // We can have 01 or 01.00 or 59, but nothing comes behind the .

	minutes, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("timespan's minutes field was incorrect, was %s", s)
	}
	if minutes < 0 || minutes > 59 {
		return 0, fmt.Errorf("timespan's minutes field was incorrect, was %s", s)
	}
	return time.Duration(minutes) * time.Minute, nil
}

// unmarshalSeconds deals with this crazy output format. Instead of having some multiplier, the number
// of precision characters behind the decimal indicates your multiplier. This can be between 0 and 7, but
// really only has 3, 4 and 7. There is something called a tick, which is 100 Nanoseconds and the precision
// at len 4 is 100 * Microsecond (don't know if that has a name).
func (t *Timespan) unmarshalSeconds(s string) (time.Duration, error) {
	// "03" = 3 * time.Second
	// "00.099" = 99 * time.Millisecond
	// "03.0123" == 3 * time.Second + 12300 * time.Microsecond
	sp := strings.Split(s, ".")
	switch len(sp) {
	case 1:
		seconds, err := strconv.Atoi(s)
		if err != nil {
			return 0, fmt.Errorf("timespan's seconds field was incorrect, was %s", s)
		}
		return time.Duration(seconds) * time.Second, nil
	case 2:
		seconds, err := strconv.Atoi(sp[0])
		if err != nil {
			return 0, fmt.Errorf("timespan's seconds field was incorrect, was %s", s)
		}
		n, err := strconv.Atoi(sp[1])
		if err != nil {
			return 0, fmt.Errorf("timespan's seconds field was incorrect, was %s", s)
		}
		var prec time.Duration
		switch len(sp[1]) {
		case 1:
			prec = time.Duration(n) * (100 * time.Millisecond)
		case 2:
			prec = time.Duration(n) * (10 * time.Millisecond)
		case 3:
			prec = time.Duration(n) * time.Millisecond
		case 4:
			prec = time.Duration(n) * 100 * time.Microsecond
		case 5:
			prec = time.Duration(n) * 10 * time.Microsecond
		case 6:
			prec = time.Duration(n) * time.Microsecond
		case 7:
			prec = time.Duration(n) * tick
		case 8:
			prec = time.Duration(n) * (10 * time.Nanosecond)
		case 9:
			prec = time.Duration(n) * time.Nanosecond
		default:
			return 0, fmt.Errorf("timespan's seconds field did not have 1-9 numbers after the decimal, had %v", s)
		}

		return time.Duration(seconds)*time.Second + prec, nil
	}
	return 0, fmt.Errorf("timespan's seconds field did not have the requisite '.'s, was %s", s)
}

// Convert Timespan into reflect value.
func (t *Timespan) Convert(v reflect.Value) error {
	pt := v.Type()
	switch {
	case pt.AssignableTo(reflect.TypeOf(time.Duration(0))):
		if t.value != nil {
			v.Set(reflect.ValueOf(*t.value))
		}
		return nil
	case pt.ConvertibleTo(reflect.TypeOf(new(time.Duration))):
		if t.value != nil {
			pt := t.value
			v.Set(reflect.ValueOf(pt))
		}
		return nil
	case pt.ConvertibleTo(reflect.TypeOf(Timespan{})):
		v.Set(reflect.ValueOf(*t))
		return nil
	case pt.ConvertibleTo(reflect.TypeOf(&Timespan{})):
		v.Set(reflect.ValueOf(t))
		return nil
	}
	return convertError(t, v)
}

func TimespanString(d time.Duration) string {
	return NewTimespan(d).Marshal()
}

// GetType returns the type of the value.
func (t *Timespan) GetType() types.Column {
	return types.Timespan
}
