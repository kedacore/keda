// Licensed to ClickHouse, Inc. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. ClickHouse, Inc. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package clickhouse

import (
	std_driver "database/sql/driver"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/column"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

func Named(name string, value any) driver.NamedValue {
	return driver.NamedValue{
		Name:  name,
		Value: value,
	}
}

type TimeUnit uint8

const (
	Seconds TimeUnit = iota
	MilliSeconds
	MicroSeconds
	NanoSeconds
)

type GroupSet struct {
	Value []any
}

type ArraySet []any

func DateNamed(name string, value time.Time, scale TimeUnit) driver.NamedDateValue {
	return driver.NamedDateValue{
		Name:  name,
		Value: value,
		Scale: uint8(scale),
	}
}

var (
	bindNumericRe    = regexp.MustCompile(`\$[0-9]+`)
	bindPositionalRe = regexp.MustCompile(`[^\\][?]`)
)

func bind(tz *time.Location, query string, args ...any) (string, error) {
	if len(args) == 0 {
		return query, nil
	}
	var (
		haveNumeric    bool
		havePositional bool
	)

	allArgumentsNamed, err := checkAllNamedArguments(args...)
	if err != nil {
		return "", err
	}

	if allArgumentsNamed {
		return bindNamed(tz, query, args...)
	}

	haveNumeric = bindNumericRe.MatchString(query)
	havePositional = bindPositionalRe.MatchString(query)
	if haveNumeric && havePositional {
		return "", ErrBindMixedParamsFormats
	}
	if haveNumeric {
		return bindNumeric(tz, query, args...)
	}
	return bindPositional(tz, query, args...)
}

func checkAllNamedArguments(args ...any) (bool, error) {
	var (
		haveNamed     bool
		haveAnonymous bool
	)
	for _, v := range args {
		switch v.(type) {
		case driver.NamedValue, driver.NamedDateValue:
			haveNamed = true
		default:
			haveAnonymous = true
		}
		if haveNamed && haveAnonymous {
			return haveNamed, ErrBindMixedParamsFormats
		}
	}
	return haveNamed, nil
}

func bindPositional(tz *time.Location, query string, args ...any) (_ string, err error) {
	var (
		lastMatchIndex = -1 // Position of previous match for copying
		argIndex       = 0  // Index for the argument at current position
		buf            = make([]byte, 0, len(query))
		unbindCount    = 0 // Number of positional arguments that couldn't be matched
	)

	for i := 0; i < len(query); i++ {
		// It's fine looping through the query string as bytes, because the (fixed) characters we're looking for
		// are in the ASCII range to won't take up more than one byte.
		if query[i] == '?' {
			if i > 0 && query[i-1] == '\\' {
				// Copy all previous index to here characters
				buf = append(buf, query[lastMatchIndex+1:i-1]...)
				buf = append(buf, '?')
			} else {
				// Copy all previous index to here characters
				buf = append(buf, query[lastMatchIndex+1:i]...)

				// Append the argument value
				if argIndex < len(args) {
					v := args[argIndex]
					if fn, ok := v.(std_driver.Valuer); ok {
						if v, err = fn.Value(); err != nil {
							return "", nil
						}
					}

					value, err := format(tz, Seconds, v)
					if err != nil {
						return "", err
					}

					buf = append(buf, value...)
					argIndex++
				} else {
					unbindCount++
				}
			}

			lastMatchIndex = i
		}
	}

	// If there were no replacements, quick return without copying the string
	if lastMatchIndex < 0 {
		return query, nil
	}

	// Append the remainder
	buf = append(buf, query[lastMatchIndex+1:]...)

	if unbindCount > 0 {
		return "", fmt.Errorf("have no arg for param ? at last %d positions", unbindCount)
	}

	return string(buf), nil
}

func bindNumeric(tz *time.Location, query string, args ...any) (_ string, err error) {
	var (
		unbind = make(map[string]struct{})
		params = make(map[string]string)
	)
	for i, v := range args {
		if fn, ok := v.(std_driver.Valuer); ok {
			if v, err = fn.Value(); err != nil {
				return "", nil
			}
		}
		val, err := format(tz, Seconds, v)
		if err != nil {
			return "", err
		}
		params[fmt.Sprintf("$%d", i+1)] = val
	}
	query = bindNumericRe.ReplaceAllStringFunc(query, func(n string) string {
		if _, found := params[n]; !found {
			unbind[n] = struct{}{}
			return ""
		}
		return params[n]
	})
	for param := range unbind {
		return "", fmt.Errorf("have no arg for %s param", param)
	}
	return query, nil
}

var bindNamedRe = regexp.MustCompile(`@[a-zA-Z0-9\_]+`)

func bindNamed(tz *time.Location, query string, args ...any) (_ string, err error) {
	var (
		unbind = make(map[string]struct{})
		params = make(map[string]string)
	)
	for _, v := range args {
		switch v := v.(type) {
		case driver.NamedValue:
			value := v.Value
			if fn, ok := v.Value.(std_driver.Valuer); ok {
				if value, err = fn.Value(); err != nil {
					return "", err
				}
			}
			val, err := format(tz, Seconds, value)
			if err != nil {
				return "", err
			}
			params["@"+v.Name] = val
		case driver.NamedDateValue:
			val, err := format(tz, TimeUnit(v.Scale), v.Value)
			if err != nil {
				return "", err
			}
			params["@"+v.Name] = val
		}
	}
	query = bindNamedRe.ReplaceAllStringFunc(query, func(n string) string {
		if _, found := params[n]; !found {
			unbind[n] = struct{}{}
			return ""
		}
		return params[n]
	})
	for param := range unbind {
		return "", fmt.Errorf("have no arg for %q param", param)
	}
	return query, nil
}

func formatTime(tz *time.Location, scale TimeUnit, value time.Time) (string, error) {
	switch value.Location().String() {
	case "Local", "":
		// It's required to pass timestamp as string due to decimal overflow for higher precision,
		// but zero-value string "toDateTime('0')" will be not parsed by ClickHouse.
		if value.Unix() == 0 {
			return "toDateTime(0)", nil
		}

		switch scale {
		case Seconds:
			return fmt.Sprintf("toDateTime('%d')", value.Unix()), nil
		case MilliSeconds:
			return fmt.Sprintf("toDateTime64('%d', 3)", value.UnixMilli()), nil
		case MicroSeconds:
			return fmt.Sprintf("toDateTime64('%d', 6)", value.UnixMicro()), nil
		case NanoSeconds:
			return fmt.Sprintf("toDateTime64('%d', 9)", value.UnixNano()), nil
		}
	case tz.String():
		if scale == Seconds {
			return value.Format("toDateTime('2006-01-02 15:04:05')"), nil
		}
		return fmt.Sprintf("toDateTime64('%s', %d)", value.Format(fmt.Sprintf("2006-01-02 15:04:05.%0*d", int(scale*3), 0)), int(scale*3)), nil
	}
	if scale == Seconds {
		return fmt.Sprintf("toDateTime('%s', '%s')", value.Format("2006-01-02 15:04:05"), value.Location().String()), nil
	}
	return fmt.Sprintf("toDateTime64('%s', %d, '%s')", value.Format(fmt.Sprintf("2006-01-02 15:04:05.%0*d", int(scale*3), 0)), int(scale*3), value.Location().String()), nil
}

var stringQuoteReplacer = strings.NewReplacer(`\`, `\\`, `'`, `\'`)

func format(tz *time.Location, scale TimeUnit, v any) (string, error) {
	quote := func(v string) string {
		return "'" + stringQuoteReplacer.Replace(v) + "'"
	}
	switch v := v.(type) {
	case nil:
		return "NULL", nil
	case string:
		return quote(v), nil
	case time.Time:
		return formatTime(tz, scale, v)
	case bool:
		if v {
			return "1", nil
		}
		return "0", nil
	case GroupSet:
		val, err := join(tz, scale, v.Value)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s)", val), nil
	case []GroupSet:
		val, err := join(tz, scale, v)
		if err != nil {
			return "", err
		}
		return val, err
	case ArraySet:
		val, err := join(tz, scale, v)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("[%s]", val), nil
	case fmt.Stringer:
		if v := reflect.ValueOf(v); v.Kind() == reflect.Pointer &&
			v.IsNil() &&
			v.Type().Elem().Implements(reflect.TypeOf((*fmt.Stringer)(nil)).Elem()) {
			return "NULL", nil
		}
		return quote(v.String()), nil
	case column.OrderedMap:
		values := make([]string, 0)
		for key := range v.Keys() {
			name, err := format(tz, scale, key)
			if err != nil {
				return "", err
			}
			value, _ := v.Get(key)
			val, err := format(tz, scale, value)
			if err != nil {
				return "", err
			}
			values = append(values, fmt.Sprintf("%s, %s", name, val))
		}

		return "map(" + strings.Join(values, ", ") + ")", nil
	case column.IterableOrderedMap:
		values := make([]string, 0)
		iter := v.Iterator()
		for iter.Next() {
			key, value := iter.Key(), iter.Value()
			name, err := format(tz, scale, key)
			if err != nil {
				return "", err
			}
			val, err := format(tz, scale, value)
			if err != nil {
				return "", err
			}
			values = append(values, fmt.Sprintf("%s, %s", name, val))
		}

		return "map(" + strings.Join(values, ", ") + ")", nil
	}
	switch v := reflect.ValueOf(v); v.Kind() {
	case reflect.String:
		return quote(v.String()), nil
	case reflect.Slice, reflect.Array:
		values := make([]string, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			val, err := format(tz, scale, v.Index(i).Interface())
			if err != nil {
				return "", err
			}
			values = append(values, val)
		}
		return fmt.Sprintf("[%s]", strings.Join(values, ", ")), nil
	case reflect.Map: // map
		values := make([]string, 0, len(v.MapKeys()))
		for _, key := range v.MapKeys() {
			name := fmt.Sprint(key.Interface())
			if key.Kind() == reflect.String {
				name = fmt.Sprintf("'%s'", name)
			}
			val, err := format(tz, scale, v.MapIndex(key).Interface())
			if err != nil {
				return "", err
			}
			values = append(values, fmt.Sprintf("%s, %s", name, val))
		}
		return "map(" + strings.Join(values, ", ") + ")", nil
	case reflect.Ptr:
		if v.IsNil() {
			return "NULL", nil
		}
		return format(tz, scale, v.Elem().Interface())
	}
	return fmt.Sprint(v), nil
}

func join[E any](tz *time.Location, scale TimeUnit, values []E) (string, error) {
	items := make([]string, len(values), len(values))
	for i := range values {
		val, err := format(tz, scale, values[i])
		if err != nil {
			return "", err
		}
		items[i] = val
	}
	return strings.Join(items, ", "), nil
}

func rebind(in []std_driver.NamedValue) []any {
	args := make([]any, 0, len(in))
	for _, v := range in {
		switch {
		case len(v.Name) != 0:
			args = append(args, driver.NamedValue{
				Name:  v.Name,
				Value: v.Value,
			})

		default:
			args = append(args, v.Value)
		}
	}
	return args
}
