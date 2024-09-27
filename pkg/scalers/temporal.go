package scalers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
	sdk "go.temporal.io/sdk/client"
	sdklog "go.temporal.io/sdk/log"
	"google.golang.org/grpc"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const (
	temporalDefaultTargetQueueLength     = 5
	temporalDefaultActivationQueueLength = 0
	temporalDefaultNamespace             = "default"
	temporalDefaultSelectAllActive       = true
	temporalDefaultSelectUnversioned     = true
)

var (
	temporalDefauleQueueTypes = []sdk.TaskQueueType{
		sdk.TaskQueueTypeActivity,
		sdk.TaskQueueTypeWorkflow,
		sdk.TaskQueueTypeNexus,
	}
)

type temporalScaler struct {
	metricType v2.MetricTargetType
	metadata   *temporalMetadata
	tcl        sdk.Client
	logger     logr.Logger
}

type temporalMetadata struct {
	activationLagThreshold int64
	endpoint               string
	namespace              string
	triggerIndex           int
	targetQueueSize        int64
	queueName              string
	queueTypes             []string
	buildIDs               []string
	allActive              bool
	unversioned            bool
	apiKey                 *string
}

func NewTemporalScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	logger := InitializeLogger(config, "temporal_scaler")

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get scaler metric type: %w", err)
	}

	meta, err := parseTemporalMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Temporal metadata: %w", err)
	}

	c, err := getTemporalClient(meta)
	if err != nil {
		return nil, fmt.Errorf("failed to create Temporal client connection: %w", err)
	}

	return &temporalScaler{
		metricType: metricType,
		metadata:   meta,
		tcl:        c,
		logger:     logger,
	}, nil
}

func (s *temporalScaler) Close(_ context.Context) error {
	if s.tcl != nil {
		s.tcl.Close()
	}
	return nil
}

func (s *temporalScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(fmt.Sprintf("temporal-%s-%s", s.metadata.namespace, s.metadata.queueName))
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, metricName),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetQueueSize),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric,
		Type:     externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

func (s *temporalScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	queueSize, err := s.getQueueSize(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get Temporal queue size: %w", err)
	}

	metric := GenerateMetricInMili(metricName, float64(queueSize))

	return []external_metrics.ExternalMetricValue{metric}, queueSize > s.metadata.activationLagThreshold, nil
}

func (s *temporalScaler) getQueueSize(ctx context.Context) (int64, error) {
	queueType := getQueueTypes(s.metadata.queueTypes)

	var selection *sdk.TaskQueueVersionSelection
	if s.metadata.allActive || s.metadata.unversioned || len(s.metadata.buildIDs) > 0 {
		selection = &sdk.TaskQueueVersionSelection{
			AllActive:   s.metadata.allActive,
			Unversioned: s.metadata.unversioned,
			BuildIDs:    s.metadata.buildIDs,
		}
	}

	resp, err := s.tcl.DescribeTaskQueueEnhanced(ctx, sdk.DescribeTaskQueueEnhancedOptions{
		TaskQueue:      s.metadata.queueName,
		ReportStats:    true,
		Versions:       selection,
		TaskQueueTypes: queueType,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get Temporal queue size: %w", err)
	}

	return getCombinedBacklogCount(resp), nil
}

func getQueueTypes(queueTypes []string) []sdk.TaskQueueType {
	var taskQueueTypes []sdk.TaskQueueType
	for _, t := range queueTypes {
		var taskQueueType sdk.TaskQueueType
		switch t {
		case "workflow":
			taskQueueType = sdk.TaskQueueTypeWorkflow
		case "activity":
			taskQueueType = sdk.TaskQueueTypeActivity
		case "nexus":
			taskQueueType = sdk.TaskQueueTypeNexus
		}
		taskQueueTypes = append(taskQueueTypes, taskQueueType)
	}

	if len(taskQueueTypes) == 0 {
		return temporalDefauleQueueTypes
	}
	return taskQueueTypes
}

func getCombinedBacklogCount(description sdk.TaskQueueDescription) int64 {
	var count int64
	for _, versionInfo := range description.VersionsInfo {
		for _, typeInfo := range versionInfo.TypesInfo {
			if typeInfo.Stats != nil {
				count += typeInfo.Stats.ApproximateBacklogCount
			}
		}
	}
	return count
}

func getTemporalClient(meta *temporalMetadata) (sdk.Client, error) {
	return sdk.Dial(sdk.Options{
		HostPort:  meta.endpoint,
		Namespace: meta.namespace,
		Logger:    sdklog.NewStructuredLogger(slog.Default()),
		ConnectionOptions: sdk.ConnectionOptions{
			DialOptions: []grpc.DialOption{
				grpc.WithConnectParams(grpc.ConnectParams{
					MinConnectTimeout: 5 * time.Second,
				}),
			},
		},
	})
}

func parseTemporalMetadata(config *scalersconfig.ScalerConfig, logger logr.Logger) (*temporalMetadata, error) {
	meta := &temporalMetadata{}
	meta.activationLagThreshold = temporalDefaultActivationQueueLength
	meta.targetQueueSize = temporalDefaultTargetQueueLength

	if config.TriggerMetadata["endpoint"] == "" {
		return nil, errors.New("no Temporal gRPC endpoint provided")
	}
	meta.endpoint = config.TriggerMetadata["endpoint"]

	if config.TriggerMetadata["namespace"] == "" {
		meta.namespace = temporalDefaultNamespace
	} else {
		meta.namespace = config.TriggerMetadata["namespace"]
	}

	if size, ok := config.TriggerMetadata["targetQueueSize"]; ok {
		queueSize, err := strconv.ParseInt(size, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid targetQueueSize - must be an integer")
		}
		meta.targetQueueSize = queueSize
	}

	if size, ok := config.TriggerMetadata["activationTargetQueueSize"]; ok {
		activationTargetQueueSize, err := strconv.ParseInt(size, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid activationTargetQueueSize - must be an integer")
		}
		meta.activationLagThreshold = activationTargetQueueSize
	}

	if queueName, ok := config.TriggerMetadata["queueName"]; ok {
		meta.queueName = queueName
	} else {
		return nil, errors.New("no queueName provided")
	}

	// if buildIds is provided, it will be used to filter the queue and make sure
	// selectAllActive and selectUnversioned are set to false to avoid considering
	if buildIds, ok := config.TriggerMetadata["buildIds"]; ok && buildIds != "" {
		meta.buildIDs = strings.Split(buildIds, ",")
	}

	if val, ok := config.TriggerMetadata["selectAllActive"]; ok && val != "" {
		allActive, err := strconv.ParseBool(val)
		if err != nil {
			meta.allActive = temporalDefaultSelectAllActive
			logger.Error(err, "Error parsing Temoral queue metadata selectAllActive, using default %n", temporalDefaultSelectAllActive)
		} else {
			meta.allActive = allActive
		}
	}

	if val, ok := config.TriggerMetadata["selectUnversioned"]; ok && val != "" {
		unversioned, err := strconv.ParseBool(val)
		if err != nil {
			meta.unversioned = temporalDefaultSelectUnversioned
			logger.Error(err, "Error parsing Temoral queue metadata selectUnversioned, using default %n", temporalDefaultSelectUnversioned)
		} else {
			meta.unversioned = unversioned
		}
	}

	// optional, valide queueTypes are workflow, activity, nexus
	if val, ok := config.TriggerMetadata["queueTypes"]; ok && val != "" {
		meta.queueTypes = strings.Split(val, ",")
	}

	meta.triggerIndex = config.TriggerIndex
	return meta, nil
}
