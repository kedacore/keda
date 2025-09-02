// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package internal

import (
	"fmt"
	"net/http"
	"strings"
)

// FakeChallenge is used by fake servers to fake the authentication challenge.
type FakeChallenge struct{}

// Do initiates an authentication challenge IFF req doesn't contain an Authorization header.
// When the last return value is true, the fake server will use the returned response/error.
func (m *FakeChallenge) Do(req *http.Request) (*http.Response, error, bool) {
	if req.Header.Get("Authorization") != "" {
		// presence of an authorization header means we don't need to elicit a challenge
		return nil, nil, false
	}

	resp := &http.Response{
		Request:    req,
		Status:     "fake unauthorized",
		StatusCode: http.StatusUnauthorized,
		Body:       http.NoBody,
		Header:     http.Header{},
	}

	s := strings.Split(req.URL.Host, ".")
	resource := fmt.Sprintf("%s://%s", req.URL.Scheme, strings.Join(s[len(s)-3:], "."))
	resp.Header.Set("WWW-Authenticate", fmt.Sprintf(`Bearer authorization="https://fake.local/tenant" resource="%s"`, resource))

	return resp, nil, true
}
