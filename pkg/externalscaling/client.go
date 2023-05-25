package externalscaling

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"

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
// returns true if the connection was successful, false if we hit a timeut from context
func (c *GrpcClient) WaitForConnectionReady(ctx context.Context, logger logr.Logger) bool {
	currentState := c.connection.GetState()
	if currentState != connectivity.Ready {
		logger.Info("Waiting for establishing a gRPC connection to server")
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
