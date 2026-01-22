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

package proto

import (
	"fmt"
	"time"

	chproto "github.com/ClickHouse/ch-go/proto"
)

type Progress struct {
	Rows       uint64
	Bytes      uint64
	TotalRows  uint64
	WroteRows  uint64
	WroteBytes uint64
	Elapsed    time.Duration
	withClient bool
}

func (p *Progress) Decode(reader *chproto.Reader, revision uint64) (err error) {
	if p.Rows, err = reader.UVarInt(); err != nil {
		return err
	}
	if p.Bytes, err = reader.UVarInt(); err != nil {
		return err
	}
	if p.TotalRows, err = reader.UVarInt(); err != nil {
		return err
	}
	if revision >= DBMS_MIN_REVISION_WITH_CLIENT_WRITE_INFO {
		p.withClient = true
		if p.WroteRows, err = reader.UVarInt(); err != nil {
			return err
		}
		if p.WroteBytes, err = reader.UVarInt(); err != nil {
			return err
		}
	}

	if revision >= DBMS_MIN_PROTOCOL_VERSION_WITH_SERVER_QUERY_TIME_IN_PROGRES {
		var n uint64
		if n, err = reader.UVarInt(); err != nil {
			return err
		}
		p.Elapsed = time.Duration(n) * time.Nanosecond
	}

	return nil
}

func (p *Progress) String() string {
	if !p.withClient {
		return fmt.Sprintf("rows=%d, bytes=%d, total rows=%d, elapsed=%s", p.Rows, p.Bytes, p.TotalRows, p.Elapsed.String())
	}
	return fmt.Sprintf("rows=%d, bytes=%d, total rows=%d, wrote rows=%d wrote bytes=%d elapsed=%s",
		p.Rows,
		p.Bytes,
		p.TotalRows,
		p.WroteRows,
		p.WroteBytes,
		p.Elapsed.String(),
	)
}
