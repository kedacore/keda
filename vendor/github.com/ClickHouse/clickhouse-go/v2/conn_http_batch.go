// Licensed to ClickHouse, Inc. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. ClickHouse, Inc. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package clickhouse

import (
	"context"
	"fmt"
	"io"
	"slices"

	"github.com/ClickHouse/clickhouse-go/v2/lib/column"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

// release is ignored, because http used by std with empty release function.
// Also opts ignored because all options unused in http batch.
func (h *httpConnect) prepareBatch(ctx context.Context, query string, opts driver.PrepareBatchOptions, release func(*connect, error), acquire func(context.Context) (*connect, error)) (driver.Batch, error) {
	query, tableName, queryColumns, err := extractNormalizedInsertQueryAndColumns(query)
	if err != nil {
		return nil, err
	}

	describeTableQuery := fmt.Sprintf("DESCRIBE TABLE %s", tableName)
	r, err := h.query(ctx, release, describeTableQuery)
	if err != nil {
		return nil, err
	}

	block := &proto.Block{}

	columns := make(map[string]string)
	var colNames []string
	for r.Next() {
		var (
			colName      string
			colType      string
			default_type string
			ignore       string
		)

		if err = r.Scan(&colName, &colType, &default_type, &ignore, &ignore, &ignore, &ignore); err != nil {
			return nil, err
		}
		// these column types cannot be specified in INSERT queries
		if default_type == "MATERIALIZED" || default_type == "ALIAS" {
			continue
		}
		colNames = append(colNames, colName)
		columns[colName] = colType
	}

	switch len(queryColumns) {
	case 0:
		for _, colName := range colNames {
			if err = block.AddColumn(colName, column.Type(columns[colName])); err != nil {
				return nil, err
			}
		}
	default:
		// user has requested specific columns so only include these
		for _, colName := range queryColumns {
			if colType, ok := columns[colName]; ok {
				if err = block.AddColumn(colName, column.Type(colType)); err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("column %s is not present in the table %s", colName, tableName)
			}
		}
	}

	return &httpBatch{
		ctx:       ctx,
		conn:      h,
		structMap: &structMap{},
		block:     block,
		query:     query,
	}, nil
}

type httpBatch struct {
	query     string
	err       error
	ctx       context.Context
	conn      *httpConnect
	structMap *structMap
	sent      bool
	block     *proto.Block
}

// Flush TODO: noop on http currently - requires streaming to be implemented
func (b *httpBatch) Flush() error {
	return nil
}

func (b *httpBatch) Abort() error {
	defer func() {
		b.sent = true
	}()
	if b.sent {
		return ErrBatchAlreadySent
	}
	return nil
}

func (b *httpBatch) Append(v ...any) error {
	if b.sent {
		return ErrBatchAlreadySent
	}
	if err := b.block.Append(v...); err != nil {
		return err
	}
	return nil
}

func (b *httpBatch) AppendStruct(v any) error {
	values, err := b.structMap.Map("AppendStruct", b.block.ColumnsNames(), v, false)
	if err != nil {
		return err
	}
	return b.Append(values...)
}

func (b *httpBatch) Column(idx int) driver.BatchColumn {
	if len(b.block.Columns) <= idx {
		return &batchColumn{
			err: &OpError{
				Op:  "batch.Column",
				Err: fmt.Errorf("invalid column index %d", idx),
			},
		}
	}
	return &batchColumn{
		batch:  b,
		column: b.block.Columns[idx],
		release: func(err error) {
			b.err = err
		},
	}
}

func (b *httpBatch) IsSent() bool {
	return b.sent
}

func (b *httpBatch) Send() (err error) {
	defer func() {
		b.sent = true
	}()
	if b.sent {
		return ErrBatchAlreadySent
	}
	if b.err != nil {
		return b.err
	}
	options := queryOptions(b.ctx)

	headers := make(map[string]string)

	r, pw := io.Pipe()
	crw := b.conn.compressionPool.Get()
	w := crw.reset(pw)

	defer b.conn.compressionPool.Put(crw)

	switch b.conn.compression {
	case CompressionGZIP, CompressionDeflate, CompressionBrotli:
		headers["Content-Encoding"] = b.conn.compression.String()
	case CompressionZSTD, CompressionLZ4:
		options.settings["decompress"] = "1"
		options.settings["compress"] = "1"
	}

	go func() {
		var err error = nil
		defer pw.CloseWithError(err)
		defer w.Close()
		b.conn.buffer.Reset()
		if b.block.Rows() != 0 {
			if err = b.conn.writeData(b.block); err != nil {
				return
			}
		}
		if err = b.conn.writeData(&proto.Block{}); err != nil {
			return
		}
		if _, err = w.Write(b.conn.buffer.Buf); err != nil {
			return
		}
	}()

	options.settings["query"] = b.query
	headers["Content-Type"] = "application/octet-stream"
	for k, v := range b.conn.headers {
		headers[k] = v
	}
	res, err := b.conn.sendStreamQuery(b.ctx, r, &options, headers)

	if res != nil {
		defer res.Body.Close()
		// we don't care about result, so just discard it to reuse connection
		_, _ = io.Copy(io.Discard, res.Body)
	}

	return err
}

func (b *httpBatch) Rows() int {
	return b.block.Rows()
}

func (b *httpBatch) Columns() []column.Interface {
	return slices.Clone(b.block.Columns)
}

var _ driver.Batch = (*httpBatch)(nil)
