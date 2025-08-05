package scalers

import (
	"context"
	"fmt"
	"strconv"
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

const (
	defaultCloudeyeMetricCollectionTime = 300
	defaultCloudeyeMetricFilter         = "average"
	defaultCloudeyeMetricPeriod         = "300"

	defaultHuaweiCloud = "myhuaweicloud.com"
)

type huaweiCloudeyeScaler struct {
	metricType v2.MetricTargetType
	metadata   *huaweiCloudeyeMetadata
	logger     logr.Logger
}

type huaweiCloudeyeMetadata struct {
	namespace      string
	metricsName    string
	dimensionName  string
	dimensionValue string

	targetMetricValue           float64
	activationTargetMetricValue float64

	metricCollectionTime int64
	metricFilter         string
	metricPeriod         string

	huaweiAuthorization huaweiAuthorizationMetadata

	triggerIndex int
}

type huaweiAuthorizationMetadata struct {
	IdentityEndpoint string

	// user project id
	ProjectID string

	DomainID string

	// region
	Region string

	// Cloud name
	Domain string

	// Cloud name
	Cloud string

	AccessKey string // Access Key
	SecretKey string // Secret key
}

// NewHuaweiCloudeyeScaler creates a new huaweiCloudeyeScaler
func NewHuaweiCloudeyeScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "huawei_cloudeye_scaler")

	meta, err := parseHuaweiCloudeyeMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing Cloudeye metadata: %w", err)
	}

	return &huaweiCloudeyeScaler{
		metricType: metricType,
		metadata:   meta,
		logger:     logger,
	}, nil
}

func parseHuaweiCloudeyeMetadata(config *scalersconfig.ScalerConfig, logger logr.Logger) (*huaweiCloudeyeMetadata, error) {
	meta := huaweiCloudeyeMetadata{}

	meta.metricCollectionTime = defaultCloudeyeMetricCollectionTime
	meta.metricFilter = defaultCloudeyeMetricFilter
	meta.metricPeriod = defaultCloudeyeMetricPeriod

	if val, ok := config.TriggerMetadata["namespace"]; ok && val != "" {
		meta.namespace = val
	} else {
		return nil, fmt.Errorf("namespace not given")
	}

	if val, ok := config.TriggerMetadata["metricName"]; ok && val != "" {
		meta.metricsName = val
	} else {
		return nil, fmt.Errorf("metric Name not given")
	}

	if val, ok := config.TriggerMetadata["dimensionName"]; ok && val != "" {
		meta.dimensionName = val
	} else {
		return nil, fmt.Errorf("dimension Name not given")
	}

	if val, ok := config.TriggerMetadata["dimensionValue"]; ok && val != "" {
		meta.dimensionValue = val
	} else {
		return nil, fmt.Errorf("dimension Value not given")
	}

	if val, ok := config.TriggerMetadata["targetMetricValue"]; ok && val != "" {
		targetMetricValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			logger.Error(err, "Error parsing targetMetricValue metadata")
		} else {
			meta.targetMetricValue = targetMetricValue
		}
	} else {
		return nil, fmt.Errorf("target Metric Value not given")
	}

	meta.activationTargetMetricValue = 0
	if val, ok := config.TriggerMetadata["activationTargetMetricValue"]; ok && val != "" {
		activationTargetMetricValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			logger.Error(err, "Error parsing activationTargetMetricValue metadata")
		}
		meta.activationTargetMetricValue = activationTargetMetricValue
	}

	if val, ok := config.TriggerMetadata["minMetricValue"]; ok && val != "" {
		minMetricValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			logger.Error(err, "Error parsing minMetricValue metadata")
		} else {
			logger.Error(err, "minMetricValue is deprecated and will be removed in next versions, please use activationTargetMetricValue instead")
			meta.activationTargetMetricValue = minMetricValue
		}
	} else {
		return nil, fmt.Errorf("min Metric Value not given")
	}

	if val, ok := config.TriggerMetadata["metricCollectionTime"]; ok && val != "" {
		metricCollectionTime, err := strconv.Atoi(val)
		if err != nil {
			logger.Error(err, "Error parsing metricCollectionTime metadata")
		} else {
			meta.metricCollectionTime = int64(metricCollectionTime)
		}
	}

	if val, ok := config.TriggerMetadata["metricFilter"]; ok && val != "" {
		meta.metricFilter = val
	}

	if val, ok := config.TriggerMetadata["metricPeriod"]; ok && val != "" {
		_, err := strconv.Atoi(val)
		if err != nil {
			logger.Error(err, "Error parsing metricPeriod metadata")
		} else {
			meta.metricPeriod = val
		}
	}

	auth, err := gethuaweiAuthorization(config.AuthParams)
	if err != nil {
		return nil, err
	}

	meta.huaweiAuthorization = auth
	meta.triggerIndex = config.TriggerIndex
	return &meta, nil
}

