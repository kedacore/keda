package scalers

import (
	"context"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/local"

	"github.com/kedacore/keda/v2/pkg/metricscollector"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type fakeEtcdWatchServer struct {
	etcdserverpb.UnimplementedWatchServer
}

func (fakeEtcdWatchServer) Watch(stream etcdserverpb.Watch_WatchServer) error {
	if _, err := stream.Recv(); err != nil {
		return err
	}
	header := &etcdserverpb.ResponseHeader{Revision: 1}
	if err := stream.Send(&etcdserverpb.WatchResponse{Header: header, WatchId: 1, Created: true}); err != nil {
		return err
	}
	if err := stream.Send(&etcdserverpb.WatchResponse{Header: header, WatchId: 1, Events: []*mvccpb.Event{{
		Type: mvccpb.PUT,
		Kv:   &mvccpb.KeyValue{Key: []byte("watch-key"), Value: []byte("5"), ModRevision: 1},
	}}}); err != nil {
		return err
	}
	<-stream.Context().Done()
	return nil
}

func TestEtcdRunLabelsWatchStream(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer(grpc.Creds(local.NewCredentials()))
	etcdserverpb.RegisterWatchServer(server, fakeEtcdWatchServer{})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)

	streamCtxs := make(chan context.Context, 1)
	captureStreamCtx := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		select {
		case streamCtxs <- ctx:
		default:
		}
		return streamer(ctx, desc, cc, method, opts...)
	}
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{listener.Addr().String()},
		DialTimeout: 5 * time.Second,
		DialOptions: []grpc.DialOption{
			grpc.WithTransportCredentials(local.NewCredentials()),
			grpc.WithChainStreamInterceptor(captureStreamCtx),
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	scaler := &etcdScaler{
		metadata: &etcdMetadata{WatchKey: "watch-key", WatchProgressNotifyInterval: defaultWatchProgressNotifyInterval},
		client:   client,
		scalerConfig: scalersconfig.ScalerConfig{
			TriggerType:             "etcd",
			TriggerName:             "etcd-trigger",
			ScalableObjectNamespace: "etcd-namespace",
			ScalableObjectName:      "etcd-resource",
		},
		logger: logr.Discard(),
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	active := make(chan bool)
	go scaler.Run(ctx, active)

	select {
	case isActive := <-active:
		assert.True(t, isActive)
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for the watch event")
	}

	var streamCtx context.Context
	select {
	case streamCtx = <-streamCtxs:
	case <-time.After(time.Second):
		t.Fatal("watch stream was not opened through the client interceptors")
	}
	assert.Equal(t, "etcd", streamCtx.Value(metricscollector.ScalerContextKey))
	assert.Equal(t, "etcd-trigger", streamCtx.Value(metricscollector.TriggerNameContextKey))
	assert.Equal(t, "", streamCtx.Value(metricscollector.MetricNameContextKey))
	assert.Equal(t, "etcd-namespace", streamCtx.Value(metricscollector.NamespaceContextKey))
	assert.Equal(t, "etcd-resource", streamCtx.Value(metricscollector.ScaledResourceContextKey))

	cancel()
	select {
	case _, open := <-active:
		assert.False(t, open, "Run must close the active channel once its context is cancelled")
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after its context was cancelled")
	}
}

type parseEtcdMetadataTestData struct {
	metadata  map[string]string
	endpoints []string
	isError   bool
}

type parseEtcdAuthParamsTestData struct {
	authParams map[string]string
	isError    bool
	enableTLS  string
}

type etcdMetricIdentifier struct {
	metadataTestData *parseEtcdMetadataTestData
	triggerIndex     int
	name             string
}

// A complete valid metadata example for reference
var validEtcdMetadata = map[string]string{
	"endpoints":                   "172.0.0.1:2379,172.0.0.2:2379,172.0.0.3:2379",
	"watchKey":                    "length",
	"value":                       "5.5",
	"activationValue":             "0.5",
	"watchProgressNotifyInterval": "600",
}

var parseEtcdMetadataTestDataset = []parseEtcdMetadataTestData{
	// success
	{map[string]string{"endpoints": "172.0.0.1:2379,172.0.0.2:2379,172.0.0.3:2379", "watchKey": "length", "value": "5.5", "activationValue": "0.5", "watchProgressNotifyInterval": "600"}, []string{"172.0.0.1:2379", "172.0.0.2:2379", "172.0.0.3:2379"}, false},
	// success
	{map[string]string{"endpoints": "172.0.0.1:2379", "watchKey": "var", "value": "5.5", "activationValue": "0.5", "watchProgressNotifyInterval": "600"}, []string{"172.0.0.1:2379"}, false},
	// failure, endpoints missed
	{map[string]string{"endpoints": "", "watchKey": "length", "value": "5", "activationValue": "0", "watchProgressNotifyInterval": "600"}, []string{""}, true},
	// failure, watchKey missed
	{map[string]string{"endpoints": "172.0.0.1:2379", "watchKey": "", "value": "5", "activationValue": "0", "watchProgressNotifyInterval": "600"}, []string{"172.0.0.1:2379"}, true},
	// failure, value invalid
	{map[string]string{"endpoints": "172.0.0.1:2379", "watchKey": "length", "value": "a", "activationValue": "0", "watchProgressNotifyInterval": "600"}, []string{"172.0.0.1:2379"}, true},
	// failure, activationValue invalid
	{map[string]string{"endpoints": "172.0.0.1:2379", "watchKey": "length", "value": "5", "activationValue": "b", "watchProgressNotifyInterval": "600"}, []string{"172.0.0.1:2379"}, true},
	// failure, watchProgressNotifyInterval invalid
	{map[string]string{"endpoints": "172.0.0.1:2379", "watchKey": "length", "value": "5", "activationValue": "0", "watchProgressNotifyInterval": "0"}, []string{"172.0.0.1:2379"}, true},
}

var parseEtcdAuthParamsTestDataset = []parseEtcdAuthParamsTestData{
	// success, TLS only
	{map[string]string{"tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, false, etcdTLSEnable},
	// success, TLS cert/key and assumed public CA
	{map[string]string{"tls": "enable", "cert": "ceert", "key": "keey"}, false, etcdTLSEnable},
	// success, TLS cert/key + key password and assumed public CA
	{map[string]string{"tls": "enable", "cert": "ceert", "key": "keey", "keyPassword": "keeyPassword"}, false, etcdTLSEnable},
	// success, TLS CA only
	{map[string]string{"tls": "enable", "ca": "caaa"}, false, etcdTLSEnable},
	// failure, TLS missing cert
	{map[string]string{"tls": "enable", "ca": "caaa", "key": "keey"}, true, etcdTLSDisable},
	// failure, TLS missing key
	{map[string]string{"tls": "enable", "ca": "caaa", "cert": "ceert"}, true, etcdTLSDisable},
	// failure, TLS invalid
	{map[string]string{"tls": "yes", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, etcdTLSDisable},
	// success, username and password
	{map[string]string{"username": "root", "password": "admin"}, false, etcdTLSDisable},
	// failure, missing password
	{map[string]string{"username": "root"}, true, etcdTLSDisable},
	// failure, missing username
	{map[string]string{"password": "admin"}, true, etcdTLSDisable},
}

var etcdMetricIdentifiers = []etcdMetricIdentifier{
	{&parseEtcdMetadataTestDataset[0], 0, "s0-etcd-length"},
	{&parseEtcdMetadataTestDataset[1], 1, "s1-etcd-var"},
}

func TestParseEtcdMetadata(t *testing.T) {
	for _, testData := range parseEtcdMetadataTestDataset {
		meta, err := parseEtcdMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error but got success %v", testData)
		}
		if err == nil && !reflect.DeepEqual(meta.Endpoints, testData.endpoints) {
			t.Errorf("Expected  %v but got %v\n", testData.endpoints, meta.Endpoints)
		}
	}
}

func TestParseEtcdAuthParams(t *testing.T) {
	for _, testData := range parseEtcdAuthParamsTestDataset {
		meta, err := parseEtcdMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: validEtcdMetadata, AuthParams: testData.authParams})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		if meta != nil && meta.EnableTLS != testData.enableTLS {
			t.Errorf("Expected enableTLS to be set to %v but got %v\n", testData.enableTLS, meta.EnableTLS)
		}
		if meta != nil && meta.EnableTLS == etcdTLSEnable {
			if meta.Ca != testData.authParams["ca"] {
				t.Errorf("Expected ca to be set to %v but got %v\n", testData.authParams["ca"], meta.EnableTLS)
			}
			if meta.Cert != testData.authParams["cert"] {
				t.Errorf("Expected cert to be set to %v but got %v\n", testData.authParams["cert"], meta.Cert)
			}
			if meta.Key != testData.authParams["key"] {
				t.Errorf("Expected key to be set to %v but got %v\n", testData.authParams["key"], meta.Key)
			}
			if meta.KeyPassword != testData.authParams["keyPassword"] {
				t.Errorf("Expected key to be set to %v but got %v\n", testData.authParams["keyPassword"], meta.Key)
			}
		}
	}
}

func TestEtcdGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range etcdMetricIdentifiers {
		meta, err := parseEtcdMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockEtcdScaler := etcdScaler{metadata: meta, logger: logr.Logger{}}

		metricSpec := mockEtcdScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
