// Package nexus provides client and server implementations of the Nexus [HTTP API]
//
// [HTTP API]: https://github.com/nexus-rpc/api
package nexus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Package version.
const version = "v0.0.11"

const (
	// Nexus specific headers.
	headerOperationState = "Nexus-Operation-State"
	headerOperationID    = "Nexus-Operation-Id"
	headerRequestID      = "Nexus-Request-Id"
	headerLink           = "Nexus-Link"

	// HeaderRequestTimeout is the total time to complete a Nexus HTTP request.
	HeaderRequestTimeout = "Request-Timeout"
	// HeaderOperationTimeout is the total time to complete a Nexus operation.
	// Unlike HeaderRequestTimeout, this applies to the whole operation, not just a single HTTP request.
	HeaderOperationTimeout = "Operation-Timeout"
)

const contentTypeJSON = "application/json"

// Query param for passing a callback URL.
const (
	queryCallbackURL = "callback"
	// Query param for passing wait duration.
	queryWait = "wait"
)

const (
	statusOperationRunning = http.StatusPreconditionFailed
	// HTTP status code for failed operation responses.
	statusOperationFailed = http.StatusFailedDependency
	StatusUpstreamTimeout = 520
)

// A Failure represents failed handler invocations as well as `failed` or `canceled` operation results.
type Failure struct {
	// A simple text message.
	Message string `json:"message"`
	// A key-value mapping for additional context. Useful for decoding the 'details' field, if needed.
	Metadata map[string]string `json:"metadata,omitempty"`
	// Additional JSON serializable structured data.
	Details json.RawMessage `json:"details,omitempty"`
}

// UnsuccessfulOperationError represents "failed" and "canceled" operation results.
type UnsuccessfulOperationError struct {
	State   OperationState
	Failure Failure
}

// Error implements the error interface.
func (e *UnsuccessfulOperationError) Error() string {
	if e.Failure.Message != "" {
		return fmt.Sprintf("operation %s: %s", e.State, e.Failure.Message)
	}
	return fmt.Sprintf("operation %s", e.State)
}

// ErrOperationStillRunning indicates that an operation is still running while trying to get its result.
var ErrOperationStillRunning = errors.New("operation still running")

// OperationInfo conveys information about an operation.
type OperationInfo struct {
	// ID of the operation.
	ID string `json:"id"`
	// State of the operation.
	State OperationState `json:"state"`
}

// OperationState represents the variable states of an operation.
type OperationState string

const (
	// "running" operation state. Indicates an operation is started and not yet completed.
	OperationStateRunning OperationState = "running"
	// "succeeded" operation state. Indicates an operation completed successfully.
	OperationStateSucceeded OperationState = "succeeded"
	// "failed" operation state. Indicates an operation completed as failed.
	OperationStateFailed OperationState = "failed"
	// "canceled" operation state. Indicates an operation completed as canceled.
	OperationStateCanceled OperationState = "canceled"
)

// isMediaTypeJSON returns true if the given content type's media type is application/json.
func isMediaTypeJSON(contentType string) bool {
	if contentType == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "application/json"
}

// isMediaTypeOctetStream returns true if the given content type's media type is application/octet-stream.
func isMediaTypeOctetStream(contentType string) bool {
	if contentType == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "application/octet-stream"
}

// Header is a mapping of string to string.
// It is used throughout the framework to transmit metadata.
type Header map[string]string

// Get is a case-insensitive key lookup from the header map.
func (h Header) Get(k string) string {
	return h[strings.ToLower(k)]
}

func prefixStrippedHTTPHeaderToNexusHeader(httpHeader http.Header, prefix string) Header {
	header := Header{}
	for k, v := range httpHeader {
		lowerK := strings.ToLower(k)
		if strings.HasPrefix(lowerK, prefix) {
			// Nexus headers can only have single values, ignore multiple values.
			header[lowerK[len(prefix):]] = v[0]
		}
	}
	return header
}

func addContentHeaderToHTTPHeader(nexusHeader Header, httpHeader http.Header) http.Header {
	for k, v := range nexusHeader {
		httpHeader.Set("Content-"+k, v)
	}
	return httpHeader
}

func addCallbackHeaderToHTTPHeader(nexusHeader Header, httpHeader http.Header) http.Header {
	for k, v := range nexusHeader {
		httpHeader.Set("Nexus-Callback-"+k, v)
	}
	return httpHeader
}

func addLinksToHTTPHeader(links []Link, httpHeader http.Header) error {
	for _, link := range links {
		encodedLink, err := encodeLink(link)
		if err != nil {
			return err
		}
		httpHeader.Add(headerLink, encodedLink)
	}
	return nil
}

