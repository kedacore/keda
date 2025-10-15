package otto

import (
	"math"
	"time"
)

// Date

const (
	// TODO Be like V8?
	// builtinDateDateTimeLayout = "Mon Jan 2 2006 15:04:05 GMT-0700 (MST)".
	builtinDateDateTimeLayout = time.RFC1123 // "Mon, 02 Jan 2006 15:04:05 MST"
	builtinDateDateLayout     = "Mon, 02 Jan 2006"
	builtinDateTimeLayout     = "15:04:05 MST"
)

// utcTimeZone is the time zone used for UTC calculations.
// It is GMT not UTC as that's what Javascript does because toUTCString is
// actually an alias to toGMTString.
var utcTimeZone = time.FixedZone("GMT", 0)

func builtinDate(call FunctionCall) Value {
	date := &dateObject{}
	date.Set(newDateTime([]Value{}, time.Local)) //nolint:gosmopolitan
	return stringValue(date.Time().Format(builtinDateDateTimeLayout))
}

func builtinNewDate(obj *object, argumentList []Value) Value {
	return objectValue(obj.runtime.newDate(newDateTime(argumentList, time.Local))) //nolint:gosmopolitan
}

func builtinDateToString(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return stringValue("Invalid Date")
	}
	return stringValue(date.Time().Local().Format(builtinDateDateTimeLayout)) //nolint:gosmopolitan
}

func builtinDateToDateString(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return stringValue("Invalid Date")
	}
	return stringValue(date.Time().Local().Format(builtinDateDateLayout)) //nolint:gosmopolitan
}

func builtinDateToTimeString(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return stringValue("Invalid Date")
	}
	return stringValue(date.Time().Local().Format(builtinDateTimeLayout)) //nolint:gosmopolitan
}

func builtinDateToUTCString(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return stringValue("Invalid Date")
	}
	return stringValue(date.Time().In(utcTimeZone).Format(builtinDateDateTimeLayout))
}

func builtinDateToISOString(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return stringValue("Invalid Date")
	}
	return stringValue(date.Time().Format("2006-01-02T15:04:05.000Z"))
}

func builtinDateToJSON(call FunctionCall) Value {
	obj := call.thisObject()
	value := obj.DefaultValue(defaultValueHintNumber) // FIXME object.primitiveNumberValue
	// FIXME fv.isFinite
	if fv := value.float64(); math.IsNaN(fv) || math.IsInf(fv, 0) {
		return nullValue
	}

	toISOString := obj.get("toISOString")
	if !toISOString.isCallable() {
		// FIXME
		panic(call.runtime.panicTypeError("Date.toJSON toISOString %q is not callable", toISOString))
	}
	return toISOString.call(call.runtime, objectValue(obj), []Value{})
}

func builtinDateToGMTString(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return stringValue("Invalid Date")
	}
	return stringValue(date.Time().Format("Mon, 02 Jan 2006 15:04:05 GMT"))
}

func builtinDateGetTime(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return NaNValue()
	}
	// We do this (convert away from a float) so the user
	// does not get something back in exponential notation
	return int64Value(date.Epoch())
}

func builtinDateSetTime(call FunctionCall) Value {
	obj := call.thisObject()
	date := dateObjectOf(call.runtime, call.thisObject())
	date.Set(call.Argument(0).float64())
	obj.value = date
	return date.Value()
}

func builtinDateBeforeSet(call FunctionCall, argumentLimit int, timeLocal bool) (*object, *dateObject, *ecmaTime, []int) {
	obj := call.thisObject()
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return nil, nil, nil, nil
	}

	if argumentLimit > len(call.ArgumentList) {
		argumentLimit = len(call.ArgumentList)
	}

	if argumentLimit == 0 {
		obj.value = invalidDateObject
		return nil, nil, nil, nil
	}

	valueList := make([]int, argumentLimit)
	for index := range argumentLimit {
		value := call.ArgumentList[index]
		nm := value.number()
		switch nm.kind {
		case numberInteger, numberFloat:
		default:
			obj.value = invalidDateObject
			return nil, nil, nil, nil
		}
		valueList[index] = int(nm.int64)
	}
	baseTime := date.Time()
	if timeLocal {
		baseTime = baseTime.Local() //nolint:gosmopolitan
	}
	ecmaTime := newEcmaTime(baseTime)
	return obj, &date, &ecmaTime, valueList
}

