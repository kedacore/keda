package kusto

// Reader provides a Reader object for Querying Kusto and turning it into Go objects and types.

import (
	"context"
	"encoding/json"
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

// A set of rows with values.
// Replace indicates whether the existing result set should be cleared and replaced with this row.
type Rows struct {
	Values  value.Values
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

	rows chan Rows

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

		rows:       make(chan Rows, 1000),
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
				for k, values := range sent.inRows {
					select {
					case <-r.ctx.Done():
					case r.rows <- Rows{Values: values, Replace: k == 0 && sent.inTableFragmentType == "DataReplace"}:
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
				if sent.inCompletion.HasErrors {
					errMsg, _ := json.MarshalIndent(sent.inCompletion.OneAPIErrors[0].Error, "", "  ")
					r.setError(fmt.Errorf("Query completed with error: %s", errMsg))
					sent.done()
					close(r.rows)
					return
				} else {
					r.mu.Lock()
					r.dsCompletion = sent.inCompletion
					sent.done()
					r.mu.Unlock()
				}
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

// Do calls f for every row returned by the query. If f returns a non-nil error,
// iteration stops.
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

// Stop is called to stop any further iteration. Always defer a Stop() call after
// receiving a RowIterator.
func (r *RowIterator) Stop() {
	r.cancel()
	return
}

// Next gets the next Row from the query. io.EOF is returned if there are no more entries in the output.
// Once Next() returns an error, all subsequent calls will return the same error.
func (r *RowIterator) Next() (*table.Row, error) {
	if err := r.getError(); err != nil {
		return nil, err
	}

	if r.mock != nil {
		if r.ctx.Err() != nil {
			return nil, r.ctx.Err()
		}
		return r.mock.nextRow()
	}

	select {
	case <-r.ctx.Done():
		return nil, r.ctx.Err()
	case kvs, ok := <-r.rows:
		if !ok {
			if err := r.getError(); err != nil {
				return nil, err
			}
			return nil, io.EOF
		}
		return &table.Row{ColumnTypes: r.columns, Values: kvs.Values, Op: r.op, Replace: kvs.Replace}, nil
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

// getNonPrimary will return a non-primary dataTable if it exists from the last query. The non-primary table kinds
// are defined as constants starting with TK<name>.
// Returns io.ErrUnexpectedEOF if not found. May not have all tables until RowIterator has reached io.EOF.
func (r *RowIterator) getNonPrimary(ctx context.Context, tableKind, tableName frames.TableKind) (v2.DataTable, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, table := range r.nonPrimary {
		if table.TableKind == tableKind && table.TableName == tableName {
			return table, nil
		}
	}
	return v2.DataTable{}, io.ErrUnexpectedEOF
}

func isTest() bool {
	if flag.Lookup("test.v") == nil {
		return false
	}
	return true
}
