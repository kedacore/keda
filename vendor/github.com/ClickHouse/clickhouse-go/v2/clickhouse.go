package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	_ "time/tzdata"

	"github.com/ClickHouse/clickhouse-go/v2/contributors"
	"github.com/ClickHouse/clickhouse-go/v2/lib/column"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

type Conn = driver.Conn

type (
	Progress      = proto.Progress
	Exception     = proto.Exception
	ProfileInfo   = proto.ProfileInfo
	ServerVersion = proto.ServerHandshake
)

var (
	ErrBatchInvalid              = errors.New("clickhouse: batch is invalid. check appended data is correct")
	ErrBatchAlreadySent          = errors.New("clickhouse: batch has already been sent")
	ErrBatchNotSent              = errors.New("clickhouse: invalid retry, batch not sent yet")
	ErrAcquireConnTimeout        = errors.New("clickhouse: acquire conn timeout. you can increase the number of max open conn or the dial timeout")
	ErrUnsupportedServerRevision = errors.New("clickhouse: unsupported server revision")
	ErrBindMixedParamsFormats    = errors.New("clickhouse [bind]: mixed named, numeric or positional parameters")
	ErrAcquireConnNoAddress      = errors.New("clickhouse: no valid address supplied")
	ErrServerUnexpectedData      = errors.New("code: 101, message: Unexpected packet Data received from client")
	ErrConnectionClosed          = errors.New("clickhouse: connection is closed")
)

type OpError struct {
	Op         string
	ColumnName string
	Err        error
}

func (e *OpError) Error() string {
	switch err := e.Err.(type) {
	case *column.Error:
		return fmt.Sprintf("clickhouse [%s]: (%s %s) %s", e.Op, e.ColumnName, err.ColumnType, err.Err)
	case *column.ColumnConverterError:
		var hint string
		if len(err.Hint) != 0 {
			hint += ". " + err.Hint
		}
		return fmt.Sprintf("clickhouse [%s]: (%s) converting %s to %s is unsupported%s",
			err.Op, e.ColumnName,
			err.From, err.To,
			hint,
		)
	}
	return fmt.Sprintf("clickhouse [%s]: %s", e.Op, e.Err)
}

func Open(opt *Options) (driver.Conn, error) {
	if opt == nil {
		opt = &Options{}
	}
	o := opt.setDefaults()

	conn := &clickhouse{
		opt:       o,
		idle:      newConnPool(o.ConnMaxLifetime, o.MaxIdleConns),
		open:      make(chan struct{}, o.MaxOpenConns),
		closeOnce: &sync.Once{},
		closed:    &atomic.Bool{},
	}

	return conn, nil
}

// nativeTransport represents an implementation (TCP or HTTP) that can be pooled by the main clickhouse struct.
// Implementations are not expected to be thread safe, which is why we provide acquire/release functions.
type nativeTransport interface {
	serverVersion() (*ServerVersion, error)
	query(ctx context.Context, release nativeTransportRelease, query string, args ...any) (*rows, error)
	queryRow(ctx context.Context, release nativeTransportRelease, query string, args ...any) *row
	prepareBatch(ctx context.Context, release nativeTransportRelease, acquire nativeTransportAcquire, query string, opts driver.PrepareBatchOptions) (driver.Batch, error)
	exec(ctx context.Context, query string, args ...any) error
	asyncInsert(ctx context.Context, query string, wait bool, args ...any) error
	ping(context.Context) error
	isBad() bool
	connID() int
	connectedAtTime() time.Time
	isReleased() bool
	setReleased(released bool)
	getLogger() *slog.Logger
	// freeBuffer is called if Options.FreeBufOnConnRelease is set
	freeBuffer()
	close() error
}
type nativeTransportAcquire func(context.Context) (nativeTransport, error)
type nativeTransportRelease func(nativeTransport, error)

// connectionPooler is an connection pool maintain
// idle connections.
type connectionPooler interface {
	Get(ctx context.Context) (nativeTransport, error)
	Put(conn nativeTransport)
	Len() int
	Cap() int
	Close() error
}

