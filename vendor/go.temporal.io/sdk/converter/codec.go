package converter

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	commonpb "go.temporal.io/api/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// PayloadCodec is an codec that encodes or decodes the given payloads.
//
// For example, NewZlibCodec returns a PayloadCodec that can be used for
// compression.
// These can be used (and even chained) in NewCodecDataConverter.
type PayloadCodec interface {
	// Encode optionally encodes the given payloads which are guaranteed to never
	// be nil. The parameters must not be mutated.
	Encode([]*commonpb.Payload) ([]*commonpb.Payload, error)

	// Decode optionally decodes the given payloads which are guaranteed to never
	// be nil. The parameters must not be mutated.
	//
	// For compatibility reasons, implementers should take care not to decode
	// payloads that were not previously encoded.
	Decode([]*commonpb.Payload) ([]*commonpb.Payload, error)
}

// ZlibCodecOptions are options for NewZlibCodec. All fields are optional.
type ZlibCodecOptions struct {
	// If true, the zlib codec will encode the contents even if there is no size
	// benefit. Otherwise, the zlib codec will only use the encoded value if it
	// is smaller.
	AlwaysEncode bool
}

type zlibCodec struct{ options ZlibCodecOptions }

// NewZlibCodec creates a PayloadCodec for use in NewCodecDataConverter
// to support zlib payload compression.
//
// While this serves as a reasonable example of a compression encoder, callers
// may prefer alternative compression algorithms for lots of small payloads.
func NewZlibCodec(options ZlibCodecOptions) PayloadCodec { return &zlibCodec{options} }

func (z *zlibCodec) Encode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	result := make([]*commonpb.Payload, len(payloads))
	for i, p := range payloads {
		// Marshal and write
		b, err := proto.Marshal(p)
		if err != nil {
			return payloads, err
		}
		var buf bytes.Buffer
		w := zlib.NewWriter(&buf)
		_, err = w.Write(b)
		if closeErr := w.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		if err != nil {
			return payloads, err
		}
		// Only set if smaller than original amount or has option to always encode
		if buf.Len() < len(b) || z.options.AlwaysEncode {
			result[i] = &commonpb.Payload{
				Metadata: map[string][]byte{MetadataEncoding: []byte("binary/zlib")},
				Data:     buf.Bytes(),
			}
		} else {
			result[i] = p
		}
	}
	return result, nil
}

func (*zlibCodec) Decode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	result := make([]*commonpb.Payload, len(payloads))
	for i, p := range payloads {
		// Only if it's our encoding
		if string(p.Metadata[MetadataEncoding]) != "binary/zlib" {
			result[i] = p
			continue
		}
		r, err := zlib.NewReader(bytes.NewReader(p.Data))
		if err != nil {
			return payloads, err
		}
		// Read all and unmarshal
		b, err := io.ReadAll(r)
		if closeErr := r.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		if err != nil {
			return payloads, err
		}
		result[i] = &commonpb.Payload{}
		err = proto.Unmarshal(b, result[i])
		if err != nil {
			return payloads, err
		}
	}
	return result, nil
}

// CodecDataConverter is a DataConverter that wraps an underlying data
// converter and supports chained encoding of just the payload without regard
// for serialization to/from actual types.
type CodecDataConverter struct {
	parent DataConverter
	codecs []PayloadCodec
}

// NewCodecDataConverter wraps the given parent DataConverter and performs
// encoding/decoding on the payload via the given codecs. When encoding for
// ToPayload(s), the codecs are applied last to first meaning the earlier
// encoders wrap the later ones. When decoding for FromPayload(s) and
// ToString(s), the decoders are applied first to last to reverse the effect.
func NewCodecDataConverter(parent DataConverter, codecs ...PayloadCodec) DataConverter {
	return &CodecDataConverter{parent, codecs}
}

func (e *CodecDataConverter) encode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	var err error
	// Iterate backwards encoding
	for i := len(e.codecs) - 1; i >= 0; i-- {
		if payloads, err = e.codecs[i].Encode(payloads); err != nil {
			return payloads, err
		}
	}
	return payloads, nil
}

