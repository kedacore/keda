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
