package common

import (
	//"bytes"

	"encoding"
	"encoding/base64"
	gojson "encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"go.temporal.io/api/internal/protojson"
	"go.temporal.io/api/internal/protojson/json"
	//	"google.golang.org/protobuf/encoding/protojson"
)

const (
	payloadMetadataKey      = "metadata"
	payloadDataKey          = "data"
	shorthandMessageTypeKey = "_protoMessageType"
	binaryNullBase64        = "YmluYXJ5L251bGw="
)

var _ protojson.ProtoJSONMaybeMarshaler = (*Payload)(nil)
var _ protojson.ProtoJSONMaybeMarshaler = (*Payloads)(nil)
var _ protojson.ProtoJSONMaybeUnmarshaler = (*Payload)(nil)
var _ protojson.ProtoJSONMaybeUnmarshaler = (*Payloads)(nil)

// !!! This file is copied from internal/temporalcommonv1 to common/v1.
// !!! DO NOT EDIT at common/v1/payload_json.go.
func marshalSingular(enc *json.Encoder, value interface{}) error {
	switch vv := value.(type) {
	case string:
		return enc.WriteString(vv)
	case bool:
		enc.WriteBool(vv)
	case int:
		enc.WriteInt(int64(vv))
	case int64:
		enc.WriteInt(vv)
	case uint:
		enc.WriteUint(uint64(vv))
	case uint64:
		enc.WriteUint(vv)
	case float32:
		enc.WriteFloat(float64(vv), 32)
	case float64:
		enc.WriteFloat(vv, 64)
	default:
		return fmt.Errorf("cannot marshal type %[1]T value %[1]v", vv)
	}
	return nil
}

func marshalStruct(enc *json.Encoder, vv reflect.Value) error {
	enc.StartObject()
	defer enc.EndObject()
	ty := vv.Type()

Loop:
	for i, n := 0, vv.NumField(); i < n; i++ {
		f := vv.Field(i)
		name := f.String()
		// lowercase. private field
		if name[0] >= 'a' && name[0] <= 'z' {
			continue
		}

		// Handle encoding/json struct tags
		tag, present := ty.Field(i).Tag.Lookup("json")
		if present {
			opts := strings.Split(tag, ",")
			for j := range opts {
				if opts[j] == "omitempty" && vv.IsZero() {
					continue Loop
				} else if opts[j] == "-" {
					continue Loop
				}
				// name overridden
				name = opts[j]
			}
		}
		if err := enc.WriteName(name); err != nil {
			return fmt.Errorf("unable to write name %s: %w", name, err)
		}
		if err := marshalSingular(enc, f.Interface()); err != nil {
			return fmt.Errorf("unable to marshal value for name %s: %w", name, err)
		}
	}
	return nil
}

type keyVal struct {
	k string
	v reflect.Value
}

// Map keys must be either strings or integers. We don't use encoding.TextMarshaler so we don't care
func marshalMap(enc *json.Encoder, vv reflect.Value) error {
	enc.StartObject()
	defer enc.EndObject()

	sv := make([]keyVal, vv.Len())
	iter := vv.MapRange()
	for i := 0; iter.Next(); i++ {
		k := iter.Key()
		sv[i].v = iter.Value()

		if k.Kind() == reflect.String {
			sv[i].k = k.String()
		} else if tm, ok := k.Interface().(encoding.TextMarshaler); ok {
			if k.Kind() == reflect.Pointer && k.IsNil() {
				return nil
			}
			buf, err := tm.MarshalText()
			sv[i].k = string(buf)
			if err != nil {
				return err
			}
		} else {
			switch k.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				sv[i].k = strconv.FormatInt(k.Int(), 10)
				return nil
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
				sv[i].k = strconv.FormatUint(k.Uint(), 10)
				return nil
			default:
				return fmt.Errorf("map key type %T not supported", k)
			}
		}
	}
	// Sort keys just like encoding/json
	sort.Slice(sv, func(i, j int) bool { return sv[i].k < sv[j].k })

	for i := 0; i < len(sv); i++ {
		if err := enc.WriteName(sv[i].k); err != nil {
			return fmt.Errorf("unable to write name %s: %w", sv[i].k, err)
		}
		if err := marshalValue(enc, sv[i].v); err != nil {
			return fmt.Errorf("unable to marshal value for name %s: %w", sv[i].k, err)
		}
	}
	return nil
}

