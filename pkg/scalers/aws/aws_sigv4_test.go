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
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kedacore/keda/v2/pkg/util"
)

func TestSigV4RoundTripper(t *testing.T) {
	transport := util.CreateHTTPTransport(false)

	cli := &http.Client{Transport: transport}

	req, err := http.NewRequest(http.MethodGet, "https://aps-workspaces.us-west-2.amazonaws.com/workspaces/ws-38377ca8-8db3-4b58-812d-b65a81837bb8/api/v1/query?query=vector(10)", strings.NewReader("Hello, world!"))
	require.NoError(t, err)
	r, err := cli.Do(req)
	require.NotEmpty(t, r)
	require.NoError(t, err)
	defer r.Body.Close()

	require.NotNil(t, req)
}
