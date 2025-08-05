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

	pb "github.com/kedacore/keda/v2/keda-scalers/externalscaler"
	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

var serverRootCA = `-----BEGIN CERTIFICATE-----
MIIDPTCCAiWgAwIBAgIUTPXiztn8CG3+NQdeEIlpA1F9Ec4wDQYJKoZIhvcNAQEL
BQAwLTEOMAwGA1UEAwwFa2VkYTExCzAJBgNVBAYTAlVTMQ4wDAYDVQQHDAVFYXJ0
aDAgFw0yMzAzMjYxMzU1MDJaGA8yMDUxMDgwMjEzNTUwMlowLTEOMAwGA1UEAwwF
a2VkYTExCzAJBgNVBAYTAlVTMQ4wDAYDVQQHDAVFYXJ0aDCCASIwDQYJKoZIhvcN
AQEBBQADggEPADCCAQoCggEBAOaQl2+EEZycNQlO1nEeFgheUZ20gTVAButKjvEK
vIZv+x4AdwNaOcKahro5b09QinoGakTJEOrpks+VUSqQpJ+zVmja5vpIb92gnmGQ
B3rl7nD9rP/bLffa5bNDhmMR7vRd88PYvopn6+hTyX3EGkvbCZD8WNs5f8jslzek
0xj4U4LgC9T1pBykNl5nZs5Fd4CdaO+vi3cywmgjaiPzDOYMbYH4pzflH7aNsEEc
IYs9fQ8SwzsocXpKUS+bTg9OmrDMAwan+mxz6m15BxvJzHQqmp/aE70BSkinBwCg
dgzgQUwg6ko/jnJixP4tkr8p8nURBL7GNvuVIS7Z2EjD240CAwEAAaNTMFEwHQYD
VR0OBBYEFN03E9o2ne0s5GZSZ7rZczisME5RMB8GA1UdIwQYMBaAFN03E9o2ne0s
5GZSZ7rZczisME5RMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEB
ADM6vkgjttrU9bYRviwdgohvGNYegDXfyKt25gwYn5+UwUtqjjztTWMr6aAKLNob
Dqjb0BDR5Ow0kD9KXyO4m0gTBzxrHzDnFeTQxE2h8Gl/VRGueJQ2sfmU8oG2/3GV
4nEWLAu1XMqXcFQWT9X+JS3Wxqc1DLrAeX8u0ZIx5Lkk4kPV3d3BP8KyX+AQGt57
p0kdhXOTNW1kUPCrtnc0uBJNHqlev3KkHebH20G7iAZFCCpul9cyLK1fftCBuVE/
jtq3TnHHw+BroQPje3zF/MZTAA8Z9RejkpALMtoHeE68ar07FPlC8wZXDlfQXzYS
PWO1PoIiMX1UsfdZ35JCOF4=
-----END CERTIFICATE-----
`
var clientCert = `-----BEGIN CERTIFICATE-----
MIIC9jCCAd4CFEGcfEWHP2ckC/kIgUEDUPkAVHHIMA0GCSqGSIb3DQEBCwUAMC0x
DjAMBgNVBAMMBWtlZGEyMQswCQYDVQQGEwJVUzEOMAwGA1UEBwwFRWFydGgwIBcN
MjMwMzI2MTM1NTAzWhgPMjA1MTA2MDcxMzU1MDNaMEAxCzAJBgNVBAYTAlVTMQsw
CQYDVQQIDAJDQTEUMBIGA1UECgwLTXlPcmcsIEluYy4xDjAMBgNVBAMMBWtlZGEy
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA3k7zr+QbOHXMqhyUM6Oz
SuGGqQttIGZEs12eLSRlGY9vL+pf/G3CubkGsTtp5b5tmP4CNYcGtU8wSJMn23Bq
BbXECpDXh6cuo+56VVAMJyNqZoIeS4JfKX5mvj4WrfpVJ3e+o6lrebAICK1qqTwq
z6N6ZUkeF552aTh8RAgEiKJlylmKdHc/IF3+oLc2aA7IAl+zxYOTCrjIiUrc54OB
gYCkCSb0P4SPHE8ryB0pN8S3LdtZrgVIzewd24joKbXL7hBZ3ltaj0t2kK2CTCxb
I3te0JdIaPV2TxCnK9dxLRkFXNjz/7V45rbRLXtSPoirKseUR2zbo5kfquY0J2hv
nwIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQCiZI/N/60Gj8711V+Uhj1j9iw3s4oU
qT4r9NozotrhjIMe14rhkJ1k+1x7pyLX8QHgO0WxiAp8tKX0kcUQO/ZQfTAM6FpW
cevGTxrVk5CcilafIBzaZF5Mz6diBxbTnhFS+hZXiwavkImBK4zZY9aUcVIjJfPv
xSaEVvMdofrhaio9M0deYzQ/Bf/uMkR3Fxs4qbhsg3gkbepFm3yJoSANzXCMvDnv
mauSvQwA0SRKECr46F8dSeFE1uIbN4ZNgrisBTVkoPYZuF7pAsSsjqGM0phUKiI8
5thG2dnqJSunC+vZW8QY+x3eq4XzFEpIYcaV9YpiGHbv8N6gzJydL/GB
-----END CERTIFICATE-----
`

type parseExternalScalerMetadataTestData struct {
	metadata   map[string]string
	isError    bool
	authParams map[string]string
}

var testExternalScalerMetadata = []parseExternalScalerMetadataTestData{
	{map[string]string{}, true, map[string]string{}},
	// all properly formed
	{map[string]string{"scalerAddress": "myservice", "test1": "7", "test2": "SAMPLE_CREDS", "enableTLS": "true", "insecureSkipVerify": "true"}, false, map[string]string{"caCert": serverRootCA, "tlsClientCert": clientCert}},
	// missing scalerAddress
	{map[string]string{"test1": "1", "test2": "SAMPLE_CREDS"}, true, map[string]string{}},
}

func TestExternalScalerParseMetadata(t *testing.T) {
	for _, testData := range testExternalScalerMetadata {
		metadata, err := parseExternalScalerMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: map[string]string{}, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}

		if testData.metadata["unsafeSsl"] == "true" && !metadata.UnsafeSsl {
			t.Error("Expected unsafeSsl to be true but got", metadata.UnsafeSsl)
		}
		if testData.metadata["enableTLS"] == "true" && !metadata.EnableTLS {
			t.Error("Expected enableTLS to be true but got", metadata.EnableTLS)
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
		pushScaler, _ := NewExternalPushScaler(&scalersconfig.ScalerConfig{ScalableObjectName: "app", ScalableObjectNamespace: "namespace", TriggerMetadata: map[string]string{"scalerAddress": servers[id].address}, ResolvedEnv: map[string]string{}})
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
		currentCount := atomic.LoadInt64(&resultCount)
		if currentCount == serverCount*iterationCount {
			t.Logf("resultCount == %d", currentCount)
			return
		}

		retries++
		if retries > 10 {
			currentCount = atomic.LoadInt64(&resultCount)
			t.Fatalf("Expected resultCount to be %d after %d retries, but got %d", serverCount*iterationCount, retries, currentCount)
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
	grpcClient, err := grpc.NewClient(address,
		grpc.WithDefaultServiceConfig(grpcConfig),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Errorf("connect grpc server %s failed:%s", address, err)
		return
	}
	graceDone := make(chan struct{})
	go func() {
		// server stop will lead to Idle.
		<-waitForState(context.TODO(), grpcClient, connectivity.Idle, connectivity.Shutdown)
		grpcClient.Close()
		// after close the state to shut down.
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