func marshalValue(enc *json.Encoder, vv reflect.Value) error {
	switch vv.Kind() {
	case reflect.Slice, reflect.Array:
		if vv.IsNil() || vv.Len() == 0 {
			enc.WriteNull()
			return nil
		}
		enc.StartArray()
		defer enc.EndArray()
		for i := 0; i < vv.Len(); i++ {
			if err := marshalValue(enc, vv.Index(i)); err != nil {
				return err
			}
		}
	case reflect.Interface, reflect.Pointer:
		if vv.IsNil() {
			enc.WriteNull()
		} else {
			marshalValue(enc, vv.Elem())
		}
	case reflect.Struct:
		marshalStruct(enc, vv)
	case reflect.Map:
		if vv.IsNil() || vv.Len() == 0 {
			enc.StartObject()
			enc.EndObject()
			return nil
		}
		marshalMap(enc, vv)
	case reflect.Bool,
		reflect.String,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128:
		marshalSingular(enc, vv.Interface())
	default:
		return fmt.Errorf("cannot marshal %[1]T value %[1]v", vv.Interface())
	}

	return nil
}

func marshal(enc *json.Encoder, value interface{}) error {
	if value == nil {
		// nil data means we send the binary/null encoding
		enc.StartObject()
		defer enc.EndObject()
		if err := enc.WriteName("metadata"); err != nil {
			return err
		}

		enc.StartObject()
		defer enc.EndObject()
		if err := enc.WriteName("encoding"); err != nil {
			return err
		}
		// base64(binary/null)
		return enc.WriteString(binaryNullBase64)
	}
	return marshalValue(enc, reflect.ValueOf(value))
}

// Key on the marshaler metadata specifying whether shorthand is enabled.
//
// WARNING: This is internal API and should not be called externally.
const EnablePayloadShorthandMetadataKey = "__temporal_enable_payload_shorthand"

// MaybeMarshalProtoJSON implements
// [go.temporal.io/api/internal/temporaljsonpb.ProtoJSONMaybeMarshaler.MaybeMarshalProtoJSON].
//
// WARNING: This is internal API and should not be called externally.
func (p *Payloads) MaybeMarshalProtoJSON(meta map[string]interface{}, enc *json.Encoder) (handled bool, err error) {
	// If this is nil, ignore
	if p == nil {
		return false, nil
	}

	// Skip unless explicitly enabled
	if _, enabled := meta[EnablePayloadShorthandMetadataKey].(bool); !enabled {
		return false, nil
	}

	// We only support marshalling to shorthand if all payloads are handled or
	// there are no payloads, so check if all can be handled first.
	vals := make([]any, len(p.Payloads))
	for i, payload := range p.Payloads {
		handled, vals[i], err = payload.toJSONShorthand()
		if !handled || err != nil {
			return handled, err
		}
	}

	enc.StartArray()
	defer enc.EndArray()

	for _, val := range vals {
		if err = marshal(enc, val); err != nil {
			return true, err
		}
	}
	return true, err
}

// MaybeUnmarshalProtoJSON implements
// [go.temporal.io/api/internal/temporaljsonpb.ProtoJSONMaybeUnmarshaler.MaybeUnmarshalProtoJSON].
//
// WARNING: This is internal API and should not be called externally.
func (p *Payloads) MaybeUnmarshalProtoJSON(meta map[string]interface{}, dec *json.Decoder) (handled bool, err error) {
	// If this is nil, ignore (should never be)
	if p == nil {
		return false, nil
	}
	// Skip unless explicitly enabled
	if _, enabled := meta[EnablePayloadShorthandMetadataKey].(bool); !enabled {
		return false, nil
	}
	tok, err := dec.Peek()
	if err != nil {
		return true, err
	}

	if tok.Kind() == json.Null {
		// Null is accepted as empty list
		_, _ = dec.Read()
		return true, nil
	} else if tok.Kind() != json.ArrayOpen {
		// If this isn't an array, then it's not shorthand
		return false, nil
	}
	_, _ = dec.Read()
	for {
		tok, err := dec.Peek()
		if err != nil {
			return true, err
		}
		if tok.Kind() == json.ArrayClose {
			_, _ = dec.Read()
			break
		}
		var pl Payload
		if err := pl.fromJSONMaybeShorthand(dec); err != nil {
			return true, fmt.Errorf("unable to unmarshal payload: %w", err)
		}
		p.Payloads = append(p.Payloads, &pl)
	}

	return true, nil
}

