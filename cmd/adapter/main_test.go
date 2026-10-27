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

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apidiscoveryv2 "k8s.io/api/apidiscovery/v2"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/apiserver"

	kedaprovider "github.com/kedacore/keda/v2/pkg/provider"
)

// aggregatedDiscoveryResources returns the resource names of each group version
// the server lists in its aggregated discovery document.
func aggregatedDiscoveryResources(t *testing.T, server *apiserver.CustomMetricsAdapterServer) map[string][]string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/apis", nil)
	req.Header.Set("Accept", "application/json;g=apidiscovery.k8s.io;v=v2;as=APIGroupDiscoveryList")
	rec := httptest.NewRecorder()
	server.GenericAPIServer.Handler.GoRestfulContainer.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var list apidiscoveryv2.APIGroupDiscoveryList
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))

	resources := map[string][]string{}
	for _, group := range list.Items {
		for _, version := range group.Versions {
			names := []string{}
			for _, r := range version.Resources {
				names = append(names, r.Resource)
			}
			resources[group.Name+"/"+version.Version] = names
		}
	}
	return resources
}

func TestAddExternalMetricsAggregatedDiscovery(t *testing.T) {
	genericConfig := genericapiserver.NewConfig(apiserver.Codecs)
	genericConfig.ExternalAddress = "127.0.0.1:443"
	genericConfig.LoopbackClientConfig = &rest.Config{}
	config := &apiserver.Config{GenericConfig: genericConfig}

	p := &kedaprovider.KedaProvider{}
	server, err := config.Complete(nil).New("keda-adapter", nil, p)
	require.NoError(t, err)

	// custom-metrics-apiserver leaves the group out of aggregated discovery on its own.
	assert.NotContains(t, aggregatedDiscoveryResources(t, server), "external.metrics.k8s.io/v1beta1")

	require.NoError(t, addExternalMetricsAggregatedDiscovery(server, p))
	assert.Equal(t, []string{"externalmetrics"}, aggregatedDiscoveryResources(t, server)["external.metrics.k8s.io/v1beta1"])
}
