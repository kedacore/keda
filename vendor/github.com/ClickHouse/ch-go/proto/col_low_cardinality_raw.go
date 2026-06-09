package proto

import "github.com/go-faster/errors"

// ColLowCardinalityRaw is non-generic version of ColLowCardinality.
type ColLowCardinalityRaw struct {
	Index Column // dictionary
	Key   CardinalityKey

	// Keeping all key column variants as fields to reuse
	// memory more efficiently.

	Keys8  ColUInt8
	Keys16 ColUInt16
	Keys32 ColUInt32
	Keys64 ColUInt64
}

func (c *ColLowCardinalityRaw) DecodeState(r *Reader) error {
	keySerialization, err := r.Int64()
	if err != nil {
		return errors.Wrap(err, "version")
	}
	if keySerialization != int64(sharedDictionariesWithAdditionalKeys) {
		return errors.Errorf("got version %d, expected %d",
			keySerialization, sharedDictionariesWithAdditionalKeys,
		)
	}
	if s, ok := c.Index.(StateDecoder); ok {
		if err := s.DecodeState(r); err != nil {
			return errors.Wrap(err, "state")
		}
	}
	return nil
}

func (c ColLowCardinalityRaw) EncodeState(b *Buffer) {
	// Writing key serialization version.
	b.PutInt64(int64(sharedDictionariesWithAdditionalKeys))
	if s, ok := c.Index.(StateEncoder); ok {
		s.EncodeState(b)
	}
}

func (c *ColLowCardinalityRaw) AppendKey(i int) {
	switch c.Key {
	case KeyUInt8:
		c.Keys8 = append(c.Keys8, uint8(i))
	case KeyUInt16:
		c.Keys16 = append(c.Keys16, uint16(i))
	case KeyUInt32:
		c.Keys32 = append(c.Keys32, uint32(i))
	case KeyUInt64:
		c.Keys64 = append(c.Keys64, uint64(i))
	default:
		panic("invalid key type")
	}
}

func (c *ColLowCardinalityRaw) Keys() Column {
	switch c.Key {
	case KeyUInt8:
		return &c.Keys8
	case KeyUInt16:
		return &c.Keys16
	case KeyUInt32:
		return &c.Keys32
	case KeyUInt64:
		return &c.Keys64
	default:
		panic("invalid key type")
	}
}

func (c ColLowCardinalityRaw) Type() ColumnType {
	return ColumnTypeLowCardinality.Sub(c.Index.Type())
}

func (c ColLowCardinalityRaw) Rows() int {
	return c.Keys().Rows()
}

func (c *ColLowCardinalityRaw) DecodeColumn(r *Reader, rows int) error {
	if rows == 0 {
		// Skipping entirely of no rows.
		return nil
	}
	meta, err := r.Int64()
	if err != nil {
		return errors.Wrap(err, "meta")
	}
	if (meta & cardinalityNeedGlobalDictionaryBit) == 1 {
		return errors.New("global dictionary is not supported")
	}
	if (meta & cardinalityHasAdditionalKeysBit) == 0 {
		return errors.New("additional keys bit is missing")
	}

	key := CardinalityKey(meta & cardinalityKeyMask)
	if !key.IsACardinalityKey() {
		return errors.Errorf("invalid low cardinality keys type %d", key)
	}
	c.Key = key

	indexRows, err := r.Int64()
	if err != nil {
		return errors.Wrap(err, "index size")
	}
	if err := checkRows(int(indexRows)); err != nil {
		return errors.Wrap(err, "index size")
	}
	if err := c.Index.DecodeColumn(r, int(indexRows)); err != nil {
		return errors.Wrap(err, "index column")
	}

	keyRows, err := r.Int64()
	if err != nil {
		return errors.Wrap(err, "keys size")
	}
	if err := checkRows(int(keyRows)); err != nil {
		return errors.Wrap(err, "index size")
	}
	if err := c.Keys().DecodeColumn(r, int(keyRows)); err != nil {
		return errors.Wrap(err, "keys column")
	}

	return nil
}

func (c *ColLowCardinalityRaw) Reset() {
	c.Index.Reset()
	c.Keys8.Reset()
	c.Keys16.Reset()
	c.Keys32.Reset()
	c.Keys64.Reset()
}

func (c ColLowCardinalityRaw) EncodeColumn(b *Buffer) {
	if c.Rows() == 0 {
		// Skipping encoding entirely.
		return
	}

	// Meta encodes whether reader should update
	// low cardinality metadata and keys column type.
	meta := cardinalityUpdateAll | int64(c.Key)
	b.PutInt64(meta)

	// Writing index (dictionary).
	b.PutInt64(int64(c.Index.Rows()))
	c.Index.EncodeColumn(b)

	// Sequence of values as indexes in dictionary.
	k := c.Keys()
	b.PutInt64(int64(k.Rows()))
	k.EncodeColumn(b)
}

func (c ColLowCardinalityRaw) WriteColumn(w *Writer) {
	if c.Rows() == 0 {
		// Skipping encoding entirely.
		return
	}

	w.ChainBuffer(func(b *Buffer) {
		// Meta encodes whether reader should update
		// low cardinality metadata and keys column type.
		meta := cardinalityUpdateAll | int64(c.Key)
		b.PutInt64(meta)

		// Writing index (dictionary).
		b.PutInt64(int64(c.Index.Rows()))
	})
	c.Index.WriteColumn(w)

	// Sequence of values as indexes in dictionary.
	k := c.Keys()
	w.ChainBuffer(func(b *Buffer) {
		b.PutInt64(int64(k.Rows()))
	})
	k.WriteColumn(w)
}
