package clickhouse

import (
	"context"
	"log/slog"
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

func (c *connect) profileEvents(ctx context.Context, scanEvents bool) ([]ProfileEvent, error) {
	block, err := c.readData(ctx, proto.ServerProfileEvents, false)
	if err != nil {
		return nil, err
	}
	c.logger.Debug("profile events received", slog.Int("rows", block.Rows()))
	if !scanEvents {
		c.logger.Debug("profile events: skipping scan")
		return nil, nil
	}
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
