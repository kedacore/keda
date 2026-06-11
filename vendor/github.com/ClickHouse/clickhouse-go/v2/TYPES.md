The following table aims to capture the Golang types supported for each ClickHouse Column Type.

Whilst each ClickHouse type often has a logical Golang type, we aim to support implicit conversions where possible and provided no precision loss will be incurred - thus alleviating the need for users to ensure their data aligns perfectly with ClickHouse types.

This effort is ongoing and can be separated in to insertion (`Append`/`AppendRow`) and read time (via a `Scan`). Should you need support for a specific conversion, please raise an issue.

## Append Support

All types can be inserted as a value or pointer.

|               | **ClickHouse Type** | String | Decimal | Bool | FixedString | UInt8 | UInt16 | UInt32 | UInt64 | UInt128 | UInt256 | Int8 | Int16 | Int32 | Int64 | Int128 | Int256 | Float32 | Float64 | UUID | Date | Date32 | DateTime | DateTime64 | Time | Time64 | Enum8 | Enum16 | Point | Ring | Polygon | MultiPolygon | LineString | MultiLineString |
|---------------|---------------------|--------|---------|------|-------------|-------|--------|--------|--------|---------|---------|------|-------|-------|-------|--------|--------|---------|---------|------|------|--------|----------|------------|------|--------|-------|--------|-------|------|---------|--------------|------------|-----------------|
| **Golang Type** |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |      |        |       |        |       |      |         |              |            |                 |
| uint          |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| unit64        |                     |        |         |      |             |       |        |        |    X   |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| uint32        |                     |        |         |      |             |       |        |    X   |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| uint16        |                     |        |         |      |             |       |    X   |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| uint8         |                     |        |         |      |             |   X   |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| int           |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |        |       |      |         |              |
| int64         |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |   X   |        |        |         |         |      |      |        |     X    |      X     |       |        |       |        |       |      |         |              |
| int32         |                     |        |         |      |             |       |        |        |        |         |         |      |       |   X   |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| int16         |                     |        |         |      |             |       |        |        |        |         |         |      |   X   |       |       |        |        |         |         |      |      |        |          |            |       |    X   |       |      |         |              |
| int8          |                     |        |         |      |             |       |        |        |        |         |         |   X  |       |       |       |        |        |         |         |      |      |        |          |            |   X   |        |       |      |         |              |
| float32       |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |    X    |         |      |      |        |          |            |       |        |       |      |         |              |
| float64       |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |    X    |      |      |        |          |            |       |        |       |      |         |              |
| string        |                     |    X   |         |      |      X      |       |        |        |        |         |         |      |       |       |       |        |        |         |         |   X  |   X  |    X   |     X    |      X     |   X   |    X   |       |        |       |      |         |              |
| bool          |                     |        |         | X    |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |        |       |      |         |              |
| time.Time     |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |   X  |    X   |     X    |      X     |       |        |       |        |       |      |         |              |
| time.Duration |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |   X   |    X   |       |        |       |      |         |              |
| big.Int       |                     |        |         |      |             |       |        |        |        |    X    |    X    |      |       |       |       |    X   |    X   |         |         |      |      |        |          |            |       |        |       |      |         |              |
| decimal.Decimal |                     |        |    X    |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| uuid.UUID     |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |   X  |      |        |          |            |       |        |       |      |         |              |
| orb.Point     |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |   X   |      |         |              |
| orb.Polygon   |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |    X    |              |
| orb.Ring      |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |   X  |         |              |
| orb.MultiPolygon |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |       X      |            |                 |
| orb.LineString |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |     X      |                 |
| orb.MultiLineString |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |            |        X        |
| []byte        |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |       X      |            |                 |
 | fmt.Stringer  |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |        |       |      |         |              |
