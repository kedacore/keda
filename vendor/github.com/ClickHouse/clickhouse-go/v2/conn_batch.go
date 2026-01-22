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
	"os"
	"regexp"
	"slices"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/ClickHouse/clickhouse-go/v2/lib/column"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

var insertMatch = regexp.MustCompile(`(?i)(INSERT\s+INTO\s+[^( ]+(?:\s*\([^()]*(?:\([^()]*\)[^()]*)*\))?)(?:\s*VALUES)?`)
var columnMatch = regexp.MustCompile(`INSERT INTO .+\s\((?P<Columns>.+)\)$`)

func (c *connect) prepareBatch(ctx context.Context, query string, opts driver.PrepareBatchOptions, release func(*connect, error), acquire func(context.Context) (*connect, error)) (driver.Batch, error) {
	query, _, queryColumns, verr := extractNormalizedInsertQueryAndColumns(query)
	if verr != nil {
		return nil, verr
	}

	options := queryOptions(ctx)
	if deadline, ok := ctx.Deadline(); ok {
		c.conn.SetDeadline(deadline)
		defer c.conn.SetDeadline(time.Time{})
	}
	if err := c.sendQuery(query, &options); err != nil {
		release(c, err)
		return nil, err
	}
	var (
		onProcess  = options.onProcess()
		block, err = c.firstBlock(ctx, onProcess)
	)
	if err != nil {
		release(c, err)
		return nil, err
	}
	// resort batch to specified columns
	if err = block.SortColumns(queryColumns); err != nil {
		return nil, err
	}

	b := &batch{
		ctx:          ctx,
		query:        query,
		conn:         c,
		block:        block,
		released:     false,
		connRelease:  release,
		connAcquire:  acquire,
		onProcess:    onProcess,
		closeOnFlush: opts.CloseOnFlush,
	}

	if opts.ReleaseConnection {
		b.release(b.closeQuery())
	}

	return b, nil
}

type batch struct {
	err          error
	ctx          context.Context
	query        string
	conn         *connect
	sent         bool // sent signalize that batch is send to ClickHouse.
	released     bool // released signalize that conn was returned to pool and can't be used.
	closeOnFlush bool // closeOnFlush signalize that batch should close query and release conn when use Flush
	block        *proto.Block
	connRelease  func(*connect, error)
	connAcquire  func(context.Context) (*connect, error)
	onProcess    *onProcess
}

func (b *batch) release(err error) {
	if !b.released {
		b.released = true
		b.connRelease(b.conn, err)
	}
}

func (b *batch) Abort() error {
	defer func() {
		b.sent = true
		b.release(os.ErrProcessDone)
	}()
	if b.sent {
		return ErrBatchAlreadySent
	}
	return nil
}

func (b *batch) Append(v ...any) error {
	if b.sent {
		return ErrBatchAlreadySent
	}
	if b.err != nil {
		return b.err
	}

	if len(v) > 0 {
		if r, ok := v[0].(*rows); ok {
			return b.appendRowsBlocks(r)
		}
	}

	if err := b.block.Append(v...); err != nil {
		b.err = errors.Wrap(ErrBatchInvalid, err.Error())
		b.release(err)
		return err
	}
	return nil
}

// appendRowsBlocks is an experimental feature that allows rows blocks be appended directly to the batch.
// This API is not stable and may be changed in the future.
// See: tests/batch_block_test.go
func (b *batch) appendRowsBlocks(r *rows) error {
	var lastReadLock *proto.Block
	var blockNum int

	for r.Next() {
		if lastReadLock == nil { // make sure the first block is logged
			b.conn.debugf("[batch.appendRowsBlocks] blockNum = %d", blockNum)
		}

		// rows.Next() will read the next block from the server only if the current block is empty
		// only if new block is available we should flush the current block
		// the last block will be handled by the batch.Send() method
		if lastReadLock != nil && lastReadLock != r.block {
			if err := b.Flush(); err != nil {
				return err
			}
			blockNum++
			b.conn.debugf("[batch.appendRowsBlocks] blockNum = %d", blockNum)
		}

		b.block = r.block
		lastReadLock = r.block
	}

	return nil
}

