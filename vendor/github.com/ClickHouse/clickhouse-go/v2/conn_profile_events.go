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
	"reflect"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

type ProfileEvent struct {
	Hostname    string
	CurrentTime time.Time
	ThreadID    uint64
	Type        string
	Name        string
	Value       int64
}

func (c *connect) profileEvents(ctx context.Context) ([]ProfileEvent, error) {
	block, err := c.readData(ctx, proto.ServerProfileEvents, false)
	if err != nil {
		return nil, err
	}
	c.debugf("[profile events] rows=%d", block.Rows())
	var (
		events []ProfileEvent
		names  = block.ColumnsNames()
	)
	for r := 0; r < block.Rows(); r++ {
		var event ProfileEvent
		for i, b := range block.Columns {
			switch names[i] {
			case "host_name":
				if err := b.ScanRow(&event.Hostname, r); err != nil {
					return nil, err
				}
			case "current_time":
				if err := b.ScanRow(&event.CurrentTime, r); err != nil {
					return nil, err
				}
			case "thread_id":
				if err := b.ScanRow(&event.ThreadID, r); err != nil {
					return nil, err
				}
			case "type":
				if err := b.ScanRow(&event.Type, r); err != nil {
					return nil, err
				}
			case "name":
				if err := b.ScanRow(&event.Name, r); err != nil {
					return nil, err
				}
			case "value":
				switch b.ScanType().Kind() {
				case reflect.Uint64:
					var v uint64
					if err := b.ScanRow(&v, r); err != nil {
						return nil, err
					}
					event.Value = int64(v)
				default:
					if err := b.ScanRow(&event.Value, r); err != nil {
						return nil, err
					}
				}
			}
		}
		events = append(events, event)
	}
	return events, nil
}
