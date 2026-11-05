package scalers

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"golang.org/x/oauth2"

	"github.com/kedacore/keda/v2/pkg/scalers/gcp"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

func TestGCSServiceAccountStorageRedirects(t *testing.T) {
	for _, tc := range []struct {
		name     string
		redirect string
		wantErr  string
	}{
		{name: "same origin", redirect: "same-origin"},
		{name: "different port", redirect: "different-port", wantErr: "redirect to a different origin"},
		{name: "HTTPS downgrade", redirect: "downgrade", wantErr: "redirect to a different origin"},
		{name: "redirect loop", redirect: "loop", wantErr: "stopped after 10 redirects"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var externalRequests atomic.Int32
			externalHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				externalRequests.Add(1)
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"items":[{"size":"1"}]}`)
			})
			externalHTTPS := httptest.NewTLSServer(externalHandler)
			t.Cleanup(externalHTTPS.Close)
			externalHTTP := httptest.NewServer(externalHandler)
			t.Cleanup(externalHTTP.Close)

			var storageRequests atomic.Int32
			endpoint := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				storageRequests.Add(1)
				assert.Equal(t, "Bearer storage-access-token", r.Header.Get("Authorization"))
				if r.URL.Path == "/objects" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"items":[{"size":"1"}]}`)
					return
				}
				assert.Equal(t, "/storage/v1/b/tenant-bucket/o", r.URL.Path)
				location := "/objects"
				switch tc.redirect {
				case "different-port":
					location = strings.Replace(externalHTTPS.URL, "127.0.0.1", "storage.googleapis.com", 1) + "/objects"
				case "downgrade":
					location = strings.Replace(externalHTTP.URL, "127.0.0.1", "storage.googleapis.com", 1) + "/objects"
				case "loop":
					location = "/storage/v1/b/tenant-bucket/o"
				}
				http.Redirect(w, r, location, http.StatusFound)
			}))
			t.Cleanup(endpoint.Close)
			t.Setenv("STORAGE_EMULATOR_HOST", "")
			t.Setenv("GOOGLE_API_USE_CLIENT_CERTIFICATE", "false")
			t.Setenv("GOOGLE_API_USE_MTLS_ENDPOINT", "never")
			t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", filepath.Join(t.TempDir(), "missing-operator-credentials.json"))
			transport := endpoint.Client().Transport.(*http.Transport).Clone()
			transport.Proxy = nil
			// Keep the real storage URLs while routing data requests to trusted
			// local TLS fixtures, without enabling unauthenticated emulator mode.
			transport.TLSClientConfig.ServerName = endpoint.Certificate().DNSNames[0]
			transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(address)
				if err != nil {
					return nil, err
				}
				if host != "storage.googleapis.com" {
					return nil, errors.New("unexpected storage request host")
				}
				if port == "443" {
					address = endpoint.Listener.Addr().String()
				} else {
					address = net.JoinHostPort("127.0.0.1", port)
				}
				return (&net.Dialer{}).DialContext(ctx, network, address)
			}
			previousTransport := http.DefaultTransport
			http.DefaultTransport = transport
			t.Cleanup(func() { http.DefaultTransport = previousTransport })

			var exchanges atomic.Int32
			stsClient := &http.Client{Transport: gcpServiceAccountRoundTripper(func(r *http.Request) (*http.Response, error) {
				exchanges.Add(1)
				assert.Equal(t, "https://sts.googleapis.com/v1/token", r.URL.String())
				require.NoError(t, r.ParseForm())
				assert.Equal(t, storage.ScopeReadOnly, r.Form.Get("scope"))
				assert.Equal(t, "storage-assertion", r.Form.Get("subject_token"))
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"access_token":"storage-access-token","token_type":"Bearer","expires_in":3600}`)),
				}, nil
			})}
			auth := &gcp.AuthorizationMetadata{ServiceAccountTokenProvider: &scalersconfig.ServiceAccountTokenProvider{
				Audience: "https://iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/kubernetes/providers/cluster",
				GetToken: func(context.Context) (string, error) { return "storage-assertion", nil },
			}}
			client, err := newGcsServiceAccountClient(context.WithValue(t.Context(), oauth2.HTTPClient, stsClient), auth)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, client.Close()) })
			scaler := &gcsScaler{
				client: client, bucket: client.Bucket("tenant-bucket"), metadata: &gcsMetadata{}, logger: logr.Discard(),
			}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			count, err := scaler.getItemCount(ctx, 10)
			if tc.wantErr == "" {
				require.NoError(t, err)
				assert.Equal(t, int64(1), count)
				assert.Equal(t, int32(2), storageRequests.Load())
			} else {
				require.ErrorContains(t, err, tc.wantErr)
				assert.Zero(t, count)
			}
			assert.Equal(t, int32(1), exchanges.Load())
			assert.Zero(t, externalRequests.Load(), "storage credentials must not leave the original origin")
		})
	}
}

func TestGCSServiceAccountStorageEmulatorDoesNotUseCredentials(t *testing.T) {
	var requests atomic.Int32
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		assert.Empty(t, r.Header.Get("Authorization"), "emulator requests must remain unauthenticated")
		assert.Equal(t, "/storage/v1/b/tenant-bucket/o", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"items":[{"size":"1"}]}`)
	}))
	t.Cleanup(endpoint.Close)
	t.Setenv("STORAGE_EMULATOR_HOST", endpoint.URL)
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", filepath.Join(t.TempDir(), "missing-operator-credentials.json"))
	var tokenRequests atomic.Int32
	auth := &gcp.AuthorizationMetadata{ServiceAccountTokenProvider: &scalersconfig.ServiceAccountTokenProvider{
		Audience: "https://iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/kubernetes/providers/cluster",
		GetToken: func(context.Context) (string, error) {
			tokenRequests.Inc()
			return "", errors.New("unexpected federation token request")
		},
	}}
	client, err := newGcsServiceAccountClient(t.Context(), auth)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	scaler := &gcsScaler{
		client: client, bucket: client.Bucket("tenant-bucket"), metadata: &gcsMetadata{}, logger: logr.Discard(),
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	count, err := scaler.getItemCount(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	assert.Equal(t, int32(1), requests.Load())
	assert.Zero(t, tokenRequests.Load())
}
