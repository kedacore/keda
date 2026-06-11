package clickhouse

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"reflect"
	"sync/atomic"
	"syscall"

	"github.com/ClickHouse/clickhouse-go/v2/lib/column"
	chdriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

var globalConnID int64

type stdConnOpener struct {
	err    error
	opt    *Options
	logger *slog.Logger
}

func (o *stdConnOpener) Driver() driver.Driver {
	return &stdDriver{
		opt:    o.opt,
		logger: o.logger,
	}
}

func (o *stdConnOpener) Connect(ctx context.Context) (_ driver.Conn, err error) {
	if o.err != nil {
		o.logger.Error("opener error", slog.Any("error", o.err))
		return nil, o.err
	}
	var (
		conn     stdConnect
		connID   = int(atomic.AddInt64(&globalConnID, 1))
		dialFunc func(ctx context.Context, addr string, num int, opt *Options) (stdConnect, error)
	)

	switch o.opt.Protocol {
	case HTTP:
		dialFunc = func(ctx context.Context, addr string, num int, opt *Options) (stdConnect, error) {
			return dialHttp(ctx, addr, num, opt)
		}
	default:
		dialFunc = func(ctx context.Context, addr string, num int, opt *Options) (stdConnect, error) {
			return dial(ctx, addr, num, opt)
		}
	}

	if len(o.opt.Addr) == 0 {
		return nil, ErrAcquireConnNoAddress
	}

	for i := range o.opt.Addr {
		var num int
		switch o.opt.ConnOpenStrategy {
		case ConnOpenInOrder:
			num = i
		case ConnOpenRoundRobin:
			num = (connID + i) % len(o.opt.Addr)
		case ConnOpenRandom:
			random := rand.Int()
			num = (random + i) % len(o.opt.Addr)
		}
		if conn, err = dialFunc(ctx, o.opt.Addr[num], connID, o.opt); err == nil {
			// Create a logger with connection-specific context
			connLogger := o.logger.With(
				slog.Int("conn_num", num),
				slog.String("addr", o.opt.Addr[num]),
			)
			return &stdDriver{
				conn:   conn,
				logger: connLogger,
			}, nil
		} else {
			o.logger.Error("connection error",
				slog.String("addr", o.opt.Addr[num]),
				slog.Int("conn_id", connID),
				slog.Any("error", err))
		}
	}

	return nil, err
}

var _ driver.Connector = (*stdConnOpener)(nil)

func init() {
	sql.Register("clickhouse", &stdDriver{logger: newNoopLogger()})
}

// isConnBrokenError returns true if the error class indicates that the
// db connection is no longer usable and should be marked bad
func isConnBrokenError(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNRESET) {
		return true
	}
	if _, ok := err.(*net.OpError); ok {
		return true
	}
	return false
}

func Connector(opt *Options) driver.Connector {
	if opt == nil {
		opt = &Options{}
	}

	o := opt.setDefaults()
	logger := o.logger().With(slog.String("component", "std-driver"))

	return &stdConnOpener{
		opt:    o,
		logger: logger,
	}
}

func OpenDB(opt *Options) *sql.DB {
	if opt == nil {
		opt = &Options{}
	}

	o := opt.setDefaults()
	logger := o.logger().With(slog.String("component", "std-driver"))

	db := sql.OpenDB(&stdConnOpener{
		opt:    o,
		logger: logger,
	})

	// Ok to set these configs irrespective of values in opt.
	// Because opt.setDefaults() would have set some sane values
	// for these configs.
	db.SetMaxIdleConns(o.MaxIdleConns)
	db.SetMaxOpenConns(o.MaxOpenConns)
	db.SetConnMaxLifetime(o.ConnMaxLifetime)

	return db
}

type stdConnect interface {
	isBad() bool
	close() error
	query(ctx context.Context, release nativeTransportRelease, query string, args ...any) (*rows, error)
	exec(ctx context.Context, query string, args ...any) error
	ping(ctx context.Context) (err error)
	prepareBatch(ctx context.Context, release nativeTransportRelease, acquire nativeTransportAcquire, query string, options chdriver.PrepareBatchOptions) (chdriver.Batch, error)
	asyncInsert(ctx context.Context, query string, wait bool, args ...any) error
}

type stdDriver struct {
	opt    *Options
	conn   stdConnect
	commit func() error
	logger *slog.Logger
}

var _ driver.Conn = (*stdDriver)(nil)
var _ driver.ConnBeginTx = (*stdDriver)(nil)
var _ driver.ExecerContext = (*stdDriver)(nil)
var _ driver.QueryerContext = (*stdDriver)(nil)
var _ driver.ConnPrepareContext = (*stdDriver)(nil)

