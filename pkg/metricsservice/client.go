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
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/go-logr/logr"
	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/keepalive"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"k8s.io/metrics/pkg/apis/external_metrics/v1beta1"

	"github.com/kedacore/keda/v2/pkg/metricsservice/api"
	"github.com/kedacore/keda/v2/pkg/metricsservice/utils"
)

type GrpcClient struct {
	client           api.MetricsServiceClient
	rawMetricsClient api.RawMetricsServiceClient
	connection       *grpc.ClientConn
}

type Measurement struct {
	Name      string
	Value     float64
	Timestamp *time.Time
}

func NewGrpcClient(ctx context.Context, url, certDir, authority, confOptions string, clientMetrics *grpcprom.ClientMetrics, rawStream bool) (*GrpcClient, error) {
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
	if confOptions == "" {
		confOptions = defaultConfig
	}

	creds, err := utils.LoadGrpcTLSCredentials(ctx, certDir, false)
	if err != nil {
		return nil, err
	}
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultServiceConfig(confOptions),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second, // Send keepalive pings every 30s
			Timeout:             10 * time.Second, // Wait 10s for ping ack before considering connection dead
			PermitWithoutStream: true,             // Send pings even without active RPCs
		}),
		grpc.WithConnectParams(grpc.ConnectParams{
			MinConnectTimeout: 5 * time.Second,
		}),
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
	grpcClient := GrpcClient{client: api.NewMetricsServiceClient(conn), connection: conn}
	if rawStream {
		grpcClient.rawMetricsClient = api.NewRawMetricsServiceClient(conn)
	}

	return &grpcClient, nil
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

// Subscribe will create a subscription on KEDA side indicating that this particular metric (identified by SO's ns, SO's name and trigger name)
// should be sent in the raw metric stream
func (c *GrpcClient) Subscribe(ctx context.Context, subscriber, scaledObjectName, scaledObjectNamespace, metricName string) (bool, error) {
	if c.rawMetricsClient == nil {
		return false, errors.New("rawMetricsClient is not initialized, initialize the client using NewGrpcClient(...,true)")
	}
	req := &api.SubscriptionRequest{
		Subscriber: subscriber,
		MetricMetadata: &api.ScaledObjectRef{
			Name:       scaledObjectName,
			Namespace:  scaledObjectNamespace,
			MetricName: metricName,
		},
	}
	ack, err := c.rawMetricsClient.SubscribeMetric(ctx, req)
	if err != nil {
		return false, err
	}
	return ack.WasSubscribed, nil
}

// Unsubscribe cancels the subscription on KEDA side -> will stop sending the metric
func (c *GrpcClient) Unsubscribe(ctx context.Context, subscriber, scaledObjectName, scaledObjectNamespace, metricName string) (bool, error) {
	if c.rawMetricsClient == nil {
		return false, errors.New("rawMetricsClient is not initialized, initialize the client using NewGrpcClient(...,true)")
	}
	req := &api.SubscriptionRequest{
		Subscriber: subscriber,
		MetricMetadata: &api.ScaledObjectRef{
			Name:       scaledObjectName,
			Namespace:  scaledObjectNamespace,
			MetricName: metricName,
		},
	}
	ack, err := c.rawMetricsClient.UnsubscribeMetric(ctx, req)
	if err != nil {
		return false, err
	}
	return ack.WasSubscribed, nil
}

// GetRawMetricsStream opens the gRPC connection for receiving the raw metrics from KEDA
// channel with metrics is returned as well as the channel that indicates closed connection or error
// it is up to the caller to make sure the connection is reopened in case of error
// if true is sent to done channel the connection was closed by server
// false represents the gRPC connection error
// note: no metrics will be sent until Subscribe is called
func (c *GrpcClient) GetRawMetricsStream(ctx context.Context, subscriber string) (chan Measurement, chan bool, error) {
	if c.rawMetricsClient == nil {
		return nil, nil, errors.New("rawMetricsClient is not initialized, initialize the client using NewGrpcClient(...,true)")
	}
	logger := log.WithName("GrpcClient").WithValues("subscriber", subscriber)
	req := &api.RawMetricsRequest{
		Subscriber: subscriber,
	}
	metricStream, err := c.rawMetricsClient.GetRawMetricsStream(ctx, req)
	doneChan := make(chan bool)
	metricsChan := make(chan Measurement)
	if err != nil {
		return nil, nil, err
	}

	go func() {
		for {
			resp, e := metricStream.Recv()
			if e == io.EOF {
				logger.Info("Received EOF - stream was closed by server")
				select {
				case doneChan <- true:
				default:
				}
				break
			}
			if e != nil {
				logger.Error(e, "error receiving metric response")
				select {
				case doneChan <- false:
				default:
				}
				break
			}
			for _, m := range resp.Metrics {
				if m == nil {
					continue
				}
				measurement := Measurement{
					Name:  m.Metadata.MetricName,
					Value: m.Value,
				}
				if m.Timestamp != nil {
					t := m.Timestamp.AsTime()
					measurement.Timestamp = &t
				}
				metricsChan <- measurement
				logger.V(10).Info("Received raw metric", "name", m.Metadata.MetricName, "value", m.Value)
			}
		}
	}()

	return metricsChan, doneChan, nil
}

// WaitForConnectionReady waits for the gRPC connection to reach the Ready state.
// It uses WaitForStateChange to efficiently wait for state transitions rather than
// busy-polling, which allows the gRPC transport to properly handle reconnection
// with its built-in backoff logic.
// Returns true if the connection is ready, false if the context is cancelled.
func (c *GrpcClient) WaitForConnectionReady(ctx context.Context, logger logr.Logger) bool {
	for {
		currentState := c.connection.GetState()
		if currentState == connectivity.Ready {
			return true
		}

		logger.Info("Waiting for establishing a gRPC connection to KEDA Metrics Server",
			"currentState", currentState.String())

		// Trigger a connection attempt if the connection is idle
		if currentState == connectivity.Idle {
			c.connection.Connect()
		}

		// Wait for the state to change or context to be cancelled.
		// WaitForStateChange blocks until the connection transitions away from
		// the given state, allowing gRPC's internal reconnection backoff to
		// work properly instead of interfering with it via busy-polling.
		if !c.connection.WaitForStateChange(ctx, currentState) {
			// Context was cancelled
			return false
		}
	}
}

// GetServerURL returns url of the gRPC server this client is connected to
func (c *GrpcClient) GetServerURL() string {
	return c.connection.Target()
}
