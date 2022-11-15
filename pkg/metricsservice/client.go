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

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"k8s.io/metrics/pkg/apis/external_metrics/v1beta1"

	"github.com/kedacore/keda/v2/pkg/metricsservice/api"
)

type GrpcClient struct {
	client api.MetricsServiceClient
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

	return &GrpcClient{client: api.NewMetricsServiceClient(conn)}, nil
}

func (c *GrpcClient) GetMetrics(ctx context.Context, scaledObjectName, scaledObjectNamespace, metricName string) (*external_metrics.ExternalMetricValueList, error) {
	v1beta1ExtMetrics, err := c.client.GetMetrics(ctx, &api.ScaledObjectRef{Name: scaledObjectName, Namespace: scaledObjectNamespace, MetricName: metricName})
	if err != nil {
		return nil, err
	}

	extMetrics := &external_metrics.ExternalMetricValueList{}
	err = v1beta1.Convert_v1beta1_ExternalMetricValueList_To_external_metrics_ExternalMetricValueList(v1beta1ExtMetrics, extMetrics, nil)
	if err != nil {
		return nil, fmt.Errorf("error when converting metric values %s", err)
	}

	return extMetrics, nil
}
