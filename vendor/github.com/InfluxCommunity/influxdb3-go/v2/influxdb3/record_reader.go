package influxdb3

import (
	"context"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/flight"
)

// RecordReader is an interface for reading Arrow record batches.
type RecordReader interface {
	Next() bool
	RecordBatch() arrow.RecordBatch
	Err() error
	Schema() *arrow.Schema
}

// cancelingRecordReader is a RecordReader that cancels the context when done.
type cancelingRecordReader struct {
	reader *flight.Reader
	cancel context.CancelFunc
}

func (cr *cancelingRecordReader) Next() bool {
	n := cr.reader.Next()
	if !n && cr.cancel != nil {
		cr.cancel()
		cr.cancel = nil
	}
	return n
}

func (cr *cancelingRecordReader) RecordBatch() arrow.RecordBatch { //nolint:ireturn
	return cr.reader.RecordBatch()
}
func (cr *cancelingRecordReader) Err() error {
	return cr.reader.Err()
}
func (cr *cancelingRecordReader) Schema() *arrow.Schema {
	return cr.reader.Schema()
}

func (cr *cancelingRecordReader) Reader() *flight.Reader {
	return cr.reader
}
