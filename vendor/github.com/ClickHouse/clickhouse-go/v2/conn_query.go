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
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

func (c *connect) query(ctx context.Context, release func(*connect, error), query string, args ...any) (*rows, error) {
	var (
		options                    = queryOptions(ctx)
		onProcess                  = options.onProcess()
		queryParamsProtocolSupport = c.revision >= proto.DBMS_MIN_PROTOCOL_VERSION_WITH_PARAMETERS
		body, err                  = bindQueryOrAppendParameters(queryParamsProtocolSupport, &options, query, c.server.Timezone, args...)
	)

	if err != nil {
		c.debugf("[bindQuery] error: %v", err)
		release(c, err)
		return nil, err
	}

	// set a read deadline - alternative to context.Read operation will fail if no data is received after deadline.
	c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	defer c.conn.SetReadDeadline(time.Time{})
	// context level deadlines override any read deadline
	if deadline, ok := ctx.Deadline(); ok {
		c.conn.SetDeadline(deadline)
		defer c.conn.SetDeadline(time.Time{})
	}

	if err = c.sendQuery(body, &options); err != nil {
		release(c, err)
		return nil, err
	}

	init, err := c.firstBlock(ctx, onProcess)

	if err != nil {
		c.debugf("[query] first block error: %v", err)
		release(c, err)
		return nil, err
	}
	bufferSize := c.blockBufferSize
	if options.blockBufferSize > 0 {
		// allow block buffer sze to be overridden per query
		bufferSize = options.blockBufferSize
	}
	var (
		errors = make(chan error, 1)
		stream = make(chan *proto.Block, bufferSize)
	)

	go func() {
		onProcess.data = func(b *proto.Block) {
			stream <- b
		}
		err := c.process(ctx, onProcess)
		if err != nil {
			c.debugf("[query] process error: %v", err)
			errors <- err
		}
		close(stream)
		close(errors)
		release(c, err)
	}()

	return &rows{
		block:     init,
		stream:    stream,
		errors:    errors,
		columns:   init.ColumnsNames(),
		structMap: c.structMap,
	}, nil
}

func (c *connect) queryRow(ctx context.Context, release func(*connect, error), query string, args ...any) *row {
	rows, err := c.query(ctx, release, query, args...)
	if err != nil {
		return &row{
			err: err,
		}
	}
	return &row{
		rows: rows,
	}
}
