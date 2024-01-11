package scalers

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/newrelic/newrelic-client-go/newrelic"
	"github.com/newrelic/newrelic-client-go/pkg/nrdb"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	account           = "account"
	queryKeyParamater = "queryKey"
	regionParameter   = "region"
	nrql              = "nrql"
	threshold         = "threshold"
	noDataError       = "noDataError"
	scalerName        = "new-relic"
)

type newrelicScaler struct {
	metricType v2.MetricTargetType
	metadata   *newrelicMetadata
	nrClient   *newrelic.NewRelic
	logger     logr.Logger
}

type newrelicMetadata struct {
	account             int
	region              string
	queryKey            string
	noDataError         bool
	nrql                string
	threshold           float64
	activationThreshold float64
	triggerIndex        int
}

func NewNewRelicScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, fmt.Sprintf("%s_scaler", scalerName))

	meta, err := parseNewRelicMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s metadata: %w", scalerName, err)
	}

	nrClient, err := newrelic.New(
		newrelic.ConfigPersonalAPIKey(meta.queryKey),
		newrelic.ConfigRegion(meta.region))

	if err != nil {
		log.Fatal("error initializing client:", err)
	}

	logMsg := fmt.Sprintf("Initializing New Relic Scaler (account %d in region %s)", meta.account, meta.region)

	logger.Info(logMsg)

	return &newrelicScaler{
		metricType: metricType,
		metadata:   meta,
		nrClient:   nrClient,
		logger:     logger}, nil
}

func parseNewRelicMetadata(config *ScalerConfig, logger logr.Logger) (*newrelicMetadata, error) {
	meta := newrelicMetadata{}
	var err error

	val, err := GetFromAuthOrMeta(config, account)
	if err != nil {
		return nil, err
	}

	t, err := strconv.Atoi(val)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %w", account, err)
	}
	meta.account = t

	if val, ok := config.TriggerMetadata[nrql]; ok && val != "" {
		meta.nrql = val
	} else {
		return nil, fmt.Errorf("no %s given", nrql)
	}

	queryKey, err := GetFromAuthOrMeta(config, queryKeyParamater)
	if err != nil {
		return nil, err
	}
	meta.queryKey = queryKey

	region, err := GetFromAuthOrMeta(config, regionParameter)
	if err != nil {
		region = "US"
		logger.Info("Using default 'US' region")
	}
	meta.region = region

	if val, ok := config.TriggerMetadata[threshold]; ok && val != "" {
		t, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s", threshold)
		}
		meta.threshold = t
	} else {
		if config.AsMetricSource {
			meta.threshold = 0
		} else {
			return nil, fmt.Errorf("missing %s value", threshold)
		}
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
	if val, ok := config.TriggerMetadata[noDataError]; ok {
		noDataError, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("noDataError has invalid value")
		}
		meta.noDataError = noDataError
	} else {
		meta.noDataError = false
	}
	meta.triggerIndex = config.TriggerIndex
	return &meta, nil
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

func (s *newrelicScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	val, err := s.executeNewRelicQuery(ctx)
	if err != nil {
		s.logger.Error(err, "error executing NRQL query")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, val)

	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.activationThreshold, nil
}

func (s *newrelicScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(scalerName)

	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, metricName),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.threshold),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}