func (e *CodecDataConverter) decode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	var err error
	// Iterate forwards decoding
	for _, codec := range e.codecs {
		if payloads, err = codec.Decode(payloads); err != nil {
			return payloads, err
		}
	}
	return payloads, nil
}

// ToPayload implements DataConverter.ToPayload performing encoding on the
// result of the parent's ToPayload call.
func (e *CodecDataConverter) ToPayload(value interface{}) (*commonpb.Payload, error) {
	payload, err := e.parent.ToPayload(value)
	if payload == nil || err != nil {
		return payload, err
	}

	encodedPayloads, err := e.encode([]*commonpb.Payload{payload})
	if err != nil {
		return payload, err
	}
	if len(encodedPayloads) != 1 {
		return payload, fmt.Errorf("received %d payloads from codec, expected 1", len(encodedPayloads))
	}
	return encodedPayloads[0], err
}

// ToPayloads implements DataConverter.ToPayloads performing encoding on the
// result of the parent's ToPayloads call.
func (e *CodecDataConverter) ToPayloads(value ...interface{}) (*commonpb.Payloads, error) {
	payloads, err := e.parent.ToPayloads(value...)
	if payloads == nil || err != nil {
		return payloads, err
	}
	encodedPayloads, err := e.encode(payloads.Payloads)
	return &commonpb.Payloads{Payloads: encodedPayloads}, err
}

// FromPayload implements DataConverter.FromPayload performing decoding on the
// given payload before sending to the parent FromPayload.
func (e *CodecDataConverter) FromPayload(payload *commonpb.Payload, valuePtr interface{}) error {
	if payload == nil {
		return nil
	}
	decodedPayloads, err := e.decode([]*commonpb.Payload{payload})
	if err != nil {
		return err
	}
	if len(decodedPayloads) != 1 {
		return fmt.Errorf("received %d payloads from codec, expected 1", len(decodedPayloads))
	}
	return e.parent.FromPayload(decodedPayloads[0], valuePtr)
}

// FromPayloads implements DataConverter.FromPayloads performing decoding on the
// given payloads before sending to the parent FromPayloads.
func (e *CodecDataConverter) FromPayloads(payloads *commonpb.Payloads, valuePtrs ...interface{}) error {
	if payloads == nil {
		return e.parent.FromPayloads(payloads, valuePtrs...)
	}
	decodedPayloads, err := e.decode(payloads.Payloads)
	if err != nil {
		return err
	}
	return e.parent.FromPayloads(&commonpb.Payloads{Payloads: decodedPayloads}, valuePtrs...)
}

// ToString implements DataConverter.ToString performing decoding on the given
// payload before sending to the parent ToString.
func (e *CodecDataConverter) ToString(payload *commonpb.Payload) string {
	decodedPayloads, err := e.decode([]*commonpb.Payload{payload})
	if err != nil {
		return err.Error()
	}
	if len(decodedPayloads) != 1 {
		return fmt.Errorf("received %d payloads from codec, expected 1", len(decodedPayloads)).Error()
	}
	return e.parent.ToString(decodedPayloads[0])
}

// ToStrings implements DataConverter.ToStrings using ToString for each value.
func (e *CodecDataConverter) ToStrings(payloads *commonpb.Payloads) []string {
	if payloads == nil {
		return nil
	}
	strs := make([]string, len(payloads.Payloads))
	// Perform decoding one by one here so that we return individual errors
	for i, payload := range payloads.Payloads {
		strs[i] = e.ToString(payload)
	}
	return strs
}

const remotePayloadCodecEncodePath = "/encode"
const remotePayloadCodecDecodePath = "/decode"

type codecHTTPHandler struct {
	codecs []PayloadCodec
}