func (b *batch) AppendStruct(v any) error {
	if b.err != nil {
		return b.err
	}
	values, err := b.conn.structMap.Map("AppendStruct", b.block.ColumnsNames(), v, false)
	if err != nil {
		return err
	}
	return b.Append(values...)
}

func (b *batch) IsSent() bool {
	return b.sent
}

func (b *batch) Column(idx int) driver.BatchColumn {
	if len(b.block.Columns) <= idx {
		err := &OpError{
			Op:  "batch.Column",
			Err: fmt.Errorf("invalid column index %d", idx),
		}

		b.release(err)

		return &batchColumn{
			err: err,
		}
	}
	return &batchColumn{
		batch:  b,
		column: b.block.Columns[idx],
		release: func(err error) {
			b.err = err
			b.release(err)
		},
	}
}

func (b *batch) Send() (err error) {
	stopCW := contextWatchdog(b.ctx, func() {
		// close TCP connection on context cancel. There is no other way simple way to interrupt underlying operations.
		// as verified in the test, this is safe to do and cleanups resources later on
		if b.conn != nil {
			_ = b.conn.conn.Close()
		}
	})

	defer func() {
		stopCW()
		b.sent = true
		b.release(err)
	}()
	if b.err != nil {
		return b.err
	}
	if b.sent || b.released {
		if err = b.resetConnection(); err != nil {
			return err
		}
	}
	if b.block.Rows() != 0 {
		if err = b.conn.sendData(b.block, ""); err != nil {
			// there might be an error caused by context cancellation
			// in this case we should return context error instead of net.OpError
			if ctxErr := b.ctx.Err(); ctxErr != nil {
				return ctxErr
			}

			return err
		}
	}
	if err = b.closeQuery(); err != nil {
		return err
	}
	return nil
}

func (b *batch) resetConnection() (err error) {
	// acquire a new conn
	if b.conn, err = b.connAcquire(b.ctx); err != nil {
		return err
	}

	defer func() {
		b.released = false
	}()

	options := queryOptions(b.ctx)
	if deadline, ok := b.ctx.Deadline(); ok {
		b.conn.conn.SetDeadline(deadline)
		defer b.conn.conn.SetDeadline(time.Time{})
	}

	if err = b.conn.sendQuery(b.query, &options); err != nil {
		b.release(err)
		return err
	}

	if _, err = b.conn.firstBlock(b.ctx, b.onProcess); err != nil {
		b.release(err)
		return err
	}

	return nil
}

func (b *batch) Flush() error {
	if b.sent {
		return ErrBatchAlreadySent
	}
	if b.err != nil {
		return b.err
	}
	if b.released {
		if err := b.resetConnection(); err != nil {
			return err
		}
	}
	if b.block.Rows() != 0 {
		if err := b.conn.sendData(b.block, ""); err != nil {
			// broken pipe/conn reset aren't generally recoverable on retry
			if errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNRESET) {
				b.release(err)
			}
			return err
		}
		if b.closeOnFlush {
			b.release(b.closeQuery())
		}
	}
	b.block.Reset()
	return nil
}

func (b *batch) Rows() int {
	return b.block.Rows()
}

func (b *batch) Columns() []column.Interface {
	return slices.Clone(b.block.Columns)
}

func (b *batch) closeQuery() error {
	if err := b.conn.sendData(&proto.Block{}, ""); err != nil {
		return err
	}

	if err := b.conn.process(b.ctx, b.onProcess); err != nil {
		return err
	}

	return nil
}

type batchColumn struct {
	err     error
	batch   driver.Batch
	column  column.Interface
	release func(error)
}

func (b *batchColumn) Append(v any) (err error) {
	if b.err != nil {
		return b.err
	}
	if b.batch.IsSent() {
		return ErrBatchAlreadySent
	}
	if _, err = b.column.Append(v); err != nil {
		b.release(err)
		return err
	}
	return nil
}

func (b *batchColumn) AppendRow(v any) (err error) {
	if b.err != nil {
		return b.err
	}
	if b.batch.IsSent() {
		return ErrBatchAlreadySent
	}
	if err = b.column.AppendRow(v); err != nil {
		b.release(err)
		return err
	}
	return nil
}

var (
	_ (driver.Batch)       = (*batch)(nil)
	_ (driver.BatchColumn) = (*batchColumn)(nil)
)
