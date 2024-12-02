package scalers

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	libsSrv "github.com/dysnix/predictkube-libs/external/grpc/server"
	pb "github.com/dysnix/predictkube-proto/external/proto/services"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

const (
	defaultTestPort = 50051
	// nosemgrep: detected-jwt-token, detected-generic-api-key
	testAPIKey = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJ0ZXN0IENyZWF0ZUNsaWVudCIsImV4cCI6MTY0NjkxNzI3Nywic3ViIjoiODM4NjY5ODAtM2UzNS0xMWVjLTlmMjQtYWNkZTQ4MDAxMTIyIn0.5QEuO6_ysdk2abGvk3Xp7Q25M4H4pIFXeqP2E7n9rKI"
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

func (s *server) stop() error {
	s.grpcSrv.GracefulStop()
	return libsSrv.CheckNetErrClosing(s.listener.Close())
}

func runMockGrpcPredictServer() (*server, *grpc.Server) {
	// nosemgrep
	grpcServer := grpc.NewServer()
	mockGrpcServer := &server{
		grpcSrv: grpcServer,
		port:    defaultTestPort,
	}

	defer func() {
		if r := recover(); r != nil {
			_ = mockGrpcServer.stop()
			panic(r)
		}
	}()

	pb.RegisterMlEngineServiceServer(grpcServer, mockGrpcServer)

	serverURL := fmt.Sprintf("0.0.0.0:%d", mockGrpcServer.port)
	listener, err := net.Listen("tcp4", serverURL)
	if err != nil {
		log.Printf("Failed to listen: %v", err)
		return nil, nil
	}
	mockGrpcServer.listener = listener

	go func() {
		log.Printf("🚀 starting mock grpc server. On host 0.0.0.0, with port: %d", mockGrpcServer.port)
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("GRPC server listen error: %v", err)
		}
	}()

	time.Sleep(time.Second)

	return mockGrpcServer, grpcServer
}

type predictKubeMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

var testPredictKubeMetadata = []predictKubeMetadataTestData{
	// all properly formed
	{
		map[string]string{"predictHorizon": "2h", "historyTimeWindow": "7d", "prometheusAddress": "http://demo.robustperception.io:9090", "queryStep": "2m", "threshold": "2000", "query": "up"},
		map[string]string{"apiKey": testAPIKey}, false,
	},
	// missing prometheusAddress
	{
		map[string]string{"predictHorizon": "2h", "historyTimeWindow": "7d", "prometheusAddress": "", "queryStep": "2m", "threshold": "2000", "query": "up"},
		map[string]string{"apiKey": testAPIKey}, true,
	},
	// malformed threshold
	{
		map[string]string{"predictHorizon": "2h", "historyTimeWindow": "7d", "prometheusAddress": "http://localhost:9090", "queryStep": "2m", "threshold": "one", "query": "up"},
		map[string]string{"apiKey": testAPIKey}, true,
	},
	// malformed activation threshold
	{
		map[string]string{"predictHorizon": "2h", "historyTimeWindow": "7d", "prometheusAddress": "http://localhost:9090", "queryStep": "2m", "threshold": "1", "activationThreshold": "one", "query": "up"},
		map[string]string{"apiKey": testAPIKey}, true,
	},
	// missing query
	{
		map[string]string{"predictHorizon": "2h", "historyTimeWindow": "7d", "prometheusAddress": "http://localhost:9090", "queryStep": "2m", "threshold": "one", "query": ""},
		map[string]string{"apiKey": testAPIKey}, true,
	},
}

