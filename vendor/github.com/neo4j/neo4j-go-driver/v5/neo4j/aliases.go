/*
 * Copyright (c) "Neo4j"
 * Neo4j Sweden AB [https://neo4j.com]
 *
 * This file is part of Neo4j.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package neo4j

import (
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j/db"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
)

// Aliases to simplify client usage (fewer imports) and to provide some backwards
// compatibility with 1.x driver.
//
// A separate dbtype package is needed to avoid circular package references and to avoid
// unnecessary copying/conversions between structs since serializing/deserializing is
// handled within bolt package and bolt package is used from this package.
type (
	Point2D       = dbtype.Point2D
	Point3D       = dbtype.Point3D
	Date          = dbtype.Date
	LocalTime     = dbtype.LocalTime
	LocalDateTime = dbtype.LocalDateTime
	Time          = dbtype.Time
	OffsetTime    = dbtype.Time
	Duration      = dbtype.Duration
	Node          = dbtype.Node
	Relationship  = dbtype.Relationship
	Path          = dbtype.Path
	Record        = db.Record
	InvalidValue  = dbtype.InvalidValue
)

// DateOf creates a neo4j.Date from time.Time.
// Hour, minute, second and nanoseconds are set to zero and location is set to UTC.
//
// Conversion can also be done by casting a time.Time to neo4j.Date but beware that time
// components and location will be left as is but ignored when used as query parameter.
func DateOf(t time.Time) Date {
	y, m, d := t.Date()
	t = time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return dbtype.Date(t)
}

// LocalTimeOf creates a neo4j.LocalTime from time.Time.
// Year, month and day are set to zero and location is set to local.
//
// Conversion can also be done by casting a time.Time to neo4j.LocalTime but beware that date
// components and location will be left as is but ignored when used as query parameter.
func LocalTimeOf(t time.Time) LocalTime {
	t = time.Date(0, 0, 0, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local)
	return dbtype.LocalTime(t)
}

// LocalDateTimeOf creates a neo4j.Local from time.Time.
//
// Conversion can also be done by casting a time.Time to neo4j.LocalTime but beware that location
// will be left as is but interpreted as local when used as query parameter.
func LocalDateTimeOf(t time.Time) LocalDateTime {
	t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local)
	return dbtype.LocalDateTime(t)
}

// OffsetTimeOf creates a neo4j.OffsetTime from time.Time.
// Year, month and day are set to zero and location is set to "Offset" using zone offset from
// time.Time.
//
// Conversion can also be done by casting a time.Time to neo4j.OffsetTime but beware that date
// components and location will be left as is but ignored when used as query parameter. Since
// location will contain the original value, the value "offset" will not be used by the driver
// but the actual name of the location in time.Time and that offset.
func OffsetTimeOf(t time.Time) OffsetTime {
	_, offset := t.Zone()
	l := time.FixedZone("Offset", int(offset))
	t = time.Date(0, 0, 0, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), l)
	return dbtype.Time(t)
}

// DurationOf creates neo4j.Duration from specified time parts.
func DurationOf(months, days, seconds int64, nanos int) Duration {
	return Duration{Months: months, Days: days, Seconds: seconds, Nanos: nanos}
}
