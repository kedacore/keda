package scalers

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/kedacore/keda/pkg/scalers/liiklus"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"log"
	"net"
	"testing"
)

type parseLiiklusMetadataTestData struct {
	metadata       map[string]string
	err            error
	liiklusAddress string
	group          string
	topic          string
	threshold      int64
}

var parseLiiklusMetadataTestDataset = []parseLiiklusMetadataTestData{
	{map[string]string{}, errors.New("no topic provided"), "", "", "", 0},
	{map[string]string{"topic": "foo"}, errors.New("no liiklus API address provided"), "", "", "", 0},
	{map[string]string{"topic": "foo", "address": "bar:6565"}, errors.New("no consumer group provided"), "", "", "", 0},
	{map[string]string{"topic": "foo", "address": "bar:6565", "group": "mygroup"}, nil, "bar:6565", "mygroup", "foo", 10},
	{map[string]string{"topic": "foo", "address": "bar:6565", "group": "mygroup", "lagThreshold": "15"}, nil, "bar:6565", "mygroup", "foo", 15},
}

func TestLiiklusParseMetadata(t *testing.T) {
	for _, testData := range parseLiiklusMetadataTestDataset {
		meta, err := parseLiiklusMetadata(testData.metadata)
		if err != nil && testData.err == nil {
			t.Error("Expected success but got error", err)
			continue
		}
		if testData.err != nil && err == nil {
			t.Error("Expected error but got success")
			continue
		}
		if testData.err != nil && err != nil && testData.err.Error() != err.Error() {
			t.Errorf("Expected error %v but got %v", testData.err, err)
			continue
		}
		if err != nil {
			continue
		}
		if testData.liiklusAddress != meta.address {
			t.Errorf("Expected address %q but got %q\n", testData.liiklusAddress, meta.address)
			continue
		}
		if meta.group != testData.group {
			t.Errorf("Expected group %q but got %q\n", testData.group, meta.group)
			continue
		}
		if meta.topic != testData.topic {
			t.Errorf("Expected topic %q but got %q\n", testData.topic, meta.topic)
			continue
		}
		if meta.lagThreshold != testData.threshold {
			t.Errorf("Expected threshold %d but got %d\n", testData.threshold, meta.lagThreshold)
			continue
		}
	}
}

type test_liiklus_server struct {
	offsets    map[uint32]uint64
	endOffsets map[uint32]uint64
	address    string
}

func (l *test_liiklus_server) GetOffsets(context.Context, *liiklus.GetOffsetsRequest) (*liiklus.GetOffsetsReply, error) {
	return &liiklus.GetOffsetsReply{
		Offsets: l.offsets,
	}, nil
}
func (l *test_liiklus_server) Publish(context.Context, *liiklus.PublishRequest) (*liiklus.PublishReply, error) {
	return nil, nil
}
func (l *test_liiklus_server) Subscribe(*liiklus.SubscribeRequest, liiklus.LiiklusService_SubscribeServer) error {
	return nil
}
func (l *test_liiklus_server) Receive(*liiklus.ReceiveRequest, liiklus.LiiklusService_ReceiveServer) error {
	return nil
}
func (l *test_liiklus_server) Ack(context.Context, *liiklus.AckRequest) (*empty.Empty, error) {
	return nil, nil
}
func (l *test_liiklus_server) GetEndOffsets(context.Context, *liiklus.GetEndOffsetsRequest) (*liiklus.GetEndOffsetsReply, error) {
	return &liiklus.GetEndOffsetsReply{
		Offsets: l.endOffsets,
	}, nil
}

func TestLiiklusScalerActiveBehavior(t *testing.T) {

	srv, liiklus, err := createTestServer()
	if err != nil {
		t.Errorf("failed to create server: %v", err)
		return
	}
	defer srv.Stop()

	liiklus.endOffsets = map[uint32]uint64{0: 2}
	liiklus.offsets = map[uint32]uint64{0: 1}

	scaler, err := NewLiiklusScaler(nil, map[string]string{"topic": "foo", "address": liiklus.address, "group": "mygroup"})
	if err != nil {
		t.Errorf("failed to create scaler: %v", err)
		return
	}
	defer scaler.Close()
	active, err := scaler.IsActive(context.Background())
	if err != nil {
		t.Errorf("error calling IsActive: %v", err)
		return
	}
	if !active {
		t.Error("expected IsActive to return true")
	}

	liiklus.endOffsets = map[uint32]uint64{0: 2}
	liiklus.offsets = map[uint32]uint64{0: 2}
	active, err = scaler.IsActive(context.Background())
	if err != nil {
		t.Errorf("error calling IsActive: %v", err)
		return
	}
	if active {
		t.Error("expected IsActive to return false")
	}
}

func TestLiiklusScalerGetMetricsBehavior(t *testing.T) {

	srv, liiklus, err := createTestServer()
	if err != nil {
		t.Errorf("failed to create server: %v", err)
		return
	}
	defer srv.Stop()

	scaler, err := NewLiiklusScaler(nil, map[string]string{"topic": "foo", "address": liiklus.address, "group": "mygroup"})
	if err != nil {
		t.Errorf("failed to create scaler: %v", err)
		return
	}
	defer scaler.Close()

	liiklus.endOffsets = map[uint32]uint64{0: 20, 1: 30}
	liiklus.offsets = map[uint32]uint64{0: 18, 1: 25}
	values, err := scaler.GetMetrics(context.Background(), "m", nil)
	if err != nil {
		t.Errorf("error calling IsActive: %v", err)
		return
	}

	if values[0].Value.Value() != 2+5 {
		t.Errorf("got wrong metric values: %v", values)
	}

	// Test metrics capping
	liiklus.endOffsets = map[uint32]uint64{0: 20, 1: 30}
	liiklus.offsets = map[uint32]uint64{0: 1, 1: 15}
	values, err = scaler.GetMetrics(context.Background(), "m", nil)
	if err != nil {
		t.Errorf("error calling IsActive: %v", err)
		return
	}

	if values[0].Value.Value() != 10+10 {
		t.Errorf("got wrong metric values: %v", values)
	}

}

func createTestServer() (*grpc.Server, *test_liiklus_server, error) {
	port := ":6565"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	server := &test_liiklus_server{
		endOffsets: make(map[uint32]uint64),
		offsets:    make(map[uint32]uint64),
		address:    "localhost" + port,
	}
	liiklus.RegisterLiiklusServiceServer(s, server)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	return s, server, err
}
