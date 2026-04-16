package scalers

import (
	"context"
	"errors"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	workflowservicemock "go.temporal.io/api/workflowservicemock/v1"
	sdk "go.temporal.io/sdk/client"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

var (
	temporalEndpoint  = "localhost:7233"
	temporalNamespace = "v2"
	temporalQueueName = "default"

)

type parseTemporalMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type temporalMetricIdentifier struct {
	metadataTestData *parseTemporalMetadataTestData
	triggerIndex     int
	name             string
}

var testTemporalMetadata = []parseTemporalMetadataTestData{
	// nothing passed
	{map[string]string{}, true},
	// Missing taskQueue, should fail
	{map[string]string{"endpoint": temporalEndpoint, "namespace": temporalNamespace}, true},
	// Missing namespace, should success
	{map[string]string{"endpoint": temporalEndpoint, "taskQueue": temporalQueueName}, false},
	// Missing endpoint, should fail
	{map[string]string{"taskQueue": temporalQueueName, "namespace": temporalNamespace}, true},
	// invalid minConnectTimeout
	{map[string]string{"endpoint": temporalEndpoint, "taskQueue": temporalQueueName, "namespace": temporalNamespace, "minConnectTimeout": "-1"}, true},
	// All good.
	{map[string]string{"endpoint": temporalEndpoint, "taskQueue": temporalQueueName, "namespace": temporalNamespace}, false},
	// All good + activationLagThreshold
	{map[string]string{"endpoint": temporalEndpoint, "taskQueue": temporalQueueName, "namespace": temporalNamespace, "activationTargetQueueSize": "10"}, false},
	// workerVersioningType=deployment without buildId
	{map[string]string{"endpoint": temporalEndpoint, "taskQueue": temporalQueueName, "namespace": temporalNamespace, "workerVersioningType": "deployment", "deploymentName": "my-deploy"}, true},
	// workerVersioningType=deployment without deploymentName
	{map[string]string{"endpoint": temporalEndpoint, "taskQueue": temporalQueueName, "namespace": temporalNamespace, "workerVersioningType": "deployment", "buildId": "v1"}, true},
	// workerVersioningType=deployment + queueTypes
	{map[string]string{"endpoint": temporalEndpoint, "taskQueue": temporalQueueName, "namespace": temporalNamespace, "workerVersioningType": "deployment", "deploymentName": "my-deploy", "buildId": "v1", "queueTypes": "workflow"}, true},
	// workerVersioningType=build-id + deploymentName
	{map[string]string{"endpoint": temporalEndpoint, "taskQueue": temporalQueueName, "namespace": temporalNamespace, "workerVersioningType": "build-id", "buildId": "v1", "deploymentName": "my-deploy"}, true},
	// unversioned + buildId (missing workerVersioningType)
	{map[string]string{"endpoint": temporalEndpoint, "taskQueue": temporalQueueName, "namespace": temporalNamespace, "buildId": "v1"}, true},
	// unversioned + deploymentName (missing workerVersioningType)
	{map[string]string{"endpoint": temporalEndpoint, "taskQueue": temporalQueueName, "namespace": temporalNamespace, "deploymentName": "my-deploy"}, true},
	// unknown workerVersioningType
	{map[string]string{"endpoint": temporalEndpoint, "taskQueue": temporalQueueName, "namespace": temporalNamespace, "workerVersioningType": "invalid"}, true},
	// valid deployment config
	{map[string]string{"endpoint": temporalEndpoint, "taskQueue": temporalQueueName, "namespace": temporalNamespace, "workerVersioningType": "deployment", "deploymentName": "my-deploy", "buildId": "v1"}, false},
	// valid build-id config
	{map[string]string{"endpoint": temporalEndpoint, "taskQueue": temporalQueueName, "namespace": temporalNamespace, "workerVersioningType": "build-id", "buildId": "v1"}, false},
	// invalid queueType value
	{map[string]string{"endpoint": temporalEndpoint, "taskQueue": temporalQueueName, "namespace": temporalNamespace, "queueTypes": "worflow"}, true},
}

var temporalMetricIdentifiers = []temporalMetricIdentifier{
	{&testTemporalMetadata[5], 0, "s0-temporal-v2-default"},
	{&testTemporalMetadata[5], 1, "s1-temporal-v2-default"},
	{&testTemporalMetadata[14], 0, "s0-temporal-v2-default-my-deploy-v1"},
	{&testTemporalMetadata[15], 0, "s0-temporal-v2-default-v1"},
}

func TestTemporalParseMetadata(t *testing.T) {
	for _, testData := range testTemporalMetadata {
		metadata := &scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata}
		_, err := parseTemporalMetadata(metadata)

		if err != nil && !testData.isError {
			t.Error("Expected success but got err", err)
		}
		if err == nil && testData.isError {
			t.Error("Expected error but got success")
		}
	}
}

func TestTemporalGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range temporalMetricIdentifiers {
		metadata, err := parseTemporalMetadata(&scalersconfig.ScalerConfig{
			TriggerMetadata: testData.metadataTestData.metadata,
			TriggerIndex:    testData.triggerIndex,
		})

		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockScaler := temporalScaler{
			metadata: metadata,
		}
		metricSpec := mockScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name

		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestParseTemporalMetadata(t *testing.T) {
	cases := []struct {
		name        string
		metadata    map[string]string
		wantMeta    *temporalMetadata
		authParams  map[string]string
		resolvedEnv map[string]string
		wantErr     bool
	}{
		{
			name: "empty queue name",
			metadata: map[string]string{
				"endpoint":  "test:7233",
				"namespace": "default",
			},
			wantMeta: nil,
			wantErr:  true,
		},
		{
			name: "empty namespace",
			metadata: map[string]string{
				"endpoint":  "test:7233",
				"taskQueue": "testxx",
			},
			wantMeta: &temporalMetadata{
				Endpoint:                  "test:7233",
				Namespace:                 "default",
				TaskQueue:                 "testxx",
				TargetQueueSize:           5,
				ActivationTargetQueueSize: 0,
				MinConnectTimeout:         5,
			},
			wantErr: false,
		},
		{
			name: "activationTargetQueueSize should not be 0",
			metadata: map[string]string{
				"endpoint":                  "test:7233",
				"namespace":                 "default",
				"taskQueue":                 "testxx",
				"activationTargetQueueSize": "12",
			},
			wantMeta: &temporalMetadata{
				Endpoint:                  "test:7233",
				Namespace:                 "default",
				TaskQueue:                 "testxx",
				TargetQueueSize:           5,
				ActivationTargetQueueSize: 12,
				MinConnectTimeout:         5,
			},
			wantErr: false,
		},
		{
			name: "apiKey should not be empty",
			metadata: map[string]string{
				"endpoint":  "test:7233",
				"namespace": "default",
				"taskQueue": "testxx",
			},
			wantMeta: &temporalMetadata{
				Endpoint:                  "test:7233",
				Namespace:                 "default",
				TaskQueue:                 "testxx",
				TargetQueueSize:           5,
				ActivationTargetQueueSize: 0,
				APIKey:                    "test01",
				MinConnectTimeout:         5,
			},
			authParams: map[string]string{
				"apiKey": "test01",
			},
			wantErr: false,
		},
		{
			name: "queue type should not be empty",
			metadata: map[string]string{
				"endpoint":   "test:7233",
				"namespace":  "default",
				"taskQueue":  "testxx",
				"queueTypes": "workflow,activity",
			},
			wantMeta: &temporalMetadata{
				Endpoint:                  "test:7233",
				Namespace:                 "default",
				TaskQueue:                 "testxx",
				TargetQueueSize:           5,
				ActivationTargetQueueSize: 0,
				QueueTypes:                []string{"workflow", "activity"},
				MinConnectTimeout:         5,
			},
			wantErr: false,
		},
		{
			name: "read config from env",
			resolvedEnv: map[string]string{
				"endpoint":  "test:7233",
				"namespace": "default",
				"taskQueue": "testxx",
			},
			metadata: map[string]string{
				"endpointFromEnv":  "endpoint",
				"namespaceFromEnv": "namespace",
				"taskQueueFromEnv": "taskQueue",
			},
			wantMeta: &temporalMetadata{
				Endpoint:                  "test:7233",
				Namespace:                 "default",
				TaskQueue:                 "testxx",
				TargetQueueSize:           5,
				ActivationTargetQueueSize: 0,
				APIKey:                    "test01",
				MinConnectTimeout:         5,
			},
			authParams: map[string]string{
				"apiKey": "test01",
			},
			wantErr: false,
		},
		{
			name: "apiKey provided",
			metadata: map[string]string{
				"endpoint":  "test:7233",
				"namespace": "default",
				"taskQueue": "testxx",
				"apiKey":    "test-api-key",
			},
			wantMeta: &temporalMetadata{
				Endpoint:                  "test:7233",
				Namespace:                 "default",
				TaskQueue:                 "testxx",
				TargetQueueSize:           5,
				ActivationTargetQueueSize: 0,
				APIKey:                    "test-api-key",
				MinConnectTimeout:         5,
			},
			authParams: map[string]string{
				"apiKey": "test-api-key",
			},
			wantErr: false,
		},
		{
			name: "with tlsServerName",
			metadata: map[string]string{
				"endpoint":      "test:7233",
				"namespace":     "default",
				"taskQueue":     "testxx",
				"tlsServerName": "my-namespace.tmpr.cloud",
			},
			wantMeta: &temporalMetadata{
				Endpoint:                  "test:7233",
				Namespace:                 "default",
				TaskQueue:                 "testxx",
				TargetQueueSize:           5,
				ActivationTargetQueueSize: 0,
				MinConnectTimeout:         5,
				TLSServerName:             "my-namespace.tmpr.cloud",
			},
			wantErr: false,
		},
		{
			name: "with tlsServerName and apiKey",
			metadata: map[string]string{
				"endpoint":      "test:7233",
				"namespace":     "default",
				"taskQueue":     "testxx",
				"tlsServerName": "my-namespace.tmpr.cloud",
			},
			authParams: map[string]string{
				"apiKey": "test01",
			},
			wantMeta: &temporalMetadata{
				Endpoint:                  "test:7233",
				Namespace:                 "default",
				TaskQueue:                 "testxx",
				TargetQueueSize:           5,
				ActivationTargetQueueSize: 0,
				APIKey:                    "test01",
				MinConnectTimeout:         5,
				TLSServerName:             "my-namespace.tmpr.cloud",
			},
			wantErr: false,
		},
		{
			name: "with tlsServerName and certificate",
			metadata: map[string]string{
				"endpoint":      "test:7233",
				"namespace":     "default",
				"taskQueue":     "testxx",
				"tlsServerName": "my-namespace.tmpr.cloud",
			},
			authParams: map[string]string{
				"cert":        "cert-data",
				"key":         "key-data",
				"keyPassword": "password",
				"ca":          "ca-data",
			},
			wantMeta: &temporalMetadata{
				Endpoint:                  "test:7233",
				Namespace:                 "default",
				TaskQueue:                 "testxx",
				TargetQueueSize:           5,
				ActivationTargetQueueSize: 0,
				Cert:                      "cert-data",
				Key:                       "key-data",
				KeyPassword:               "password",
				CA:                        "ca-data",
				MinConnectTimeout:         5,
				TLSServerName:             "my-namespace.tmpr.cloud",
			},
			wantErr: false,
		},
		{
			name: "apiKey and cert cannot be used together",
			metadata: map[string]string{
				"endpoint":  "test:7233",
				"namespace": "default",
				"taskQueue": "testxx",
			},
			authParams: map[string]string{
				"apiKey": "test-api-key",
				"cert":   "cert-data",
				"key":    "key-data",
			},
			wantMeta: nil,
			wantErr:  true,
		},
	}

	for _, testCase := range cases {
		c := testCase
		t.Run(c.name, func(t *testing.T) {
			config := &scalersconfig.ScalerConfig{
				TriggerMetadata: c.metadata,
				AuthParams:      c.authParams,
				ResolvedEnv:     c.resolvedEnv,
			}
			meta, err := parseTemporalMetadata(config)
			if c.wantErr {
				assert.Error(t, err)
				assert.Nil(t, meta)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.wantMeta, meta)
			}
		})
	}
}

