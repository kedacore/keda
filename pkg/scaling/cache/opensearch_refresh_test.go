/*
Copyright 2026 The KEDA Authors

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

package cache

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type pausedOpensearchScaler struct {
	scalers.Scaler
	entered chan struct{}
	proceed chan struct{}
}

func (s *pausedOpensearchScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	close(s.entered)
	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	case <-s.proceed:
		return s.Scaler.GetMetricsAndActivity(ctx, metricName)
	}
}

func TestScalersCache_OpensearchRefreshWithSelectedReader(t *testing.T) {
	for _, test := range []struct {
		name  string
		key   string
		value string
		path  string
	}{
		{"query", "query", `{"query":{"match_all":{}}}`, "/jobs/_search"},
		{"template", "searchTemplateName", "pending-jobs", "/jobs/_search/template"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/" {
					_, _ = w.Write([]byte(`{"version":{"distribution":"opensearch","number":"2.19.0"}}`))
					return
				}
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, test.path, r.URL.Path)
				_, _ = w.Write([]byte(`{"took":1,"timed_out":false,"_shards":{"total":1,"successful":1,"failed":0},"hits":{"total":{"value":42,"relation":"eq"},"hits":[]}}`))
			}))
			defer server.Close()
			config := &scalersconfig.ScalerConfig{
				TriggerType: "opensearch",
				TriggerMetadata: map[string]string{
					"addresses":     server.URL,
					"index":         "jobs",
					"valueLocation": "hits.total.value",
					"targetValue":   "10",
					test.key:        test.value,
				},
				AuthParams: map[string]string{"username": "admin", "password": "secret"},
			}
			factory := func(context.Context) (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
				scaler, err := scalers.NewOpensearchScaler(config)
				return scaler, config, err
			}
			original, _, err := factory(ctx)
			require.NoError(t, err)
			paused := &pausedOpensearchScaler{
				Scaler:  original,
				entered: make(chan struct{}),
				proceed: make(chan struct{}),
			}
			cache := newCacheWithScaler(paused)
			cache.Scalers[0].Factory = factory
			t.Cleanup(func() { cache.Close(context.Background()) })

			done := make(chan struct{})
			go func() {
				defer close(done)
				metrics, active, _, err := cache.GetMetricsAndActivityForScaler(ctx, 0, "pending-jobs")
				assert.NoError(t, err)
				if assert.Len(t, metrics, 1) {
					assert.Equal(t, float64(42), metrics[0].Value.AsApproximateFloat64())
				}
				assert.True(t, active)
			}()
			select {
			case <-paused.entered:
			case <-ctx.Done():
				t.Fatal("reader did not select the original scaler")
			}

			// Refresh closes the selected scaler while the reader is outside the cache lock.
			_, err = cache.refreshScaler(ctx, 0)
			require.NoError(t, err)
			close(paused.proceed)
			select {
			case <-done:
			case <-ctx.Done():
				t.Fatal("reader did not complete after scaler refresh")
			}
		})
	}
}
