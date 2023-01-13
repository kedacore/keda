package azure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	"github.com/kedacore/keda/v2/pkg/util"
)

const (
	MSIURL             = "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=%s"
	MSIURLWithClientID = "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=%s&client_id=%s"
)

var globalHTTPTimeout time.Duration

func init() {
	valueStr, found := os.LookupEnv("KEDA_HTTP_DEFAULT_TIMEOUT")
	globalHTTPTimeoutMS := 3000
	if found && valueStr != "" {
		value, err := strconv.Atoi(valueStr)
		if err == nil {
			globalHTTPTimeoutMS = value
		}
	}
	globalHTTPTimeout = time.Duration(globalHTTPTimeoutMS) * time.Millisecond
}

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
		return token, fmt.Errorf("error getting aad-pod-identity token - %w", err)
	}
	req.Header = map[string][]string{
		"Metadata": {"true"},
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return token, fmt.Errorf("error getting aad-pod-identity token - %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return token, fmt.Errorf("error getting aad-pod-identity token - %w", err)
	}

	err = json.Unmarshal(body, &token)
	if err != nil {
		return token, fmt.Errorf("error getting aad-pod-identity token - %w", errors.New(string(body)))
	}

	return token, nil
}

type ManagedIdentityWrapper struct {
	cred *azidentity.ManagedIdentityCredential
}

func ManagedIdentityWrapperCredential(clientID string) (*ManagedIdentityWrapper, error) {
	opts := &azidentity.ManagedIdentityCredentialOptions{}
	if clientID != "" {
		opts.ID = azidentity.ClientID(clientID)
	}

	msiCred, err := azidentity.NewManagedIdentityCredential(opts)
	if err != nil {
		return nil, err
	}
	return &ManagedIdentityWrapper{
		cred: msiCred,
	}, nil
}

func (w *ManagedIdentityWrapper) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	c, cancel := context.WithTimeout(ctx, globalHTTPTimeout)
	defer cancel()
	tk, err := w.cred.GetToken(c, opts)
	if ctxErr := c.Err(); errors.Is(ctxErr, context.DeadlineExceeded) {
		// timeout: signal the chain to try its next credential, if any
		err = azidentity.NewCredentialUnavailableError("managed identity timed out")
	}
	return tk, err
}
