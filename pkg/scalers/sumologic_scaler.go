package scalers

import (
	"context"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/kedacore/keda/v2/pkg/scalers/sumologic"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type sumologicScaler struct {
	client *sumologic.Client
}

type sumologicMetadata struct {
	AccessID  string `keda:"name=access_id,        order=authParams"`
	AccessKey string `keda:"name=access_key,        order=authParams"`
	Host      string `keda:"name=host,            order=triggerMetadata"`
	UnsafeSsl bool   `keda:"name=unsafeSsl,       order=triggerMetadata, optional"`
}

func NewSumologicScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	meta := &sumologicMetadata{}
	client, err := sumologic.NewClient(&sumologic.Config{
		Host:      meta.Host,
		AccessID:  meta.AccessID,
		AccessKey: meta.AccessKey,
		UnsafeSsl: meta.UnsafeSsl,
	}, config)
	if err != nil {
		return nil, err
	}
	return &sumologicScaler{
		client: client,
	}, nil
}

func (s *sumologicScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	return nil, false, nil
}

func (s *sumologicScaler) GetMetricSpecForScaling(ctx context.Context) []v2.MetricSpec {
	return nil
}

func (s *sumologicScaler) Close(ctx context.Context) error {
	return nil
}