// MaybeMarshalProtoJSON implements
// [go.temporal.io/api/internal/temporaljsonpb.ProtoJSONMaybeMarshaler.MaybeMarshalProtoJSON].
//
// WARNING: This is internal API and should not be called externally.
func (p *Payload) MaybeMarshalProtoJSON(meta map[string]interface{}, enc *json.Encoder) (handled bool, err error) {
	// If this is nil, ignore
	if p == nil {
		return false, nil
	}
	// Skip unless explicitly enabled
	if _, enabled := meta[EnablePayloadShorthandMetadataKey].(bool); !enabled {
		return false, nil
	}
	// If any are not handled or there is an error, return
	handled, val, err := p.toJSONShorthand()
	if !handled || err != nil {
		return handled, err
	}
	return true, marshal(enc, val)
}

// MaybeUnmarshalProtoJSON implements
// [go.temporal.io/api/internal/temporaljsonpb.ProtoJSONMaybeUnmarshaler.MaybeUnmarshalProtoJSON].
//
// WARNING: This is internal API and should not be called externally.
func (p *Payload) MaybeUnmarshalProtoJSON(meta map[string]interface{}, dec *json.Decoder) (handled bool, err error) {
	// If this is nil, ignore (should never be)
	if p == nil {
		return false, nil
	}
	// Skip unless explicitly enabled
	if _, enabled := meta[EnablePayloadShorthandMetadataKey].(bool); !enabled {
		return false, nil
	}
	// Always considered handled, unmarshaler ignored (unknown fields always
	// disallowed for non-shorthand payloads at this time)
	p.fromJSONMaybeShorthand(dec)
	return true, nil
}

func (p *Payload) toJSONShorthand() (handled bool, value interface{}, err error) {
	// Only support binary null, plain JSON and proto JSON
	switch string(p.Metadata["encoding"]) {
	case "binary/null":
		// Leave value as nil
		handled = true
	case "json/plain":
		// Must only have this single metadata
		if len(p.Metadata) != 1 {
			return false, nil, nil
		}
		// We unmarshal because we may have to indent. We let this error fail the
		// marshaller.
		handled = true
		err = gojson.Unmarshal(p.Data, &value)
	case "json/protobuf":
		// Must have the message type and no other metadata
		msgType := string(p.Metadata["messageType"])
		if msgType == "" || len(p.Metadata) != 2 {
			return false, nil, nil
		}
		// Since this is a proto object, this must unmarshal to a object. We let
		// this error fail the marshaller.
		var valueMap map[string]interface{}
		handled = true
		err = gojson.Unmarshal(p.Data, &valueMap)
		// Put the message type on the object
		if valueMap != nil {
			valueMap[shorthandMessageTypeKey] = msgType
		}
		value = valueMap
	default:
		return false, nil, fmt.Errorf("unsupported encoding %s", string(p.Metadata["encoding"]))
	}
	return
}

func unmarshalArray(dec *json.Decoder) (interface{}, error) {
	var arr []interface{}
	for {
		tok, err := dec.Read()
		if err != nil {
			return nil, err
		}
		if tok.Kind() == json.ArrayClose {
			return arr, nil
		}
		obj, err := unmarshalValue(dec, tok)
		if err != nil {
			return nil, err
		}
		arr = append(arr, obj)
	}

}

func unmarshalValue(dec *json.Decoder, tok json.Token) (interface{}, error) {
	switch tok.Kind() {
	case json.Null:
		return nil, nil
	case json.Bool:
		return tok.Bool(), nil
	case json.Number:
		i64, ok := tok.Int(64)
		if ok {
			return i64, nil
		}
		f64, ok := tok.Float(64)
		if ok {
			return f64, nil
		}
		return nil, fmt.Errorf("unable to parse number from %s", tok.Kind())
	case json.String:
		return tok.ParsedString(), nil
	case json.ObjectOpen:
		out := map[string]interface{}{}
		if err := unmarshalMap(dec, out); err != nil {
			return nil, err
		}
		return out, nil
	case json.ArrayOpen:
		return unmarshalArray(dec)
	default:
		return nil, fmt.Errorf("unexpected %s token %v", tok.Kind(), tok)
	}
}

// Payloads are a map of string to things. All keys are strings however, so we can take shortcuts here.
func unmarshalMap(dec *json.Decoder, out map[string]interface{}) error {
	for {
		tok, err := dec.Read()
		if err != nil {
			return err
		}
		switch tok.Kind() {
		default:
			return fmt.Errorf("unexpected %s token", tok.Kind())
		case json.ObjectClose:
			return nil
		case json.Name:
			key := tok.Name()
			tok, err = dec.Read()
			if err != nil {
				return fmt.Errorf("unexpected error unmarshalling value for map key %q: %w", key, err)
			}
			val, err := unmarshalValue(dec, tok)
			if err != nil {
				return fmt.Errorf("unable to unmarshal value for map key %q: %w", key, err)
			}
			out[key] = val
		}
	}
}