func (e *codecHTTPHandler) encode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	var err error
	for i := len(e.codecs) - 1; i >= 0; i-- {
		if payloads, err = e.codecs[i].Encode(payloads); err != nil {
			return payloads, err
		}
	}
	return payloads, nil
}

func (e *codecHTTPHandler) decode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	var err error
	for _, codec := range e.codecs {
		if payloads, err = codec.Decode(payloads); err != nil {
			return payloads, err
		}
	}
	return payloads, nil
}

// ServeHTTP implements the http.Handler interface.
func (e *codecHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	path := r.URL.Path

	if !strings.HasSuffix(path, remotePayloadCodecEncodePath) &&
		!strings.HasSuffix(path, remotePayloadCodecDecodePath) {
		http.NotFound(w, r)
		return
	}

	var payloadspb commonpb.Payloads
	var err error

	if r.Body == nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	bs, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err = protojson.Unmarshal(bs, &payloadspb); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	payloads := payloadspb.Payloads

	switch {
	case strings.HasSuffix(path, remotePayloadCodecEncodePath):
		if payloads, err = e.encode(payloads); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	case strings.HasSuffix(path, remotePayloadCodecDecodePath):
		if payloads, err = e.decode(payloads); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	default:
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(commonpb.Payloads{Payloads: payloads})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

// NewPayloadCodecHTTPHandler creates a http.Handler for a PayloadCodec.
// This can be used to provide a remote data converter.
func NewPayloadCodecHTTPHandler(e ...PayloadCodec) http.Handler {
	return &codecHTTPHandler{codecs: e}
}

// RemotePayloadCodecOptions are options for RemotePayloadCodec.
// Client is optional.
type RemotePayloadCodecOptions struct {
	Endpoint      string
	ModifyRequest func(*http.Request) error
	Client        http.Client
}

type remotePayloadCodec struct {
	options RemotePayloadCodecOptions
}

// NewRemotePayloadCodec creates a PayloadCodec using the remote endpoint configured by RemotePayloadCodecOptions.
func NewRemotePayloadCodec(options RemotePayloadCodecOptions) PayloadCodec {
	return &remotePayloadCodec{options}
}

// Encode uses the remote payload codec endpoint to encode payloads.
func (pc *remotePayloadCodec) Encode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	return pc.encodeOrDecode(pc.options.Endpoint+remotePayloadCodecEncodePath, payloads)
}

// Decode uses the remote payload codec endpoint to decode payloads.
func (pc *remotePayloadCodec) Decode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	return pc.encodeOrDecode(pc.options.Endpoint+remotePayloadCodecDecodePath, payloads)
}