type clickhouse struct {
	opt    *Options
	connID int64

	idle connectionPooler
	open chan struct{}

	closeOnce *sync.Once
	closed    *atomic.Bool
}

func (clickhouse) Contributors() []string {
	list := contributors.List
	if len(list[len(list)-1]) == 0 {
		return list[:len(list)-1]
	}
	return list
}

func (ch *clickhouse) ServerVersion() (*driver.ServerVersion, error) {
	var (
		ctx, cancel = context.WithTimeout(context.Background(), ch.opt.DialTimeout)
		conn, err   = ch.acquire(ctx)
	)
	defer cancel()
	if err != nil {
		return nil, err
	}
	defer ch.release(conn, nil)
	return conn.serverVersion()
}

func (ch *clickhouse) Query(ctx context.Context, query string, args ...any) (rows driver.Rows, err error) {
	conn, err := ch.acquire(ctx)
	if err != nil {
		return nil, err
	}
	conn.getLogger().Debug("executing query", slog.String("sql", query))
	return conn.query(ctx, ch.release, query, args...)
}

func (ch *clickhouse) QueryRow(ctx context.Context, query string, args ...any) driver.Row {
	conn, err := ch.acquire(ctx)
	if err != nil {
		return &row{
			err: err,
		}
	}

	conn.getLogger().Debug("executing query row", slog.String("sql", query))
	return conn.queryRow(ctx, ch.release, query, args...)
}

func (ch *clickhouse) Exec(ctx context.Context, query string, args ...any) error {
	conn, err := ch.acquire(ctx)
	if err != nil {
		return err
	}
	conn.getLogger().Debug("executing statement", slog.String("sql", query))

	if asyncOpt := queryOptionsAsync(ctx); asyncOpt.ok {
		err = conn.asyncInsert(ctx, query, asyncOpt.wait, args...)
	} else {
		err = conn.exec(ctx, query, args...)
	}

	if err != nil {
		ch.release(conn, err)
		return err
	}

	ch.release(conn, nil)
	return nil
}

func (ch *clickhouse) PrepareBatch(ctx context.Context, query string, opts ...driver.PrepareBatchOption) (driver.Batch, error) {
	conn, err := ch.acquire(ctx)
	if err != nil {
		return nil, err
	}
	conn.getLogger().Debug("preparing batch", slog.String("sql", query))
	batch, err := conn.prepareBatch(ctx, ch.release, ch.acquire, query, getPrepareBatchOptions(opts...))
	if err != nil {
		return nil, err
	}
	return batch, nil
}

func getPrepareBatchOptions(opts ...driver.PrepareBatchOption) driver.PrepareBatchOptions {
	var options driver.PrepareBatchOptions

	for _, opt := range opts {
		opt(&options)
	}

	return options
}

// Deprecated: use context aware `WithAsync()` for any async operations
func (ch *clickhouse) AsyncInsert(ctx context.Context, query string, wait bool, args ...any) error {
	conn, err := ch.acquire(ctx)
	if err != nil {
		return err
	}
	conn.getLogger().Debug("async insert", slog.String("sql", query), slog.Bool("wait", wait))
	if err := conn.asyncInsert(ctx, query, wait, args...); err != nil {
		ch.release(conn, err)
		return err
	}
	ch.release(conn, nil)
	return nil
}

func (ch *clickhouse) Ping(ctx context.Context) (err error) {
	conn, err := ch.acquire(ctx)
	if err != nil {
		return err
	}
	conn.getLogger().Debug("ping")
	if err := conn.ping(ctx); err != nil {
		ch.release(conn, err)
		return err
	}
	ch.release(conn, nil)
	return nil
}

func (ch *clickhouse) Stats() driver.Stats {
	return driver.Stats{
		Open:         len(ch.open),
		MaxOpenConns: cap(ch.open),

		Idle:         ch.idle.Len(),
		MaxIdleConns: ch.idle.Cap(),
	}
}

