package authentication

import "time"

// Type describes the authentication type used in a scaler
type Type string

const (
	// APIKeyAuthType is a auth type using an API key
	APIKeyAuthType Type = "apiKey"
	// BasicAuthType is a auth type using basic auth
	BasicAuthType Type = "basic"
	// TLSAuthType is a auth type using TLS
	TLSAuthType Type = "tls"
	// BearerAuthType is a auth type using a bearer token
	BearerAuthType Type = "bearer"
)

// TransportType is type of http transport
type TransportType int

const (
	NetHTTP  TransportType = iota // NetHTTP standard Go net/http client.
	FastHTTP                      // FastHTTP Fast http client.
)

type AuthMeta struct {
	// bearer auth
	EnableBearerAuth bool
	BearerToken      string

	// basic auth
	EnableBasicAuth bool
	Username        string
	Password        string // +optional

	// client certification
	EnableTLS bool
	Cert      string
	Key       string
	CA        string
}

type HTTPTransport struct {
	MaxIdleConnDuration time.Duration
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
}