func TestTemporalDefaultQueueTypes(t *testing.T) {
	metadata, err := parseTemporalMetadata(&scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{
			"endpoint": "localhost:7233", "taskQueue": "testcc",
		},
	})

	assert.NoError(t, err, "error should be nil")
	assert.Empty(t, metadata.QueueTypes, "queueTypes should be empty")

	assert.Len(t, getQueueTypes(metadata.QueueTypes), 3, "all queue types should be there")

	metadata.QueueTypes = []string{"workflow"}
	assert.Len(t, getQueueTypes(metadata.QueueTypes), 1, "only one type should be there")
}

func makeVersionTaskQueue(backlog int64) *workflowservice.DescribeWorkerDeploymentVersionResponse_VersionTaskQueue {
	return &workflowservice.DescribeWorkerDeploymentVersionResponse_VersionTaskQueue{
		Stats: &taskqueuepb.TaskQueueStats{
			ApproximateBacklogCount: backlog,
		},
	}
}

func TestGetDeploymentBacklogCount(t *testing.T) {
	cases := []struct {
		name          string
		taskQueues    []*workflowservice.DescribeWorkerDeploymentVersionResponse_VersionTaskQueue
		svcErr        error
		wantBacklog   int64
		wantErr       bool
	}{
		{
			name:        "single task queue",
			taskQueues:  []*workflowservice.DescribeWorkerDeploymentVersionResponse_VersionTaskQueue{makeVersionTaskQueue(42)},
			wantBacklog: 42,
		},
		{
			name: "multiple task queues summed",
			taskQueues: []*workflowservice.DescribeWorkerDeploymentVersionResponse_VersionTaskQueue{
				makeVersionTaskQueue(10),
				makeVersionTaskQueue(20),
				makeVersionTaskQueue(5),
			},
			wantBacklog: 35,
		},
		{
			name:        "no task queues returns zero",
			taskQueues:  nil,
			wantBacklog: 0,
		},
		{
			name: "nil stats entry skipped",
			taskQueues: []*workflowservice.DescribeWorkerDeploymentVersionResponse_VersionTaskQueue{
				{Stats: nil},
				makeVersionTaskQueue(7),
			},
			wantBacklog: 7,
		},
		{
			name:    "service error propagated",
			svcErr:  errors.New("rpc error"),
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSvc := workflowservicemock.NewMockWorkflowServiceClient(ctrl)
			if tc.svcErr != nil {
				mockSvc.EXPECT().
					DescribeWorkerDeploymentVersion(gomock.Any(), gomock.Any()).
					Return(nil, tc.svcErr)
			} else {
				mockSvc.EXPECT().
					DescribeWorkerDeploymentVersion(gomock.Any(), gomock.Any()).
					Return(&workflowservice.DescribeWorkerDeploymentVersionResponse{
						VersionTaskQueues: tc.taskQueues,
					}, nil)
			}

			got, err := getDeploymentBacklogCount(context.Background(), mockSvc, "default", "my-deployment", "v1.0.0")
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantBacklog, got)
			}
		})
	}
}

