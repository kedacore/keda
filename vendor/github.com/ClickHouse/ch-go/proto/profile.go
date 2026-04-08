package proto

import "github.com/go-faster/errors"

type Profile struct {
	Rows                      uint64
	Blocks                    uint64
	Bytes                     uint64
	AppliedLimit              bool
	RowsBeforeLimit           uint64
	CalculatedRowsBeforeLimit bool
}

func (p *Profile) DecodeAware(r *Reader, _ int) error {
	{
		v, err := r.UVarInt()
		if err != nil {
			return errors.Wrap(err, "rows")
		}
		p.Rows = v
	}
	{
		v, err := r.UVarInt()
		if err != nil {
			return errors.Wrap(err, "blocks")
		}
		p.Blocks = v
	}
	{
		v, err := r.UVarInt()
		if err != nil {
			return errors.Wrap(err, "bytes")
		}
		p.Bytes = v
	}
	{
		v, err := r.Bool()
		if err != nil {
			return errors.Wrap(err, "applied limit")
		}
		p.AppliedLimit = v
	}
	{
		v, err := r.UVarInt()
		if err != nil {
			return errors.Wrap(err, "rows before limit")
		}
		p.RowsBeforeLimit = v
	}
	{
		v, err := r.Bool()
		if err != nil {
			return errors.Wrap(err, "calculated rows before limit")
		}
		p.CalculatedRowsBeforeLimit = v
	}

	return nil
}

func (p Profile) EncodeAware(b *Buffer, _ int) {
	ServerCodeProfile.Encode(b)
	b.PutUVarInt(p.Rows)
	b.PutUVarInt(p.Blocks)
	b.PutUVarInt(p.Bytes)
	b.PutBool(p.AppliedLimit)
	b.PutUVarInt(p.RowsBeforeLimit)
	b.PutBool(p.CalculatedRowsBeforeLimit)
}
