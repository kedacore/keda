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
	"database/sql"
	"io"

	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

type rows struct {
	err       error
	row       int
	block     *proto.Block
	totals    *proto.Block
	errors    chan error
	stream    chan *proto.Block
	columns   []string
	structMap *structMap
}

func (r *rows) Next() (result bool) {
	defer func() {
		if !result {
			r.Close()
		}
	}()
	if r.block == nil {
		return false
	}
next:
	if r.row >= r.block.Rows() {
		if r.stream == nil {
			return false
		}
		select {
		case err := <-r.errors:
			if err != nil {
				r.err = err
				return false
			}
		case block := <-r.stream:
			if block == nil {
				return false
			}
			if block.Packet == proto.ServerTotals {
				r.row, r.block, r.totals = 0, nil, block
				return false
			}
			r.row, r.block = 0, block
		}
		goto next
	}
	r.row++
	return r.row <= r.block.Rows()
}

func (r *rows) Scan(dest ...any) error {
	if r.block == nil || (r.row == 0 && r.row >= r.block.Rows()) { // call without next when result is empty
		return io.EOF
	}
	return scan(r.block, r.row, dest...)
}

func (r *rows) ScanStruct(dest any) error {
	values, err := r.structMap.Map("ScanStruct", r.columns, dest, true)
	if err != nil {
		return err
	}
	return r.Scan(values...)
}

func (r *rows) Totals(dest ...any) error {
	if r.totals == nil {
		return sql.ErrNoRows
	}
	return scan(r.totals, 1, dest...)
}

func (r *rows) Columns() []string {
	return r.columns
}

func (r *rows) Close() error {
	if r.errors == nil && r.stream == nil {
		return r.err
	}

	if r.errors == nil {
		for range r.stream {
		}
		return nil
	}

	if r.stream == nil {
		for err := range r.errors {
			r.err = err
		}
		return r.err
	}

	errorsClosed := false
	streamClosed := false
	for {
		select {
		case _, ok := <-r.stream:
			if !ok {
				streamClosed = true
			}
		case err, ok := <-r.errors:
			if err != nil {
				r.err = err
			}
			if !ok {
				errorsClosed = true
			}
		}

		if errorsClosed && streamClosed {
			return r.err
		}
	}
}

func (r *rows) Err() error {
	return r.err
}

type row struct {
	err  error
	rows *rows
}

func (r *row) Err() error {
	return r.err
}

func (r *row) ScanStruct(dest any) error {
	if r.err != nil {
		return r.err
	}
	values, err := r.rows.structMap.Map("ScanStruct", r.rows.columns, dest, true)
	if err != nil {
		return err
	}
	return r.Scan(values...)
}

func (r *row) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if !r.rows.Next() {
		r.rows.Close()
		if err := r.rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	if err := r.rows.Scan(dest...); err != nil {
		return err
	}
	return r.rows.Close()
}