// mockBacklogClient implements temporalBacklogClient for testing the
// unversioned and build-id backlog paths.
type mockBacklogClient struct {
	resp sdk.TaskQueueDescription
	err  error
}

func (m *mockBacklogClient) DescribeTaskQueueEnhanced(_ context.Context, _ sdk.DescribeTaskQueueEnhancedOptions) (sdk.TaskQueueDescription, error) {
	return m.resp, m.err
}

func (m *mockBacklogClient) WorkflowService() workflowservice.WorkflowServiceClient { return nil }
func (m *mockBacklogClient) CheckHealth(_ context.Context, _ *sdk.CheckHealthRequest) (*sdk.CheckHealthResponse, error) {
	return nil, nil
}
func (m *mockBacklogClient) Close() {}

func makeTaskQueueDescription(backlogs ...int64) sdk.TaskQueueDescription {
	typesInfo := make(map[sdk.TaskQueueType]sdk.TaskQueueTypeInfo)
	for i, b := range backlogs {
		typesInfo[sdk.TaskQueueType(i)] = sdk.TaskQueueTypeInfo{
			Stats: &sdk.TaskQueueStats{ApproximateBacklogCount: b},
		}
	}
	return sdk.TaskQueueDescription{
		VersionsInfo: map[string]sdk.TaskQueueVersionInfo{
			"": {TypesInfo: typesInfo},
		},
	}
}

func TestGetUnversionedBacklogCount(t *testing.T) {
	cases := []struct {
		name        string
		resp        sdk.TaskQueueDescription
		err         error
		wantBacklog int64
		wantErr     bool
	}{
		{
			name:        "returns backlog count",
			resp:        makeTaskQueueDescription(10, 20),
			wantBacklog: 30,
		},
		{
			name:        "empty response returns zero",
			resp:        sdk.TaskQueueDescription{},
			wantBacklog: 0,
		},
		{
			name:    "service error propagated",
			err:     errors.New("rpc error"),
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := &mockBacklogClient{resp: tc.resp, err: tc.err}
			got, err := getUnversionedBacklogCount(context.Background(), client, "test-queue", nil)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantBacklog, got)
			}
		})
	}
}

