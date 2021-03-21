package authentication

// AuthenticationType describes the authentication types available in scalers
type AuthenticationType string

const (
	APIKeyAuth AuthenticationType = "apiKey"
	BasicAuth  AuthenticationType = "basic"
	TLSAuth    AuthenticationType = "tls"
	BearerAuth AuthenticationType = "bearer"
)
