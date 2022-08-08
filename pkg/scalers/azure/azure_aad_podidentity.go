package azure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/kedacore/keda/v2/pkg/util"
)

const (
	MSIURL             = "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=%s"
	MSIURLWithClientID = "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=%s&client_id=%s"
)

// GetAzureADPodIdentityToken returns the AADToken for resource
func GetAzureADPodIdentityToken(ctx context.Context, httpClient util.HTTPDoer, identityID, audience string) (AADToken, error) {
	var token AADToken

	var urlStr string
	if identityID == "" {
		urlStr = fmt.Sprintf(MSIURL, url.QueryEscape(audience))
	} else {
		urlStr = fmt.Sprintf(MSIURLWithClientID, url.QueryEscape(audience), url.QueryEscape(identityID))
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return token, err
	}
	req.Header = map[string][]string{
		"Metadata": {"true"},
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return token, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return token, err
	}

	err = json.Unmarshal(body, &token)
	if err != nil {
		return token, errors.New(string(body))
	}

	return token, nil
}
