package kusto

// Reader provides a Reader object for Querying Kusto and turning it into Go objects and types.

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/Azure/azure-kusto-go/kusto/data/errors"
	"github.com/Azure/azure-kusto-go/kusto/data/table"
	"github.com/Azure/azure-kusto-go/kusto/data/value"
	"github.com/Azure/azure-kusto-go/kusto/internal/frames"
	v2 "github.com/Azure/azure-kusto-go/kusto/internal/frames/v2"
)

// send allows us to send a table on a channel and know when everything has been written.
type send struct {
	inColumns           table.Columns
	inRows              []value.Values
	inRowErrors         []errors.Error
	inTableFragmentType string
	inProgress          v2.TableProgress
	inNonPrimary        v2.DataTable
	inCompletion        v2.DataSetCompletion
	inErr               error

	wg *sync.WaitGroup
}

func (s send) done() {
	if s.wg != nil {
		s.wg.Done()
	}
}

// Row is a row of data from Kusto, or an error.
// Replace indicates whether the existing result set should be cleared and replaced with this row.
type Row struct {
	Values  value.Values
	Error   *errors.Error
	Replace bool
}

// RowIterator is used to iterate over the returned Row objects returned by Kusto.
type RowIterator struct {
	op     errors.Op
	ctx    context.Context
	cancel context.CancelFunc

	// RequestHeader is the http.Header sent in the request to the server.
	RequestHeader http.Header
	// ResponseHeader is the http.header sent in the response from the server.
	ResponseHeader http.Header

	// The following channels represent input entering the RowIterator.
	inColumns    chan send
	inRows       chan send
	inProgress   chan send
	inNonPrimary chan send
	inCompletion chan send
	inErr        chan send

	rows chan Row

	mu sync.Mutex

	// progressive indicates if we are receiving a progressive stream or not.
	progressive bool
	// progress provides a progress indicator if the frames are progressive.
	progress v2.TableProgress
	// nonPrimary contains dataTables that are not the primary table.
	nonPrimary map[frames.TableKind]v2.DataTable
	// dsCompletion is the completion frame for a non-progressive query.
	dsCompletion v2.DataSetCompletion

	columns table.Columns

	// error holds an error that was encountered. Once this is set, all calls on Rowiterator will
	// just return the error here.
	error error

	// mock hold our MockRows data if it has been provided for tests.
	mock *MockRows
}

func newRowIterator(ctx context.Context, cancel context.CancelFunc, execResp execResp, header v2.DataSetHeader, op errors.Op) (*RowIterator, chan struct{}) {
	ri := &RowIterator{
		RequestHeader:  execResp.reqHeader,
		ResponseHeader: execResp.respHeader,

		op:           op,
		ctx:          ctx,
		cancel:       cancel,
		progressive:  header.IsProgressive,
		inColumns:    make(chan send, 1),
		inRows:       make(chan send, 100),
		inProgress:   make(chan send, 1),
		inNonPrimary: make(chan send, 1),
		inCompletion: make(chan send, 1),
		inErr:        make(chan send),

		rows:       make(chan Row, 1000),
		nonPrimary: make(map[frames.TableKind]v2.DataTable),
	}
	columnsReady := ri.start()
	return ri, columnsReady
}

func (r *RowIterator) start() chan struct{} {
	done := make(chan struct{})
	once := sync.Once{}
	closeDone := func() {
		once.Do(func() { close(done) })
	}

	go func() {
		defer closeDone() // Catchall

		for {
			select {
			case <-r.ctx.Done():
			case sent := <-r.inColumns:
				r.columns = sent.inColumns
				sent.done()
				closeDone()
			case sent, ok := <-r.inRows:
				if !ok {
					close(r.rows)
					return
				}
				if sent.inRows != nil {
					for k, values := range sent.inRows {
						select {
						case <-r.ctx.Done():
						case r.rows <- Row{Values: values, Replace: k == 0 && sent.inTableFragmentType == "DataReplace"}:
						}
					}
				}

				if sent.inRowErrors != nil {
					for _, e := range sent.inRowErrors {
						e := e // capture so we can send reference
						select {
						case <-r.ctx.Done():
						case r.rows <- Row{Error: &e}:
						}
					}
				}
				sent.done()
			case sent := <-r.inProgress:
				r.mu.Lock()
				r.progress = sent.inProgress
				sent.done()
				r.mu.Unlock()
			case sent := <-r.inNonPrimary:
				r.mu.Lock()
				r.nonPrimary[sent.inNonPrimary.TableKind] = sent.inNonPrimary
				sent.done()
				r.mu.Unlock()
			case sent := <-r.inCompletion:
				r.mu.Lock()
				r.dsCompletion = sent.inCompletion
				sent.done()
				r.mu.Unlock()
			case sent := <-r.inErr:
				r.setError(sent.inErr)
				sent.done()
				close(r.rows)
				return
			}
		}
	}()
	return done
}

