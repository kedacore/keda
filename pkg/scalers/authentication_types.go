package scalers

type authenticationType string

const (
	apiKeyAuth authenticationType = "apiKey"
	basicAuth  authenticationType = "basic"
	tlsAuth    authenticationType = "tls"
)
