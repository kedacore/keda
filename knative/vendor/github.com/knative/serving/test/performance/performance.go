/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package performance

import (
	"context"
	"fmt"
	"testing"
	"time"

	pkgTest "github.com/knative/pkg/test"
	"github.com/knative/pkg/test/logging"
	"github.com/knative/serving/test"
	"github.com/knative/test-infra/shared/junit"
	"github.com/knative/test-infra/shared/prometheus"

	// Mysteriously required to support GCP auth (required by k8s libs). Apparently just importing it is enough. @_@ side effects @_@. https://github.com/kubernetes/client-go/issues/242
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

const (
	istioNS      = "istio-system"
	monitoringNS = "knative-monitoring"
	gateway      = "istio-ingressgateway"
	// Property name used by testgrid.
	perfLatency = "perf_latency"
	duration    = 1 * time.Minute
)

// Client is the client used in the performance tests.
type Client struct {
	E2EClients *test.Clients
	PromClient *prometheus.PromProxy
}

// Setup creates all the clients that we need to interact with in our tests
func Setup(ctx context.Context, t *testing.T, promReqd bool) (*Client, error) {
	clients, err := test.NewClients(pkgTest.Flags.Kubeconfig, pkgTest.Flags.Cluster, test.ServingNamespace)
	if err != nil {
		return nil, err
	}

	var p *prometheus.PromProxy
	if promReqd {
		t.Log("Creating prometheus proxy client")
		p = &prometheus.PromProxy{Namespace: monitoringNS}
		p.Setup(ctx, logging.GetContextLogger(t.Name()))
	}
	return &Client{E2EClients: clients, PromClient: p}, nil
}

// TearDown cleans up resources used
func TearDown(t *testing.T, client *Client, names test.ResourceNames) {
	test.TearDown(client.E2EClients, names)

	if client.PromClient != nil {
		client.PromClient.Teardown(logging.GetContextLogger(t.Name()))
	}
}

// CreatePerfTestCase creates a perf test case with the provided name and value
func CreatePerfTestCase(metricValue float32, metricName, testName string) junit.TestCase {
	tp := []junit.TestProperty{{Name: perfLatency, Value: fmt.Sprintf("%f", metricValue)}}
	tc := junit.TestCase{
		ClassName:  testName,
		Name:       fmt.Sprintf("%s/%s", testName, metricName),
		Properties: junit.TestProperties{Properties: tp}}
	return tc
}
