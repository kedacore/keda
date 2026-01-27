package proto

import "github.com/go-faster/errors"

// Exception is server-side error.
type Exception struct {
	Code    Error
	Name    string
	Message string
	Stack   string
	Nested  bool
}

// DecodeAware decodes exception.
func (e *Exception) DecodeAware(r *Reader, _ int) error {
	code, err := r.Int32()
	if err != nil {
		return errors.Wrap(err, "code")
	}
	e.Code = Error(code)

	{
		s, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "name")
		}
		e.Name = s
	}
	{
		s, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "message")
		}
		e.Message = s
	}
	{
		s, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "stack trace")
		}
		e.Stack = s
	}
	nested, err := r.Bool()
	if err != nil {
		return errors.Wrap(err, "nested")
	}
	e.Nested = nested

	return nil
}

// EncodeAware encodes exception.
func (e *Exception) EncodeAware(b *Buffer, _ int) {
	b.PutInt32(int32(e.Code))
	b.PutString(e.Name)
	b.PutString(e.Message)
	b.PutString(e.Stack)
	b.PutBool(e.Nested)
}
