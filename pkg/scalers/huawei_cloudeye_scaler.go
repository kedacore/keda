package scalers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Huawei/gophercloud"
	"github.com/Huawei/gophercloud/auth/aksk"
	"github.com/Huawei/gophercloud/openstack"
	"github.com/Huawei/gophercloud/openstack/ces/v1/metricdata"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	defaultCloudeyeMetricCollectionTime = 300
	defaultCloudeyeMetricFilter         = "average"
	defaultCloudeyeMetricPeriod         = "300"

	defaultHuaweiCloud = "myhuaweicloud.com"
)

type huaweiCloudeyeScaler struct {
	metadata *huaweiCloudeyeMetadata
}

type huaweiCloudeyeMetadata struct {
	namespace      string
	metricsName    string
	dimensionName  string
	dimensionValue string

	targetMetricValue float64
	minMetricValue    float64

	metricCollectionTime int64
	metricFilter         string
	metricPeriod         string

	huaweiAuthorization huaweiAuthorizationMetadata
}

type huaweiAuthorizationMetadata struct {
	IdentityEndpoint string

	// user project id
	ProjectID string

	DomainID string

	// region
	Region string

	//Cloud name
	Domain string

	//Cloud name
	Cloud string

	AccessKey string //Access Key
	SecretKey string //Secret key
}

var cloudeyeLog = logf.Log.WithName("huawei_cloudeye_scaler")

// NewHuaweiCloudeyeScaler creates a new huaweiCloudeyeScaler
func NewHuaweiCloudeyeScaler(metadata, authParams map[string]string) (Scaler, error) {
	meta, err := parseHuaweiCloudeyeMetadata(metadata, authParams)
	if err != nil {
		return nil, fmt.Errorf("Error parsing Cloudeye metadata: %s", err)
	}

	return &huaweiCloudeyeScaler{
		metadata: meta,
	}, nil
}

func parseHuaweiCloudeyeMetadata(metadata, authParams map[string]string) (*huaweiCloudeyeMetadata, error) {
	meta := huaweiCloudeyeMetadata{}

	meta.metricCollectionTime = defaultCloudeyeMetricCollectionTime
	meta.metricFilter = defaultCloudeyeMetricFilter
	meta.metricPeriod = defaultCloudeyeMetricPeriod

	if val, ok := metadata["namespace"]; ok && val != "" {
		meta.namespace = val
	} else {
		return nil, fmt.Errorf("Namespace not given")
	}

	if val, ok := metadata["metricName"]; ok && val != "" {
		meta.metricsName = val
	} else {
		return nil, fmt.Errorf("Metric Name not given")
	}

	if val, ok := metadata["dimensionName"]; ok && val != "" {
		meta.dimensionName = val
	} else {
		return nil, fmt.Errorf("Dimension Name not given")
	}

	if val, ok := metadata["dimensionValue"]; ok && val != "" {
		meta.dimensionValue = val
	} else {
		return nil, fmt.Errorf("Dimension Value not given")
	}

	if val, ok := metadata["targetMetricValue"]; ok && val != "" {
		targetMetricValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			cloudeyeLog.Error(err, "Error parsing targetMetricValue metadata")
		} else {
			meta.targetMetricValue = targetMetricValue
		}
	} else {
		return nil, fmt.Errorf("target Metric Value not given")
	}

	if val, ok := metadata["minMetricValue"]; ok && val != "" {
		minMetricValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			cloudeyeLog.Error(err, "Error parsing minMetricValue metadata")
		} else {
			meta.minMetricValue = minMetricValue
		}
	} else {
		return nil, fmt.Errorf("Min Metric Value not given")
	}

	if val, ok := metadata["metricCollectionTime"]; ok && val != "" {
		metricCollectionTime, err := strconv.Atoi(val)
		if err != nil {
			cloudeyeLog.Error(err, "Error parsing metricCollectionTime metadata")
		} else {
			meta.metricCollectionTime = int64(metricCollectionTime)
		}
	}

	if val, ok := metadata["metricFilter"]; ok && val != "" {
		meta.metricFilter = val
	}

	if val, ok := metadata["metricPeriod"]; ok && val != "" {
		_, err := strconv.Atoi(val)
		if err != nil {
			cloudeyeLog.Error(err, "Error parsing metricPeriod metadata")
		} else {
			meta.metricPeriod = val
		}
	}

	auth, err := gethuaweiAuthorization(authParams)
	if err != nil {
		return nil, err
	}

	meta.huaweiAuthorization = auth

	return &meta, nil
}

