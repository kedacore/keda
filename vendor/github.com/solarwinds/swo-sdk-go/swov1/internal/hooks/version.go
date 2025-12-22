package hooks

import "net/http"

// APIVersion is the SWO API version extracted from gen.lock at build time
const APIVersion = "1.0.11"

type VersionHook struct{}

var (
	_ beforeRequestHook = (*VersionHook)(nil)
)

func (v *VersionHook) BeforeRequest(_ BeforeRequestContext, req *http.Request) (*http.Request, error) {
	req.Header.Set("X-SWO-API-Version", APIVersion)
	return req, nil
}