func TestPredictKubeParseMetadata(t *testing.T) {
	for _, testData := range testPredictKubeMetadata {
		_, err := parsePredictKubeMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
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
	triggerIndex     int
	name             string
}

var predictKubeMetricIdentifiers = []predictKubeMetricIdentifier{
	{&testPredictKubeMetadata[0], 0, fmt.Sprintf("s0-predictkube-%s", predictKubeMetricPrefix)},
	{&testPredictKubeMetadata[0], 1, fmt.Sprintf("s1-predictkube-%s", predictKubeMetricPrefix)},
}

func TestPredictKubeGetMetricSpecForScaling(t *testing.T) {
	mockPredictServer, grpcServer := runMockGrpcPredictServer()
	if mockPredictServer == nil || grpcServer == nil {
		t.Fatal("Failed to start mock server")
	}

	defer func() {
		_ = mockPredictServer.stop()
		grpcServer.GracefulStop()
	}()

	err := waitForServer(fmt.Sprintf("0.0.0.0:%d", defaultTestPort), 5*time.Second)
	if err != nil {
		t.Fatalf("Server failed to start: %v", err)
	}

	// Mock Prometheus server
	mockPrometheus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
	}))
	defer mockPrometheus.Close()

	mlEngineHost = "0.0.0.0"
	mlEnginePort = mockPredictServer.port

	for _, testData := range predictKubeMetricIdentifiers {
		metadata := make(map[string]string)
		for k, v := range testData.metadataTestData.metadata {
			if k == "prometheusAddress" {
				metadata[k] = mockPrometheus.URL
			} else {
				metadata[k] = v
			}
		}

		mockPredictKubeScaler, err := NewPredictKubeScaler(
			context.Background(),
			&scalersconfig.ScalerConfig{
				TriggerMetadata: metadata,
				AuthParams:      testData.metadataTestData.authParams,
				TriggerIndex:    testData.triggerIndex,
			},
		)
		assert.NoError(t, err)

		metricSpec := mockPredictKubeScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
			return
		}
	}
}

func waitForServer(address string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for server")
		default:
			conn, err := net.Dial("tcp", address)
			if err == nil {
				conn.Close()
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func TestPredictKubeGetMetrics(t *testing.T) {
	mockPredictServer, grpcServer := runMockGrpcPredictServer()
	if mockPredictServer == nil || grpcServer == nil {
		t.Fatal("Failed to start mock server")
	}

	defer func() {
		_ = mockPredictServer.stop()
		grpcServer.GracefulStop()
	}()

	err := waitForServer(fmt.Sprintf("0.0.0.0:%d", defaultTestPort), 5*time.Second)
	if err != nil {
		t.Fatalf("Server failed to start: %v", err)
	}

	// Mock Prometheus server
	mockPrometheus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
	}))
	defer mockPrometheus.Close()

	grpcConf.Conn.Insecure = true
	mlEngineHost = "0.0.0.0"
	mlEnginePort = defaultTestPort

	for _, testData := range predictKubeMetricIdentifiers {
		testData := testData
		t.Run(fmt.Sprintf("trigger_index_%d", testData.triggerIndex), func(t *testing.T) {
			metadata := make(map[string]string)
			for k, v := range testData.metadataTestData.metadata {
				if k == "prometheusAddress" {
					metadata[k] = mockPrometheus.URL
				} else {
					metadata[k] = v
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			mockPredictKubeScaler, err := NewPredictKubeScaler(
				ctx,
				&scalersconfig.ScalerConfig{
					TriggerMetadata: metadata,
					AuthParams:      testData.metadataTestData.authParams,
					TriggerIndex:    testData.triggerIndex,
				},
			)
			if err != nil {
				t.Fatalf("Failed to create scaler: %v", err)
			}

			result, _, err := mockPredictKubeScaler.GetMetricsAndActivity(ctx, predictKubeMetricPrefix)
			if err != nil {
				t.Fatalf("Failed to get metrics: %v", err)
			}

			if len(result) == 0 {
				t.Fatal("Expected non-empty result")
			}

			assert.Equal(t, result[0].Value, *resource.NewMilliQuantity(mockPredictServer.val*1000, resource.DecimalSI))
			t.Logf("get: %v, want: %v, predictMetric: %d",
				result[0].Value,
				*resource.NewQuantity(mockPredictServer.val, resource.DecimalSI),
				mockPredictServer.val)
		})
	}
}
