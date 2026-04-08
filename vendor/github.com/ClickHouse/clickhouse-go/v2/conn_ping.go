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
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

// Connection::ping
// https://github.com/ClickHouse/ClickHouse/blob/master/src/Client/Connection.cpp
func (c *connect) ping(ctx context.Context) (err error) {
	// set a read deadline - alternative to context.Read operation will fail if no data is received after deadline.
	c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	defer c.conn.SetReadDeadline(time.Time{})
	// context level deadlines override any read deadline
	if deadline, ok := ctx.Deadline(); ok {
		c.conn.SetDeadline(deadline)
		defer c.conn.SetDeadline(time.Time{})
	}
	c.debugf("[ping] -> ping")
	c.buffer.PutByte(proto.ClientPing)
	if err := c.flush(); err != nil {
		return err
	}

	var packet byte
	for {
		if packet, err = c.reader.ReadByte(); err != nil {
			return err
		}
		switch packet {
		case proto.ServerException:
			return c.exception()
		case proto.ServerProgress:
			if _, err = c.progress(); err != nil {
				return err
			}
		case proto.ServerPong:
			c.debugf("[ping] <- pong")
			return nil
		default:
			return fmt.Errorf("unexpected packet %d", packet)
		}
	}
}
