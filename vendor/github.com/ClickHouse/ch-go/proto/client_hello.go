package proto

import "github.com/go-faster/errors"

// ClientHello represents ClientCodeHello message.
type ClientHello struct {
	Name string

	Major int // client major version
	Minor int // client minor version

	// ProtocolVersion is TCP protocol version of client.
	//
	// Usually it is equal to the latest compatible server revision, but
	// should not be confused with it.
	ProtocolVersion int

	Database string
	User     string
	Password string
}

// Encode to Buffer.
func (c ClientHello) Encode(b *Buffer) {
	ClientCodeHello.Encode(b)
	b.PutString(c.Name)
	b.PutInt(c.Major)
	b.PutInt(c.Minor)
	b.PutInt(c.ProtocolVersion)
	b.PutString(c.Database)
	b.PutString(c.User)
	b.PutString(c.Password)
}

func (c *ClientHello) Decode(r *Reader) error {
	{
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "name")
		}
		c.Name = v
	}
	{
		v, err := r.Int()
		if err != nil {
			return errors.Wrap(err, "major")
		}
		c.Major = v
	}
	{
		v, err := r.Int()
		if err != nil {
			return errors.Wrap(err, "minor")
		}
		c.Minor = v
	}
	{
		v, err := r.Int()
		if err != nil {
			return errors.Wrap(err, "protocol version")
		}
		c.ProtocolVersion = v
	}
	{
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "database")
		}
		c.Database = v
	}
	{
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "user")
		}
		c.User = v
	}
	{
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "password")
		}
		c.Password = v
	}
	return nil
}
