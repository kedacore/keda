/*
Copyright 2024 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
This file contains all the logic for caching aws.Config across all the (AWS)
triggers. The first time when an aws.Config is requested, it's cached based on
the authentication info (roleArn, Key&Secret, keda itself) and it's returned
every time when an aws.Config is requested for the same authentication info.
This is required because if we don't cache and share them, each scaler
generates and refresh it's own token although all the tokens grants the same
permissions
*/
package aws

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/amp"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	httputils "github.com/kedacore/keda/v2/pkg/util"
)

// roundTripper adds custom round tripper to sign requests
type roundTripper struct {
	client *amp.Client
}

var (
	// ErrAwsAMPNoAwsRegion is returned when "awsRegion" is missing from the config.
	ErrAwsAMPNoAwsRegion = errors.New("no awsRegion given")
)

// RoundTrip adds the roundTrip logic so that the request is SigV4 signed
func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	cred, err := rt.client.Options().Credentials.Retrieve(req.Context())
	if err != nil {
		return nil, err
	}

	// We need to sign the request because giving an empty string (not a hashed empty string)
	// fails in the backend as not signed request hence the following value is used
	// "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" is the sha256 of ""
	const reqHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	err = rt.client.Options().HTTPSignerV4.SignHTTP(req.Context(), cred, req, reqHash, "aps", rt.client.Options().Region, time.Now())
	if err != nil {
		return nil, err
	}
	// Create default transport
	transport := httputils.CreateHTTPTransport(false)

	// Send signed request
	return transport.RoundTrip(req)
}

// parseAwsAMPMetadata parses the data to get the AWS specific auth info and metadata
func parseAwsAMPMetadata(config *scalersconfig.ScalerConfig, awsRegion string) (*AuthorizationMetadata, error) {
	auth, err := GetAwsAuthorization(config.TriggerUniqueKey, awsRegion, config.PodIdentity, config.TriggerMetadata, config.AuthParams, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}
	return &auth, nil
}

// NewSigV4RoundTripper returns a new http.RoundTripper that will sign requests
// using Amazon's Signature Verification V4 signing procedure. The request will
// then be handed off to the next RoundTripper provided by next. If next is nil,
// http.DefaultTransport will be used.
//
// Credentials for signing are retrieving used the default AWS credential chain.
// If credentials could not be found, an error will be returned.
func NewSigV4RoundTripper(config *scalersconfig.ScalerConfig, awsRegion string) (http.RoundTripper, error) {
	// parseAwsAMPMetadata can return an error if AWS info is missing
	// but this can happen if we check for them on not AWS scalers
	// which is probably the reason to create a SigV4RoundTripper.
	// To prevent failures we check if the metadata is nil
	// (missing AWS info) and we hide the error
	awsAuthorization, _ := parseAwsAMPMetadata(config, awsRegion)
	if awsAuthorization == nil {
		return nil, nil
	}
	awsCfg, err := GetAwsConfig(context.Background(), *awsAuthorization)
	if err != nil {
		return nil, err
	}

	client := amp.NewFromConfig(*awsCfg, func(_ *amp.Options) {})
	rt := &roundTripper{
		client: client,
	}

	return rt, nil
}
