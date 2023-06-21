package scalers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	temporalScalerName = "temporal"
)

type temporalScaler struct {
	metricType     v2.MetricTargetType
	metadata       *temporalMetadata
	temporalClient client.Client
	logger         logr.Logger
}

type temporalMetadata struct {
	address             string
	threshold           float64
	activationThreshold float64
	scalerIndex         int
	temporalAuth        *authentication.AuthMeta
	unsafeSsl           bool
	serverName          string
	namespace           string
}

func NewTemporalScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, fmt.Sprintf("%s_scaler", temporalScalerName))

	meta, err := parseTemporalMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s metadata: %w", temporalScalerName, err)
	}

	clientOptions, err := parseClientOptions(meta)
	if err != nil {
		return nil, fmt.Errorf("unable to configure Temporal Client: %w. %+v", err, meta)
	}

	temporalClient, err := client.Dial(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("unable to create Temporal Client: %w. %s", err, meta.address)
	}

	logger.Info(fmt.Sprintf("Initializing Temporal Scaler connected to %s", meta.address))

	return &temporalScaler{
		metricType:     metricType,
		metadata:       meta,
		temporalClient: temporalClient,
		logger:         logger}, nil
}

func parseTemporalMetadata(config *ScalerConfig, logger logr.Logger) (*temporalMetadata, error) {
	meta := &temporalMetadata{}

	// Mandatory fields
	if val, ok := config.TriggerMetadata["address"]; ok {
		meta.address = val
	} else {
		return nil, fmt.Errorf("missing address value")
	}

	if val, ok := config.TriggerMetadata[threshold]; ok && val != "" {
		t, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing threshold %s %w", val, err)
		}
		meta.threshold = t
	} else {
		return nil, fmt.Errorf("missing threshold value")
	}

	// Optional fields
	meta.activationThreshold = 0
	if val, ok := config.TriggerMetadata["activationThreshold"]; ok {
		activationThreshold, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing activationThreshold %s %w", val, err)
		}
		meta.activationThreshold = activationThreshold
	}

	if val, ok := config.TriggerMetadata["namespace"]; ok {
		meta.namespace = val
	} else {
		meta.namespace = "default"
	}

	meta.unsafeSsl = false
	if val, ok := config.TriggerMetadata[unsafeSsl]; ok && val != "" {
		unsafeSslValue, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %w", unsafeSsl, err)
		}

		meta.unsafeSsl = unsafeSslValue
	}

	meta.scalerIndex = config.ScalerIndex
	meta.serverName = config.TriggerMetadata["serverName"]

	auth, err := authentication.GetAuthConfigs(config.TriggerMetadata, config.AuthParams)
	if err != nil {
		return nil, err
	}
	meta.temporalAuth = auth

	logger.Info("Parsed Temporal Scaler metadata: ", meta)

	return meta, nil
}

func (s *temporalScaler) Close(context.Context) error {
	return nil
}

func parseClientOptions(metadata *temporalMetadata) (client.Options, error) {
	// Read configuration from trigger metadata. EnableTLS will be set if the trigger metadata contains
	// the required TLS configuration.
	if !metadata.temporalAuth.EnableTLS {
		return client.Options{
			HostPort:  metadata.address,
			Namespace: metadata.namespace,
		}, nil
	}

	// This implementation assumes that a PEM format is base64 encoded in the trigger metadata
	cert, err := tls.X509KeyPair([]byte(metadata.temporalAuth.Cert), []byte(metadata.temporalAuth.Key))
	if err != nil {
		return client.Options{}, fmt.Errorf("failed loading client cert and key: %w", err)
	}

	serverCAPool := x509.NewCertPool()
	b := []byte(metadata.temporalAuth.CA)
	if !serverCAPool.AppendCertsFromPEM(b) {
		return client.Options{}, fmt.Errorf("server CA PEM file invalid")
	}

	return client.Options{
		HostPort:  metadata.address,
		Namespace: metadata.namespace,
		ConnectionOptions: client.ConnectionOptions{
			TLS: &tls.Config{
				Certificates:       []tls.Certificate{cert},
				RootCAs:            serverCAPool,
				ServerName:         metadata.serverName,
				MinVersion:         tls.VersionTLS13,
				InsecureSkipVerify: metadata.unsafeSsl,
			},
		},
	}, nil
}

func (s *temporalScaler) executeTemporalQuery(ctx context.Context) (float64, error) {
	size := float64(0)
	if _, err := s.temporalClient.CheckHealth(ctx, &client.CheckHealthRequest{}); err != nil {
		s.logger.Info("Health is bad")
	} else {
		s.logger.Info("Health is good")
	}

	openWorkFlows, workflowErr := s.temporalClient.ListOpenWorkflow(ctx, &workflowservice.ListOpenWorkflowExecutionsRequest{})
	if workflowErr != nil {
		s.logger.Error(workflowErr, "error getting list of open workflows")
	} else {
		size = float64(openWorkFlows.Size())
		logMsg := fmt.Sprintf("Open Workflows: %v", size)
		s.logger.Info(logMsg)
	}

	defer s.temporalClient.Close()
	return size, workflowErr
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
	metricName := kedautil.NormalizeString(temporalScalerName)

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
