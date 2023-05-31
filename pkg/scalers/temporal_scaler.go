package scalers

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

const (
	temporalHost = "http://20.81.100.134:8088/"
)

type temporalScaler struct {
	metricType v2.MetricTargetType
	metadata   *temporalMetadata
	logger     logr.Logger
}

type temporalMetadata struct {
	address             string
	threshold           float64
	activationThreshold float64
	scalerIndex         int
}

func NewTemporalScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, fmt.Sprintf("%s_scaler", scalerName))

	meta, err := parseTemporalMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s metadata: %w", scalerName, err)
	}

	if err != nil {
		log.Fatal("error initializing client:", err)
	}

	logMsg := fmt.Sprintf("Initializing Temporal Scaler")

	logger.Info(logMsg)

	return &temporalScaler{
		metricType: metricType,
		metadata:   meta,
		logger:     logger}, nil
}

func parseTemporalMetadata(config *ScalerConfig, logger logr.Logger) (*temporalMetadata, error) {
	meta := temporalMetadata{}

	if val, ok := config.TriggerMetadata[threshold]; ok && val != "" {
		t, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s", threshold)
		}
		meta.threshold = t
	} else {
		return nil, fmt.Errorf("missing %s value", threshold)
	}

	meta.activationThreshold = 0
	if val, ok := config.TriggerMetadata["activationThreshold"]; ok {
		activationThreshold, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("queryValue parsing error %w", err)
		}
		meta.activationThreshold = activationThreshold
	}

	// If Query Return an Empty Data , shall we treat it as an error or not
	// default is NO error is returned when query result is empty/no data
	if val, ok := config.TriggerMetadata["address"]; ok {
		address := val
		meta.address = address
	} else {
		meta.address = "ERROR"
	}
	meta.scalerIndex = config.ScalerIndex
	return &meta, nil
}

func (s *temporalScaler) Close(context.Context) error {
	return nil
}

func (s *temporalScaler) executeTemporalQuery(ctx context.Context) (float64, error) {
	temporalClient, err := client.Dial(client.Options{
		HostPort: s.metadata.address,
	})
	if err != nil {
		log.Fatalln("Unable to create Temporal Client", err)
	}

	if _, err := temporalClient.CheckHealth(context.Background(), &client.CheckHealthRequest{}); err != nil {
		/* health is bad */
		log.Println("Health is bad")
	} else {
		/* health is good */
		log.Println("Health is good")
	}

	if openWorkFlows, err := temporalClient.ListOpenWorkflow(context.Background(), &workflowservice.ListOpenWorkflowExecutionsRequest{}); err != nil {
		log.Println(err.Error())
	} else {
		log.Println(openWorkFlows.Size())
		return float64(openWorkFlows.Size()), nil
	}

	defer temporalClient.Close()
	return 0, nil
}

func (s *temporalScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	val, err := s.executeTemporalQuery(ctx)
	if err != nil {
		s.logger.Error(err, "error executing Temporal query")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, val)

	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.activationThreshold, nil
}

func (s *temporalScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(scalerName)

	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, metricName),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.threshold),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}
