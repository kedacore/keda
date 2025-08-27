package idtool

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"strings"
)

// ID is used to identify many SignalFx resources, including time series.
type ID int64

// String returns the string representation commonly used instead of an int64
func (id ID) String() string {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(id))
	return strings.TrimRight(base64.URLEncoding.EncodeToString(b), "=")
}

// UnmarshalJSON assumes that the id is always serialized in the string format.
func (id *ID) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	id2 := IDFromString(s)
	*id = id2
	return nil
}

// IDFromString creates an ID from a pseudo-base64 string
func IDFromString(idstr string) ID {
	if idstr != "" {
		if idstr[len(idstr)-1] != '=' {
			idstr = idstr + "="
		}
		buff, err := base64.URLEncoding.DecodeString(idstr)
		if err == nil {
			output := binary.BigEndian.Uint64(buff)
			return ID(output)
		}
	}
	return ID(0)
}