func builtinDateParse(call FunctionCall) Value {
	date := call.Argument(0).string()
	return float64Value(dateParse(date))
}

func builtinDateUTC(call FunctionCall) Value {
	return float64Value(newDateTime(call.ArgumentList, time.UTC))
}

func builtinDateNow(call FunctionCall) Value {
	call.ArgumentList = []Value(nil)
	return builtinDateUTC(call)
}

// This is a placeholder.
func builtinDateToLocaleString(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return stringValue("Invalid Date")
	}
	return stringValue(date.Time().Local().Format("2006-01-02 15:04:05")) //nolint:gosmopolitan
}

// This is a placeholder.
func builtinDateToLocaleDateString(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return stringValue("Invalid Date")
	}
	return stringValue(date.Time().Local().Format("2006-01-02")) //nolint:gosmopolitan
}

// This is a placeholder.
func builtinDateToLocaleTimeString(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return stringValue("Invalid Date")
	}
	return stringValue(date.Time().Local().Format("15:04:05")) //nolint:gosmopolitan
}

func builtinDateValueOf(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return NaNValue()
	}
	return date.Value()
}

func builtinDateGetYear(call FunctionCall) Value {
	// Will throw a TypeError is ThisObject is nil or
	// does not have Class of "Date"
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return NaNValue()
	}
	return intValue(date.Time().Local().Year() - 1900) //nolint:gosmopolitan
}

func builtinDateGetFullYear(call FunctionCall) Value {
	// Will throw a TypeError is ThisObject is nil or
	// does not have Class of "Date"
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return NaNValue()
	}
	return intValue(date.Time().Local().Year()) //nolint:gosmopolitan
}

func builtinDateGetUTCFullYear(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return NaNValue()
	}
	return intValue(date.Time().Year())
}

func builtinDateGetMonth(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return NaNValue()
	}
	return intValue(dateFromGoMonth(date.Time().Local().Month())) //nolint:gosmopolitan
}

func builtinDateGetUTCMonth(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return NaNValue()
	}
	return intValue(dateFromGoMonth(date.Time().Month()))
}

func builtinDateGetDate(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return NaNValue()
	}
	return intValue(date.Time().Local().Day()) //nolint:gosmopolitan
}

func builtinDateGetUTCDate(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return NaNValue()
	}
	return intValue(date.Time().Day())
}

func builtinDateGetDay(call FunctionCall) Value {
	// Actually day of the week
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return NaNValue()
	}
	return intValue(dateFromGoDay(date.Time().Local().Weekday())) //nolint:gosmopolitan
}

func builtinDateGetUTCDay(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return NaNValue()
	}
	return intValue(dateFromGoDay(date.Time().Weekday()))
}

func builtinDateGetHours(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return NaNValue()
	}
	return intValue(date.Time().Local().Hour()) //nolint:gosmopolitan
}

func builtinDateGetUTCHours(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return NaNValue()
	}
	return intValue(date.Time().Hour())
}

func builtinDateGetMinutes(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return NaNValue()
	}
	return intValue(date.Time().Local().Minute()) //nolint:gosmopolitan
}

func builtinDateGetUTCMinutes(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return NaNValue()
	}
	return intValue(date.Time().Minute())
}

func builtinDateGetSeconds(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return NaNValue()
	}
	return intValue(date.Time().Local().Second()) //nolint:gosmopolitan
}

func builtinDateGetUTCSeconds(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return NaNValue()
	}
	return intValue(date.Time().Second())
}

func builtinDateGetMilliseconds(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return NaNValue()
	}
	return intValue(date.Time().Local().Nanosecond() / (100 * 100 * 100)) //nolint:gosmopolitan
}

