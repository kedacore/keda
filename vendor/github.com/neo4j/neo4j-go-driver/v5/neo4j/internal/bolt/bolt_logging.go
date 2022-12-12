package bolt

import (
	"encoding/json"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/db"
	"strconv"
	"strings"
)

type loggableDictionary map[string]any

func (d loggableDictionary) String() string {
	if credentials, ok := d["credentials"]; ok {
		d["credentials"] = "<redacted>"
		defer func() {
			d["credentials"] = credentials
		}()
	}
	return serializeTrace(d)
}

type loggableStringDictionary map[string]string

func (sd loggableStringDictionary) String() string {
	if credentials, ok := sd["credentials"]; ok {
		sd["credentials"] = "<redacted>"
		defer func() {
			sd["credentials"] = credentials
		}()
	}
	return serializeTrace(sd)
}

type loggableList []any

func (l loggableList) String() string {
	return serializeTrace(l)
}

type loggableStringList []string

func (s loggableStringList) String() string {
	return serializeTrace(s)
}

type loggableSuccess success
type loggedSuccess struct {
	Server       string              `json:"server,omitempty"`
	ConnectionId string              `json:"connection_id,omitempty"`
	Fields       []string            `json:"fields,omitempty"`
	TFirst       string              `json:"t_first,omitempty"`
	Bookmark     string              `json:"bookmark,omitempty"`
	TLast        string              `json:"t_last,omitempty"`
	HasMore      bool                `json:"has_more,omitempty"`
	Db           string              `json:"db,omitempty"`
	Qid          int64               `json:"qid,omitempty"`
	ConfigHints  loggableDictionary  `json:"hints,omitempty"`
	RoutingTable *loggedRoutingTable `json:"routing_table,omitempty"`
}

func (s loggableSuccess) String() string {
	success := loggedSuccess{
		Server:       s.server,
		ConnectionId: s.connectionId,
		Fields:       s.fields,
		TFirst:       formatOmittingZero(s.tfirst),
		Bookmark:     s.bookmark,
		TLast:        formatOmittingZero(s.tlast),
		HasMore:      s.hasMore,
		Db:           s.db,
		ConfigHints:  s.configurationHints,
	}
	if s.qid > -1 {
		success.Qid = s.qid
	}
	routingTable := s.routingTable
	if routingTable != nil {
		success.RoutingTable = &loggedRoutingTable{
			TimeToLive:   routingTable.TimeToLive,
			DatabaseName: routingTable.DatabaseName,
			Routers:      routingTable.Routers,
			Readers:      routingTable.Readers,
			Writers:      routingTable.Writers,
		}
	}
	return serializeTrace(success)
}

type loggedRoutingTable struct {
	TimeToLive   int      `json:"ttl,omitempty"`
	DatabaseName string   `json:"db,omitempty"`
	Routers      []string `json:"routers,omitempty"`
	Readers      []string `json:"readers,omitempty"`
	Writers      []string `json:"writers,omitempty"`
}

func formatOmittingZero(i int64) string {
	if i == 0 {
		return ""
	}
	return strconv.FormatInt(i, 10)
}

type loggableFailure db.Neo4jError

func (f loggableFailure) String() string {
	return serializeTrace(map[string]any{
		"code":    f.Code,
		"message": f.Msg,
	})
}

func serializeTrace(v any) string {
	builder := strings.Builder{}
	encoder := json.NewEncoder(&builder)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(v)
	return strings.TrimSpace(builder.String())
}
