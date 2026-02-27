package proto

import (
	"github.com/go-faster/errors"
)

type Query struct {
	ID          string
	Body        string
	Secret      string
	Stage       Stage
	Compression Compression
	Info        ClientInfo
	Settings    []Setting
	Parameters  []Parameter
}

type Parameter struct {
	Key   string
	Value string
}

func (p Parameter) Encode(b *Buffer) {
	Setting{
		Key:    p.Key,
		Value:  p.Value,
		Custom: true,
	}.Encode(b)
}

func (p *Parameter) Decode(r *Reader) error {
	var s Setting
	if err := s.Decode(r); err != nil {
		return errors.Wrap(err, "as setting")
	}

	p.Key = s.Key
	p.Value = s.Value

	return nil
}

// src/Core/BaseSettings.h:191 (BaseSettingsHelpers.Flags)
const (
	settingFlagImportant = 0x01
	settingFlagCustom    = 0x02
	settingFlagObsolete  = 0x04
)

type Setting struct {
	Key   string
	Value string

	Important bool
	Custom    bool
	Obsolete  bool
}

func (s Setting) Encode(b *Buffer) {
	b.PutString(s.Key)
	{
		var flags uint64
		if s.Important {
			flags |= settingFlagImportant
		}
		if s.Custom {
			flags |= settingFlagCustom
		}
		if s.Obsolete {
			flags |= settingFlagObsolete
		}
		b.PutUVarInt(flags)
	}
	b.PutString(s.Value)
}

func (s *Setting) Decode(r *Reader) error {
	key, err := r.Str()
	if err != nil {
		return errors.Wrap(err, "key")
	}

	if key == "" {
		// End of settings.
		return nil
	}

	flags, err := r.UVarInt()
	if err != nil {
		return errors.Wrap(err, "flags")
	}

	v, err := r.Str()
	if err != nil {
		return errors.Wrapf(err, "value (%s)", key)
	}

	s.Key = key
	s.Important = flags&settingFlagImportant != 0
	s.Custom = flags&settingFlagCustom != 0
	s.Obsolete = flags&settingFlagObsolete != 0
	s.Value = v

	return nil
}

func (q *Query) DecodeAware(r *Reader, version int) error {
	{
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "query id")
		}
		q.ID = v
	}
	if FeatureClientWriteInfo.In(version) {
		if err := q.Info.DecodeAware(r, version); err != nil {
			return errors.Wrap(err, "client info")
		}
	}
	if !FeatureSettingsSerializedAsStrings.In(version) {
		return errors.New("unsupported version")
	}
	for {
		var s Setting
		if err := s.Decode(r); err != nil {
			return errors.Wrap(err, "setting")
		}
		if s.Key == "" {
			break
		}
		q.Settings = append(q.Settings, s)
	}
	if FeatureInterServerSecret.In(version) {
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "inter-server secret")
		}
		q.Secret = v
	}

	{
		v, err := r.UVarInt()
		if err != nil {
			return errors.Wrap(err, "stage")
		}
		q.Stage = Stage(v)
		if !q.Stage.IsAStage() {
			return errors.Errorf("unknown stage %d", v)
		}
	}
	{
		v, err := r.UVarInt()
		if err != nil {
			return errors.Wrap(err, "compression")
		}
		q.Compression = Compression(v)
		if !q.Compression.IsACompression() {
			return errors.Errorf("unknown compression %d", v)
		}
	}

	{
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "query body")
		}
		q.Body = v
	}
	if FeatureParameters.In(version) {
		for {
			var p Parameter
			if err := p.Decode(r); err != nil {
				return errors.Wrap(err, "parameter")
			}
			if p.Key == "" {
				break
			}
			q.Parameters = append(q.Parameters, p)
		}
	}
	return nil
}

func (q Query) EncodeAware(b *Buffer, version int) {
	ClientCodeQuery.Encode(b)
	b.PutString(q.ID)
	if FeatureClientWriteInfo.In(version) {
		q.Info.EncodeAware(b, version)
	}

	if FeatureSettingsSerializedAsStrings.In(version) {
		for _, s := range q.Settings {
			s.Encode(b)
		}
	}
	b.PutString("") // end of settings

	if FeatureInterServerSecret.In(version) {
		b.PutString(q.Secret)
	}

	StageComplete.Encode(b)
	q.Compression.Encode(b)

	b.PutString(q.Body)

	if FeatureParameters.In(version) {
		for _, p := range q.Parameters {
			p.Encode(b)
		}
		b.PutString("") // end of parameters
	}
}
