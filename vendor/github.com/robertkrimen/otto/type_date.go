package otto

import (
	"fmt"
	"math"
	"regexp"
	Time "time"
)

type dateObject struct {
	time  Time.Time
	value Value
	epoch int64
	isNaN bool
}

var invalidDateObject = dateObject{
	time:  Time.Time{},
	epoch: -1,
	value: NaNValue(),
	isNaN: true,
}

type ecmaTime struct {
	location    *Time.Location
	year        int
	month       int
	day         int
	hour        int
	minute      int
	second      int
	millisecond int
}

func newEcmaTime(goTime Time.Time) ecmaTime {
	return ecmaTime{
		year:        goTime.Year(),
		month:       dateFromGoMonth(goTime.Month()),
		day:         goTime.Day(),
		hour:        goTime.Hour(),
		minute:      goTime.Minute(),
		second:      goTime.Second(),
		millisecond: goTime.Nanosecond() / (100 * 100 * 100),
		location:    goTime.Location(),
	}
}

func (t *ecmaTime) goTime() Time.Time {
	return Time.Date(
		t.year,
		dateToGoMonth(t.month),
		t.day,
		t.hour,
		t.minute,
		t.second,
		t.millisecond*(100*100*100),
		t.location,
	)
}

func (d *dateObject) Time() Time.Time {
	return d.time
}

func (d *dateObject) Epoch() int64 {
	return d.epoch
}

func (d *dateObject) Value() Value {
	return d.value
}

// FIXME A date should only be in the range of -100,000,000 to +100,000,000 (1970): 15.9.1.1.
func (d *dateObject) SetNaN() {
	d.time = Time.Time{}
	d.epoch = -1
	d.value = NaNValue()
	d.isNaN = true
}

func (d *dateObject) SetTime(time Time.Time) {
	d.Set(timeToEpoch(time))
}

func (d *dateObject) Set(epoch float64) {
	// epoch
	d.epoch = epochToInteger(epoch)

	// time
	time, err := epochToTime(epoch)
	d.time = time // Is either a valid time, or the zero-value for time.Time

	// value & isNaN
	if err != nil {
		d.isNaN = true
		d.epoch = -1
		d.value = NaNValue()
	} else {
		d.value = int64Value(d.epoch)
	}
}

func epochToInteger(value float64) int64 {
	if value > 0 {
		return int64(math.Floor(value))
	}
	return int64(math.Ceil(value))
}

func epochToTime(value float64) (Time.Time, error) {
	epochWithMilli := value
	if math.IsNaN(epochWithMilli) || math.IsInf(epochWithMilli, 0) {
		return Time.Time{}, fmt.Errorf("invalid time %v", value)
	}

	epoch := int64(epochWithMilli / 1000)
	milli := int64(epochWithMilli) % 1000

	return Time.Unix(epoch, milli*1000000).In(utcTimeZone), nil
}

func timeToEpoch(time Time.Time) float64 {
	return float64(time.UnixMilli())
}

func (rt *runtime) newDateObject(epoch float64) *object {
	obj := rt.newObject()
	obj.class = classDateName

	// FIXME This is ugly...
	date := dateObject{}
	date.Set(epoch)
	obj.value = date
	return obj
}

func (o *object) dateValue() dateObject {
	value, _ := o.value.(dateObject)
	return value
}

func dateObjectOf(rt *runtime, date *object) dateObject {
	if date == nil {
		panic(rt.panicTypeError("Date.ObjectOf is nil"))
	}
	if date.class != classDateName {
		panic(rt.panicTypeError("Date.ObjectOf %q != %q", date.class, classDateName))
	}
	return date.dateValue()
}

// JavaScript is 0-based, Go is 1-based (15.9.1.4).
func dateToGoMonth(month int) Time.Month {
	return Time.Month(month + 1)
}

func dateFromGoMonth(month Time.Month) int {
	return int(month) - 1
}

func dateFromGoDay(day Time.Weekday) int {
	return int(day)
}

// newDateTime returns the epoch of date contained in argumentList for location.
func newDateTime(argumentList []Value, location *Time.Location) float64 {
	pick := func(index int, default_ float64) (float64, bool) {
		if index >= len(argumentList) {
			return default_, false
		}
		value := argumentList[index].float64()
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return 0, true
		}
		return value, false
	}

	switch len(argumentList) {
	case 0: // 0-argument
		time := Time.Now().In(utcTimeZone)
		return timeToEpoch(time)
	case 1: // 1-argument
		value := valueOfArrayIndex(argumentList, 0)
		value = toPrimitiveValue(value)
		if value.IsString() {
			return dateParse(value.string())
		}

		return value.float64()
	default: // 2-argument, 3-argument, ...
		var year, month, day, hour, minute, second, millisecond float64
		var invalid bool
		if year, invalid = pick(0, 1900.0); invalid {
			return math.NaN()
		}
		if month, invalid = pick(1, 0.0); invalid {
			return math.NaN()
		}
		if day, invalid = pick(2, 1.0); invalid {
			return math.NaN()
		}
		if hour, invalid = pick(3, 0.0); invalid {
			return math.NaN()
		}
		if minute, invalid = pick(4, 0.0); invalid {
			return math.NaN()
		}
		if second, invalid = pick(5, 0.0); invalid {
			return math.NaN()
		}
		if millisecond, invalid = pick(6, 0.0); invalid {
			return math.NaN()
		}

		if year >= 0 && year <= 99 {
			year += 1900
		}

		time := Time.Date(int(year), dateToGoMonth(int(month)), int(day), int(hour), int(minute), int(second), int(millisecond)*1000*1000, location)
		return timeToEpoch(time)
	}
}

var (
	dateLayoutList = []string{
		"2006",
		"2006-01",
		"2006-01-02",

		"2006T15:04",
		"2006-01T15:04",
		"2006-01-02T15:04",

		"2006T15:04:05",
		"2006-01T15:04:05",
		"2006-01-02T15:04:05",

		"2006/01",
		"2006/01/02",
		"2006/01/02 15:04:05",

		"2006T15:04:05.000",
		"2006-01T15:04:05.000",
		"2006-01-02T15:04:05.000",

		"2006T15:04-0700",
		"2006-01T15:04-0700",
		"2006-01-02T15:04-0700",

		"2006T15:04:05-0700",
		"2006-01T15:04:05-0700",
		"2006-01-02T15:04:05-0700",

		"2006T15:04:05.000-0700",
		"2006-01T15:04:05.000-0700",
		"2006-01-02T15:04:05.000-0700",

		Time.RFC1123,
	}
	matchDateTimeZone = regexp.MustCompile(`^(.*)(?:(Z)|([\+\-]\d{2}):(\d{2}))$`)
)

// dateParse returns the epoch of the parsed date.
func dateParse(date string) float64 {
	// YYYY-MM-DDTHH:mm:ss.sssZ
	var time Time.Time
	var err error

	if match := matchDateTimeZone.FindStringSubmatch(date); match != nil {
		if match[2] == "Z" {
			date = match[1] + "+0000"
		} else {
			date = match[1] + match[3] + match[4]
		}
	}

	for _, layout := range dateLayoutList {
		time, err = Time.Parse(layout, date)
		if err == nil {
			break
		}
	}

	if err != nil {
		return math.NaN()
	}

	return float64(time.UnixMilli())
}
