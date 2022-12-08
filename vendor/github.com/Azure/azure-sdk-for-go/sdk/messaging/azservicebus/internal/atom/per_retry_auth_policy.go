// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package atom

import (
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/auth"
)

type perRetryAuthPolicy struct {
	tp auth.TokenProvider
}

type ctxWithAuthKey struct{}

// Do applies the policy to the specified Request.  When implementing a Policy, mutate the
// request before calling req.Next() to move on to the next policy, and respond to the result
// before returning to the caller.
func (p *perRetryAuthPolicy) Do(req *policy.Request) (*http.Response, error) {
	if err := p.AddTokenHeader("Authorization", req.Raw(), req.Raw().URL.String()); err != nil {
		return nil, err
	}

	executeOptions, ok := req.Raw().Context().Value(ctxWithAuthKey{}).(*ExecuteOptions)

	if ok && executeOptions != nil {
		if executeOptions.ForwardTo != nil {
			if err := p.AddTokenHeader("ServiceBusSupplementaryAuthorization", req.Raw(), *executeOptions.ForwardTo); err != nil {
				return nil, err
			}
		}

		if executeOptions.ForwardToDeadLetter != nil {
			if err := p.AddTokenHeader("ServiceBusDlqSupplementaryAuthorization", req.Raw(), *executeOptions.ForwardToDeadLetter); err != nil {
				return nil, err
			}
		}
	}

	return req.Next()
}

func (p *perRetryAuthPolicy) AddTokenHeader(headerName string, req *http.Request, targetURI string) error {
	signature, err := p.tp.GetToken(targetURI)

	if err != nil {
		return err
	}

	req.Header.Add(headerName, signature.Token)
	return nil
}
