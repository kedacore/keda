package proto

import "time"

// Log from server.
type Log struct {
	QueryID  string    `json:"query_id"`
	Source   string    `json:"source"`
	Text     string    `json:"text"`
	Time     time.Time `json:"event_time"`
	Host     string    `json:"host_name"`
	ThreadID uint64    `json:"thread_id"`
	Priority int8      `json:"priority"`
}

// Logs from ServerCodeLog packet.
type Logs struct {
	Time      ColDateTime
	TimeMicro ColUInt32
	HostName  ColStr
	QueryID   ColStr
	ThreadID  ColUInt64
	Priority  ColInt8
	Source    ColStr
	Text      ColStr
}

func (s *Logs) Result() Results {
	return Results{
		{Name: "event_time", Data: &s.Time},
		{Name: "event_time_microseconds", Data: &s.TimeMicro},
		{Name: "host_name", Data: &s.HostName},
		{Name: "query_id", Data: &s.QueryID},
		{Name: "thread_id", Data: &s.ThreadID},
		{Name: "priority", Data: &s.Priority},
		{Name: "source", Data: &s.Source},
		{Name: "text", Data: &s.Text},
	}
}

func (s Logs) All() []Log {
	var out []Log
	for i := 0; i < s.Source.Rows(); i++ {
		out = append(out, Log{
			Time:     s.Time.Row(i),
			Host:     s.HostName.Row(i),
			QueryID:  s.QueryID.Row(i),
			ThreadID: s.ThreadID[i],
			Priority: s.Priority[i],
			Source:   s.Source.Row(i),
			Text:     s.Text.Row(i),
		})
	}
	return out
}
