package authentication

type AuthenticationType string

const (
	ApiKeyAuth AuthenticationType = "apiKey"
	BasicAuth  AuthenticationType = "basic"
	TlsAuth    AuthenticationType = "tls"
	BearerAuth AuthenticationType = "bearer"
)
