package kivik

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"sync"

	"github.com/go-kivik/kivik/v3/driver"
)

type iterator interface {
	Next(interface{}) error
	Close() error
}

type iter struct {
	feed iterator

	mu      sync.RWMutex
	ready   bool // Set to true once Next() has been called
	closed  bool
	lasterr error // non-nil only if closed is true
	eoq     bool

	cancel func() // cancel function to exit context goroutine when iterator is closed

	curVal interface{}
}

func (i *iter) rlock() (unlock func(), err error) {
	i.mu.RLock()
	if i.closed {
		i.mu.RUnlock()
		return nil, &Error{HTTPStatus: http.StatusBadRequest, Message: "kivik: Iterator is closed"}
	}
	if !i.ready {
		i.mu.RUnlock()
		return nil, &Error{HTTPStatus: http.StatusBadRequest, Message: "kivik: Iterator access before calling Next"}
	}
	return func() { i.mu.RUnlock() }, nil
}

// newIterator instantiates a new iterator.
//
// ctx is a possibly-cancellable context
// zeroValue is an empty instance of the data type this iterator iterates over
// feed is the iterator interface, which typically wraps a driver.X iterator
func newIterator(ctx context.Context, feed iterator, zeroValue interface{}) *iter {
	i := &iter{
		feed:   feed,
		curVal: zeroValue,
	}
	ctx, i.cancel = context.WithCancel(ctx)
	go i.awaitDone(ctx)
	return i
}

// awaitDone blocks until the rows are closed or the context is cancelled, then closes the iterator if it's still open.
func (i *iter) awaitDone(ctx context.Context) {
	<-ctx.Done()
	_ = i.close(ctx.Err())
}

// Next prepares the next iterator result value for reading. It returns true on
// success, or false if there is no next result or an error occurs while
// preparing it. Err should be consulted to distinguish between the two.
func (i *iter) Next() bool {
	doClose, ok := i.next()
	if doClose {
		_ = i.Close()
	}
	return ok
}

func (i *iter) next() (doClose, ok bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	if i.closed {
		return false, false
	}
	i.ready = true
	i.eoq = false
	err := i.feed.Next(i.curVal)
	if err == driver.EOQ {
		i.eoq = true
		err = nil
	}
	i.lasterr = err
	if i.lasterr != nil {
		return true, false
	}
	return false, true
}

// EOQ returns true if the iterator has reached the end of a query in a
// multi-query query. When EOQ is true, the row data will not have been
// updated. It is common to simply `continue` in case of EOQ, unless you care
// about the per-query metadata, such as offset, total rows, etc.
func (i *iter) EOQ() bool {
	return i.eoq
}

// Close closes the Iterator, preventing further enumeration, and freeing any
// resources (such as the http request body) of the underlying feed. If Next is
// called and there are no further results, Iterator is closed automatically and
// it will suffice to check the result of Err. Close is idempotent and does not
// affect the result of Err.
func (i *iter) Close() error {
	return i.close(nil)
}

func (i *iter) close(err error) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closed {
		return nil
	}
	i.closed = true

	if i.lasterr == nil {
		i.lasterr = err
	}

	err = i.feed.Close()

	if i.cancel != nil {
		i.cancel()
	}

	return err
}

// Err returns the error, if any, that was encountered during iteration. Err
// may be called after an explicit or implicit Close.
func (i *iter) Err() error {
	i.mu.RLock()
	defer i.mu.RUnlock()
	if i.lasterr == io.EOF {
		return nil
	}
	return i.lasterr
}

func scan(dest interface{}, val json.RawMessage) error {
	if reflect.TypeOf(dest).Kind() != reflect.Ptr {
		return errNonPtr
	}
	switch d := dest.(type) {
	case *[]byte:
		if d == nil {
			return errNilPtr
		}
		tgt := make([]byte, len(val))
		copy(tgt, val)
		*d = tgt
		return nil
	case *json.RawMessage:
		if d == nil {
			return errNilPtr
		}
		*d = val
		return nil
	}
	if err := json.Unmarshal(val, dest); err != nil {
		return err
	}
	return nil
}
