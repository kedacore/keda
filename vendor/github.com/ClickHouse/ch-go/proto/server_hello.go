package proto

import (
	"fmt"
	"strings"

	"github.com/go-faster/errors"
)

// ServerHello is answer to ClientHello and represents ServerCodeHello message.
type ServerHello struct {
	Name        string
	Major       int
	Minor       int
	Revision    int
	Timezone    string
	DisplayName string
	Patch       int
}

// Features implemented by server.
func (s ServerHello) Features() []Feature {
	var features []Feature
	for _, f := range FeatureValues() {
		if s.Has(f) {
			features = append(features, f)
		}
	}
	return features
}

// Has reports whether Feature is implemented.
func (s ServerHello) Has(f Feature) bool {
	return f.In(s.Revision)
}

func (s ServerHello) String() string {
	var b strings.Builder
	b.WriteString(s.Name)
	if s.DisplayName != "" {
		_, _ = fmt.Fprintf(&b, " (%s", s.DisplayName)
		if s.Timezone != "" {
			b.WriteString(", ")
			b.WriteString(s.Timezone)
		}
		b.WriteRune(')')
	}

	_, _ = fmt.Fprintf(&b, " %d.%d", s.Major, s.Minor)
	if s.Has(FeatureVersionPatch) {
		_, _ = fmt.Fprintf(&b, ".%d", s.Patch)
	}
	_, _ = fmt.Fprintf(&b, " (%d)", s.Revision)
	return b.String()
}

// DecodeAware decodes ServerHello message from Reader.
func (s *ServerHello) DecodeAware(r *Reader, v int) error {
	name, err := r.Str()
	if err != nil {
		return errors.Wrap(err, "str")
	}
	s.Name = name

	major, err := r.Int()
	if err != nil {
		return errors.Wrap(err, "major")
	}
	minor, err := r.Int()
	if err != nil {
		return errors.Wrap(err, "minor")
	}
	revision, err := r.Int()
	if err != nil {
		return errors.Wrap(err, "revision")
	}

	s.Major, s.Minor, s.Revision = major, minor, revision

	if FeatureTimezone.In(v) {
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "timezone")
		}
		s.Timezone = v
	}
	if FeatureDisplayName.In(v) {
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "display name")
		}
		s.DisplayName = v
	}
	if FeatureVersionPatch.In(v) {
		path, err := r.Int()
		if err != nil {
			return errors.Wrap(err, "patch")
		}
		s.Patch = path
	}

	return nil
}

func (s *ServerHello) EncodeAware(b *Buffer, v int) {
	ServerCodeHello.Encode(b)
	b.PutString(s.Name)
	b.PutInt(s.Major)
	b.PutInt(s.Minor)
	b.PutInt(s.Revision)
	if FeatureTimezone.In(v) {
		b.PutString(s.Timezone)
	}
	if FeatureDisplayName.In(v) {
		b.PutString(s.DisplayName)
	}
	if FeatureVersionPatch.In(v) {
		b.PutInt(s.Patch)
	}
}