func gethuaweiAuthorization(authParams map[string]string) (huaweiAuthorizationMetadata, error) {
	meta := huaweiAuthorizationMetadata{}

	if authParams["IdentityEndpoint"] != "" {
		meta.IdentityEndpoint = authParams["IdentityEndpoint"]
	} else {
		return meta, fmt.Errorf("identityEndpoint doesn't exist in the authParams")
	}

	if authParams["ProjectID"] != "" {
		meta.ProjectID = authParams["ProjectID"]
	} else {
		return meta, fmt.Errorf("projectID doesn't exist in the authParams")
	}

	if authParams["DomainID"] != "" {
		meta.DomainID = authParams["DomainID"]
	} else {
		return meta, fmt.Errorf("domainID doesn't exist in the authParams")
	}

	if authParams["Region"] != "" {
		meta.Region = authParams["Region"]
	} else {
		return meta, fmt.Errorf("region doesn't exist in the authParams")
	}

	if authParams["Domain"] != "" {
		meta.Domain = authParams["Domain"]
	} else {
		return meta, fmt.Errorf("domain doesn't exist in the authParams")
	}

	if authParams["Cloud"] != "" {
		meta.Cloud = authParams["Cloud"]
	} else {
		meta.Cloud = defaultHuaweiCloud
	}

	if authParams["AccessKey"] != "" {
		meta.AccessKey = authParams["AccessKey"]
	} else {
		return meta, fmt.Errorf("accessKey doesn't exist in the authParams")
	}

	if authParams["SecretKey"] != "" {
		meta.SecretKey = authParams["SecretKey"]
	} else {
		return meta, fmt.Errorf("secretKey doesn't exist in the authParams")
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
	return []external_metrics.ExternalMetricValue{metric}, metricValue > s.metadata.activationTargetMetricValue, nil
}

func (s *huaweiCloudeyeScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("huawei-cloudeye-%s", s.metadata.metricsName))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.targetMetricValue),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s *huaweiCloudeyeScaler) Close(context.Context) error {
	return nil
}

func (s *huaweiCloudeyeScaler) GetCloudeyeMetrics() (float64, error) {
	options := aksk.AKSKOptions{
		IdentityEndpoint: s.metadata.huaweiAuthorization.IdentityEndpoint,
		ProjectID:        s.metadata.huaweiAuthorization.ProjectID,
		AccessKey:        s.metadata.huaweiAuthorization.AccessKey,
		SecretKey:        s.metadata.huaweiAuthorization.SecretKey,
		Region:           s.metadata.huaweiAuthorization.Region,
		Domain:           s.metadata.huaweiAuthorization.Domain,
		DomainID:         s.metadata.huaweiAuthorization.DomainID,
		Cloud:            s.metadata.huaweiAuthorization.Cloud,
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
				Namespace: s.metadata.namespace,
				Dimensions: []map[string]string{
					{
						"name":  s.metadata.dimensionName,
						"value": s.metadata.dimensionValue,
					},
				},
				MetricName: s.metadata.metricsName,
			},
		},
		From:   time.Now().Truncate(time.Minute).Add(time.Second*-1*time.Duration(s.metadata.metricCollectionTime)).UnixNano() / 1e6,
		To:     time.Now().Truncate(time.Minute).UnixNano() / 1e6,
		Period: s.metadata.metricPeriod,
		Filter: s.metadata.metricFilter,
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
		v, ok := metricdatas[0].Datapoints[0][s.metadata.metricFilter].(float64)
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
