package proto

import (
	"time"

	"github.com/go-faster/errors"
)

// ProfileEvents is data of ServerProfileEvents packet.
type ProfileEvents struct {
	Host     ColStr
	Time     ColDateTime
	ThreadID ColUInt64
	Type     ColInt8
	Name     ColStr
	Value    ColAuto // UInt64 or Int64 depending on version
}

func (d *ProfileEvents) All() ([]ProfileEvent, error) {
	var out []ProfileEvent
	for i := range d.Type {
		e := ProfileEvent{
			Time:     d.Time.Row(i),
			Host:     d.Host.Row(i),
			ThreadID: d.ThreadID[i],
			Type:     ProfileEventType(d.Type[i]),
			Name:     d.Name.Row(i),
		}
		switch data := d.Value.Data.(type) {
		case *ColInt64:
			e.Value = (*data)[i]
		case *ColUInt64:
			e.Value = int64((*data)[i])
		default:
			return nil, errors.Errorf("unexpected type %q for metric column", data.Type())
		}
		out = append(out, e)
	}
	return out, nil
}

func (d *ProfileEvents) Result() Results {
	return Results{
		{Name: "host_name", Data: &d.Host},
		{Name: "current_time", Data: &d.Time},
		{Name: "thread_id", Data: &d.ThreadID},
		{Name: "type", Data: &d.Type},
		{Name: "name", Data: &d.Name},
		{Name: "value", Data: &d.Value},
	}
}

//go:generate go run github.com/dmarkham/enumer -type ProfileEventType -trimprefix Profile -text -json -output profile_enum.go

type ProfileEventType byte

const (
	ProfileIncrement ProfileEventType = 1
	ProfileGauge     ProfileEventType = 2
)

// ProfileEvent is detailed profiling event from Server.
type ProfileEvent struct {
	Type     ProfileEventType `json:"type"`
	Name     string           `json:"name"`
	Value    int64            `json:"value"`
	Host     string           `json:"host_name"`
	Time     time.Time        `json:"current_time"`
	ThreadID uint64           `json:"thread_id"`
}
