// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package shared

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

// NOTE: This is duplication of the azcore includeResponsePolicy, this is necessary for some APIs like directory/file
// 		 GetProperties. Under the hood, these APIs grab the raw response to construct the datalake properties
//		 (owner, group, permission) in addition to the blob properties. If we use the includeResponsePolicy, this can
//		 result in the customer not being able to retrieve the response themselves.

// CtxIncludeBlobResponseKey is used as a context key for retrieving the raw response.
type CtxIncludeBlobResponseKey struct{}

type includeBlobResponsePolicy struct {
}

// NewIncludeBlobResponsePolicy creates a policy that retrieves the raw HTTP response upon request
func NewIncludeBlobResponsePolicy() policy.Policy {
	return &includeBlobResponsePolicy{}
}

func (p *includeBlobResponsePolicy) Do(req *policy.Request) (*http.Response, error) {
	resp, err := req.Next()
	if resp == nil {
		return resp, err
	}
	if httpOutRaw := req.Raw().Context().Value(CtxIncludeBlobResponseKey{}); httpOutRaw != nil {
		httpOut := httpOutRaw.(**http.Response)
		*httpOut = resp
	}
	return resp, err
}

// WithCaptureBlobResponse applies the HTTP response retrieval annotation to the parent context.
// The resp parameter will contain the HTTP response after the request has completed.
func WithCaptureBlobResponse(parent context.Context, resp **http.Response) context.Context {
	return context.WithValue(parent, CtxIncludeBlobResponseKey{}, resp)
}
