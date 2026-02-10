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

type Log struct {
	Time      time.Time
	TimeMicro uint32
	Hostname  string
	QueryID   string
	ThreadID  uint64
	Priority  int8
	Source    string
	Text      string
}

func (c *connect) logs(ctx context.Context) ([]Log, error) {
	block, err := c.readData(ctx, proto.ServerLog, false)
	if err != nil {
		return nil, err
	}
	c.debugf("[logs] rows=%d", block.Rows())
	var (
		logs  []Log
		names = block.ColumnsNames()
	)
	for r := 0; r < block.Rows(); r++ {
		var log Log
		for i, b := range block.Columns {
			switch names[i] {
			case "event_time":
				if err := b.ScanRow(&log.Time, r); err != nil {
					return nil, err
				}
			case "event_time_microseconds":
				if err := b.ScanRow(&log.TimeMicro, r); err != nil {
					return nil, err
				}
			case "host_name":
				if err := b.ScanRow(&log.Hostname, r); err != nil {
					return nil, err
				}
			case "query_id":
				if err := b.ScanRow(&log.QueryID, r); err != nil {
					return nil, err
				}
			case "thread_id":
				if err := b.ScanRow(&log.ThreadID, r); err != nil {
					return nil, err
				}
			case "priority":
				if err := b.ScanRow(&log.Priority, r); err != nil {
					return nil, err
				}
			case "source":
				if err := b.ScanRow(&log.Source, r); err != nil {
					return nil, err
				}
			case "text":
				if err := b.ScanRow(&log.Text, r); err != nil {
					return nil, err
				}
			}
		}
		logs = append(logs, log)
	}
	return logs, nil
}
