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
	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

// Connection::sendQuery
// https://github.com/ClickHouse/ClickHouse/blob/master/src/Client/Connection.cpp
func (c *connect) sendQuery(body string, o *QueryOptions) error {
	c.debugf("[send query] compression=%q %s", c.compression, body)
	c.buffer.PutByte(proto.ClientQuery)
	q := proto.Query{
		ClientTCPProtocolVersion: ClientTCPProtocolVersion,
		ClientName:               c.opt.ClientInfo.String(),
		ClientVersion:            proto.Version{ClientVersionMajor, ClientVersionMinor, ClientVersionPatch}, //nolint:govet
		ID:                       o.queryID,
		Body:                     body,
		Span:                     o.span,
		QuotaKey:                 o.quotaKey,
		Compression:              c.compression != CompressionNone,
		InitialAddress:           c.conn.LocalAddr().String(),
		Settings:                 c.settings(o.settings),
		Parameters:               parametersToProtoParameters(o.parameters),
	}
	if err := q.Encode(c.buffer, c.revision); err != nil {
		return err
	}
	for _, table := range o.external {
		if err := c.sendData(table.Block(), table.Name()); err != nil {
			return err
		}
	}
	if err := c.sendData(&proto.Block{}, ""); err != nil {
		return err
	}
	return c.flush()
}

func parametersToProtoParameters(parameters Parameters) (s proto.Parameters) {
	for k, v := range parameters {
		s = append(s, proto.Parameter{
			Key:   k,
			Value: v,
		})
	}

	return s
}
