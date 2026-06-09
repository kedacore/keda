package proto

import (
	"fmt"

	chproto "github.com/ClickHouse/ch-go/proto"
)

type ProfileInfo struct {
	Rows                      uint64
	Bytes                     uint64
	Blocks                    uint64
	AppliedLimit              bool
	RowsBeforeLimit           uint64
	CalculatedRowsBeforeLimit bool
}

func (p *ProfileInfo) Decode(reader *chproto.Reader, revision uint64) (err error) {
	if p.Rows, err = reader.UVarInt(); err != nil {
		return err
	}
	if p.Blocks, err = reader.UVarInt(); err != nil {
		return err
	}
	if p.Bytes, err = reader.UVarInt(); err != nil {
		return err
	}
	if p.AppliedLimit, err = reader.Bool(); err != nil {
		return err
	}
	if p.RowsBeforeLimit, err = reader.UVarInt(); err != nil {
		return err
	}
	if p.CalculatedRowsBeforeLimit, err = reader.Bool(); err != nil {
		return err
	}
	return nil
}

func (p *ProfileInfo) String() string {
	return fmt.Sprintf("rows=%d, bytes=%d, blocks=%d, rows before limit=%d, applied limit=%t, calculated rows before limit=%t",
		p.Rows,
		p.Bytes,
		p.Blocks,
		p.RowsBeforeLimit,
		p.AppliedLimit,
		p.CalculatedRowsBeforeLimit,
	)
}