func builtinDateGetUTCMilliseconds(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return NaNValue()
	}
	return intValue(date.Time().Nanosecond() / (100 * 100 * 100))
}

func builtinDateGetTimezoneOffset(call FunctionCall) Value {
	date := dateObjectOf(call.runtime, call.thisObject())
	if date.isNaN {
		return NaNValue()
	}
	timeLocal := date.Time().Local() //nolint:gosmopolitan
	// Is this kosher?
	timeLocalAsUTC := time.Date(
		timeLocal.Year(),
		timeLocal.Month(),
		timeLocal.Day(),
		timeLocal.Hour(),
		timeLocal.Minute(),
		timeLocal.Second(),
		timeLocal.Nanosecond(),
		time.UTC,
	)
	return float64Value(date.Time().Sub(timeLocalAsUTC).Seconds() / 60)
}

func builtinDateSetMilliseconds(call FunctionCall) Value {
	obj, date, ecmaTime, value := builtinDateBeforeSet(call, 1, true)
	if ecmaTime == nil {
		return NaNValue()
	}

	ecmaTime.millisecond = value[0]

	date.SetTime(ecmaTime.goTime())
	obj.value = *date
	return date.Value()
}

func builtinDateSetUTCMilliseconds(call FunctionCall) Value {
	obj, date, ecmaTime, value := builtinDateBeforeSet(call, 1, false)
	if ecmaTime == nil {
		return NaNValue()
	}

	ecmaTime.millisecond = value[0]

	date.SetTime(ecmaTime.goTime())
	obj.value = *date
	return date.Value()
}

func builtinDateSetSeconds(call FunctionCall) Value {
	obj, date, ecmaTime, value := builtinDateBeforeSet(call, 2, true)
	if ecmaTime == nil {
		return NaNValue()
	}

	if len(value) > 1 {
		ecmaTime.millisecond = value[1]
	}
	ecmaTime.second = value[0]

	date.SetTime(ecmaTime.goTime())
	obj.value = *date
	return date.Value()
}

func builtinDateSetUTCSeconds(call FunctionCall) Value {
	obj, date, ecmaTime, value := builtinDateBeforeSet(call, 2, false)
	if ecmaTime == nil {
		return NaNValue()
	}

	if len(value) > 1 {
		ecmaTime.millisecond = value[1]
	}
	ecmaTime.second = value[0]

	date.SetTime(ecmaTime.goTime())
	obj.value = *date
	return date.Value()
}

func builtinDateSetMinutes(call FunctionCall) Value {
	obj, date, ecmaTime, value := builtinDateBeforeSet(call, 3, true)
	if ecmaTime == nil {
		return NaNValue()
	}

	if len(value) > 2 {
		ecmaTime.millisecond = value[2]
		ecmaTime.second = value[1]
	} else if len(value) > 1 {
		ecmaTime.second = value[1]
	}
	ecmaTime.minute = value[0]

	date.SetTime(ecmaTime.goTime())
	obj.value = *date
	return date.Value()
}

func builtinDateSetUTCMinutes(call FunctionCall) Value {
	obj, date, ecmaTime, value := builtinDateBeforeSet(call, 3, false)
	if ecmaTime == nil {
		return NaNValue()
	}

	if len(value) > 2 {
		ecmaTime.millisecond = value[2]
		ecmaTime.second = value[1]
	} else if len(value) > 1 {
		ecmaTime.second = value[1]
	}
	ecmaTime.minute = value[0]

	date.SetTime(ecmaTime.goTime())
	obj.value = *date
	return date.Value()
}

func builtinDateSetHours(call FunctionCall) Value {
	obj, date, ecmaTime, value := builtinDateBeforeSet(call, 4, true)
	if ecmaTime == nil {
		return NaNValue()
	}

	switch {
	case len(value) > 3:
		ecmaTime.millisecond = value[3]
		fallthrough
	case len(value) > 2:
		ecmaTime.second = value[2]
		fallthrough
	case len(value) > 1:
		ecmaTime.minute = value[1]
	}
	ecmaTime.hour = value[0]

	date.SetTime(ecmaTime.goTime())
	obj.value = *date
	return date.Value()
}

