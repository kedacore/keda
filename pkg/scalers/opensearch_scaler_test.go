package scalers

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

func TestOpensearchMetadataValidate(t *testing.T) {
	tests := []struct {
		name    string
		meta    opensearchMetadata
		wantErr string
	}{
		{
			name: "valid: TLS enabled with clientCert and clientKey",
			meta: opensearchMetadata{
				Addresses:          []string{"https://localhost:9200"},
				EnableTLS:          true,
				ClientCert:         "cert-data",
				ClientKey:          "key-data",
				SearchTemplateName: "my-template",
				ValueLocation:      "hits.total.value",
				TargetValue:        10,
			},
		},
		{
			name: "valid: basic auth with username and password",
			meta: opensearchMetadata{
				Addresses:     []string{"http://localhost:9200"},
				Username:      "admin",
				Password:      "secret",
				Query:         `{"query":{"match_all":{}}}`,
				ValueLocation: "hits.total.value",
				TargetValue:   10,
			},
		},
		{
			name: "invalid: enableTLS true but clientCert missing",
			meta: opensearchMetadata{
				Addresses:     []string{"https://localhost:9200"},
				EnableTLS:     true,
				ClientKey:     "key-data",
				Query:         `{"query":{"match_all":{}}}`,
				ValueLocation: "hits.total.value",
				TargetValue:   10,
			},
			wantErr: "both clientCert and clientKey must be provided when enableTLS is true",
		},
		{
			name: "invalid: enableTLS true but clientKey missing",
			meta: opensearchMetadata{
				Addresses:     []string{"https://localhost:9200"},
				EnableTLS:     true,
				ClientCert:    "cert-data",
				Query:         `{"query":{"match_all":{}}}`,
				ValueLocation: "hits.total.value",
				TargetValue:   10,
			},
			wantErr: "both clientCert and clientKey must be provided when enableTLS is true",
		},
		{
			name: "invalid: basic auth without username",
			meta: opensearchMetadata{
				Addresses:     []string{"http://localhost:9200"},
				Password:      "secret",
				Query:         `{"query":{"match_all":{}}}`,
				ValueLocation: "hits.total.value",
				TargetValue:   10,
			},
			wantErr: "both username and password must be provided for basic auth",
		},
		{
			name: "invalid: basic auth without password",
			meta: opensearchMetadata{
				Addresses:     []string{"http://localhost:9200"},
				Username:      "admin",
				Query:         `{"query":{"match_all":{}}}`,
				ValueLocation: "hits.total.value",
				TargetValue:   10,
			},
			wantErr: "both username and password must be provided for basic auth",
		},
		{
			name: "invalid: neither searchTemplateName nor query provided",
			meta: opensearchMetadata{
				Username:      "admin",
				Password:      "secret",
				ValueLocation: "hits.total.value",
				TargetValue:   10,
			},
			wantErr: "either searchTemplateName or query must be provided",
		},
		{
			name: "invalid: both searchTemplateName and query provided",
			meta: opensearchMetadata{
				Username:           "admin",
				Password:           "secret",
				SearchTemplateName: "my-template",
				Query:              `{"query":{"match_all":{}}}`,
				ValueLocation:      "hits.total.value",
				TargetValue:        10,
			},
			wantErr: "cannot provide both searchTemplateName and query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.meta.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestParseOpensearchMetadata(t *testing.T) {
	// baseConfig builds a minimal valid TriggerMetadata map.
	// Pass overrides to replace a value, or "" to delete a key.
	baseConfig := func(overrides map[string]string) map[string]string {
		base := map[string]string{
			"addresses":     "http://localhost:9200",
			"username":      "admin",
			"password":      "secret",
			"index":         "my-index",
			"query":         `{"query":{"match_all":{}}}`,
			"valueLocation": "hits.total.value",
			"targetValue":   "10",
		}
		for k, v := range overrides {
			if v == "" {
				delete(base, k)
			} else {
				base[k] = v
			}
		}
		return base
	}

	tests := []struct {
		name         string
		metadata     map[string]string
		authParams   map[string]string
		resolvedEnv  map[string]string
		triggerIndex int
		wantErr      bool
		check        func(t *testing.T, meta opensearchMetadata)
	}{
		{
			name:     "query in TriggerMetadata: all fields parsed correctly",
			metadata: baseConfig(nil),
			check: func(t *testing.T, meta opensearchMetadata) {
				assert.Equal(t, "s0-opensearch-query", meta.metricName)
				assert.Equal(t, `{"query":{"match_all":{}}}`, meta.Query)
				assert.Equal(t, "admin", meta.Username)
				assert.Equal(t, "secret", meta.Password)
				assert.Equal(t, []string{"http://localhost:9200"}, meta.Addresses)
				assert.Equal(t, []string{"my-index"}, meta.Index)
				assert.Equal(t, "hits.total.value", meta.ValueLocation)
				assert.Equal(t, float64(10), meta.TargetValue)
			},
		},
		{
			name:     "searchTemplateName: metric name derived from template name",
			metadata: baseConfig(map[string]string{"query": "", "searchTemplateName": "my-template"}),
			check: func(t *testing.T, meta opensearchMetadata) {
				assert.Equal(t, "s0-opensearch-my-template", meta.metricName)
				assert.Equal(t, "my-template", meta.SearchTemplateName)
			},
		},
		{
			name: "auth fields read from AuthParams",
			metadata: map[string]string{
				"index":         "my-index",
				"query":         `{"query":{"match_all":{}}}`,
				"valueLocation": "hits.total.value",
				"targetValue":   "10",
			},
			authParams: map[string]string{
				"addresses": "http://authparam-host:9200",
				"username":  "auth-user",
				"password":  "auth-pass",
			},
			check: func(t *testing.T, meta opensearchMetadata) {
				assert.Equal(t, []string{"http://authparam-host:9200"}, meta.Addresses)
				assert.Equal(t, "auth-user", meta.Username)
				assert.Equal(t, "auth-pass", meta.Password)
			},
		},
		{
			name: "password resolved from environment variable",
			metadata: map[string]string{
				"addresses":       "http://localhost:9200",
				"username":        "admin",
				"passwordFromEnv": "MY_PASS",
				"index":           "my-index",
				"query":           `{"query":{"match_all":{}}}`,
				"valueLocation":   "hits.total.value",
				"targetValue":     "10",
			},
			resolvedEnv: map[string]string{
				"MY_PASS": "env-secret",
			},
			check: func(t *testing.T, meta opensearchMetadata) {
				assert.Equal(t, "env-secret", meta.Password)
			},
		},
		{
			name: "TLS auth: clientCert and clientKey, no basic auth needed",
			metadata: baseConfig(map[string]string{
				"username":   "",
				"password":   "",
				"enableTLS":  "true",
				"clientCert": "cert-data",
				"clientKey":  "key-data",
			}),
			check: func(t *testing.T, meta opensearchMetadata) {
				assert.True(t, meta.EnableTLS)
				assert.Equal(t, "cert-data", meta.ClientCert)
				assert.Equal(t, "key-data", meta.ClientKey)
			},
		},
		{
			name:     "multiple indexes parsed with semicolon separator",
			metadata: baseConfig(map[string]string{"index": "idx1;idx2;idx3"}),
			check: func(t *testing.T, meta opensearchMetadata) {
				assert.Equal(t, []string{"idx1", "idx2", "idx3"}, meta.Index)
			},
		},
		{
			name:     "multiple addresses parsed with comma separator",
			metadata: baseConfig(map[string]string{"addresses": "http://a:9200,http://b:9200"}),
			check: func(t *testing.T, meta opensearchMetadata) {
				assert.Len(t, meta.Addresses, 2)
				assert.Equal(t, []string{"http://a:9200", "http://b:9200"}, meta.Addresses)
			},
		},
		{
			name: "parameters parsed with semicolon separator",
			metadata: baseConfig(map[string]string{
				"query":              "",
				"searchTemplateName": "my-template",
				"parameters":         "key1:val1;key2:val2",
			}),
			check: func(t *testing.T, meta opensearchMetadata) {
				assert.Equal(t, []string{"key1:val1", "key2:val2"}, meta.Parameters)
			},
		},
		{
			name:     "activationTargetValue is parsed",
			metadata: baseConfig(map[string]string{"activationTargetValue": "5"}),
			check: func(t *testing.T, meta opensearchMetadata) {
				assert.Equal(t, float64(5), meta.ActivationTargetValue)
			},
		},
		{
			name:     "ignoreNullValues is parsed",
			metadata: baseConfig(map[string]string{"ignoreNullValues": "true"}),
			check: func(t *testing.T, meta opensearchMetadata) {
				assert.True(t, meta.IgnoreNullValues)
			},
		},
		{
			name:     "unsafeSsl is parsed",
			metadata: baseConfig(map[string]string{"unsafeSsl": "true"}),
			check: func(t *testing.T, meta opensearchMetadata) {
				assert.True(t, meta.UnsafeSsl)
			},
		},
		{
			name:     "default values applied when optional fields absent",
			metadata: baseConfig(nil),
			check: func(t *testing.T, meta opensearchMetadata) {
				assert.False(t, meta.UnsafeSsl)
				assert.Equal(t, float64(0), meta.ActivationTargetValue)
				assert.False(t, meta.IgnoreNullValues)
			},
		},
		// Error cases
		{
			name:     "missing addresses returns error",
			metadata: baseConfig(map[string]string{"addresses": ""}),
			wantErr:  true,
		},
		{
			name:     "missing index returns error",
			metadata: baseConfig(map[string]string{"index": ""}),
			wantErr:  true,
		},
		{
			name:     "missing valueLocation returns error",
			metadata: baseConfig(map[string]string{"valueLocation": ""}),
			wantErr:  true,
		},
		{
			name:     "missing targetValue returns error",
			metadata: baseConfig(map[string]string{"targetValue": ""}),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := parseOpensearchMetadata(&scalersconfig.ScalerConfig{
				TriggerMetadata: tt.metadata,
				AuthParams:      tt.authParams,
				ResolvedEnv:     tt.resolvedEnv,
				TriggerIndex:    tt.triggerIndex,
			})
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.check != nil {
				tt.check(t, meta)
			}
		})
	}
}

func TestNewOpensearchAPIClientWithBasicAuth(t *testing.T) {
	t.Run("success: connects to a plain HTTP server", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		meta := opensearchMetadata{
			Addresses: []string{srv.URL},
			Username:  "admin",
			Password:  "secret",
		}
		client, err := newOpensearchAPIClientWithBasicAuth(meta, logr.Discard())
		assert.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("error: unreachable address causes ping failure", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		addr := srv.URL
		srv.Close()

		meta := opensearchMetadata{
			Addresses: []string{addr},
			Username:  "admin",
			Password:  "secret",
		}
		client, err := newOpensearchAPIClientWithBasicAuth(meta, logr.Discard())
		assert.Error(t, err)
		assert.Nil(t, client)
	})
}

func TestNewOpensearchAPIClientWithTLS(t *testing.T) {
	t.Run("error: invalid clientCert and clientKey fails TLS config creation", func(t *testing.T) {
		meta := opensearchMetadata{
			Addresses:  []string{"https://localhost:9200"},
			ClientCert: "not-a-valid-cert",
			ClientKey:  "not-a-valid-key",
		}
		client, err := newOpensearchAPIClientWithTLS(meta, logr.Discard())
		assert.ErrorContains(t, err, "failed to create TLS config")
		assert.Nil(t, client)
	})

	t.Run("success: connects to TLS server using CACert for verification", func(t *testing.T) {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		// Extract the server's self-signed certificate and encode it as PEM to use as CACert.
		x509Cert, err := x509.ParseCertificate(srv.TLS.Certificates[0].Certificate[0])
		assert.NoError(t, err)
		caCertPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: x509Cert.Raw}))

		meta := opensearchMetadata{
			Addresses: []string{srv.URL},
			CACert:    caCertPEM,
			UnsafeSsl: false,
		}
		client, err := newOpensearchAPIClientWithTLS(meta, logr.Discard())
		assert.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("success: connects to a TLS server with UnsafeSsl skipping cert verification", func(t *testing.T) {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		meta := opensearchMetadata{
			Addresses: []string{srv.URL},
			UnsafeSsl: true,
		}
		client, err := newOpensearchAPIClientWithTLS(meta, logr.Discard())
		assert.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("error: TLS client cannot connect to a plain HTTP server", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		// Force https:// so the TLS transport attempts a handshake with a server
		// that speaks plain HTTP — this causes the TLS handshake to fail.
		meta := opensearchMetadata{
			Addresses: []string{"https://" + srv.Listener.Addr().String()},
			UnsafeSsl: true,
		}
		client, err := newOpensearchAPIClientWithTLS(meta, logr.Discard())
		assert.Error(t, err)
		assert.ErrorContains(t, err, "tls: first record does not look like a TLS handshake")
		assert.Nil(t, client)
	})
}

func TestGetMetricsAndActivity_InvalidParameters(t *testing.T) {
	tests := []struct {
		name      string
		parameter string
	}{
		{
			name:      "no colon",
			parameter: "invalid-param",
		},
		{
			name:      "colon at beginning",
			parameter: ":value",
		},
		{
			name:      "colon at end",
			parameter: "key:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scaler := &opensearchScaler{
				metadata: opensearchMetadata{
					SearchTemplateName: "my-template",
					Parameters:         []string{tt.parameter},
					ValueLocation:      "hits.total.value",
					TargetValue:        10,
				},
				logger: logr.Discard(),
			}

			_, _, err := scaler.GetMetricsAndActivity(context.Background(), "test-metric")
			assert.ErrorContains(t, err, fmt.Sprintf("invalid parameter format %q", tt.parameter))
		})
	}
}

func TestOpensearchCheckHTTPStatus(t *testing.T) {
	scaler := &opensearchScaler{}

	t.Run("returns clear error on HTTP 401", func(t *testing.T) {
		err := scaler.checkHTTPStatus(401)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "authentication failed")
		assert.Contains(t, err.Error(), "HTTP 401")
		assert.Contains(t, err.Error(), "check username and password")
	})

	t.Run("returns clear error on HTTP 403", func(t *testing.T) {
		err := scaler.checkHTTPStatus(403)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "authorization failed")
		assert.Contains(t, err.Error(), "HTTP 403")
		assert.Contains(t, err.Error(), "insufficient permissions")
	})

	t.Run("returns nil for HTTP 200", func(t *testing.T) {
		err := scaler.checkHTTPStatus(200)
		assert.NoError(t, err)
	})

	t.Run("returns nil for other status codes", func(t *testing.T) {
		err := scaler.checkHTTPStatus(404)
		assert.NoError(t, err)
	})
}
