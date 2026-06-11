package driver

import (
	"context"
	"reflect"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/column"
	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

type ServerVersion = proto.ServerHandshake

type (
	NamedValue struct {
		Name  string
		Value any
	}

	NamedDateValue struct {
		Name  string
		Value time.Time
		Scale uint8
	}

	Stats struct {
		MaxOpenConns int
		MaxIdleConns int
		Open         int
		Idle         int
	}
)

type (
	Conn interface {
		Contributors() []string
		ServerVersion() (*ServerVersion, error)
		Select(ctx context.Context, dest any, query string, args ...any) error
		Query(ctx context.Context, query string, args ...any) (Rows, error)
		QueryRow(ctx context.Context, query string, args ...any) Row
		PrepareBatch(ctx context.Context, query string, opts ...PrepareBatchOption) (Batch, error)
		Exec(ctx context.Context, query string, args ...any) error

		// Deprecated: use context aware `WithAsync()` for any async operations
		AsyncInsert(ctx context.Context, query string, wait bool, args ...any) error
		Ping(context.Context) error
		Stats() Stats
		Close() error
	}
	Row interface {
		Err() error
		Scan(dest ...any) error
		ScanStruct(dest any) error
	}
	Rows interface {
		Next() bool
		Scan(dest ...any) error
		ScanStruct(dest any) error
		ColumnTypes() []ColumnType
		Totals(dest ...any) error
		Columns() []string
		Close() error
		Err() error
		HasData() bool
	}

	// Batch represents a prepared INSERT that buffers rows client-side and sends them to ClickHouse.
	//
	// Typical usage:
	//
	//	batch, err := conn.PrepareBatch(ctx, "INSERT INTO t")
	//	if err != nil { ... }
	//	defer batch.Close() // cleanup if Send is not reached
	//
	//	for ... {
	//		_ = batch.Append(...)
	//		// Optionally flush periodically for native protocol.
	//		// _ = batch.Flush()
	//	}
	//	_ = batch.Send()
	//
	// Notes:
	// - After Send(), the batch is considered finalized (IsSent() becomes true). Create a new batch to send more rows.
	// - For HTTP protocol, Flush() is currently a no-op. Use Send() to transmit buffered rows.
	Batch interface {
		Abort() error
		Append(v ...any) error
		AppendStruct(v any) error
		Column(int) BatchColumn

		// Flush sends the currently buffered rows but keeps the batch usable.
		//
		// For native protocol this transmits the buffered block to the server and clears the local buffer.
		// For HTTP protocol this is currently a no-op.
		Flush() error

		// Send flushes any buffered rows and finalizes the INSERT.
		// After Send() the batch is considered sent and should not be reused.
		Send() error

		// IsSent reports whether the batch has been finalized via Send(), Abort(), or Close().
		IsSent() bool
		Rows() int
		Columns() []column.Interface

		// Close ends the current INSERT and releases resources.
		//
		// It is safe (and recommended) to call Close via defer immediately after PrepareBatch.
		// Close does not guarantee that buffered rows are sent; call Send() to finalize the INSERT.
		Close() error
	}
	BatchColumn interface {
		// Append appends a value to the underlying column buffer.
		Append(any) error
		// AppendRow appends a row-oriented value to the underlying column buffer.
		AppendRow(any) error
	}
	ColumnType interface {
		Name() string
		Nullable() bool
		ScanType() reflect.Type
		DatabaseTypeName() string
	}
)