func builtinDateSetUTCHours(call FunctionCall) Value {
	obj, date, ecmaTime, value := builtinDateBeforeSet(call, 4, false)
	if ecmaTime == nil {
		return NaNValue()
	}

	switch {
	case len(value) > 3:
		ecmaTime.millisecond = value[3]
		fallthrough
	case len(value) > 2:
		ecmaTime.second = value[2]
		fallthrough
	case len(value) > 1:
		ecmaTime.minute = value[1]
	}
	ecmaTime.hour = value[0]

	date.SetTime(ecmaTime.goTime())
	obj.value = *date
	return date.Value()
}

func builtinDateSetDate(call FunctionCall) Value {
	obj, date, ecmaTime, value := builtinDateBeforeSet(call, 1, true)
	if ecmaTime == nil {
		return NaNValue()
	}

	ecmaTime.day = value[0]

	date.SetTime(ecmaTime.goTime())
	obj.value = *date
	return date.Value()
}

func builtinDateSetUTCDate(call FunctionCall) Value {
	obj, date, ecmaTime, value := builtinDateBeforeSet(call, 1, false)
	if ecmaTime == nil {
		return NaNValue()
	}

	ecmaTime.day = value[0]

	date.SetTime(ecmaTime.goTime())
	obj.value = *date
	return date.Value()
}

func builtinDateSetMonth(call FunctionCall) Value {
	obj, date, ecmaTime, value := builtinDateBeforeSet(call, 2, true)
	if ecmaTime == nil {
		return NaNValue()
	}

	if len(value) > 1 {
		ecmaTime.day = value[1]
	}
	ecmaTime.month = value[0]

	date.SetTime(ecmaTime.goTime())
	obj.value = *date
	return date.Value()
}

func builtinDateSetUTCMonth(call FunctionCall) Value {
	obj, date, ecmaTime, value := builtinDateBeforeSet(call, 2, false)
	if ecmaTime == nil {
		return NaNValue()
	}

	if len(value) > 1 {
		ecmaTime.day = value[1]
	}
	ecmaTime.month = value[0]

	date.SetTime(ecmaTime.goTime())
	obj.value = *date
	return date.Value()
}

func builtinDateSetYear(call FunctionCall) Value {
	obj, date, ecmaTime, value := builtinDateBeforeSet(call, 1, true)
	if ecmaTime == nil {
		return NaNValue()
	}

	year := value[0]
	if 0 <= year && year <= 99 {
		year += 1900
	}
	ecmaTime.year = year

	date.SetTime(ecmaTime.goTime())
	obj.value = *date
	return date.Value()
}

func builtinDateSetFullYear(call FunctionCall) Value {
	obj, date, ecmaTime, value := builtinDateBeforeSet(call, 3, true)
	if ecmaTime == nil {
		return NaNValue()
	}

	if len(value) > 2 {
		ecmaTime.day = value[2]
		ecmaTime.month = value[1]
	} else if len(value) > 1 {
		ecmaTime.month = value[1]
	}
	ecmaTime.year = value[0]

	date.SetTime(ecmaTime.goTime())
	obj.value = *date
	return date.Value()
}

func builtinDateSetUTCFullYear(call FunctionCall) Value {
	obj, date, ecmaTime, value := builtinDateBeforeSet(call, 3, false)
	if ecmaTime == nil {
		return NaNValue()
	}

	if len(value) > 2 {
		ecmaTime.day = value[2]
		ecmaTime.month = value[1]
	} else if len(value) > 1 {
		ecmaTime.month = value[1]
	}
	ecmaTime.year = value[0]

	date.SetTime(ecmaTime.goTime())
	obj.value = *date
	return date.Value()
}

// toUTCString
// toISOString
// toJSONString
// toJSON