// Protojson marshals bytes as base64-encoded strings
func unmarshalBytes(s string) ([]byte, bool) {
	enc := base64.StdEncoding
	if strings.ContainsAny(s, "-_") {
		enc = base64.URLEncoding
	}
	if len(s)%4 != 0 {
		enc = enc.WithPadding(base64.NoPadding)
	}
	b, err := enc.DecodeString(s)
	if err != nil {
		return nil, false
	}
	return b, true
}

// Attempt to unmarshal a standard payload from this map. Returns true if successful
func (p *Payload) unmarshalPayload(valueMap map[string]interface{}) bool {
	md, mdOk := valueMap[payloadMetadataKey]
	if !mdOk {
		return false
	}

	mdm, ok := md.(map[string]interface{})
	if !ok {
		return false
	}

	// Payloads must have an encoding
	enc, ok := mdm["encoding"]
	if !ok {
		return false
	}

	d, dataOk := valueMap[payloadDataKey]
	// It's ok to have no data key if the encoding is binary/null
	if mdOk && !dataOk && enc == binaryNullBase64 {
		p.Metadata = map[string][]byte{
			"encoding": []byte("binary/null"),
		}
		return true
	} else if !mdOk && !dataOk {
		return false
	} else if len(valueMap) > 2 {
		// If we change the schema of the Payload type we'll need to update this
	}

	// By this point payloads must have both data and metadata keys and no others
	if !(dataOk && mdOk && len(valueMap) == 2) {
		return false
	}

	// We're probably a payload by this point
	ds, ok := d.(string)
	if !ok {
		return false
	}

	dataBytes, ok := unmarshalBytes(ds)
	if !ok {
		return false
	}
	mdbm := make(map[string][]byte, len(mdm))
	for k, v := range mdm {
		vs, ok := v.(string)
		// metadata keys will be encoded as base64 strings so we can reject everything else
		if !ok {
			return false
		}
		vb, ok := unmarshalBytes(vs)
		if !ok {
			return false
		}
		mdbm[k] = vb
	}

	p.Metadata = mdbm
	p.Data = dataBytes
	return true
}

func (p *Payload) fromJSONMaybeShorthand(dec *json.Decoder) error {
	// We need to try to deserialize into the regular payload first. If it works
	// and there is metadata _and_ data actually present (or null with a null
	// metadata encoding), we assume it's a non-shorthand payload. If it fails
	// (which it will if not an object or there is an unknown field or if
	// 'metadata' is not string + base64 or if 'data' is not base64), we assume
	// shorthand. We are ok disallowing unknown fields for payloads here even if
	// the outer unmarshaler allows them.
	tok, err := dec.Read()
	if err != nil {
		return err
	}
	val, err := unmarshalValue(dec, tok)
	if err != nil {
		return err
	}
	switch tv := val.(type) {
	default:
		// take it as-is
		p.Metadata = map[string][]byte{"encoding": []byte("json/plain")}
		p.Data, err = gojson.Marshal(val)
		return err
	case nil:
		p.Data = nil
		p.Metadata = map[string][]byte{"encoding": []byte("binary/null")}
		return nil
	case map[string]interface{}:
		if handled := p.unmarshalPayload(tv); handled {
			// Standard payload
			return nil
		}

		// Now that we know it is shorthand, it might be a proto JSON with a message
		// type. If it does have the message type, we need to remove it and
		// re-serialize it to data. So the quickest way to check whether it has the
		// message type is to search for the key.
		if maybeMsgType, found := tv[shorthandMessageTypeKey]; found {
			msgType, ok := maybeMsgType.(string)
			if !ok {
				return fmt.Errorf("internal key %q should have type string, not %T", shorthandMessageTypeKey, maybeMsgType)
			}
			// Now we know it's a proto JSON, so remove the key and re-serialize
			delete(tv, "_protoMessageType")
			// This won't error. The resulting JSON payload data may not be exactly
			// what user passed in sans message type (e.g. user may have indented or
			// did not have same field order), but that is acceptable when going
			// from shorthand to non-shorthand.
			p.Data, _ = gojson.Marshal(tv)
			p.Metadata = map[string][]byte{
				"encoding":    []byte("json/protobuf"),
				"messageType": []byte(msgType),
			}
		} else {
			p.Metadata = map[string][]byte{"encoding": []byte("json/plain")}
			p.Data, err = gojson.Marshal(val)
			return err
		}
		return nil
	}
}
