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
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"k8s.io/metrics/pkg/apis/external_metrics/v1beta1"

	"github.com/kedacore/keda/v2/pkg/metricsservice/api"
)

type GrpcClient struct {
	client     api.MetricsServiceClient
	connection *grpc.ClientConn
}

func NewGrpcClient(url string) (*GrpcClient, error) {
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

	// TODO fix Transport layer - use TLS
	conn, err := grpc.Dial(url, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultServiceConfig(retryPolicy))
	if err != nil {
		return nil, err
	}

	return &GrpcClient{client: api.NewMetricsServiceClient(conn), connection: conn}, nil
}

func (c *GrpcClient) GetMetrics(ctx context.Context, scaledObjectName, scaledObjectNamespace, metricName string) (*external_metrics.ExternalMetricValueList, *api.PromMetricsMsg, error) {
	response, err := c.client.GetMetrics(ctx, &api.ScaledObjectRef{Name: scaledObjectName, Namespace: scaledObjectNamespace, MetricName: metricName})
	if err != nil {
		return nil, nil, err
	}

	extMetrics := &external_metrics.ExternalMetricValueList{}
	err = v1beta1.Convert_v1beta1_ExternalMetricValueList_To_external_metrics_ExternalMetricValueList(response.GetMetrics(), extMetrics, nil)
	if err != nil {
		return nil, response.GetPromMetrics(), fmt.Errorf("error when converting metric values %s", err)
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
