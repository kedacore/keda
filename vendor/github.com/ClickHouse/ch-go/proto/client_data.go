package proto

import "github.com/go-faster/errors"

type ClientData struct {
	TableName string
}

func (c ClientData) EncodeAware(b *Buffer, version int) {
	if FeatureTempTables.In(version) {
		b.PutString(c.TableName)
	}
}

func (c *ClientData) DecodeAware(r *Reader, version int) error {
	if FeatureTempTables.In(version) {
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "temp tables")
		}
		c.TableName = v
	}
	return nil
}
