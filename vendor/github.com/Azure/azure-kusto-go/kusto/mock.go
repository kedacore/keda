package kusto

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sync"

	"github.com/Azure/azure-kusto-go/kusto/data/errors"
	"github.com/Azure/azure-kusto-go/kusto/data/table"
	"github.com/Azure/azure-kusto-go/kusto/data/value"
	"github.com/Azure/azure-kusto-go/kusto/internal/frames"
	v1 "github.com/Azure/azure-kusto-go/kusto/internal/frames/v1"
)

type columnData struct {
	column   table.Column
	position int
}

type columnMap map[string]columnData

func newColumnMap(cols table.Columns) columnMap {
	m := make(columnMap, len(cols))
	for i, col := range cols {
		m[col.Name] = columnData{column: col, position: i}
	}
	return m
}

// MockRows provides the abilty to provide mocked Row data that can be played back from a RowIterator.
// This allows for creating hermetic tests from mock data or creating mock data from a real data fetch.
type MockRows struct {
	columns table.Columns
	// playback is the list of data we are going to return to the RowIterator.
	// Note: In the future, we may want to  allow adding other table data playback.
	playback []interface{}
	position int
	err      error
}

// NewMockRows is the constructor for MockRows.
func NewMockRows(columns table.Columns) (*MockRows, error) {
	if err := columns.Validate(); err != nil {
		return nil, err
	}

	return &MockRows{columns: columns}, nil
}

func (m *MockRows) nextRow() (*table.Row, error) {
	if m.err != nil {
		return nil, m.err
	}

	if m.position > len(m.playback)-1 {
		return nil, io.EOF
	}

	defer func() { m.position++ }()

	v := m.playback[m.position]
	switch t := v.(type) {
	case value.Values:
		return &table.Row{
			ColumnTypes: m.columns,
			Values:      value.Values(t),
			Op:          errors.OpQuery,
		}, nil
	case error:
		m.err = t
		return nil, t
	default:
		panic(fmt.Sprintf("bug, received a playback type we don't support: %T", v))
	}
}

// Row adds Row data that will be replayed in a RowIterator.
func (m *MockRows) Row(row value.Values) error {
	if len(row) == 0 {
		return fmt.Errorf("cannot add an empty value.Values")
	}

	if err := colToValueCheck(m.columns, row); err != nil {
		return err
	}

	m.playback = append(m.playback, row)

	return nil
}

// Struct adds Row data that will be replayed in a RowIterator by parsing the passed *struct into
// value.Values.
func (m *MockRows) Struct(p interface{}) error {
	// Check if p is a pointer to a struct.
	if t := reflect.TypeOf(p); t == nil || t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("type %T is not a pointer to a struct", p)
	}

	row, err := structToKustoValues(m.columns, p)
	if err != nil {
		return err
	}

	return m.Row(row)
}

// Error adds an error into the result stream. Nothing else added to this stream will matter
// once this is called.
func (m *MockRows) Error(err error) error {
	if err == nil {
		return fmt.Errorf("cannot add a nil error")
	}
	m.playback = append(m.playback, err)
	return nil
}

type mockConn struct {
}

func (m mockConn) queryToJson(ctx context.Context, db string, query Stmt, options *queryOptions) (string, error) {
	return "[]]", nil
}

func (m mockConn) Close() error {
	return nil
}

func (m mockConn) query(_ context.Context, _ string, _ Stmt, _ *queryOptions) (execResp, error) {
	return execResp{}, nil
}

func (m mockConn) mgmt(_ context.Context, _ string, _ Stmt, _ *mgmtOptions) (execResp, error) {
	framesCh := make(chan frames.Frame, 100)
	framesCh <- v1.DataTable{}
	close(framesCh)
	return execResp{
		reqHeader:  nil,
		respHeader: nil,
		frameCh:    framesCh,
	}, nil
}

func NewMockClient() *Client {

	kcsb := NewConnectionStringBuilder("https://sdkse2etest.eastus.kusto.windows.net")
	tkp, _ := kcsb.newTokenProvider()

	return &Client{
		conn:       mockConn{},
		ingestConn: mockConn{},
		endpoint:   "https://sdkse2etest.eastus.kusto.windows.net",
		auth:       Authorization{TokenProvider: tkp},
		mgmtConnMu: sync.Mutex{},
		http:       &http.Client{},
	}
}
