package proto

import "github.com/go-faster/errors"

// Result of Query.
type Result interface {
	DecodeResult(r *Reader, version int, b Block) error
}

// Results wrap []ResultColumn to implement Result.
type Results []ResultColumn

type autoResults struct {
	results *Results
}

func (s autoResults) DecodeResult(r *Reader, version int, b Block) error {
	return s.results.decodeAuto(r, version, b)
}

func (s Results) Rows() int {
	if len(s) == 0 {
		return 0
	}
	return s[0].Data.Rows()
}

func (s *Results) Auto() Result {
	return autoResults{results: s}
}

func (s *Results) decodeAuto(r *Reader, version int, b Block) error {
	if len(*s) > 0 {
		// Already inferred.
		return s.DecodeResult(r, version, b)
	}
	for i := 0; i < b.Columns; i++ {
		columnName, err := r.Str()
		if err != nil {
			return errors.Wrapf(err, "column [%d] name", i)
		}
		columnTypeRaw, err := r.Str()
		if err != nil {
			return errors.Wrapf(err, "column [%d] type", i)
		}
		var customSerialization bool
		if FeatureCustomSerialization.In(version) {
			if customSerialization, err = r.Bool(); err != nil {
				return errors.Wrapf(err, "column [%d] custom serialization", i)
			}
			if customSerialization {
				// Not implemented.
				return errors.Wrapf(err, "column [%d] has custom serialization (not supported)", i)
			}
		}
		var (
			colType = ColumnType(columnTypeRaw)
			col     = &ColAuto{}
		)
		if err := col.Infer(colType); err != nil {
			return errors.Wrap(err, "column type inference")
		}
		col.Data.Reset()
		if b.Rows != 0 {
			if s, ok := col.Data.(Stateful); ok {
				if err := s.DecodeState(r); err != nil {
					return errors.Wrapf(err, "%s state", columnName)
				}
			}
			if err := col.Data.DecodeColumn(r, b.Rows); err != nil {
				return errors.Wrap(err, columnName)
			}
		}
		*s = append(*s, ResultColumn{
			Name: columnName,
			Data: col.Data,
		})
	}
	return nil
}

func (s Results) DecodeResult(r *Reader, version int, b Block) error {
	var (
		noTarget        = len(s) == 0
		noRows          = b.Rows == 0
		columnsMismatch = b.Columns != len(s)
		allowMismatch   = noTarget && noRows
	)
	if columnsMismatch && !allowMismatch {
		return errors.Errorf("%d (columns) != %d (target)", b.Columns, len(s))
	}
	for i := 0; i < b.Columns; i++ {
		columnName, err := r.Str()
		if err != nil {
			return errors.Wrapf(err, "column [%d] name", i)
		}
		columnType, err := r.Str()
		if err != nil {
			return errors.Wrapf(err, "column [%d] type", i)
		}
		if FeatureCustomSerialization.In(version) {
			customSerialization, err := r.Bool()
			if err != nil {
				return errors.Wrapf(err, "column [%d] custom serialization", i)
			}
			if customSerialization {
				// Not implemented.
				return errors.Wrapf(err, "column [%d] has custom serialization (not supported)", i)
			}
		}
		if noTarget {
			// Just reading types and names.
			continue
		}

		// Checking column name and type.
		t := s[i]
		if t.Name == "" {
			// Inferring column name.
			t.Name = columnName
			s[i] = t
		}
		if t.Name != columnName {
			return errors.Errorf("[%d]: unexpected column %q (%q expected)", i, columnName, t.Name)
		}
		gotType := ColumnType(columnType)
		if infer, ok := t.Data.(Inferable); ok {
			if err := infer.Infer(gotType); err != nil {
				return errors.Wrap(err, "infer")
			}
		}
		hasType := t.Data.Type()
		if gotType.Conflicts(hasType) {
			return errors.Errorf("[%d]: %s: unexpected type %q (got) instead of %q (has)",
				i, columnName, gotType, hasType,
			)
		}
		t.Data.Reset()
		if b.Rows == 0 {
			continue
		}
		if s, ok := t.Data.(StateDecoder); ok {
			if err := s.DecodeState(r); err != nil {
				return errors.Wrapf(err, "%s state", columnName)
			}
		}
		if err := t.Data.DecodeColumn(r, b.Rows); err != nil {
			return errors.Wrap(err, columnName)
		}
	}

	return nil
}