func (std *stdDriver) Open(dsn string) (_ driver.Conn, err error) {
	var opt Options
	if err := opt.fromDSN(dsn); err != nil {
		std.logger.Error("dsn parsing error", slog.Any("error", err))
		return nil, err
	}
	o := opt.setDefaults()
	logger := o.logger().With(slog.String("component", "std-driver"))
	o.ClientInfo.Comment = []string{"database/sql"}
	return (&stdConnOpener{opt: o, logger: logger}).Connect(context.Background())
}

var _ driver.Driver = (*stdDriver)(nil)

func (std *stdDriver) ResetSession(ctx context.Context) error {
	if std.conn.isBad() {
		std.logger.Debug("resetting session because connection is bad")
		return driver.ErrBadConn
	}
	return nil
}

var _ driver.SessionResetter = (*stdDriver)(nil)

func (std *stdDriver) Ping(ctx context.Context) error {
	if std.conn.isBad() {
		std.logger.Debug("ping: connection is bad")
		return driver.ErrBadConn
	}

	return std.conn.ping(ctx)
}

var _ driver.Pinger = (*stdDriver)(nil)

func (std *stdDriver) Begin() (driver.Tx, error) {
	if std.conn.isBad() {
		std.logger.Debug("begin: connection is bad")
		return nil, driver.ErrBadConn
	}

	return std, nil
}

func (std *stdDriver) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if std.conn.isBad() {
		std.logger.Debug("begin tx: connection is bad")
		return nil, driver.ErrBadConn
	}

	return std, nil
}

func (std *stdDriver) Commit() error {
	if std.commit == nil {
		return nil
	}
	defer func() {
		std.commit = nil
	}()

	if err := std.commit(); err != nil {
		if isConnBrokenError(err) {
			std.logger.Debug("commit got EOF error: resetting connection")
			return driver.ErrBadConn
		}
		std.logger.Error("commit error", slog.Any("error", err))
		return err
	}
	return nil
}

func (std *stdDriver) Rollback() error {
	std.commit = nil
	std.conn.close()
	return nil
}

var _ driver.Tx = (*stdDriver)(nil)

func (std *stdDriver) CheckNamedValue(nv *driver.NamedValue) error { return nil }

var _ driver.NamedValueChecker = (*stdDriver)(nil)

func (std *stdDriver) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if std.conn.isBad() {
		std.logger.Debug("exec context: connection is bad")
		return nil, driver.ErrBadConn
	}

	var err error
	if asyncOpt := queryOptionsAsync(ctx); asyncOpt.ok {
		err = std.conn.asyncInsert(ctx, query, asyncOpt.wait, rebind(args)...)
	} else {
		err = std.conn.exec(ctx, query, rebind(args)...)
	}

	if err != nil {
		if isConnBrokenError(err) {
			std.logger.Error("exec context got a fatal error, resetting connection", slog.Any("error", err))
			return nil, driver.ErrBadConn
		}
		std.logger.Error("exec context error", slog.Any("error", err))
		return nil, err
	}
	return driver.RowsAffected(0), nil
}

func (std *stdDriver) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if std.conn.isBad() {
		std.logger.Debug("query context: connection is bad")
		return nil, driver.ErrBadConn
	}

	r, err := std.conn.query(ctx, func(nativeTransport, error) {}, query, rebind(args)...)
	if isConnBrokenError(err) {
		std.logger.Error("query context got a fatal error, resetting connection", slog.Any("error", err))
		return nil, driver.ErrBadConn
	}
	if err != nil {
		std.logger.Error("query context error", slog.Any("error", err))
		return nil, err
	}
	return &stdRows{
		rows:   r,
		logger: std.logger,
	}, nil
}

func (std *stdDriver) Prepare(query string) (driver.Stmt, error) {
	return std.PrepareContext(context.Background(), query)
}

func (std *stdDriver) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if std.conn.isBad() {
		std.logger.Debug("prepare context: connection is bad")
		return nil, driver.ErrBadConn
	}

	batch, err := std.conn.prepareBatch(ctx, func(nativeTransport, error) {}, func(context.Context) (nativeTransport, error) { return nil, nil }, query, chdriver.PrepareBatchOptions{})
	if err != nil {
		if isConnBrokenError(err) {
			std.logger.Error("prepare context got a fatal error, resetting connection", slog.Any("error", err))
			return nil, driver.ErrBadConn
		}
		std.logger.Error("prepare context error", slog.Any("error", err))
		return nil, err
	}
	std.commit = batch.Send
	return &stdBatch{
		batch:  batch,
		logger: std.logger,
	}, nil
}

func (std *stdDriver) Close() error {
	err := std.conn.close()
	if err != nil {
		if isConnBrokenError(err) {
			std.logger.Error("close got a fatal error, resetting connection", slog.Any("error", err))
			return driver.ErrBadConn
		}
		std.logger.Error("close error", slog.Any("error", err))
	}
	return err
}

