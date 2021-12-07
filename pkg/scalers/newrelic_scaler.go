package scalers

import (
	"context"
	"fmt"
	"log"
	"strconv"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/external_metrics"

	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/labels"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/newrelic/newrelic-client-go/newrelic"
	"github.com/newrelic/newrelic-client-go/pkg/nrdb"
)

const (
	account    = "account"
	queryKey   = "queryKey"
	region     = "region"
	metricName = "metricName"
	nrql       = "nrql"
	threshold  = "threshold"
	noDataErr  = "noDataErr"
)

type newrelicScaler struct {
	metadata *newrelicMetadata
	nrClient *newrelic.NewRelic
}

type newrelicMetadata struct {
	account     int
	region      string
	queryKey    string
	metricName  string
	noDataErr   bool
	nrql        string
	threshold   int
	scalerIndex int
}

var newrelicLog = logf.Log.WithName("new-relic_scaler")

func NewNewRelicScaler(config *ScalerConfig) (Scaler, error) {
	meta, err := parseNewRelicMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing new-relic metadata: %s", err)
	}

	nrClient, err := newrelic.New(
		newrelic.ConfigPersonalAPIKey(meta.queryKey),
		newrelic.ConfigRegion(meta.region))

	if err != nil {
		log.Fatal("error initializing client:", err)
	}

	logMsg := fmt.Sprintf("Initializing New Relic Scaler (account %d in region %s)", meta.account, meta.region)

	newrelicLog.Info(logMsg)

	return &newrelicScaler{
		metadata: meta,
		nrClient: nrClient}, nil
}

func parseNewRelicMetadata(config *ScalerConfig) (*newrelicMetadata, error) {
	meta := newrelicMetadata{}
	var err error
	if val, ok := config.TriggerMetadata[account]; ok && val != "" {
		t, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %s", account, err)
		}
		meta.account = t
	} else {
		return nil, fmt.Errorf("no %s given", account)
	}

	meta.queryKey, err = GetFromAuthOrMeta(config, queryKey)
	if err != nil {
		return nil, fmt.Errorf("no %s given", queryKey)
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

	meta.region, err = GetFromAuthOrMeta(config, region)
	if err != nil {
		meta.region = "US"
		newrelicLog.Info("Using default 'US' region")
	}

	if val, ok := config.TriggerMetadata[threshold]; ok && val != "" {
		t, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s", threshold)
		}
		meta.threshold = t
	} else {
		return nil, fmt.Errorf("missing %s value", threshold)
	}

	// If Query Return an Empty Data , shall we treat it as an error or not
	// default is NO error is returned when query result is empty/no data
	if val, ok := config.TriggerMetadata[noDataErr]; ok {
		noDataErr, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("noDataErr has invalid value")
		}
		meta.noDataErr = noDataErr
	} else {
		meta.noDataErr = false
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
	resp, err := s.nrClient.Nrdb.QueryWithContext(ctx, s.metadata.account, nrdbQuery)
	if err != nil {
		return 0, fmt.Errorf("error running NerdGraph query %s (%s)", s.metadata.nrql, err.Error())
	}
	for _, r := range resp.Results {
		val, ok := r[s.metadata.metricName].(float64)
		if ok {
			return val, nil
		}
	}
	if s.metadata.noDataErr {
		return 0, fmt.Errorf("query return no results %s", s.metadata.nrql)
	}
	return 0, nil
}

func (s *newrelicScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	val, err := s.ExecuteNewRelicQuery(ctx)
	if err != nil {
		newrelicLog.Error(err, "error executing New Relic query")
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