func getLinksFromHeader(httpHeader http.Header) ([]Link, error) {
	var links []Link
	headerValues := httpHeader.Values(headerLink)
	if len(headerValues) == 0 {
		return nil, nil
	}
	for _, encodedLink := range strings.Split(strings.Join(headerValues, ","), ",") {
		link, err := decodeLink(encodedLink)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, nil
}

func httpHeaderToNexusHeader(httpHeader http.Header, excludePrefixes ...string) Header {
	header := Header{}
headerLoop:
	for k, v := range httpHeader {
		lowerK := strings.ToLower(k)
		for _, prefix := range excludePrefixes {
			if strings.HasPrefix(lowerK, prefix) {
				continue headerLoop
			}
		}
		// Nexus headers can only have single values, ignore multiple values.
		header[lowerK] = v[0]
	}
	return header
}

func addNexusHeaderToHTTPHeader(nexusHeader Header, httpHeader http.Header) http.Header {
	for k, v := range nexusHeader {
		httpHeader.Set(k, v)
	}
	return httpHeader
}

func addContextTimeoutToHTTPHeader(ctx context.Context, httpHeader http.Header) http.Header {
	deadline, ok := ctx.Deadline()
	if !ok {
		return httpHeader
	}
	httpHeader.Set(HeaderRequestTimeout, formatDuration(time.Until(deadline)))
	return httpHeader
}

// Link contains an URL and a Type that can be used to decode the URL.
// Links can contain any arbitrary information as a percent-encoded URL.
// It can be used to pass information about the caller to the handler, or vice-versa.
type Link struct {
	// URL information about the link.
	// It must be URL percent-encoded.
	URL *url.URL
	// Type can describe an actual data type for decoding the URL.
	// Valid chars: alphanumeric, '_', '.', '/'
	Type string
}

const linkTypeKey = "type"

// decodeLink encodes the link to Nexus-Link header value.
// It follows the same format of HTTP Link header: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Link
func encodeLink(link Link) (string, error) {
	if err := validateLinkURL(link.URL); err != nil {
		return "", fmt.Errorf("failed to encode link: %w", err)
	}
	if err := validateLinkType(link.Type); err != nil {
		return "", fmt.Errorf("failed to encode link: %w", err)
	}
	return fmt.Sprintf(`<%s>; %s="%s"`, link.URL.String(), linkTypeKey, link.Type), nil
}

// decodeLink decodes the Nexus-Link header values.
// It must have the same format of HTTP Link header: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Link
func decodeLink(encodedLink string) (Link, error) {
	var link Link
	encodedLink = strings.TrimSpace(encodedLink)
	if len(encodedLink) == 0 {
		return link, fmt.Errorf("failed to parse link header: value is empty")
	}

	if encodedLink[0] != '<' {
		return link, fmt.Errorf("failed to parse link header: invalid format: %s", encodedLink)
	}
	urlEnd := strings.Index(encodedLink, ">")
	if urlEnd == -1 {
		return link, fmt.Errorf("failed to parse link header: invalid format: %s", encodedLink)
	}
	urlStr := strings.TrimSpace(encodedLink[1:urlEnd])
	if len(urlStr) == 0 {
		return link, fmt.Errorf("failed to parse link header: url is empty")
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return link, fmt.Errorf("failed to parse link header: invalid url: %s", urlStr)
	}
	if err := validateLinkURL(u); err != nil {
		return link, fmt.Errorf("failed to parse link header: %w", err)
	}
	link.URL = u

	params := strings.Split(encodedLink[urlEnd+1:], ";")
	// must contain at least one semi-colon, and first param must be empty since
	// it corresponds to the url part parsed above.
	if len(params) < 2 {
		return link, fmt.Errorf("failed to parse link header: invalid format: %s", encodedLink)
	}
	if strings.TrimSpace(params[0]) != "" {
		return link, fmt.Errorf("failed to parse link header: invalid format: %s", encodedLink)
	}

	typeKeyFound := false
	for _, param := range params[1:] {
		param = strings.TrimSpace(param)
		if len(param) == 0 {
			return link, fmt.Errorf("failed to parse link header: parameter is empty: %s", encodedLink)
		}
		kv := strings.SplitN(param, "=", 2)
		if len(kv) != 2 {
			return link, fmt.Errorf("failed to parse link header: invalid parameter format: %s", param)
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		if strings.HasPrefix(val, `"`) != strings.HasSuffix(val, `"`) {
			return link, fmt.Errorf(
				"failed to parse link header: parameter value missing double-quote: %s",
				param,
			)
		}
		if strings.HasPrefix(val, `"`) {
			val = val[1 : len(val)-1]
		}
		if key == linkTypeKey {
			if err := validateLinkType(val); err != nil {
				return link, fmt.Errorf("failed to parse link header: %w", err)
			}
			link.Type = val
			typeKeyFound = true
		}
	}
	if !typeKeyFound {
		return link, fmt.Errorf(
			"failed to parse link header: %q key not found: %s",
			linkTypeKey,
			encodedLink,
		)
	}

	return link, nil
}

func validateLinkURL(value *url.URL) error {
	if value == nil || value.String() == "" {
		return fmt.Errorf("url is empty")
	}
	_, err := url.ParseQuery(value.RawQuery)
	if err != nil {
		return fmt.Errorf("url query not percent-encoded: %s", value)
	}
	return nil
}

func validateLinkType(value string) error {
	if len(value) == 0 {
		return fmt.Errorf("link type is empty")
	}
	for _, c := range value {
		if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && !(c >= '0' && c <= '9') && c != '_' && c != '.' && c != '/' {
			return fmt.Errorf("link type contains invalid char (valid chars: alphanumeric, '_', '.', '/')")
		}
	}
	return nil
}

var durationRegexp = regexp.MustCompile(`^(\d+(?:\.\d+)?)(ms|s|m)$`)

func parseDuration(value string) (time.Duration, error) {
	m := durationRegexp.FindStringSubmatch(value)
	if len(m) == 0 {
		return 0, fmt.Errorf("invalid duration: %q", value)
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, err
	}

	switch m[2] {
	case "ms":
		return time.Millisecond * time.Duration(v), nil
	case "s":
		return time.Millisecond * time.Duration(v*1e3), nil
	case "m":
		return time.Millisecond * time.Duration(v*1e3*60), nil
	}
	panic("unreachable")
}

// formatDuration converts a duration into a string representation in millisecond resolution.
func formatDuration(d time.Duration) string {
	return strconv.FormatInt(d.Milliseconds(), 10) + "ms"
}
