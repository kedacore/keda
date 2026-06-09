package clickhouse

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/internal/circular"
)

var errQueueEmpty = errors.New("clickhouse: connection pool queue is empty")

type connPool struct {
	mu    sync.RWMutex
	conns *circular.Queue[nativeTransport]

	ticker   *time.Ticker
	finish   chan struct{}
	finished chan struct{}

	maxConnLifetime time.Duration
}

func newConnPool(lifetime time.Duration, capacity int) *connPool {
	pool := &connPool{
		conns:           circular.New[nativeTransport](capacity),
		ticker:          time.NewTicker(lifetime),
		finish:          make(chan struct{}),
		finished:        make(chan struct{}),
		maxConnLifetime: lifetime,
	}

	go pool.runDrainPool()

	return pool
}

func (i *connPool) Len() int {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.conns.Len()
}

func (i *connPool) Cap() int {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.conns.Cap()
}

func (i *connPool) Get(ctx context.Context) (nativeTransport, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	// check if pool was closed while we waited on the lock
	// return early if pool already closed
	// otherwise, pool wont close again while we hold lock
	// and so we continue
	if i.closed() {
		return nil, ErrConnectionClosed
	}

	// this loop continues until either:
	// a) the provided context is cancelled
	// b) the underlying circular queue is empty
	// c) it finds a non-expired connection
	for {
		if err := ctx.Err(); err != nil {
			// context has been cancelled
			return nil, context.Cause(ctx)
		}

		// Try to pull a connection
		conn, ok := i.conns.Pull()
		if !ok {
			return nil, errQueueEmpty // queue is empty
		}

		if !i.isExpired(conn) {
			return conn, nil
		}

		conn.close()
	}
}

func (i *connPool) Put(conn nativeTransport) {
	if i.isExpired(conn) || conn.isBad() {
		conn.close()
		return
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	if i.closed() {
		return
	}

	// Try to push the connection
	if !i.conns.Push(conn) {
		// Buffer is full, close the connection
		conn.close()
	}
}

func (i *connPool) Close() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.closed() {
		return nil
	}

	close(i.finish)

	<-i.finished

	// Drain all remaining connections from the pool
	i.drainPool()

	return nil
}

func (i *connPool) closed() bool {
	select {
	case <-i.finished:
		return true
	default:
		return false
	}
}

func (i *connPool) runDrainPool() {
	defer func() {
		i.ticker.Stop()
		close(i.finished)
	}()

	for {
		select {
		case <-i.ticker.C:
			i.mu.Lock()
			i.drainPool()
			i.mu.Unlock()
		case <-i.finish:
			return
		}
	}
}

// drainPool removes connections from the pool.
// If the pool is closed, it removes all connections.
// Otherwise, it only removes expired connections.
// Must be called with i.mu held.
func (i *connPool) drainPool() {
	if i.closed() {
		// Close all connections
		for conn := range i.conns.Clear() {
			conn.close()
		}
		return
	}

	// Remove only expired connections
	for conn := range i.conns.DeleteFunc(func(conn nativeTransport) bool {
		return i.isExpired(conn)
	}) {
		conn.close()
	}
}

func (i *connPool) isExpired(conn nativeTransport) bool {
	return !time.Now().Before(i.expires(conn))
}

func (i *connPool) expires(conn nativeTransport) time.Time {
	return conn.connectedAtTime().Add(i.maxConnLifetime)
}
