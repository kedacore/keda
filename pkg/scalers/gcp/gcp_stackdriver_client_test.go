package gcp

import (
	"context"
	"net"
	"testing"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"google.golang.org/api/iterator"
	option "google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type fakeMetricServer struct {
	monitoringpb.UnimplementedMetricServiceServer
}

func (fakeMetricServer) ListTimeSeries(context.Context, *monitoringpb.ListTimeSeriesRequest) (*monitoringpb.ListTimeSeriesResponse, error) {
	return &monitoringpb.ListTimeSeriesResponse{}, nil
}

func TestGRPCClientOptionsDisableGoogleTelemetry(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	monitoringpb.RegisterMetricServiceServer(server, fakeMetricServer{})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)

	reader := sdkmetric.NewManualReader()
	previous := otel.GetMeterProvider()
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	t.Cleanup(func() { otel.SetMeterProvider(previous) })

	opts := append(grpcClientOptions(),
		option.WithEndpoint(listener.Addr().String()),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	client, err := monitoring.NewMetricClient(t.Context(), opts...)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	_, err = client.ListTimeSeries(t.Context(), &monitoringpb.ListTimeSeriesRequest{Name: "projects/test"}).Next()
	require.ErrorIs(t, err, iterator.Done)

	var got metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &got))
	for _, scope := range got.ScopeMetrics {
		assert.NotEqual(t, otelgrpc.ScopeName, scope.Scope.Name, "Google's default otelgrpc handler must be disabled")
	}
}

func TestBuildMQLQuery(t *testing.T) {
	for _, tc := range []struct {
		name         string
		resourceType string
		metric       string
		resourceName string
		aggregation  string
		timeHorizon  time.Duration

		expected string
		isError  bool
	}{
		{
			"topic with aggregation",
			"topic", "pubsub.googleapis.com/topic/x", "mytopic", "count", time.Minute,
			"fetch pubsub_topic | metric 'pubsub.googleapis.com/topic/x' | filter (resource.project_id == 'myproject' && resource.topic_id == 'mytopic')" +
				" | within 1m0s | align delta(3m0s) | every 3m0s | group_by [], count(value)",
			false,
		},
		{
			"topic without aggregation",
			"topic", "pubsub.googleapis.com/topic/x", "mytopic", "", time.Duration(0),
			"fetch pubsub_topic | metric 'pubsub.googleapis.com/topic/x' | filter (resource.project_id == 'myproject' && resource.topic_id == 'mytopic')" +
				" | within 2m0s",
			false,
		},
		{
			"subscription with aggregation",
			"subscription", "pubsub.googleapis.com/subscription/x", "mysubscription", "percentile99", time.Duration(0),
			"fetch pubsub_subscription | metric 'pubsub.googleapis.com/subscription/x' | filter (resource.project_id == 'myproject' && resource.subscription_id == 'mysubscription')" +
				" | within 5m0s | align delta(3m0s) | every 3m0s | group_by [], percentile(value, 99)",
			false,
		},
		{
			"subscription without aggregation",
			"subscription", "pubsub.googleapis.com/subscription/x", "mysubscription", "", time.Minute * 4,
			"fetch pubsub_subscription | metric 'pubsub.googleapis.com/subscription/x' | filter (resource.project_id == 'myproject' && resource.subscription_id == 'mysubscription')" +
				" | within 4m0s",
			false,
		},
		{
			"invalid percentile",
			"topic", "pubsub.googleapis.com/topic/x", "mytopic", "percentile101", time.Minute,
			"invalid percentile value: 101",
			true,
		},
		{
			"unsupported aggregation function",
			"topic", "pubsub.googleapis.com/topic/x", "mytopic", "max", time.Duration(0),
			"unsupported aggregation function: max",
			true,
		},
	} {
		s := &StackDriverClient{}
		t.Run(tc.name, func(t *testing.T) {
			q, err := s.BuildMQLQuery("myproject", tc.resourceType, tc.metric, tc.resourceName, tc.aggregation, tc.timeHorizon)
			if tc.isError {
				assert.Error(t, err)
				assert.Equal(t, tc.expected, err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, q)
			}
		})
	}
}

func TestGetActualProjectID(t *testing.T) {
	// There are three ways to get projectID
	// This is ordered from highest priority to lowest priority
	pidFromMetadata := "myproject0"
	pidFromClient := "myproject1"
	pidFromClientCreds := "myproject2"

	for _, tc := range []struct {
		name      string
		projectID string
		client    *StackDriverClient
		expected  string
	}{
		{
			"all projectID present",
			pidFromMetadata,
			&StackDriverClient{
				projectID: pidFromClient,
				credentials: GoogleApplicationCredentials{
					ProjectID: pidFromClientCreds,
				},
			},
			pidFromMetadata,
		},
		{
			"both projectID from metadata and client present",
			pidFromMetadata,
			&StackDriverClient{
				projectID: pidFromClient,
			},
			pidFromMetadata,
		},
		{
			"both projectID from metadata and client credentials present",
			pidFromMetadata,
			&StackDriverClient{
				credentials: GoogleApplicationCredentials{
					ProjectID: pidFromClientCreds,
				},
			},
			pidFromMetadata,
		},
		{
			"both projectID from client and client credentials present",
			"",
			&StackDriverClient{
				projectID: pidFromClient,
				credentials: GoogleApplicationCredentials{
					ProjectID: pidFromClientCreds,
				},
			},
			pidFromClient,
		},
		{
			"projectID from metadata only",
			pidFromMetadata,
			&StackDriverClient{},
			pidFromMetadata,
		},
		{
			"projectID from client only",
			"",
			&StackDriverClient{
				projectID: pidFromClient,
			},
			pidFromClient,
		},
		{
			"projectID from client credentials only",
			"",
			&StackDriverClient{
				projectID: "",
				credentials: GoogleApplicationCredentials{
					ProjectID: pidFromClientCreds,
				},
			},
			pidFromClientCreds,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pid := getActualProjectID(tc.client, tc.projectID)
			assert.Equal(t, pid, tc.expected)
		})
	}
}
