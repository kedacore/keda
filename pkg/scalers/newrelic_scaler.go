package scalers

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/newrelic/newrelic-client-go/newrelic"
	"github.com/newrelic/newrelic-client-go/pkg/nrdb"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	account     = "account"
	queryKey    = "queryKey"
	region      = "region"
	nrql        = "nrql"
	threshold   = "threshold"
	noDataError = "noDataError"
	scalerName  = "new-relic"
)

type newrelicScaler struct {
	metricType v2beta2.MetricTargetType
	metadata   *newrelicMetadata
	nrClient   *newrelic.NewRelic
}

type newrelicMetadata struct {
	account     int
	region      string
	queryKey    string
	noDataError bool
	nrql        string
	threshold   int64
	scalerIndex int
}

var newrelicLog = logf.Log.WithName(fmt.Sprintf("%s_scaler", scalerName))

func NewNewRelicScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	meta, err := parseNewRelicMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s metadata: %s", scalerName, err)
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
		metricType: metricType,
		metadata:   meta,
		nrClient:   nrClient}, nil
}

func parseNewRelicMetadata(config *ScalerConfig) (*newrelicMetadata, error) {
	meta := newrelicMetadata{}
	var err error

	val, err := GetFromAuthOrMeta(config, account)
	if err != nil {
		return nil, fmt.Errorf("no %s given", account)
	}

	t, err := strconv.Atoi(val)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %s", account, err)
	}
	meta.account = t

	if val, ok := config.TriggerMetadata[nrql]; ok && val != "" {
		meta.nrql = val
	} else {
		return nil, fmt.Errorf("no %s given", nrql)
	}

	meta.queryKey, err = GetFromAuthOrMeta(config, queryKey)
	if err != nil {
		return nil, fmt.Errorf("no %s given", queryKey)
	}

	meta.region, err = GetFromAuthOrMeta(config, region)
	if err != nil {
		meta.region = "US"
		newrelicLog.Info("Using default 'US' region")
	}

	if val, ok := config.TriggerMetadata[threshold]; ok && val != "" {
		t, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s", threshold)
		}
		meta.threshold = t
	} else {
		return nil, fmt.Errorf("missing %s value", threshold)
	}

	// If Query Return an Empty Data , shall we treat it as an error or not
	// default is NO error is returned when query result is empty/no data
	if val, ok := config.TriggerMetadata[noDataError]; ok {
		noDataError, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("noDataError has invalid value")
		}
		meta.noDataError = noDataError
	} else {
		meta.noDataError = false
	}
	meta.scalerIndex = config.ScalerIndex
	return &meta, nil
}

func (s *newrelicScaler) IsActive(ctx context.Context) (bool, error) {
	val, err := s.executeNewRelicQuery(ctx)
	if err != nil {
		newrelicLog.Error(err, "error executing NRQL")
		return false, err
	}
	return val > 0, nil
}

func (s *newrelicScaler) Close(context.Context) error {
	return nil
}

func (s *newrelicScaler) executeNewRelicQuery(ctx context.Context) (float64, error) {
	nrdbQuery := nrdb.NRQL(s.metadata.nrql)
	resp, err := s.nrClient.Nrdb.QueryWithContext(ctx, s.metadata.account, nrdbQuery)
	if err != nil {
		return 0, fmt.Errorf("error running NRQL %s (%s)", s.metadata.nrql, err.Error())
	}
	// Only use the first result from the query, as the query should not be multi row
	for _, v := range resp.Results[0] {
		val, ok := v.(float64)
		if ok {
			return val, nil
		}
	}
	if s.metadata.noDataError {
		return 0, fmt.Errorf("query return no results %s", s.metadata.nrql)
	}
	return 0, nil
}

func (s *newrelicScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	val, err := s.executeNewRelicQuery(ctx)
	if err != nil {
		newrelicLog.Error(err, "error executing NRQL query")
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
	metricName := kedautil.NormalizeString(scalerName)

	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, metricName),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.threshold),
	}
	metricSpec := v2beta2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2beta2.MetricSpec{metricSpec}
}
