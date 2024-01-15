package scalers

import (
	"net/http"
	"strings"
	"testing"

	"github.com/kedacore/keda/v2/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestSigV4RoundTripper(t *testing.T) {
	// rt := &roundTripper{
	// 	client: amp.New(nil),
	// 	region: "us-west-2",
	// }

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
