package scalers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/amp"

	awsutils "github.com/kedacore/keda/v2/pkg/scalers/aws"
	httputils "github.com/kedacore/keda/v2/pkg/util"
)

type awsConfigMetadata struct {
	awsRegion        string
	awsAuthorization awsutils.AuthorizationMetadata
}

// Custom round tripper to sign requests
type roundTripper struct {
	client *amp.Client
	region string
}

var (
	// ErrAwsAMPNoAwsRegion is returned when "awsRegion" is missing from the config.
	ErrAwsAMPNoAwsRegion = errors.New("no awsRegion given")
)

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	cred, err := rt.client.Options().Credentials.Retrieve(req.Context())
	if err != nil {
		return nil, err
	}
	// Sign request
	hasher := sha256.New()
	reqCxt := v4.SetPayloadHash(req.Context(), hex.EncodeToString(hasher.Sum(nil)))
	reqHash := v4.GetPayloadHash(reqCxt)
	err = rt.client.Options().HTTPSignerV4.SignHTTP(req.Context(), cred, req, reqHash, "aps", rt.region, time.Now())
	if err != nil {
		return nil, err
	}
	// Create default transport
	transport := httputils.CreateHTTPTransport(false)

	// Send signed request
	return transport.RoundTrip(req)
}

func parseAwsAMPMetadata(config *ScalerConfig) (*awsConfigMetadata, error) {
	meta := awsConfigMetadata{}

	auth, err := awsutils.GetAwsAuthorization(config.TriggerUniqueKey, config.PodIdentity, config.TriggerMetadata, config.AuthParams, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}

	meta.awsAuthorization = auth
	return &meta, nil
}

// NewSigV4RoundTripper returns a new http.RoundTripper that will sign requests
// using Amazon's Signature Verification V4 signing procedure. The request will
// then be handed off to the next RoundTripper provided by next. If next is nil,
// http.DefaultTransport will be used.
//
// Credentials for signing are retrieving used the default AWS credential chain.
// If credentials could not be found, an error will be returned.
func NewSigV4RoundTripper(config *ScalerConfig) (http.RoundTripper, error) {
	metadata, err := parseAwsAMPMetadata(config)
	if err != nil {
		return nil, err
	}
	awsCfg, err := awsutils.GetAwsConfig(context.Background(), metadata.awsRegion, metadata.awsAuthorization)
	if err != nil {
		return nil, err
	}

	client := amp.NewFromConfig(*awsCfg, func(o *amp.Options) {})
	rt := &roundTripper{
		client: client,
		region: metadata.awsRegion,
	}

	return rt, nil
}
