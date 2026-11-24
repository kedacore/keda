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

package k8s

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/scale"
)

func TestScaleUsesResolvedResource(t *testing.T) {
	for _, gv := range []schema.GroupVersion{
		{Group: "apps", Version: "v1"},
		{Group: "apps.kruise.io", Version: "v1"},
		{Group: "apps.kruise.io", Version: "v1beta1"},
	} {
		t.Run(gv.String(), func(t *testing.T) {
			target := gv.WithResource("statefulsets")
			groupPath := "/apis/" + target.GroupVersion().String()
			scalePath := groupPath + "/namespaces/demo/statefulsets/web/scale"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodGet && r.URL.Path == scalePath:
					assert.Equal(t, "32s", r.URL.Query().Get("timeout"))
					assert.Equal(t, "17", r.URL.Query().Get("resourceVersion"))
					_, _ = w.Write([]byte(`{"apiVersion":"autoscaling/v1","kind":"Scale","metadata":{"name":"web","namespace":"demo","resourceVersion":"17"},"spec":{"replicas":2},"status":{"selector":"app=web"}}`))
				case r.Method == http.MethodGet && r.URL.Path == groupPath:
					list := metav1.APIResourceList{GroupVersion: target.GroupVersion().String(), APIResources: []metav1.APIResource{
						{Name: "statefulsets/scale", Group: "autoscaling", Version: "v1", Kind: "Scale"},
					}}
					assert.NoError(t, json.NewEncoder(w).Encode(list))
				case r.Method == http.MethodPut && r.URL.Path == scalePath:
					assert.Equal(t, "32s", r.URL.Query().Get("timeout"))
					assert.Equal(t, "scale-test", r.URL.Query().Get("fieldManager"))
					var body autoscalingv1.Scale
					if assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
						assert.Equal(t, "autoscaling/v1", body.APIVersion)
						assert.Equal(t, "17", body.ResourceVersion)
						assert.EqualValues(t, 3, body.Spec.Replicas)
						body.ResourceVersion = "18"
						assert.NoError(t, json.NewEncoder(w).Encode(body))
					}
				default:
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			config := &rest.Config{Host: server.URL}
			resources, err := newScaleResourceClient(config)
			require.NoError(t, err)
			require.Zero(t, config.Timeout)
			client := &versionedScaleClient{
				client:    resources,
				kinds:     scale.NewDiscoveryScaleKindResolver(discovery.NewDiscoveryClientForConfigOrDie(config)),
				converter: scale.NewScaleConverter(),
			}
			scales := client.Scales("demo")
			current, err := scales.Get(t.Context(), target, "web", metav1.GetOptions{ResourceVersion: "17"})
			require.NoError(t, err)
			assert.EqualValues(t, 2, current.Spec.Replicas)
			assert.Equal(t, "app=web", current.Status.Selector)
			current.Spec.Replicas = 3
			updated, err := scales.Update(t.Context(), target, current, metav1.UpdateOptions{FieldManager: "scale-test"})
			require.NoError(t, err)
			assert.Equal(t, "18", updated.ResourceVersion)
		})
	}
}

func TestScaleRequestHonorsConfiguredTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	resources, err := newScaleResourceClient(&rest.Config{Host: server.URL, Timeout: 100 * time.Millisecond})
	require.NoError(t, err)
	client := &versionedScaleClient{client: resources}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	target := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}
	_, err = client.Scales("demo").Get(ctx, target, "web", metav1.GetOptions{})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.NoError(t, ctx.Err(), "the client timeout should expire before the caller's deadline")
}