| sql.NullString |                     |    X   |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |        |       |      |         |              |
| sql.NullTime  |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |   X  |    X   |     X    |      X     |       |        |       |        |       |      |         |              |
| sql.NullFloat64 |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |    X    |      |      |        |          |            |       |        |       |        |       |      |         |              |
| sql.NullInt64 |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |   X   |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| sql.NullInt32 |                     |        |         |      |             |       |        |        |        |         |         |      |       |   X   |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| sql.NullInt16 |                     |        |         |      |             |       |        |        |        |         |         |      |   X   |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| sql.NullBool  |                     |        |         | X    |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |

## Scan Support

All types can be read into a pointer or pointer to a pointer.

|               | **ClickHouse Type** | String | Decimal | Bool | FixedString | UInt8 | UInt16 | UInt32 | UInt64 | UInt128 | UInt256 | Int8 | Int16 | Int32 | Int64 | Int128 | Int256 | Float32 | Float64 | UUID | Date | Date32 | DateTime | DateTime64 | Time | Time64 | Enum8 | Enum16 | Point | Ring | Polygon | MultiPolygon | LineString | MultiLineString |
|---------------|---------------------|--------|---------|------|-------------|-------|--------|--------|--------|---------|---------|------|-------|-------|-------|--------|--------|---------|---------|------|------|--------|----------|------------|------|--------|-------|--------|-------|------|---------|--------------|------------|-----------------|
| **Golang Type** |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |      |        |       |        |       |      |         |              |            |                 |
| uint          |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| unit64        |                     |        |         |      |             |       |        |        |    X   |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| uint32        |                     |        |         |      |             |       |        |    X   |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| uint16        |                     |        |         |      |             |       |    X   |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| uint8         |                     |        |         |      |             |   X   |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| int           |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| int64         |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |   X   |        |        |         |         |      |      |        |     x    |      x     |       |        |       |      |         |              |
| int32         |                     |        |         |      |             |       |        |        |        |         |         |      |       |   X   |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| int16         |                     |        |         |      |             |       |        |        |        |         |         |      |   X   |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| int8          |                     |        |         |      |             |       |        |        |        |         |         |   X  |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| float32       |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |    X    |         |      |      |        |          |            |       |        |       |      |         |              |
| float64       |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |    X    |      |      |        |          |            |       |        |       |      |         |              |
| string        |                     |    X   |         |      |      X      |       |        |        |        |         |         |      |       |       |       |        |        |         |         |   X  |      |        |          |            |   X   |    X   |       |        |       |      |         |              |
| bool          |                     |        |         | X    |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |        |       |      |         |              |
| time.Time     |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |   X  |    X   |     X    |      X     |       |        |       |        |       |      |         |              |
| time.Duration |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |   X   |    X   |       |        |       |      |         |              |
| big.Int       |                     |        |         |      |             |       |        |        |        |    X    |    X    |      |       |       |       |    X   |    X   |         |         |      |      |        |          |            |       |        |       |        |       |      |         |              |
| decimal.Decimal |                     |        |    X    |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| uuid.UUID     |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |   X  |      |        |          |            |       |        |       |      |         |              |
| orb.Point     |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |   X   |      |         |              |
| orb.Polygon   |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |    X    |              |
| orb.Ring      |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |   X  |         |              |
| orb.MultiPolygon |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |       X      |            |                 |
| orb.LineString |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |     X      |                 |
| orb.MultiLineString |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |            |        X        |
| sql.Scan      |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |   X  |   X  |    X   |     X    |      X     |       |        |       |        |       |      |         |              |            |                 |
| sql.NullString |                     |    X   |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |        |       |      |         |              |
| sql.NullTime  |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |   X  |    X   |     X    |      X     |       |        |       |        |       |      |         |              |
| sql.NullFloat64 |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |       |        |        |         |    X    |      |      |        |          |            |       |        |       |        |       |      |         |              |
| sql.NullInt64 |                     |        |         |      |             |       |        |        |        |         |         |      |       |       |   X   |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| sql.NullInt32 |                     |        |         |      |             |       |        |        |        |         |         |      |       |   X   |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| sql.NullInt16 |                     |        |         |      |             |       |        |        |        |         |         |      |   X   |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |
| sql.NullBool  |                     |        |         | X    |             |       |        |        |        |         |         |      |       |       |       |        |        |         |         |      |      |        |          |            |       |        |       |      |         |              |

