package proto

import "github.com/go-faster/errors"

// ColRaw is Column that performs zero decoding or encoding.
// T, Size are required.
//
// TODO: support strings and T, Size inference.
//
// Useful for copying from one source to another.
type ColRaw struct {
	T    ColumnType // type of column
	Size int        // size of single value

	Data  []byte // raw value of column
	Count int    // count of rows
}

func (c ColRaw) Type() ColumnType       { return c.T }
func (c ColRaw) Rows() int              { return c.Count }
func (c ColRaw) EncodeColumn(b *Buffer) { b.Buf = append(b.Buf, c.Data...) }

func (c *ColRaw) DecodeColumn(r *Reader, rows int) error {
	c.Count = rows
	c.Data = append(c.Data[:0], make([]byte, c.Size*rows)...)
	if err := r.ReadFull(c.Data); err != nil {
		return errors.Wrap(err, "read full")
	}
	return nil
}

func (c *ColRaw) Reset() {
	c.Count = 0
	c.Data = c.Data[:0]
}