// Mock is used to tell the RowIterator to return specific data for tests. This is useful when building
// fakes of the client's Query() call for hermetic tests. This can only be called in a test or it will panic.
func (r *RowIterator) Mock(m *MockRows) error {
	if !isTest() {
		panic("cannot call Mock outside a test")
	}
	if r.mock != nil {
		return fmt.Errorf("RowIterator already has mock data")
	}
	r.ctx, r.cancel = context.WithCancel(context.Background())

	r.mock = m
	return nil
}

// Deprecated: Use DoOnRowOrError() instead for more robust error handling. In a future version, this will be removed, and NextRowOrError will replace it.
// Do calls f for every row returned by the query. If f returns a non-nil error, iteration stops.
// This method will fail on errors inline within the rows, even though they could potentially be recovered and more data might be available.
// This behavior is to keep the interface compatible.
func (r *RowIterator) Do(f func(r *table.Row) error) error {
	for {
		row, err := r.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if err := f(row); err != nil {
			return err
		}
	}
}

// DoOnRowOrError calls f for every row returned by the query. If errors occur inline within the rows, they are passed to f.
// Other errors will stop the iteration and be returned.
// If f returns a non-nil error, iteration stops.
func (r *RowIterator) DoOnRowOrError(f func(r *table.Row, e *errors.Error) error) error {
	for {
		row, inlineErr, err := r.NextRowOrError()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if err := f(row, inlineErr); err != nil {
			return err
		}
	}
}

// Stop is called to stop any further iteration. Always defer a Stop() call after
// receiving a RowIterator.
func (r *RowIterator) Stop() {
	r.cancel()
}

// Deprecated: Use NextRowOrError() instead for more robust error handling. In a future version, this will be removed, and NextRowOrError will replace it.
// Next gets the next Row from the query. io.EOF is returned if there are no more entries in the output.
// This method will fail on errors inline within the rows, even though they could potentially be recovered and more data might be available.
// Once Next() returns an error, all subsequent calls will return the same error.
func (r *RowIterator) Next() (row *table.Row, finalError error) {
	row, inlineErr, err := r.NextRowOrError()
	if err != nil {
		return nil, err
	}
	if inlineErr != nil {
		r.setError(inlineErr)
		return nil, inlineErr
	}
	return row, err
}

// NextRowOrError gets the next Row or service-side error from the query.
// On partial success, inlineError will be set.
// Once finalError returns non-nil, all subsequent calls will return the same error.
// finalError will be set to io.EOF is when frame parsing completed with success or partial success (data + errors).
// if finalError is not io.EOF, reading the frame has resulted in a failure state (no data is expected).
func (r *RowIterator) NextRowOrError() (row *table.Row, inlineError *errors.Error, finalError error) {
	if err := r.getError(); err != nil {
		return nil, nil, err
	}

	if r.mock != nil {
		if r.ctx.Err() != nil {
			return nil, nil, r.ctx.Err()
		}
		nextRow, err := r.mock.nextRow()
		if err != nil {
			return nil, nil, err
		}
		return nextRow, nil, nil
	}

	select {
	case <-r.ctx.Done():
		return nil, nil, r.ctx.Err()
	case kvs, ok := <-r.rows:
		if !ok {
			if err := r.getError(); err != nil {
				return nil, nil, err
			}
			return nil, nil, io.EOF
		}
		if kvs.Error != nil {
			return nil, kvs.Error, nil
		}
		return &table.Row{ColumnTypes: r.columns, Values: kvs.Values, Op: r.op, Replace: kvs.Replace}, nil, nil
	}
}

func (r *RowIterator) getError() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.error
}

func (r *RowIterator) setError(e error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.error = e
}

// Progress returns the progress of the query, 0-100%. This is only valid on Progressive data returns.
func (r *RowIterator) Progress() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.progress.TableProgress
}

// Progressive indicates if the RowIterator is unpacking progressive (streaming) frames.
func (r *RowIterator) Progressive() bool {
	return r.progressive
}

// GetNonPrimary will return a non-primary dataTable if it exists from the last query. The non-primary table and common names are defined under the frames.TableKind enum.
// Returns io.ErrUnexpectedEOF if not found. May not have all tables until RowIterator has reached io.EOF.
func (r *RowIterator) GetNonPrimary(tableKind, tableName frames.TableKind) (v2.DataTable, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, npTable := range r.nonPrimary {
		if npTable.TableKind == tableKind && npTable.TableName == tableName {
			return npTable, nil
		}
	}
	return v2.DataTable{}, io.ErrUnexpectedEOF
}

// GetExtendedProperties will return the extended properties' table from the iterator, if it exists.
// Returns io.ErrUnexpectedEOF if not found. May not have all tables until RowIterator has reached io.EOF.
func (r *RowIterator) GetExtendedProperties() (v2.DataTable, error) {
	return r.GetNonPrimary(frames.QueryProperties, frames.ExtendedProperties)
}

// GetQueryCompletionInformation will return the query completion information table from the iterator, if it exists.
// Returns io.ErrUnexpectedEOF if not found. May not have all tables until RowIterator has reached io.EOF.
func (r *RowIterator) GetQueryCompletionInformation() (v2.DataTable, error) {
	return r.GetNonPrimary(frames.QueryCompletionInformation, frames.QueryCompletionInformation)
}

func isTest() bool {
	return flag.Lookup("test.v") != nil
}
