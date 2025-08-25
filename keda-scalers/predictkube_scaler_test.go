package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	libsSrv "github.com/dysnix/predictkube-libs/external/grpc/server"
	pb "github.com/dysnix/predictkube-proto/external/proto/services"
	"github.com/phayes/freeport"
	prometheusV1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if r.RequestURI == "/api/v1/status/runtimeinfo" {
		w.WriteHeader(http.StatusOK)
		runtimeInfo := prometheusV1.RuntimeinfoResult{
			StartTime: time.Now(),
		}
		data, _ := json.Marshal(runtimeInfo)
		response := struct {
			Data json.RawMessage `json:"data"`
		}{
			Data: data,
		}
		_ = json.NewEncoder(w).Encode(response)
		return
	}
	if r.RequestURI == "/api/v1/query_range" {
		w.WriteHeader(http.StatusOK)
		result := struct {
			Type   model.ValueType `json:"resultType"`
			Result interface{}     `json:"result"`
		}{
			Type: model.ValScalar,
			Result: model.Scalar{
				Value:     model.ZeroSamplePair.Value,
				Timestamp: model.Now(),
			},
		}
		data, _ := json.Marshal(result)
		response := struct {
			Data json.RawMessage `json:"data"`
		}{
			Data: data,
		}
		_ = json.NewEncoder(w).Encode(response)
		return
	}
}))

type server struct {
	pb.UnimplementedMlEngineServiceServer
	mu       sync.Mutex
	grpcSrv  *grpc.Server
	listener net.Listener
	port     int
	val      int64
}

func (s *server) GetPredictMetric(_ context.Context, _ *pb.ReqGetPredictMetric) (res *pb.ResGetPredictMetric, err error) {
	s.mu.Lock()
	s.val = int64(rand.Intn(30000-10000) + 10000)
	predictVal := s.val
	s.mu.Unlock()

	return &pb.ResGetPredictMetric{
		ResultMetric: predictVal,
	}, nil
}

func (s *server) getPort() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.port
}

func (s *server) start() <-chan error {
	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)

		var (
			err  error
			port int
		)

		port, err = freeport.GetFreePort()
		if err != nil {
			log.Fatalf("Could not get free port for init mock grpc server: %s", err)
		}

		s.mu.Lock()
		s.port = port
		s.mu.Unlock()

		serverURL := fmt.Sprintf("0.0.0.0:%d", port)

		var listener net.Listener
		listener, err = net.Listen("tcp4", serverURL)
		if err != nil {
			log.Println("starting grpc server with error")
			errCh <- err
			return
		}

		s.mu.Lock()
		s.listener = listener
		s.mu.Unlock()

		log.Printf("🚀 starting mock grpc server. On host 0.0.0.0, with port: %d", port)

		if err := s.grpcSrv.Serve(listener); err != nil {
			log.Println(err, "serving grpc server with error")
			errCh <- err
			return
		}
	}()

	return errCh
}

func (s *server) stop() error {
	s.grpcSrv.GracefulStop()

	s.mu.Lock()
	listener := s.listener
	s.mu.Unlock()

	if listener != nil {
		return libsSrv.CheckNetErrClosing(listener.Close())
	}
	return nil
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
		map[string]string{"predictHorizon": "2h", "historyTimeWindow": "7d", "prometheusAddress": apiStub.URL, "queryStep": "2m", "threshold": "2000", "query": "up"},
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

	defer func() {
		_ = mockPredictServer.stop()
		grpcServer.GracefulStop()
	}()

	mlEngineHost = "0.0.0.0"
	mlEnginePort = mockPredictServer.getPort()

	for _, testData := range predictKubeMetricIdentifiers {
		mockPredictKubeScaler, err := NewPredictKubeScaler(
			context.Background(), &scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadataTestData.metadata,
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

		t.Log(metricSpec)
	}
}

func TestPredictKubeGetMetrics(t *testing.T) {
	grpcConf.Conn.Insecure = true

	mockPredictServer, grpcServer := runMockGrpcPredictServer()
	<-time.After(time.Second * 3)
	defer func() {
		_ = mockPredictServer.stop()
		grpcServer.GracefulStop()
	}()

	mlEngineHost = "0.0.0.0"
	mlEnginePort = mockPredictServer.getPort()

	for _, testData := range predictKubeMetricIdentifiers {
		mockPredictKubeScaler, err := NewPredictKubeScaler(
			context.Background(), &scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadataTestData.metadata,
				AuthParams:      testData.metadataTestData.authParams,
				TriggerIndex:    testData.triggerIndex,
			},
		)
		assert.NoError(t, err)

		result, _, err := mockPredictKubeScaler.GetMetricsAndActivity(context.Background(), predictKubeMetricPrefix)
		assert.NoError(t, err)
		assert.Equal(t, len(result), 1)

		mockPredictServer.mu.Lock()
		predictVal := mockPredictServer.val
		mockPredictServer.mu.Unlock()

		assert.Equal(t, result[0].Value, *resource.NewMilliQuantity(predictVal*1000, resource.DecimalSI))

		t.Logf("get: %v, want: %v, predictMetric: %d", result[0].Value, *resource.NewQuantity(predictVal, resource.DecimalSI), predictVal)
	}
}
