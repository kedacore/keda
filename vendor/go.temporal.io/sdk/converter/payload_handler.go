// Package converter provides an HTTP handler for a Temporal codec server with
// support for external payload storage.

package converter

import (
	"errors"
	"io"
	"net/http"
	"strings"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/proxy"
	"go.temporal.io/sdk/internal/extstore"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	downloadPath = "/download"
)

// PayloadHTTPHandlerOptions configures a [NewPayloadHTTPHandler].
//
// NOTE: Experimental
type PayloadHTTPHandlerOptions struct {
	// PostStorageCodecs are codecs applied after external storage from the
	// perspective of payloads going through a encoding transformation. These are
	// typically the codecs that would be configured in the proxy's codec chain.
	// When encoding, the codecs are applied last to first meaning the earlier
	// codecs wrap the later ones. When decoding, the codecs are applied first
	// to last to reverse the effect.
	//
	// NOTE: Experimental.
	PostStorageCodecs []PayloadCodec

	// PreStorageCodecs are codecs that are applied before external storage,
	// from the perspective of payloads going through an encoding transformation.
	// These are typically the codecs that would be configured in the DataConverter
	// codec chain on a Temporal client. When encoding, the codecs are applied last
	// to first meaning the earlier codecs wrap the later ones. When decoding, the
	// codecs are applied first to last to reverse the effect.
	//
	// NOTE: Experimental.
	PreStorageCodecs []PayloadCodec

	// ExternalStorage configures external payload storage, allowing payloads
	// to be stored and retrieved from external sources if they meet the size
	// threshold and driver selection criteria.
	//
	// NOTE: Experimental.
	ExternalStorage ExternalStorage
}

type payloadHTTPHandler struct {
	postStorageCodecs []PayloadCodec
	preStorageCodecs  []PayloadCodec
	retrievalVisitor  extstore.PayloadVisitor
	storageVisitor    extstore.PayloadVisitor
}

var _ http.Handler = (*payloadHTTPHandler)(nil)

// NewPayloadHTTPHandler creates an [http.Handler] that serves /encode, /decode,
// and /download routes for remote payload transformations.
//
// NOTE: Experimental
func NewPayloadHTTPHandler(options PayloadHTTPHandlerOptions) (http.Handler, error) {
	params, err := extstore.ExternalStorageToParams(options.ExternalStorage)
	if err != nil {
		return nil, err
	}

	h := &payloadHTTPHandler{
		postStorageCodecs: options.PostStorageCodecs,
		preStorageCodecs:  options.PreStorageCodecs,
		retrievalVisitor:  extstore.NewExternalRetrievalVisitor(params),
		storageVisitor:    extstore.NewExternalStorageVisitor(params),
	}
	return h, nil
}

// ServeHTTP implements [http.Handler].
func (h *payloadHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	path := r.URL.Path
	if !strings.HasSuffix(path, remotePayloadCodecDecodePath) &&
		!strings.HasSuffix(path, remotePayloadCodecEncodePath) &&
		!strings.HasSuffix(path, downloadPath) {
		http.NotFound(w, r)
		return
	}

	if r.Body == nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	bs, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var payloadspb commonpb.Payloads
	if err = protojson.Unmarshal(bs, &payloadspb); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	payloads := payloadspb.Payloads

	switch {
	case strings.HasSuffix(path, remotePayloadCodecDecodePath):
		payloads, err = h.decode(r, payloads)
	case strings.HasSuffix(path, remotePayloadCodecEncodePath):
		payloads, err = h.encode(r, payloads)
	case strings.HasSuffix(path, downloadPath):
		payloads, err = h.download(r, payloads)
	default:
		http.NotFound(w, r)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bs, err = protojson.Marshal(&commonpb.Payloads{Payloads: payloads})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err = w.Write(bs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

// decode decodes payloads through the post-storage then pre-storage codec chains.
// If preserveStorageRefs=true is set in the query string, storage references are
// returned as-is rather than being retrieved from external storage.
func (h *payloadHTTPHandler) decode(r *http.Request, payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	var err error

	payloads, err = decodePayloads(payloads, h.postStorageCodecs)
	if err != nil {
		return nil, err
	}

	if !strings.EqualFold(r.URL.Query().Get("preserveStorageRefs"), "true") {
		vpc := &proxy.VisitPayloadsContext{Context: r.Context()}
		payloads, err = h.retrievalVisitor.Visit(vpc, payloads)
		if err != nil {
			return nil, err
		}
	}

	return decodeNonReferences(payloads, h.preStorageCodecs)
}

// download retrieves payloads from external storage and decodes them through
// the pre-storage codec chain. All input payloads must be storage references.
func (h *payloadHTTPHandler) download(r *http.Request, payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	for _, p := range payloads {
		if !extstore.IsStorageReference(p) {
			return nil, errors.New("all payloads must be storage references")
		}
	}

	vpc := &proxy.VisitPayloadsContext{Context: r.Context()}
	retrieved, err := h.retrievalVisitor.Visit(vpc, payloads)
	if err != nil {
		return nil, err
	}

	return decodeNonReferences(retrieved, h.preStorageCodecs)
}

// encode encodes payloads through the pre-storage then post-storage codec chains,
// applying external storage as configured. Storage references are returned for
// payloads that meet the size threshold and driver selection criteria.
func (h *payloadHTTPHandler) encode(r *http.Request, payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	var err error

	payloads, err = encodePayloads(payloads, h.preStorageCodecs)
	if err != nil {
		return nil, err
	}

	vpc := &proxy.VisitPayloadsContext{Context: r.Context()}
	payloads, err = h.storageVisitor.Visit(vpc, payloads)
	if err != nil {
		return nil, err
	}

	payloads, err = encodePayloads(payloads, h.postStorageCodecs)
	if err != nil {
		return nil, err
	}

	return payloads, nil
}

// decodeNonReferences decodes non-storage-reference payloads through the given
// codec chain. Storage references pass through as-is.
func decodeNonReferences(payloads []*commonpb.Payload, codecs []PayloadCodec) ([]*commonpb.Payload, error) {
	result := make([]*commonpb.Payload, len(payloads))
	var nonRefIdxs []int
	var nonRefPayloads []*commonpb.Payload
	for i, p := range payloads {
		if extstore.IsStorageReference(p) {
			result[i] = p
		} else {
			nonRefIdxs = append(nonRefIdxs, i)
			nonRefPayloads = append(nonRefPayloads, p)
		}
	}
	decoded, err := decodePayloads(nonRefPayloads, codecs)
	if err != nil {
		return nil, err
	}
	for j, idx := range nonRefIdxs {
		result[idx] = decoded[j]
	}
	return result, nil
}