func (ch *clickhouse) dial(ctx context.Context) (conn nativeTransport, err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	connID := int(atomic.AddInt64(&ch.connID, 1))

	dialFunc := func(ctx context.Context, addr string, opt *Options) (DialResult, error) {
		var conn nativeTransport
		var err error
		switch opt.Protocol {
		case HTTP:
			conn, err = dialHttp(ctx, addr, connID, opt)
		default:
			conn, err = dial(ctx, addr, connID, opt)
		}

		return DialResult{conn}, err
	}

	dialStrategy := DefaultDialStrategy
	if ch.opt.DialStrategy != nil {
		dialStrategy = ch.opt.DialStrategy
	}

	result, err := dialStrategy(ctx, connID, ch.opt, dialFunc)
	if err != nil {
		return nil, err
	}
	return result.conn, nil
}

func DefaultDialStrategy(ctx context.Context, connID int, opt *Options, dial Dial) (r DialResult, err error) {
	for i := range opt.Addr {
		var num int
		switch opt.ConnOpenStrategy {
		case ConnOpenInOrder:
			num = i
		case ConnOpenRoundRobin:
			num = (connID + i) % len(opt.Addr)
		case ConnOpenRandom:
			random := rand.Int()
			num = (random + i) % len(opt.Addr)
		}

		if r, err = dial(ctx, opt.Addr[num], opt); err == nil {
			return r, nil
		}
	}

	if err == nil {
		err = ErrAcquireConnNoAddress
	}

	return r, err
}

func (ch *clickhouse) acquire(ctx context.Context) (conn nativeTransport, err error) {
	if ch.closed.Load() {
		return nil, ErrConnectionClosed
	}

	ctx, cancel := context.WithTimeoutCause(ctx, ch.opt.DialTimeout, ErrAcquireConnTimeout)
	defer cancel()

	// If context is already cancelled, just return without any work
	// done this way with single case with default. Otherwise if both ctx is cancelled and ch.open is ready,
	// Go would choose one of those at random, thus missing to return deterministically when context is cancelled
	// at this point in time.
	// Known pattern: https://go.dev/ref/spec#Select_statements
	select {
	case <-ctx.Done():
		return nil, context.Cause(ctx)
	default:
	}

	select {
	case ch.open <- struct{}{}:
	case <-ctx.Done():
		return nil, context.Cause(ctx)
	}

	conn, err = ch.idle.Get(ctx)
	if err != nil && !errors.Is(err, errQueueEmpty) {
		select {
		case <-ch.open:
		default:
		}
		return nil, err
	}

	if err == nil && conn != nil {
		if !conn.isBad() {
			conn.setReleased(false)
			conn.getLogger().Debug("connection acquired from pool")
			return conn, nil
		}

		conn.close()
	}

	if conn, err = ch.dial(ctx); err != nil {
		select {
		case <-ch.open:
		default:
		}

		return nil, err
	}

	conn.getLogger().Debug("new connection established")
	return conn, nil

}

func (ch *clickhouse) release(conn nativeTransport, err error) {
	if conn.isReleased() {
		return
	}
	conn.setReleased(true)

	if err != nil {
		conn.getLogger().Debug("connection released with error", slog.Any("error", err))
	} else {
		conn.getLogger().Debug("connection released to pool")
	}

	select {
	case <-ch.open:
	default:
	}

	if err != nil {
		conn.getLogger().Debug("connection closed due to error", slog.Any("error", err))
		conn.close()
		return
	} else if time.Since(conn.connectedAtTime()) >= ch.opt.ConnMaxLifetime {
		conn.getLogger().Debug("connection closed: lifetime expired",
			slog.Duration("age", time.Since(conn.connectedAtTime())),
			slog.Duration("max_lifetime", ch.opt.ConnMaxLifetime))
		conn.close()
		return
	}

	if ch.opt.FreeBufOnConnRelease {
		conn.getLogger().Debug("freeing connection buffer")
		conn.freeBuffer()
	}

	if ch.closed.Load() {
		conn.close()
		return
	}

	ch.idle.Put(conn)
}

func (ch *clickhouse) Close() (err error) {
	ch.closeOnce.Do(func() {
		err = ch.idle.Close()
		ch.closed.Store(true)
	})

	return
}
