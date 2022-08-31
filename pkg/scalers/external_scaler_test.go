package scalers

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "github.com/kedacore/keda/v2/pkg/scalers/externalscaler"
)

type parseExternalScalerMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

var testExternalScalerMetadata = []parseExternalScalerMetadataTestData{
	{map[string]string{}, true},
	// all properly formed
	{map[string]string{"scalerAddress": "myservice", "test1": "7", "test2": "SAMPLE_CREDS"}, false},
	// missing scalerAddress
	{map[string]string{"test1": "1", "test2": "SAMPLE_CREDS"}, true},
}

func TestExternalScalerParseMetadata(t *testing.T) {
	for _, testData := range testExternalScalerMetadata {
		_, err := parseExternalScalerMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: map[string]string{}})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestExternalPushScaler_Run(t *testing.T) {
	const serverCount = 5
	const iterationCount = 500

	servers := createGRPCServers(serverCount, t)
	replyCh := createIsActiveChannels(serverCount * iterationCount)

	// we will send serverCount * iterationCount 'isActiveResponse' and expect resultCount == serverCount * iterationCount
	var resultCount int64

	ctx, cancel := context.WithCancel(context.Background())
	for i := 0; i < serverCount*iterationCount; i++ {
		id := i % serverCount
		pushScaler, _ := NewExternalPushScaler(&ScalerConfig{ScalableObjectName: "app", ScalableObjectNamespace: "namespace", TriggerMetadata: map[string]string{"scalerAddress": servers[id].address}, ResolvedEnv: map[string]string{}})
		go pushScaler.Run(ctx, replyCh[i])
	}

	// scaler consumer
	for i, ch := range replyCh {
		go func(c chan bool, _ int) {
			for msg := range c {
				if msg {
					atomic.AddInt64(&resultCount, 1)
				}
			}
		}(ch, i)
	}

	// producer
	for _, s := range servers {
		go func(c chan bool) {
			for i := 0; i < iterationCount; i++ {
				c <- true
			}
		}(s.publish)
	}

	retries := 0
	defer cancel()
	for {
		<-time.After(time.Second * 1)
		if resultCount == serverCount*iterationCount {
			t.Logf("resultCount == %d", resultCount)
			return
		}

		retries++
		if retries > 10 {
			t.Fatalf("Expected resultCount to be %d after %d retries, but got %d", serverCount*iterationCount, retries, resultCount)
			return
		}
	}
}

type testServer struct {
	grpcServer *grpc.Server
	address    string
	publish    chan bool
}

func createGRPCServers(count int, t *testing.T) []testServer {
	result := make([]testServer, 0, count)

	for i := 0; i < count; i++ {
		grpcServer := grpc.NewServer()
		address := fmt.Sprintf("127.0.0.1:%d", 5050+i)
		lis, _ := net.Listen("tcp", address)
		activeCh := make(chan bool)
		pb.RegisterExternalScalerServer(grpcServer, &testExternalScaler{
			t:      t,
			active: activeCh,
		})

		go func() {
			if err := grpcServer.Serve(lis); err != nil {
				t.Error(err, "error from grpcServer")
			}
		}()

		result = append(result, testServer{
			grpcServer: grpcServer,
			address:    address,
			publish:    activeCh,
		})
	}

	return result
}

func createIsActiveChannels(count int) []chan bool {
	result := make([]chan bool, 0, count)
	for i := 0; i < count; i++ {
		result = append(result, make(chan bool))
	}

	return result
}

type testExternalScaler struct {
	// Embed the unimplemented server
	pb.UnimplementedExternalScalerServer

	t      *testing.T
	active chan bool
}

func (e *testExternalScaler) IsActive(context.Context, *pb.ScaledObjectRef) (*pb.IsActiveResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method IsActive not implemented")
}
func (e *testExternalScaler) StreamIsActive(_ *pb.ScaledObjectRef, epsServer pb.ExternalScaler_StreamIsActiveServer) error {
	for {
		select {
		case <-epsServer.Context().Done():
			// the call completed? exit
			return nil
		case i := <-e.active:
			err := epsServer.Send(&pb.IsActiveResponse{
				Result: i,
			})
			if err != nil {
				e.t.Error(err)
			}
		}
	}
}

func (e *testExternalScaler) GetMetricSpec(context.Context, *pb.ScaledObjectRef) (*pb.GetMetricSpecResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetMetricSpec not implemented")
}

func (e *testExternalScaler) GetMetrics(context.Context, *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetMetrics not implemented")
}

func TestWaitForState(t *testing.T) {
	grpcServer := grpc.NewServer()
	address := fmt.Sprintf("127.0.0.1:%d", 15050)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		t.Errorf("start grpcServer with %s failed:%s", address, err)
		return
	}
	activeCh := make(chan bool)
	pb.RegisterExternalScalerServer(grpcServer, &testExternalScaler{
		t:      t,
		active: activeCh,
	})

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Error(err, "error from grpcServer")
		}
	}()
	// send active data
	go func() {
		activeCh <- true
	}()

	// build client connect to server
	grpcClient, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Errorf("connect grpc server %s failed:%s", address, err)
		return
	}
	graceDone := make(chan struct{})
	go func() {
		// server stop will lead to Idle.
		<-waitForState(context.TODO(), grpcClient, connectivity.Idle, connectivity.Shutdown)
		grpcClient.Close()
		// after close the state to Shutdown.
		t.Log("close state:", grpcClient.GetState().String())
		close(graceDone)
	}()
	client := pb.NewExternalScalerClient(grpcClient)

	// request StreamIsActive interface
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()
	stream, err := client.StreamIsActive(ctx, &pb.ScaledObjectRef{})
	if err != nil {
		t.Errorf("StreamIsActive request failed:%s", err)
		return
	}

	// check result value
	resp, err := stream.Recv()
	if err != nil {
		t.Error(err)
		return
	}
	if !resp.Result {
		t.Error("StreamIsActive should receive")
	}

	// stop server
	time.Sleep(time.Second * 5)
	grpcServer.GracefulStop()

	select {
	case <-graceDone:
		// test ok.
		return
	case <-time.After(time.Second * 1):
		t.Error("waitForState should be get connectivity.Shutdown.")
	}
}
