package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"

	cl "github.com/kedacore/keda/v2/pkg/externalscaling/api"
)

var (
// logger logr.Logger
)

type GrpcClient struct {
	client     cl.ExternalCalculationClient
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

func (c *GrpcClient) Calculate(ctx context.Context, list *cl.MetricsList) (*cl.MetricsList, error) {
	if !c.WaitForConnectionReady(ctx) {
		fmt.Print("Client didnt connect to server successfully (WaitForConnectionReady)")
		return nil, fmt.Errorf("chybka v connection")
	}
	fmt.Print("client in: ", list)
	response, err := c.client.Calculate(ctx, list)
	if err != nil {
		fmt.Print("CHYBKA v Calculate...\n")
		return nil, err
	}
	fmt.Print("client response: ", response)

	listRet := response.List

	return listRet, nil
}

// WaitForConnectionReady waits for gRPC connection to be ready
// returns true if the connection was successful, false if we hit a timeut from context
func (c *GrpcClient) WaitForConnectionReady(ctx context.Context) bool {
	currentState := c.connection.GetState()
	if currentState != connectivity.Ready {
		fmt.Print("Waiting for establishing a gRPC connection to server\n")
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

func main() {
	// call calculate
	fmt.Print("client begin\n")

	ctx := context.Background()
	list := cl.MetricsList{
		MetricValues: []*cl.Metric{
			{Name: "one", Value: 1},
			{Name: "two", Value: 2},
			{Name: "three", Value: 3},
			{Name: "four", Value: 4},
		},
	}

	grpcClient, err := NewGrpcClient("localhost:50051")
	if err != nil {
		fmt.Printf("error ocured while creating new grpc client: %s", err)
		os.Exit(1)
	}

	ret, err := grpcClient.Calculate(ctx, &list)
	if err != nil {
		fmt.Print(">> error in Calculate: ", err)
	} else {
		fmt.Print("all good")
		fmt.Print("list:", ret)
	}
}
