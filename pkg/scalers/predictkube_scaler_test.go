package scalers

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/api/resource"

	libsSrv "github.com/dysnix/predictkube-libs/external/grpc/server"
	pb "github.com/dysnix/predictkube-proto/external/proto/services"
)

type server struct {
	grpcSrv  *grpc.Server
	listener net.Listener
	port     int
	val      int64
}

func (s *server) GetPredictMetric(_ context.Context, _ *pb.ReqGetPredictMetric) (res *pb.ResGetPredictMetric, err error) {
	s.val = int64(rand.Intn(30000-10000) + 10000)
	return &pb.ResGetPredictMetric{
		ResultMetric: s.val,
	}, nil
}

func (s *server) start() <-chan error {
	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)

		var (
			err error
		)

		s.port, err = freeport.GetFreePort()
		if err != nil {
			log.Fatalf("Could not get free port for init mock grpc server: %s", err)
		}

		serverURL := fmt.Sprintf("0.0.0.0:%d", s.port)
		if s.listener == nil {
			var err error
			s.listener, err = net.Listen("tcp4", serverURL)

			if err != nil {
				log.Println("starting grpc server with error")

				errCh <- err
				return
			}
		}

		log.Printf("🚀 starting mock grpc server. On host 0.0.0.0, with port: %d", s.port)

		if err := s.grpcSrv.Serve(s.listener); err != nil {
			log.Println(err, "serving grpc server with error")

			errCh <- err
			return
		}
	}()

	return errCh
}

func (s *server) stop() error {
	s.grpcSrv.GracefulStop()
	return libsSrv.CheckNetErrClosing(s.listener.Close())
}

func runMockGrpcPredictServer() (*server, *grpc.Server) {
	grpcServer := grpc.NewServer()

	mockGrpcServer := &server{grpcSrv: grpcServer}

	defer func() {
		if r := recover(); r != nil {
			_ = mockGrpcServer.stop()
			panic(r)
		}
	}()

	go func() {
		for errCh := range mockGrpcServer.start() {
			if errCh != nil {
				log.Printf("GRPC server listen error: %3v", errCh)
			}
		}
	}()

	pb.RegisterMlEngineServiceServer(grpcServer, mockGrpcServer)

	return mockGrpcServer, grpcServer
}

const testAPIKey = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJ0ZXN0IENyZWF0ZUNsaWVudCIsImV4cCI6MTY0NjkxNzI3Nywic3ViIjoiODM4NjY5ODAtM2UzNS0xMWVjLTlmMjQtYWNkZTQ4MDAxMTIyIn0.5QEuO6_ysdk2abGvk3Xp7Q25M4H4pIFXeqP2E7n9rKI"

type predictKubeMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

var testPredictKubeMetadata = []predictKubeMetadataTestData{
	// all properly formed
	{
		map[string]string{"predictHorizon": "2h", "historyTimeWindow": "7d", "prometheusAddress": "http://demo.robustperception.io:9090", "queryStep": "2m", "metricName": "http_requests_total", "threshold": "2000", "query": "up"},
		map[string]string{"apiKey": testAPIKey}, false,
	},
	// missing prometheusAddress
	{
		map[string]string{"predictHorizon": "2h", "historyTimeWindow": "7d", "prometheusAddress": "", "queryStep": "2m", "metricName": "http_requests_total", "threshold": "2000", "query": "up"},
		map[string]string{"apiKey": testAPIKey}, true,
	},
	// missing metricName
	{
		map[string]string{"predictHorizon": "2h", "historyTimeWindow": "7d", "prometheusAddress": "http://localhost:9090", "queryStep": "2m", "metricName": "", "threshold": "2000", "query": "up"},
		map[string]string{"apiKey": testAPIKey}, true,
	},
	// malformed threshold
	{
		map[string]string{"predictHorizon": "2h", "historyTimeWindow": "7d", "prometheusAddress": "http://localhost:9090", "queryStep": "2m", "metricName": "http_requests_total", "threshold": "one", "query": "up"},

		map[string]string{"apiKey": testAPIKey}, true,
	},
	// missing query
	{
		map[string]string{"predictHorizon": "2h", "historyTimeWindow": "7d", "prometheusAddress": "http://localhost:9090", "queryStep": "2m", "metricName": "http_requests_total", "threshold": "one", "query": ""},
		map[string]string{"apiKey": testAPIKey}, true,
	},
}

func TestPredictKubeParseMetadata(t *testing.T) {
	for _, testData := range testPredictKubeMetadata {
		_, err := parsePredictKubeMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

type predictKubeMetricIdentifier struct {
	metadataTestData *predictKubeMetadataTestData
	scalerIndex      int
	name             string
}

var predictKubeMetricIdentifiers = []predictKubeMetricIdentifier{
	{&testPredictKubeMetadata[0], 0, "s0-predictkube-http_requests_total"},
	{&testPredictKubeMetadata[0], 1, "s1-predictkube-http_requests_total"},
}

func TestPredictKubeGetMetricSpecForScaling(t *testing.T) {
	mockPredictServer, grpcServer := runMockGrpcPredictServer()
	defer func() {
		_ = mockPredictServer.stop()
		grpcServer.GracefulStop()
	}()

	mlEngineHost = "0.0.0.0"
	mlEnginePort = mockPredictServer.port

	for _, testData := range predictKubeMetricIdentifiers {
		mockPredictKubeScaler, err := NewPredictKubeScaler(
			context.Background(), &ScalerConfig{
				TriggerMetadata: testData.metadataTestData.metadata,
				AuthParams:      testData.metadataTestData.authParams,
				ScalerIndex:     testData.scalerIndex,
			},
		)
		assert.NoError(t, err)

		metricSpec := mockPredictKubeScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
			return
		}

		t.Log(metricSpec)
	}
}

func TestPredictKubeGetMetrics(t *testing.T) {
	mockPredictServer, grpcServer := runMockGrpcPredictServer()
	<-time.After(time.Second * 3)
	defer func() {
		_ = mockPredictServer.stop()
		grpcServer.GracefulStop()
	}()

	mlEngineHost = "0.0.0.0"
	mlEnginePort = mockPredictServer.port

	for _, testData := range predictKubeMetricIdentifiers {
		mockPredictKubeScaler, err := NewPredictKubeScaler(
			context.Background(), &ScalerConfig{
				TriggerMetadata: testData.metadataTestData.metadata,
				AuthParams:      testData.metadataTestData.authParams,
				ScalerIndex:     testData.scalerIndex,
			},
		)
		assert.NoError(t, err)

		result, err := mockPredictKubeScaler.GetMetrics(context.Background(), testData.metadataTestData.metadata["metricName"], nil)
		assert.NoError(t, err)
		assert.Equal(t, len(result), 1)
		assert.Equal(t, result[0].Value, *resource.NewQuantity(mockPredictServer.val, resource.DecimalSI))

		t.Logf("get: %v, want: %v, predictMetric: %d", result[0].Value, *resource.NewQuantity(mockPredictServer.val, resource.DecimalSI), mockPredictServer.val)
	}
}
