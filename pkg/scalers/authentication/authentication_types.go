package authentication

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
