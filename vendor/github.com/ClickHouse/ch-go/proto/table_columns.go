package proto

import "github.com/go-faster/errors"

type TableColumns struct {
	First  string
	Second string
}

func (c *TableColumns) DecodeAware(r *Reader, _ int) error {
	{
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "first")
		}
		c.First = v
	}
	{
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "second")
		}
		c.Second = v
	}
	return nil
}

func (c TableColumns) EncodeAware(b *Buffer, _ int) {
	ServerCodeTableColumns.Encode(b)
	b.PutString(c.First)
	b.PutString(c.Second)
}
