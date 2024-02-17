package azure

import (
	"context"
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
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
