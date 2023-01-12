/*
Copyright 2022 The KEDA Authors

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

package metricsservice

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"k8s.io/metrics/pkg/apis/external_metrics/v1beta1"

	"github.com/kedacore/keda/v2/pkg/metricsservice/api"
	"github.com/kedacore/keda/v2/pkg/metricsservice/utils"
)

type GrpcClient struct {
	client     api.MetricsServiceClient
	connection *grpc.ClientConn
}

func NewGrpcClient(url, certDir string) (*GrpcClient, error) {
	retryPolicy := `{
		"methodConfig": [{
		  "timeout": "3s",
		  "waitForReady": true,
		  "retryPolicy": {
			  "InitialBackoff": ".25s",
			  "MaxBackoff": "2.0s",
			  "BackoffMultiplier": 2,
			  "RetryableStatusCodes": [ "UNAVAILABLE" ]
		  }
		}]}`

	creds, err := utils.LoadGrpcTLSCredentials(certDir, false)
	if err != nil {
		return nil, err
	}
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultServiceConfig(retryPolicy),
	}
	conn, err := grpc.Dial(url, opts...)
	if err != nil {
		return nil, err
	}

	return &GrpcClient{client: api.NewMetricsServiceClient(conn), connection: conn}, nil
}

// nosemgrep: trailofbits.go.invalid-usage-of-modified-variable.invalid-usage-of-modified-variable
func (c *GrpcClient) GetMetrics(ctx context.Context, scaledObjectName, scaledObjectNamespace, metricName string) (*external_metrics.ExternalMetricValueList, *api.PromMetricsMsg, error) {
	response, err := c.client.GetMetrics(ctx, &api.ScaledObjectRef{Name: scaledObjectName, Namespace: scaledObjectNamespace, MetricName: metricName})
	if err != nil {
		// in certain cases we would like to get Prometheus metrics even if there's an error
		// so we can expose information about the error in the client
		return nil, response.GetPromMetrics(), err
	}

	extMetrics := &external_metrics.ExternalMetricValueList{}
	err = v1beta1.Convert_v1beta1_ExternalMetricValueList_To_external_metrics_ExternalMetricValueList(response.GetMetrics(), extMetrics, nil)
	if err != nil {
		// in certain cases we would like to get Prometheus metrics even if there's an error
		// so we can expose information about the error in the client
		return nil, response.GetPromMetrics(), fmt.Errorf("error when converting metric values %w", err)
	}

	return extMetrics, response.GetPromMetrics(), nil
}

// WaitForConnectionReady waits for gRPC connection to be ready
// returns true if the connection was successful, false if we hit a timeut from context
func (c *GrpcClient) WaitForConnectionReady(ctx context.Context, logger logr.Logger) bool {
	currentState := c.connection.GetState()
	if currentState != connectivity.Ready {
		logger.Info("Waiting for establishing a gRPC connection to KEDA Metrics Server")
		for {
			select {
			case <-ctx.Done():
				return false
			default:
				c.connection.Connect()
				time.Sleep(500 * time.Millisecond)
				currentState := c.connection.GetState()
				if currentState == connectivity.Ready {
					return true
				}
			}
		}
	}
	return true
}
