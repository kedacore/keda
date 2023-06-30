package externalscaling

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/external_metrics"

	cl "github.com/kedacore/keda/v2/pkg/externalscaling/api"
)

type GrpcClient struct {
	client     cl.ExternalCalculationClient
	connection *grpc.ClientConn
}

func NewGrpcClient(url string, logger logr.Logger) (*GrpcClient, error) {
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

	opts := []grpc.DialOption{
		grpc.WithDefaultServiceConfig(retryPolicy),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	conn, err := grpc.Dial(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("error in grpc.Dial: %s", err)
	}

	return &GrpcClient{client: cl.NewExternalCalculationClient(conn), connection: conn}, nil
}

func (c *GrpcClient) Calculate(ctx context.Context, list *cl.MetricsList, logger logr.Logger) (*cl.MetricsList, error) {
	response, err := c.client.Calculate(ctx, list)
	if err != nil {
		return nil, fmt.Errorf("error in externalscaling.Calculate %s", err)
	}
	return response.List, nil
}

// WaitForConnectionReady waits for gRPC connection to be ready
// returns true if the connection was successful, false if we hit a timeout from context
// TODO: add timeout instead into time.Sleep() - removed for testing
func (c *GrpcClient) WaitForConnectionReady(ctx context.Context, url string, timeout time.Duration, logger logr.Logger) bool {
	currentState := c.connection.GetState()
	if currentState != connectivity.Ready {
		logger.Info(fmt.Sprintf("Waiting for establishing a gRPC connection to server for external calculator at %s", url))
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

// ConvertToGeneratedStruct converts K8s external metrics list to gRPC generated
// external metrics list
func ConvertToGeneratedStruct(inK8sList []external_metrics.ExternalMetricValue, l logr.Logger) *cl.MetricsList {
	outExternal := cl.MetricsList{}
	for _, val := range inK8sList {
		metric := cl.Metric{Name: val.MetricName, Value: float32(val.Value.Value())}
		outExternal.MetricValues = append(outExternal.MetricValues, &metric)
	}
	return &outExternal
}

// ConvertFromGeneratedStruct converts gRPC generated external metrics list to
// K8s external_metrics list
func ConvertFromGeneratedStruct(inExternal *cl.MetricsList) []external_metrics.ExternalMetricValue {
	outK8sList := []external_metrics.ExternalMetricValue{}
	for _, inValue := range inExternal.MetricValues {
		outValue := external_metrics.ExternalMetricValue{}
		outValue.MetricName = inValue.Name
		outValue.Timestamp = v1.Now()
		outValue.Value.SetMilli(int64(inValue.Value * 1000))
		outK8sList = append(outK8sList, outValue)
	}
	return outK8sList
}
