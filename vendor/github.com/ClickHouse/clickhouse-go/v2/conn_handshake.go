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
	_ "embed"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

func (c *connect) handshake(database, username, password string) error {
	defer c.buffer.Reset()
	c.debugf("[handshake] -> %s", proto.ClientHandshake{})
	// set a read deadline - alternative to context.Read operation will fail if no data is received after deadline.
	c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	defer c.conn.SetReadDeadline(time.Time{})
	// context level deadlines override any read deadline
	c.conn.SetDeadline(time.Now().Add(c.opt.DialTimeout))
	defer c.conn.SetDeadline(time.Time{})
	{
		c.buffer.PutByte(proto.ClientHello)
		handshake := &proto.ClientHandshake{
			ProtocolVersion: ClientTCPProtocolVersion,
			ClientName:      c.opt.ClientInfo.String(),
			ClientVersion:   proto.Version{ClientVersionMajor, ClientVersionMinor, ClientVersionPatch}, //nolint:govet
		}
		handshake.Encode(c.buffer)
		{
			c.buffer.PutString(database)
			c.buffer.PutString(username)
			c.buffer.PutString(password)
		}
		if err := c.flush(); err != nil {
			return err
		}
	}
	{
		packet, err := c.reader.ReadByte()
		if err != nil {
			return err
		}
		switch packet {
		case proto.ServerException:
			return c.exception()
		case proto.ServerHello:
			if err := c.server.Decode(c.reader); err != nil {
				return err
			}
		case proto.ServerEndOfStream:
			c.debugf("[handshake] <- end of stream")
			return nil
		default:
			return fmt.Errorf("[handshake] unexpected packet [%d] from server", packet)
		}
	}
	if c.server.Revision < proto.DBMS_MIN_REVISION_WITH_CLIENT_INFO {
		return ErrUnsupportedServerRevision
	}

	if c.revision > c.server.Revision {
		c.revision = c.server.Revision
		c.debugf("[handshake] downgrade client proto")
	}
	c.debugf("[handshake] <- %s", c.server)
	return nil
}

func (c *connect) sendAddendum() error {
	if c.revision >= proto.DBMS_MIN_PROTOCOL_VERSION_WITH_QUOTA_KEY {
		c.buffer.PutString("") // todo quota key support
	}

	return c.flush()
}
