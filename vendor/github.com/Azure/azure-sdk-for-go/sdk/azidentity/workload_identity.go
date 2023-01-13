//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azidentity

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

const credNameWorkloadIdentity = "WorkloadIdentityCredential"

// WorkloadIdentityCredential supports Azure workload identity on Kubernetes.
// See [AKS documentation] for more information.
//
// [AKS documentation]: https://learn.microsoft.com/azure/aks/workload-identity-overview
type WorkloadIdentityCredential struct {
	assertion, file string
	cred            *ClientAssertionCredential
	expires         time.Time
	mtx             *sync.RWMutex
}

// WorkloadIdentityCredentialOptions contains optional parameters for WorkloadIdentityCredential.
type WorkloadIdentityCredentialOptions struct {
	azcore.ClientOptions
}

// NewWorkloadIdentityCredential constructs a WorkloadIdentityCredential. tenantID and clientID specify the identity the credential authenticates.
// file is a path to a file containing a Kubernetes service account token that authenticates the identity.
func NewWorkloadIdentityCredential(tenantID, clientID, file string, options *WorkloadIdentityCredentialOptions) (*WorkloadIdentityCredential, error) {
	if options == nil {
		options = &WorkloadIdentityCredentialOptions{}
	}
	w := WorkloadIdentityCredential{file: file, mtx: &sync.RWMutex{}}
	cred, err := NewClientAssertionCredential(tenantID, clientID, w.getAssertion, &ClientAssertionCredentialOptions{ClientOptions: options.ClientOptions})
	if err != nil {
		return nil, err
	}
	cred.name = credNameWorkloadIdentity
	w.cred = cred
	return &w, nil
}

// GetToken requests an access token from Azure Active Directory. Azure SDK clients call this method automatically.
func (w *WorkloadIdentityCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return w.cred.GetToken(ctx, opts)
}

// getAssertion returns the specified file's content, which is expected to be a Kubernetes service account token.
// Kubernetes is responsible for updating the file as service account tokens expire.
func (w *WorkloadIdentityCredential) getAssertion(context.Context) (string, error) {
	w.mtx.RLock()
	if w.expires.Before(time.Now()) {
		// ensure only one goroutine at a time updates the assertion
		w.mtx.RUnlock()
		w.mtx.Lock()
		defer w.mtx.Unlock()
		// double check because another goroutine may have acquired the write lock first and done the update
		if now := time.Now(); w.expires.Before(now) {
			content, err := os.ReadFile(w.file)
			if err != nil {
				return "", err
			}
			w.assertion = string(content)
			// Kubernetes rotates service account tokens when they reach 80% of their total TTL. The shortest TTL
			// is 1 hour. That implies the token we just read is valid for at least 12 minutes (20% of 1 hour),
			// but we add some margin for safety.
			w.expires = now.Add(10 * time.Minute)
		}
	} else {
		defer w.mtx.RUnlock()
	}
	return w.assertion, nil
}
