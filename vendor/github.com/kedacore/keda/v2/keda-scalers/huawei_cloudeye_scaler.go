package scalers

import (
	"context"
	"fmt"
	"time"

	"github.com/Huawei/gophercloud"
	"github.com/Huawei/gophercloud/auth/aksk"
	"github.com/Huawei/gophercloud/openstack"
	"github.com/Huawei/gophercloud/openstack/ces/v1/metricdata"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type huaweiCloudeyeScaler struct {
	metricType v2.MetricTargetType
	metadata   *huaweiCloudeyeMetadata
	logger     logr.Logger
}

type huaweiCloudeyeMetadata struct {
	triggerIndex int

	Namespace      string `keda:"name=namespace,      order=triggerMetadata"`
	MetricsName    string `keda:"name=metricName,     order=triggerMetadata"`
	DimensionName  string `keda:"name=dimensionName,  order=triggerMetadata"`
	DimensionValue string `keda:"name=dimensionValue, order=triggerMetadata"`

	TargetMetricValue           float64 `keda:"name=targetMetricValue,           order=triggerMetadata"`
	ActivationTargetMetricValue float64 `keda:"name=activationTargetMetricValue, order=triggerMetadata, default=0"`
	MinMetricValue              float64 `keda:"name=minMetricValue,              order=triggerMetadata, optional, deprecatedAnnounce=The 'minMetricValue' setting is DEPRECATED and will be removed in v2.20 - Use 'activationTargetMetricValue' instead"`

	MetricCollectionTime int64  `keda:"name=metricCollectionTime, order=triggerMetadata, default=300"`
	MetricFilter         string `keda:"name=metricFilter,         order=triggerMetadata, enum=average;max;min;sum, default=average"`
	MetricPeriod         string `keda:"name=metricPeriod,         order=triggerMetadata, default=300"`

	HuaweiAuthorization huaweiAuthorizationMetadata
}

type huaweiAuthorizationMetadata struct {
	IdentityEndpoint string `keda:"name=IdentityEndpoint, order=authParams"`
	ProjectID        string `keda:"name=ProjectID,        order=authParams"`
	DomainID         string `keda:"name=DomainID,         order=authParams"`
	Region           string `keda:"name=Region,           order=authParams"`
	Domain           string `keda:"name=Domain,           order=authParams"`
	Cloud            string `keda:"name=Cloud,            order=authParams, default=myhuaweicloud.com"`
	AccessKey        string `keda:"name=AccessKey,        order=authParams"`
	SecretKey        string `keda:"name=SecretKey,        order=authParams"`
}

func (h *huaweiCloudeyeMetadata) Validate() error {
	if h.MinMetricValue != 0 && h.ActivationTargetMetricValue == 0 {
		h.ActivationTargetMetricValue = h.MinMetricValue
	}
	return nil
}

// NewHuaweiCloudeyeScaler creates a new huaweiCloudeyeScaler
func NewHuaweiCloudeyeScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "huawei_cloudeye_scaler")

	meta, err := parseHuaweiCloudeyeMetadata(config) // Removed logger parameter
	if err != nil {
		return nil, fmt.Errorf("error parsing Cloudeye metadata: %w", err)
	}

	return &huaweiCloudeyeScaler{
		metricType: metricType,
		metadata:   meta,
		logger:     logger,
	}, nil
}

func parseHuaweiCloudeyeMetadata(config *scalersconfig.ScalerConfig) (*huaweiCloudeyeMetadata, error) {
	meta := &huaweiCloudeyeMetadata{}
	meta.triggerIndex = config.TriggerIndex

	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing huawei cloudeye metadata: %w", err)
	}

	return meta, nil
}

func (s *huaweiCloudeyeScaler) GetMetricsAndActivity(_ context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	metricValue, err := s.GetCloudeyeMetrics()

	if err != nil {
		s.logger.Error(err, "Error getting metric value")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, metricValue)
	return []external_metrics.ExternalMetricValue{metric}, metricValue > s.metadata.ActivationTargetMetricValue, nil
}

func (s *huaweiCloudeyeScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("huawei-cloudeye-%s", s.metadata.MetricsName))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.TargetMetricValue),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s *huaweiCloudeyeScaler) Close(context.Context) error {
	return nil
}

func (s *huaweiCloudeyeScaler) GetCloudeyeMetrics() (float64, error) {
	options := aksk.AKSKOptions{
		IdentityEndpoint: s.metadata.HuaweiAuthorization.IdentityEndpoint,
		ProjectID:        s.metadata.HuaweiAuthorization.ProjectID,
		AccessKey:        s.metadata.HuaweiAuthorization.AccessKey,
		SecretKey:        s.metadata.HuaweiAuthorization.SecretKey,
		Region:           s.metadata.HuaweiAuthorization.Region,
		Domain:           s.metadata.HuaweiAuthorization.Domain,
		DomainID:         s.metadata.HuaweiAuthorization.DomainID,
		Cloud:            s.metadata.HuaweiAuthorization.Cloud,
	}

	provider, err := openstack.AuthenticatedClient(options)
	if err != nil {
		s.logger.Error(err, "Failed to get the provider")
		return -1, err
	}
	sc, err := openstack.NewCESV1(provider, gophercloud.EndpointOpts{})

	if err != nil {
		s.logger.Error(err, "get ces client failed")
		if ue, ok := err.(*gophercloud.UnifiedError); ok {
			s.logger.Info("ErrCode:", ue.ErrorCode())
			s.logger.Info("Message:", ue.Message())
		}
		return -1, err
	}

	opts := metricdata.BatchQueryOpts{
		Metrics: []metricdata.Metric{
			{
				Namespace: s.metadata.Namespace,
				Dimensions: []map[string]string{
					{
						"name":  s.metadata.DimensionName,
						"value": s.metadata.DimensionValue,
					},
				},
				MetricName: s.metadata.MetricsName,
			},
		},
		From:   time.Now().Truncate(time.Minute).Add(time.Second*-1*time.Duration(s.metadata.MetricCollectionTime)).UnixNano() / 1e6,
		To:     time.Now().Truncate(time.Minute).UnixNano() / 1e6,
		Period: s.metadata.MetricPeriod,
		Filter: s.metadata.MetricFilter,
	}

	metricdatas, err := metricdata.BatchQuery(sc, opts).ExtractMetricDatas()
	if err != nil {
		s.logger.Error(err, "query metrics failed")
		if ue, ok := err.(*gophercloud.UnifiedError); ok {
			s.logger.Info("ErrCode:", ue.ErrorCode())
			s.logger.Info("Message:", ue.Message())
		}
		return -1, err
	}

	s.logger.V(1).Info("Received Metric Data", "data", metricdatas)

	var metricValue float64

	if len(metricdatas[0].Datapoints) > 0 {
		v, ok := metricdatas[0].Datapoints[0][s.metadata.MetricFilter].(float64)
		if ok {
			metricValue = v
		} else {
			return -1, fmt.Errorf("metric Data not float64")
		}
	} else {
		return -1, fmt.Errorf("metric Data not received")
	}

	return metricValue, nil
}