func (pc *remotePayloadCodec) encodeOrDecode(endpoint string, payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	requestPayloads, err := json.Marshal(commonpb.Payloads{Payloads: payloads})
	if err != nil {
		return payloads, fmt.Errorf("unable to marshal payloads: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(requestPayloads))
	if err != nil {
		return payloads, fmt.Errorf("unable to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if pc.options.ModifyRequest != nil {
		err = pc.options.ModifyRequest(req)
		if err != nil {
			return payloads, err
		}
	}

	response, err := pc.options.Client.Do(req)
	if err != nil {
		return payloads, err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode == 200 {
		bs, err := io.ReadAll(response.Body)
		if err != nil {
			return payloads, fmt.Errorf("failed to read response body: %w", err)
		}
		var resultPayloads commonpb.Payloads
		err = protojson.Unmarshal(bs, &resultPayloads)
		if err != nil {
			return payloads, fmt.Errorf("unable to unmarshal payloads: %w", err)
		}
		if len(payloads) != len(resultPayloads.Payloads) {
			return payloads, fmt.Errorf("received %d payloads from remote codec, expected %d", len(resultPayloads.Payloads), len(payloads))
		}
		return resultPayloads.Payloads, nil
	}

	message, _ := io.ReadAll(response.Body)
	return payloads, fmt.Errorf("%s: %s", http.StatusText(response.StatusCode), message)
}

// Fields Endpoint, ModifyRequest, Client of RemotePayloadCodecOptions are also
// exposed here in RemoteDataConverterOptions for backwards compatibility.

// RemoteDataConverterOptions are options for NewRemoteDataConverter.
type RemoteDataConverterOptions struct {
	Endpoint      string
	ModifyRequest func(*http.Request) error
	Client        http.Client
}

type remoteDataConverter struct {
	parent       DataConverter
	payloadCodec PayloadCodec
}

// NewRemoteDataConverter wraps the given parent DataConverter and performs
// encoding/decoding on the payload via the remote endpoint.
func NewRemoteDataConverter(parent DataConverter, options RemoteDataConverterOptions) DataConverter {
	options.Endpoint = strings.TrimSuffix(options.Endpoint, "/")
	payloadCodec := NewRemotePayloadCodec(RemotePayloadCodecOptions(options))
	return &remoteDataConverter{parent, payloadCodec}
}

// ToPayload implements DataConverter.ToPayload performing remote encoding on the
// result of the parent's ToPayload call.
func (rdc *remoteDataConverter) ToPayload(value interface{}) (*commonpb.Payload, error) {
	payload, err := rdc.parent.ToPayload(value)
	if payload == nil || err != nil {
		return payload, err
	}
	encodedPayloads, err := rdc.payloadCodec.Encode([]*commonpb.Payload{payload})
	if err != nil {
		return payload, err
	}
	return encodedPayloads[0], err
}

// ToPayloads implements DataConverter.ToPayloads performing remote encoding on the
// result of the parent's ToPayloads call.
func (rdc *remoteDataConverter) ToPayloads(value ...interface{}) (*commonpb.Payloads, error) {
	payloads, err := rdc.parent.ToPayloads(value...)
	if payloads == nil || err != nil {
		return payloads, err
	}
	encodedPayloads, err := rdc.payloadCodec.Encode(payloads.Payloads)
	return &commonpb.Payloads{Payloads: encodedPayloads}, err
}

// FromPayload implements DataConverter.FromPayload performing remote decoding on the
// given payload before sending to the parent FromPayload.
func (rdc *remoteDataConverter) FromPayload(payload *commonpb.Payload, valuePtr interface{}) error {
	decodedPayloads, err := rdc.payloadCodec.Decode([]*commonpb.Payload{payload})
	if err != nil {
		return err
	}
	return rdc.parent.FromPayload(decodedPayloads[0], valuePtr)
}

// FromPayloads implements DataConverter.FromPayloads performing remote decoding on the
// given payloads before sending to the parent FromPayloads.
func (rdc *remoteDataConverter) FromPayloads(payloads *commonpb.Payloads, valuePtrs ...interface{}) error {
	if payloads == nil {
		return rdc.parent.FromPayloads(payloads, valuePtrs...)
	}

	decodedPayloads, err := rdc.payloadCodec.Decode(payloads.Payloads)
	if err != nil {
		return err
	}
	return rdc.parent.FromPayloads(&commonpb.Payloads{Payloads: decodedPayloads}, valuePtrs...)
}

// ToString implements DataConverter.ToString performing remote decoding on the given
// payload before sending to the parent ToString.
func (rdc *remoteDataConverter) ToString(payload *commonpb.Payload) string {
	if payload == nil {
		return rdc.parent.ToString(payload)
	}

	decodedPayloads, err := rdc.payloadCodec.Decode([]*commonpb.Payload{payload})
	if err != nil {
		return err.Error()
	}
	return rdc.parent.ToString(decodedPayloads[0])
}

// ToStrings implements DataConverter.ToStrings using ToString for each value.
func (rdc *remoteDataConverter) ToStrings(payloads *commonpb.Payloads) []string {
	if payloads == nil {
		return nil
	}

	strs := make([]string, len(payloads.Payloads))
	// Perform decoding one by one here so that we return individual errors
	for i, payload := range payloads.Payloads {
		strs[i] = rdc.ToString(payload)
	}
	return strs
}