---

## Time and Time64 Example

ClickHouse `Time` and `Time64` types represent time-of-day values (without date). In Go, these map to `time.Duration` representing elapsed time since midnight.

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func main() {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{"localhost:9000"},
	})
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	ctx := clickhouse.Context(context.Background(), clickhouse.WithSettings(clickhouse.Settings{
		"enable_time_time64_type": 1,
	}))

	// Create table with Time and Time64 columns
	err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS time_example (
			id UInt32,
			t_time Time,                    -- second precision
			t_time64_3 Time64(3),            -- millisecond precision
			t_time64_6 Time64(6),            -- microsecond precision
			t_time64_9 Time64(9),            -- nanosecond precision
			arr_time Array(Time),            -- array of Time values
			nullable_time Nullable(Time64(9)) -- nullable Time64
		) ENGINE = MergeTree() ORDER BY id
	`)
	if err != nil {
		panic(err)
	}

	// Insert data using time.Duration
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO time_example")
	if err != nil {
		panic(err)
	}

	// Time values as time.Duration (duration since midnight)
	timeValue := 12*time.Hour + 34*time.Minute + 56*time.Second
	time64Value := 15*time.Hour + 30*time.Minute + 45*time.Second + 123456789*time.Nanosecond

	timeArray := []time.Duration{
		6 * time.Hour,
		12*time.Hour + 30*time.Minute,
		18*time.Hour + 45*time.Minute + 30*time.Second,
	}

	err = batch.Append(
		uint32(1),
		timeValue,
		time64Value,
		time64Value,
		time64Value,
		timeArray,
		&time64Value, // nullable value
	)
	if err != nil {
		panic(err)
	}

	// Insert NULL for nullable column
	err = batch.Append(
		uint32(2),
		timeValue,
		time64Value,
		time64Value,
		time64Value,
		timeArray,
		nil, // NULL value
	)
	if err != nil {
		panic(err)
	}

	err = batch.Send()
	if err != nil {
		panic(err)
	}

	// Query data
	rows, err := conn.Query(ctx, "SELECT id, t_time, t_time64_9, arr_time, nullable_time FROM time_example ORDER BY id")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id            uint32
			tTime         time.Duration
			tTime64       time.Duration
			arrTime       []time.Duration
			nullableTime  *time.Duration
		)
		if err := rows.Scan(&id, &tTime, &tTime64, &arrTime, &nullableTime); err != nil {
			panic(err)
		}

		fmt.Printf("ID: %d\n", id)
		fmt.Printf("  Time (second precision): %v\n", tTime)
		fmt.Printf("  Time64(9) (nanosecond precision): %v\n", tTime64)
		fmt.Printf("  Array(Time): %v\n", arrTime)
		if nullableTime != nil {
			fmt.Printf("  Nullable Time64: %v\n", *nullableTime)
		} else {
			fmt.Printf("  Nullable Time64: NULL\n")
		}
	}
}

// Output:
// ID: 1
//   Time (second precision): 12h34m56s
//   Time64(9) (nanosecond precision): 15h30m45.123456789s
//   Array(Time): [6h0m0s 12h30m0s 18h45m30s]
//   Nullable Time64: 15h30m45.123456789s
// ID: 2
//   Time (second precision): 12h34m56s
//   Time64(9) (nanosecond precision): 15h30m45.123456789s
//   Array(Time): [6h0m0s 12h30m0s 18h45m30s]
//   Nullable Time64: NULL

// Key points:
// - Time has second precision (truncates to seconds)
// - Time64(N) supports precision from 0 (second) to 9 (nanosecond)
// - Use time.Duration for both Time and Time64 types
// - String input format is also supported: "HH:MM:SS" or "HH:MM:SS.sss..."
// - Time values represent time-of-day only (no date component)
// - Timezone is not applicable (values are timezone-agnostic)
```
