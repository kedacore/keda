package scalers

import (
	"context"
	"fmt"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"log"
	"strconv"

	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/labels"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/newrelic/newrelic-client-go/newrelic"
	"github.com/newrelic/newrelic-client-go/pkg/nrdb"
)

const (
	nrAccount  = "nrAccount"
	nrQueryKey = "nrQueryKey"
	nrRegion   = "nrRegion"
	metricName = "metricName"
	nrLogLevel = "nrLogLevel"
	nrql       = "nrql"
	threshold  = "threshold"
)

type newrelicScaler struct {
	metadata *newrelicMetadata
	nrClient *newrelic.NewRelic
}

type newrelicMetadata struct {
	nrAccount  int
	nrRegion   string
	nrLogLevel string
	nrQueryKey string
	metricName string
	nrql       string
	threshold  int

	scalerIndex int
}

var newrelicLog = logf.Log.WithName("newrelic_scaler")

func NewNewRelicScaler(config *ScalerConfig) (Scaler, error) {
	meta, err := parseNewRelicMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing prometheus metadata: %s", err)
	}

	nrClient, err := newrelic.New(
		newrelic.ConfigPersonalAPIKey(meta.nrQueryKey),
		newrelic.ConfigRegion(meta.nrRegion),
		newrelic.ConfigLogLevel(meta.nrLogLevel))

	if err != nil {
		log.Fatal("error initializing client:", err)
	}

	return &newrelicScaler{
		metadata: meta,
		nrClient: nrClient}, nil
}

func parseNewRelicMetadata(config *ScalerConfig) (*newrelicMetadata, error) {
	meta := newrelicMetadata{}
	var err error
	if val, ok := config.TriggerMetadata[nrAccount]; ok && val != "" {
		t, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %s", nrAccount, err)
		}
		meta.nrAccount = t
	} else {
		return nil, fmt.Errorf("no %s given >%s<", nrAccount, val)
	}

	meta.nrQueryKey, err = GetFromAuthOrMeta(config, nrQueryKey)

	if err != nil {
		return nil, fmt.Errorf("no %s given", nrQueryKey)
	}

	if val, ok := config.TriggerMetadata[metricName]; ok && val != "" {
		meta.metricName = val
	} else {
		return nil, fmt.Errorf("no %s given", metricName)
	}

	if val, ok := config.TriggerMetadata[nrql]; ok && val != "" {
		meta.nrql = val
	} else {
		return nil, fmt.Errorf("no %s given", nrql)
	}

	meta.nrQueryKey, err = GetFromAuthOrMeta(config, nrRegion)

	if err != nil {
		meta.nrRegion = "US"
		newrelicLog.Info("Using default \"US\" region")
	}

	meta.nrQueryKey, err = GetFromAuthOrMeta(config, nrLogLevel)
	if err != nil {
		meta.nrLogLevel = "INFO"
	}

	if val, ok := config.TriggerMetadata[threshold]; ok && val != "" {
		t, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %s", threshold, err)
		}

		meta.threshold = t
	}

	meta.scalerIndex = config.ScalerIndex

	return &meta, nil
}

func (s *newrelicScaler) IsActive(ctx context.Context) (bool, error) {
	val, err := s.ExecuteNewRelicQuery(ctx)
	if err != nil {
		newrelicLog.Error(err, "error executing newrelic query")
		return false, err
	}
	return val > 0, nil
}

func (s *newrelicScaler) Close(context.Context) error {
	return nil
}

func (s *newrelicScaler) ExecuteNewRelicQuery(ctx context.Context) (float64, error) {

	nrdbQuery := nrdb.NRQL(s.metadata.nrql)
	resp, err := s.nrClient.Nrdb.Query(s.metadata.nrAccount, nrdbQuery)
	if err != nil {
		log.Fatal("error running NerdGraph query: ", err)
	}
	for _, r := range resp.Results {
		//fmt.Printf("%f", r)
		val, ok := r[s.metadata.metricName].(float64)
		if ok {
			newrelicLog.Info("Result of the query %s is %s", s.metadata.nrql, val)
			return val, nil
		} else {

			return 0, fmt.Errorf("metric not found on result")
		}
	}

	return 0, fmt.Errorf("query return no results")
}

func (s *newrelicScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	val, err := s.ExecuteNewRelicQuery(ctx)
	if err != nil {
		prometheusLog.Error(err, "error executing prometheus query")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(val), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (s *newrelicScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	targetMetricValue := resource.NewQuantity(int64(s.metadata.threshold), resource.DecimalSI)
	metricName := kedautil.NormalizeString(fmt.Sprintf("newrelic-%s", s.metadata.metricName))
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, metricName),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetMetricValue,
		},
	}
	metricSpec := v2beta2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2beta2.MetricSpec{metricSpec}
}