func gethuaweiAuthorization(authParams map[string]string) (huaweiAuthorizationMetadata, error) {
	meta := huaweiAuthorizationMetadata{}

	if authParams["IdentityEndpoint"] != "" {
		meta.IdentityEndpoint = authParams["IdentityEndpoint"]
	} else {
		return meta, fmt.Errorf("IdentityEndpoint doesn't exist in the authParams")
	}

	if authParams["ProjectID"] != "" {
		meta.ProjectID = authParams["ProjectID"]
	} else {
		return meta, fmt.Errorf("ProjectID doesn't exist in the authParams")
	}

	if authParams["DomainID"] != "" {
		meta.DomainID = authParams["DomainID"]
	} else {
		return meta, fmt.Errorf("DomainID doesn't exist in the authParams")
	}

	if authParams["Region"] != "" {
		meta.Region = authParams["Region"]
	} else {
		return meta, fmt.Errorf("Region doesn't exist in the authParams")
	}

	if authParams["Domain"] != "" {
		meta.Domain = authParams["Domain"]
	} else {
		return meta, fmt.Errorf("Domain doesn't exist in the authParams")
	}

	if authParams["Cloud"] != "" {
		meta.Cloud = authParams["Cloud"]
	} else {
		meta.Cloud = defaultHuaweiCloud
	}

	if authParams["AccessKey"] != "" {
		meta.AccessKey = authParams["AccessKey"]
	} else {
		return meta, fmt.Errorf("AccessKey doesn't exist in the authParams")
	}

	if authParams["SecretKey"] != "" {
		meta.SecretKey = authParams["SecretKey"]
	} else {
		return meta, fmt.Errorf("SecretKey doesn't exist in the authParams")
	}

	return meta, nil
}

func (h *huaweiCloudeyeScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	metricValue, err := h.GetCloudeyeMetrics()

	if err != nil {
		cloudeyeLog.Error(err, "Error getting metric value")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(metricValue), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (h *huaweiCloudeyeScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetMetricValue := resource.NewQuantity(int64(h.metadata.targetMetricValue), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: fmt.Sprintf("%s-%s-%s-%s-%s", "huawei-cloudeye", strings.ReplaceAll(h.metadata.namespace, ".", "-"),
				h.metadata.metricsName,
				h.metadata.dimensionName, h.metadata.dimensionValue),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetMetricValue,
		},
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

func (h *huaweiCloudeyeScaler) IsActive(ctx context.Context) (bool, error) {
	val, err := h.GetCloudeyeMetrics()

	if err != nil {
		return false, err
	}

	return val > h.metadata.minMetricValue, nil
}

func (h *huaweiCloudeyeScaler) Close() error {
	return nil
}

func (h *huaweiCloudeyeScaler) GetCloudeyeMetrics() (float64, error) {
	options := aksk.AKSKOptions{
		IdentityEndpoint: h.metadata.huaweiAuthorization.IdentityEndpoint,
		ProjectID:        h.metadata.huaweiAuthorization.ProjectID,
		AccessKey:        h.metadata.huaweiAuthorization.AccessKey,
		SecretKey:        h.metadata.huaweiAuthorization.SecretKey,
		Region:           h.metadata.huaweiAuthorization.Region,
		Domain:           h.metadata.huaweiAuthorization.Domain,
		DomainID:         h.metadata.huaweiAuthorization.DomainID,
		Cloud:            h.metadata.huaweiAuthorization.Cloud,
	}

	provider, err := openstack.AuthenticatedClient(options)
	if err != nil {
		cloudeyeLog.Error(err, "Failed to get the provider")
		return -1, err
	}
	sc, err := openstack.NewCESV1(provider, gophercloud.EndpointOpts{})

	if err != nil {
		cloudeyeLog.Error(err, "get ces client failed")
		if ue, ok := err.(*gophercloud.UnifiedError); ok {
			cloudeyeLog.Info("ErrCode:", ue.ErrorCode())
			cloudeyeLog.Info("Message:", ue.Message())
		}
		return -1, err
	}

	opts := metricdata.BatchQueryOpts{
		Metrics: []metricdata.Metric{
			{
				Namespace: h.metadata.namespace,
				Dimensions: []map[string]string{
					{
						"name":  h.metadata.dimensionName,
						"value": h.metadata.dimensionValue,
					},
				},
				MetricName: h.metadata.metricsName,
			},
		},
		From:   time.Now().Truncate(time.Minute).Add(time.Second*-1*time.Duration(h.metadata.metricCollectionTime)).UnixNano() / 1e6,
		To:     time.Now().Truncate(time.Minute).UnixNano() / 1e6,
		Period: h.metadata.metricPeriod,
		Filter: h.metadata.metricFilter,
	}

	metricdatas, err := metricdata.BatchQuery(sc, opts).ExtractMetricDatas()
	if err != nil {
		cloudeyeLog.Error(err, "query metrics failed")
		if ue, ok := err.(*gophercloud.UnifiedError); ok {
			cloudeyeLog.Info("ErrCode:", ue.ErrorCode())
			cloudeyeLog.Info("Message:", ue.Message())
		}
		return -1, err
	}

	cloudeyeLog.V(1).Info("Received Metric Data", "data", metricdatas)

	var metricValue float64

	if metricdatas[0].Datapoints != nil && len(metricdatas[0].Datapoints) > 0 {
		v, ok := metricdatas[0].Datapoints[0][h.metadata.metricFilter].(float64)
		if ok {
			metricValue = v
		} else {
			return -1, fmt.Errorf("Metric Data not float64")
		}
	} else {
		return -1, fmt.Errorf("Metric Data not received")
	}

	return metricValue, nil

}