type stdBatch struct {
	batch  chdriver.Batch
	logger *slog.Logger
}

func (s *stdBatch) NumInput() int { return -1 }
func (s *stdBatch) Exec(args []driver.Value) (driver.Result, error) {
	values := make([]any, 0, len(args))
	for _, v := range args {
		values = append(values, v)
	}
	if err := s.batch.Append(values...); err != nil {
		return nil, err
	}
	return driver.RowsAffected(0), nil
}

func (s *stdBatch) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	values := make([]driver.Value, 0, len(args))
	for _, v := range args {
		values = append(values, v.Value)
	}
	return s.Exec(values)
}

var _ driver.StmtExecContext = (*stdBatch)(nil)

func (s *stdBatch) Query(args []driver.Value) (driver.Rows, error) {
	// Note: not implementing driver.StmtQueryContext accordingly
	return nil, errors.New("only Exec method supported in batch mode")
}

func (s *stdBatch) Close() error { return nil }

type stdRows struct {
	rows   *rows
	logger *slog.Logger
}

func (r *stdRows) Columns() []string {
	return r.rows.Columns()
}

func (r *stdRows) ColumnTypeScanType(idx int) reflect.Type {
	return r.rows.block.Columns[idx].ScanType()
}

var _ driver.RowsColumnTypeScanType = (*stdRows)(nil)

func (r *stdRows) ColumnTypeDatabaseTypeName(idx int) string {
	return string(r.rows.block.Columns[idx].Type())
}

func (r *stdRows) ColumnTypeNullable(idx int) (nullable, ok bool) {
	_, ok = r.rows.block.Columns[idx].(*column.Nullable)
	return ok, true
}

func (r *stdRows) ColumnTypePrecisionScale(idx int) (precision, scale int64, ok bool) {
	switch col := r.rows.block.Columns[idx].(type) {
	case *column.Decimal:
		return col.Precision(), col.Scale(), true
	case *column.DateTime64:
		p, ok := col.Precision()
		return p, 0, ok
	case interface{ Base() column.Interface }:
		switch col := col.Base().(type) {
		case *column.Decimal:
			return col.Precision(), col.Scale(), true
		case *column.DateTime64:
			p, ok := col.Precision()
			return p, 0, ok
		}
	}
	return 0, 0, false
}

var _ driver.Rows = (*stdRows)(nil)
var _ driver.RowsNextResultSet = (*stdRows)(nil)
var _ driver.RowsColumnTypeDatabaseTypeName = (*stdRows)(nil)
var _ driver.RowsColumnTypeNullable = (*stdRows)(nil)
var _ driver.RowsColumnTypePrecisionScale = (*stdRows)(nil)

func (r *stdRows) Next(dest []driver.Value) error {
	if len(r.rows.block.Columns) != len(dest) {
		err := fmt.Errorf("expected %d destination arguments in Next, not %d", len(r.rows.block.Columns), len(dest))
		r.logger.Error("next length error", slog.Any("error", err))
		return &OpError{
			Op:  "Next",
			Err: err,
		}
	}
	if r.rows.Next() {
		for i := range dest {
			nullable, ok := r.ColumnTypeNullable(i)
			switch value := r.rows.block.Columns[i].Row(r.rows.row-1, nullable && ok).(type) {
			case driver.Valuer:
				v, err := value.Value()
				if err != nil {
					r.logger.Error("next row error", slog.Any("error", err))
					return err
				}
				dest[i] = v
			default:
				// We don't know what is the destination type at this stage,
				// but destination type might be a sql.Null* type that expects to receive a value
				// instead of a pointer to a value. ClickHouse-go returns pointers to values for nullable columns.
				//
				// This is a compatibility layer to make sure that the driver works with the standard library.
				// Due to reflection used it has a performance cost.
				if nullable {
					if value == nil {
						dest[i] = nil
						continue
					}
					rv := reflect.ValueOf(value)
					value = rv.Elem().Interface()
				}

				dest[i] = value
			}
		}
		return nil
	}
	if err := r.rows.Err(); err != nil {
		r.logger.Error("next rows error", slog.Any("error", err))
		return err
	}
	return io.EOF
}

func (r *stdRows) HasNextResultSet() bool {
	return r.rows.totals != nil
}

func (r *stdRows) NextResultSet() error {
	switch {
	case r.rows.totals != nil:
		r.rows.block = r.rows.totals
		r.rows.totals = nil
	default:
		return io.EOF
	}
	return nil
}

var _ driver.RowsNextResultSet = (*stdRows)(nil)

func (r *stdRows) Close() error {
	err := r.rows.Close()
	if err != nil {
		r.logger.Error("rows close error", slog.Any("error", err))
	}
	return err
}

var _ driver.Rows = (*stdRows)(nil)