func TestGetBuildIDBacklogCount(t *testing.T) {
	cases := []struct {
		name        string
		resp        sdk.TaskQueueDescription
		err         error
		buildID     string
		wantBacklog int64
		wantErr     bool
	}{
		{
			name:        "returns backlog count with build ID",
			resp:        makeTaskQueueDescription(15),
			buildID:     "v1.0",
			wantBacklog: 15,
		},
		{
			name:        "empty build ID still works",
			resp:        makeTaskQueueDescription(5),
			buildID:     "",
			wantBacklog: 5,
		},
		{
			name:    "service error propagated",
			err:     errors.New("rpc error"),
			buildID: "v1.0",
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := &mockBacklogClient{resp: tc.resp, err: tc.err}
			got, err := getBuildIDBacklogCount(context.Background(), client, "test-queue", nil, tc.buildID)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantBacklog, got)
			}
		})
	}
}

func TestGetCombinedBacklogCount(t *testing.T) {
	cases := []struct {
		name        string
		description sdk.TaskQueueDescription
		want        int64
	}{
		{
			name:        "empty versions info",
			description: sdk.TaskQueueDescription{},
			want:        0,
		},
		{
			name:        "single version single type",
			description: makeTaskQueueDescription(42),
			want:        42,
		},
		{
			name:        "single version multiple types summed",
			description: makeTaskQueueDescription(10, 20, 5),
			want:        35,
		},
		{
			name: "multiple versions summed",
			description: sdk.TaskQueueDescription{
				VersionsInfo: map[string]sdk.TaskQueueVersionInfo{
					"v1": {TypesInfo: map[sdk.TaskQueueType]sdk.TaskQueueTypeInfo{
						sdk.TaskQueueTypeWorkflow: {Stats: &sdk.TaskQueueStats{ApproximateBacklogCount: 10}},
					}},
					"v2": {TypesInfo: map[sdk.TaskQueueType]sdk.TaskQueueTypeInfo{
						sdk.TaskQueueTypeActivity: {Stats: &sdk.TaskQueueStats{ApproximateBacklogCount: 7}},
					}},
				},
			},
			want: 17,
		},
		{
			name: "nil stats entry skipped",
			description: sdk.TaskQueueDescription{
				VersionsInfo: map[string]sdk.TaskQueueVersionInfo{
					"": {TypesInfo: map[sdk.TaskQueueType]sdk.TaskQueueTypeInfo{
						sdk.TaskQueueTypeWorkflow: {Stats: nil},
						sdk.TaskQueueTypeActivity: {Stats: &sdk.TaskQueueStats{ApproximateBacklogCount: 3}},
					}},
				},
			},
			want: 3,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, getCombinedBacklogCount(tc.description))
		})
	}
}

func TestValidateTLS(t *testing.T) {
	cases := []struct {
		name    string
		meta    temporalMetadata
		wantErr bool
	}{
		{
			name: "no TLS fields is valid",
			meta: temporalMetadata{},
		},
		{
			name:    "cert without key",
			meta:    temporalMetadata{Cert: "cert-data"},
			wantErr: true,
		},
		{
			name:    "key without cert",
			meta:    temporalMetadata{Key: "key-data"},
			wantErr: true,
		},
		{
			name: "cert and key together is valid",
			meta: temporalMetadata{Cert: "cert-data", Key: "key-data"},
		},
		{
			name:    "apiKey and cert conflict",
			meta:    temporalMetadata{APIKey: "key", Cert: "cert-data", Key: "key-data"},
			wantErr: true,
		},
		{
			name: "apiKey alone is valid",
			meta: temporalMetadata{APIKey: "key"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.meta.validateTLS()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuildTLSConfig(t *testing.T) {
	cases := []struct {
		name           string
		meta           temporalMetadata
		wantNil        bool
		wantServerName string
	}{
		{
			name:    "no auth returns nil",
			meta:    temporalMetadata{},
			wantNil: true,
		},
		{
			name: "apiKey returns TLS config",
			meta: temporalMetadata{APIKey: "key"},
		},
		{
			name:           "apiKey with server name",
			meta:           temporalMetadata{APIKey: "key", TLSServerName: "my-server"},
			wantServerName: "my-server",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := buildTLSConfig(&tc.meta)
			assert.NoError(t, err)
			if tc.wantNil {
				assert.Nil(t, cfg)
			} else {
				assert.NotNil(t, cfg)
				if tc.wantServerName != "" {
					assert.Equal(t, tc.wantServerName, cfg.ServerName)
				}
			}
		})
	}
}
