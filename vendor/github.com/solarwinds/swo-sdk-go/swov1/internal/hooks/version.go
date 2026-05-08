package hooks

import (
	"net/http"
	"strings"

	"github.com/solarwinds/swo-sdk-go/swov1/internal/config"
)

const (
	APIVersionHeader = "X-SWO-API-Version"
)

var (
	_ sdkInitHook       = (*VersionHook)(nil)
	_ config.HTTPClient = (*VersionClient)(nil)
)

type VersionClient struct {
	client  config.HTTPClient
	version string
}

func (v *VersionClient) Do(req *http.Request) (*http.Response, error) {
	clonedReq := req.Clone(req.Context())
	clonedReq.Header.Set(APIVersionHeader, v.version)
	return v.client.Do(clonedReq)
}

type VersionHook struct{}

func (v *VersionHook) SDKInit(config config.SDKConfiguration) config.SDKConfiguration {
	parts := strings.Split(config.UserAgent, " ")
	if len(parts) < 5 {
		// Someone modified the UserAgent somewhere or the User-Agent format has changed.
		return config
	}
	docVersion := "unknown"
	if len(parts) > 3 {
		docVersion = parts[3]
	}

	config.Client = &VersionClient{client: config.Client, version: docVersion}

	return config
}
