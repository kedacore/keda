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
	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
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

func NewGrpcClient(ctx context.Context, url, certDir, authority string, clientMetrics *grpcprom.ClientMetrics) (*GrpcClient, error) {
	defaultConfig := `{
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

	creds, err := utils.LoadGrpcTLSCredentials(ctx, certDir, false)
	if err != nil {
		return nil, err
	}
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultServiceConfig(defaultConfig),
	}

	opts = append(
		opts,
		grpc.WithChainUnaryInterceptor(clientMetrics.UnaryClientInterceptor()),
		grpc.WithChainStreamInterceptor(clientMetrics.StreamClientInterceptor()),
	)

	if authority != "" {
		// If an Authority header override is specified, add it to the client so it is set on every request.
		// This is useful when the address used to dial the GRPC server does not match any hosts provided in the TLS certificate's
		// SAN
		opts = append(opts, grpc.WithAuthority(authority))
	}

	conn, err := grpc.NewClient(url, opts...)
	if err != nil {
		return nil, err
	}

	return &GrpcClient{client: api.NewMetricsServiceClient(conn), connection: conn}, nil
}

func (c *GrpcClient) GetMetrics(ctx context.Context, scaledObjectName, scaledObjectNamespace, metricName string) (*external_metrics.ExternalMetricValueList, error) {
	v1beta1ExtMetrics, err := c.client.GetMetrics(ctx, &api.ScaledObjectRef{Name: scaledObjectName, Namespace: scaledObjectNamespace, MetricName: metricName})
	if err != nil {
		return nil, err
	}

	extMetrics := &external_metrics.ExternalMetricValueList{}
	err = v1beta1.Convert_v1beta1_ExternalMetricValueList_To_external_metrics_ExternalMetricValueList(v1beta1ExtMetrics, extMetrics, nil)
	if err != nil {
		return nil, fmt.Errorf("error when converting metric values %w", err)
	}

	return extMetrics, nil
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

// GetServerURL returns url of the gRPC server this client is connected to
func (c *GrpcClient) GetServerURL() string {
	return c.connection.Target()
}
