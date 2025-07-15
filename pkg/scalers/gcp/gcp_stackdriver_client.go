package gcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/api/iterator"
	option "google.golang.org/api/option"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// Although the "common" value could be 1m
	// before v2.13 it was 2m, so we need to
	// keep that value to not break the behaviour
	// We need to revisit this in KEDA v3
	// https://github.com/kedacore/keda/issues/5429
	defaultTimeHorizon = 2 * time.Minute

	// Visualization of aggregation window:
	// aggregationTimeHorizon: [- - - - -]
	// alignmentPeriod:         [- - -][- - -] (may shift slightly left or right arbitrarily)

	// For aggregations, a shorter time horizon may not return any data
	aggregationTimeHorizon = 5 * time.Minute
	// To prevent the aggregation window from being too big,
	// which may result in the data being stale for too long
	alignmentPeriod = 3 * time.Minute

	// Not all aggregations are meaningful for distribution metrics,
	// so we only support a subset of them
	// https://cloud.google.com/monitoring/mql/reference#aggr-function-group
	aggregationMean       = "mean"
	aggregationMedian     = "median"
	aggregationVariance   = "variance"
	aggregationStddev     = "stddev"
	aggregationSum        = "sum"
	aggregationCount      = "count"
	aggregationPercentile = "percentile"
)

// StackDriverClient is a generic client to fetch metrics from Stackdriver. Can be used
// for a stackdriver scaler in the future
type StackDriverClient struct {
	metricsClient *monitoring.MetricClient
	queryClient   *monitoring.QueryClient
	credentials   GoogleApplicationCredentials
	projectID     string
}

// NewStackDriverClient creates a new stackdriver client with the credentials that are passed
func NewStackDriverClient(ctx context.Context, credentials string) (*StackDriverClient, error) {
	var gcpCredentials GoogleApplicationCredentials

	if err := json.Unmarshal([]byte(credentials), &gcpCredentials); err != nil {
		return nil, err
	}

	clientOption := option.WithCredentialsJSON([]byte(credentials))

	metricsClient, err := monitoring.NewMetricClient(ctx, clientOption)
	if err != nil {
		return nil, err
	}

	queryClient, err := monitoring.NewQueryClient(ctx, clientOption)
	if err != nil {
		return nil, err
	}

	return &StackDriverClient{
		metricsClient: metricsClient,
		queryClient:   queryClient,
		credentials:   gcpCredentials,
	}, nil
}

// NewStackDriverClientPodIdentity creates a new stackdriver client with the credentials underlying
func NewStackDriverClientPodIdentity(ctx context.Context) (*StackDriverClient, error) {
	metricsClient, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		return nil, err
	}
	queryClient, err := monitoring.NewQueryClient(ctx)
	if err != nil {
		return nil, err
	}
	c := metadata.NewClient(&http.Client{})

	// Running workload identity outside GKE, we can't use the metadata api and we need to use the env that it's provided from the hook
	project, found := os.LookupEnv("CLOUDSDK_CORE_PROJECT")
	if !found {
		project, err = c.ProjectIDWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return &StackDriverClient{
		metricsClient: metricsClient,
		queryClient:   queryClient,
		projectID:     project,
	}, nil
}

func NewStackdriverAggregator(period int64, aligner string, reducer string) (*monitoringpb.Aggregation, error) {
	sdAggregation := monitoringpb.Aggregation{
		AlignmentPeriod: &durationpb.Duration{
			Seconds: period,
			Nanos:   0,
		},
	}

	var err error
	perSeriesAligner, err := alignerFromString(aligner)
	if err != nil {
		return nil, err
	}
	sdAggregation.PerSeriesAligner = perSeriesAligner

	crossSeriesReducer, err := reducerFromString(reducer)
	if err != nil {
		return nil, err
	}
	sdAggregation.CrossSeriesReducer = crossSeriesReducer

	return &sdAggregation, nil
}

func alignerFromString(aligner string) (monitoringpb.Aggregation_Aligner, error) {
	switch strings.ToLower(aligner) {
	case "", "none":
		return monitoringpb.Aggregation_ALIGN_NONE, nil
	case "delta":
		return monitoringpb.Aggregation_ALIGN_DELTA, nil
	case "rate":
		return monitoringpb.Aggregation_ALIGN_RATE, nil
	case "interpolate":
		return monitoringpb.Aggregation_ALIGN_INTERPOLATE, nil
	case "next_older":
		return monitoringpb.Aggregation_ALIGN_NEXT_OLDER, nil
	case "min":
		return monitoringpb.Aggregation_ALIGN_MIN, nil
	case "max":
		return monitoringpb.Aggregation_ALIGN_MAX, nil
	case "mean":
		return monitoringpb.Aggregation_ALIGN_MEAN, nil
	case "count":
		return monitoringpb.Aggregation_ALIGN_COUNT, nil
	case "sum":
		return monitoringpb.Aggregation_ALIGN_SUM, nil
	case "stddev":
		return monitoringpb.Aggregation_ALIGN_STDDEV, nil
	case "count_true":
		return monitoringpb.Aggregation_ALIGN_COUNT_TRUE, nil
	case "count_false":
		return monitoringpb.Aggregation_ALIGN_COUNT_FALSE, nil
	case "fraction_true":
		return monitoringpb.Aggregation_ALIGN_FRACTION_TRUE, nil
	case "percentile_99":
		return monitoringpb.Aggregation_ALIGN_PERCENTILE_99, nil
	case "percentile_95":
		return monitoringpb.Aggregation_ALIGN_PERCENTILE_95, nil
	case "percentile_50":
		return monitoringpb.Aggregation_ALIGN_PERCENTILE_50, nil
	case "percentile_05":
		return monitoringpb.Aggregation_ALIGN_PERCENTILE_05, nil
	case "percent_change":
		return monitoringpb.Aggregation_ALIGN_PERCENT_CHANGE, nil
	default:
	}
	return monitoringpb.Aggregation_ALIGN_NONE, fmt.Errorf("unknown aligner: %s", aligner)
}

func reducerFromString(reducer string) (monitoringpb.Aggregation_Reducer, error) {
	switch strings.ToLower(reducer) {
	case "", "none":
		return monitoringpb.Aggregation_REDUCE_NONE, nil
	case "mean":
		return monitoringpb.Aggregation_REDUCE_MEAN, nil
	case "min":
		return monitoringpb.Aggregation_REDUCE_MIN, nil
	case "max":
		return monitoringpb.Aggregation_REDUCE_MAX, nil
	case "sum":
		return monitoringpb.Aggregation_REDUCE_SUM, nil
	case "stddev":
		return monitoringpb.Aggregation_REDUCE_STDDEV, nil
	case "count":
		return monitoringpb.Aggregation_REDUCE_COUNT, nil
	case "count_true":
		return monitoringpb.Aggregation_REDUCE_COUNT_TRUE, nil
	case "count_false":
		return monitoringpb.Aggregation_REDUCE_COUNT_FALSE, nil
	case "fraction_true":
		return monitoringpb.Aggregation_REDUCE_FRACTION_TRUE, nil
	case "percentile_99":
		return monitoringpb.Aggregation_REDUCE_PERCENTILE_99, nil
	case "percentile_95":
		return monitoringpb.Aggregation_REDUCE_PERCENTILE_95, nil
	case "percentile_50":
		return monitoringpb.Aggregation_REDUCE_PERCENTILE_50, nil
	case "percentile_05":
		return monitoringpb.Aggregation_REDUCE_PERCENTILE_05, nil
	default:
	}
	return monitoringpb.Aggregation_REDUCE_NONE, fmt.Errorf("unknown reducer: %s", reducer)
}

// GetMetrics fetches metrics from stackdriver for a specific filter for the last minute
func (s StackDriverClient) GetMetrics(
	ctx context.Context,
	filter string,
	projectID string,
	aggregation *monitoringpb.Aggregation,
	valueIfNull *float64,
	filterDuration int64) (float64, error) {
	// Set the start time (default 2 minute ago)
	if filterDuration <= 0 {
		filterDuration = 2
	}
	startTime := time.Now().UTC().Add(time.Minute * -time.Duration(filterDuration))

	// Set the end time to now
	endTime := time.Now().UTC()

	// Create a request with the filter and the GCP project ID
	var req = &monitoringpb.ListTimeSeriesRequest{
		Interval: &monitoringpb.TimeInterval{
			StartTime: &timestamppb.Timestamp{Seconds: startTime.Unix()},
			EndTime:   &timestamppb.Timestamp{Seconds: endTime.Unix()},
		},
		Aggregation: aggregation,
	}

	// Set project to perform request in and update filter with project_id
	pid := getActualProjectID(&s, projectID)
	req.Name = "projects/" + pid
	filter += ` AND resource.labels.project_id="` + pid + `"`

	// Set filter on request
	req.Filter = filter

	// Get an iterator with the list of time series
	it := s.metricsClient.ListTimeSeries(ctx, req)

	var value float64 = -1

	// Get the value from the first metric returned
	resp, err := it.Next()

	if err == iterator.Done {
		if valueIfNull == nil {
			return value, fmt.Errorf("could not find stackdriver metric with filter %s", filter)
		}
		return *valueIfNull, nil
	}

	if err != nil {
		return value, err
	}

	if len(resp.GetPoints()) > 0 {
		point := resp.GetPoints()[0]
		value, err = extractValueFromPoint(point)

		if err != nil {
			return -1, err
		}
	}

	return value, nil
}

// QueryMetrics fetches metrics from the Cloud Monitoring API
// for a specific Monitoring Query Language (MQL) query
//
// MQL provides a more expressive query language than
// the current filtering options of GetMetrics
func (s StackDriverClient) QueryMetrics(ctx context.Context, projectID, query string, valueIfNull *float64) (float64, error) {
	//nolint:staticcheck
	req := &monitoringpb.QueryTimeSeriesRequest{
		Query:    query,
		PageSize: 1,
	}
	req.Name = "projects/" + getActualProjectID(&s, projectID)
	//nolint:staticcheck
	it := s.queryClient.QueryTimeSeries(ctx, req)

	var value float64 = -1

	// Get the value from the first metric returned
	resp, err := it.Next()

	if err == iterator.Done {
		if valueIfNull == nil {
			return value, fmt.Errorf("could not find stackdriver metric with query %s", req.Query)
		}
		return *valueIfNull, nil
	}

	if err != nil {
		return value, err
	}

	if len(resp.GetPointData()) > 0 {
		point := resp.GetPointData()[0]
		value, err = extractValueFromPointData(point)

		if err != nil {
			return -1, err
		}
	}

	return value, nil
}

func getActualProjectID(s *StackDriverClient, projectID string) string {
	if len(projectID) > 0 {
		return projectID
	}
	if len(s.projectID) > 0 {
		return s.projectID
	}
	return s.credentials.ProjectID
}

// BuildMQLQuery builds a Monitoring Query Language (MQL) query for the last minute (five for aggregations),
// given a resource type, metric, resource name, and an optional aggregation
//
// example:
// fetch pubsub_topic
// | metric 'pubsub.googleapis.com/topic/message_sizes'
// | filter (resource.project_id == 'myproject' && resource.topic_id == 'mytopic')
// | within 5m
// | align delta(3m)
// | every 3m
// | group_by [], count(value)
func (s StackDriverClient) BuildMQLQuery(projectID, resourceType, metric, resourceName, aggregation string, timeHorizon time.Duration) (string, error) {
	th := timeHorizon
	if time.Duration(0) >= timeHorizon {
		th = defaultTimeHorizon
		if aggregation != "" {
			th = aggregationTimeHorizon
		}
	}

	pid := getActualProjectID(&s, projectID)
	q := fmt.Sprintf(
		"fetch pubsub_%s | metric '%s' | filter (resource.project_id == '%s' && resource.%s_id == '%s') | within %s",
		resourceType, metric, pid, resourceType, resourceName, th,
	)
	if aggregation != "" {
		agg, err := buildAggregation(aggregation)
		if err != nil {
			return "", err
		}
		// Aggregate for every `alignmentPeriod` minutes
		q += fmt.Sprintf(
			" | align delta(%s) | every %s | group_by [], %s",
			alignmentPeriod, alignmentPeriod, agg,
		)
	}

	return q, nil
}

func (s *StackDriverClient) Close() error {
	var queryClientError error
	var metricsClientError error
	if s.queryClient != nil {
		queryClientError = s.queryClient.Close()
	}
	if s.metricsClient != nil {
		metricsClientError = s.metricsClient.Close()
	}
	return errors.Join(queryClientError, metricsClientError)
}

// buildAggregation builds the aggregation part of a Monitoring Query Language (MQL) query
func buildAggregation(aggregation string) (string, error) {
	// Match against "percentileX"
	if strings.HasPrefix(aggregation, aggregationPercentile) {
		p, err := parsePercentile(aggregation)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s(value, %d)", aggregationPercentile, p), nil
	}

	switch aggregation {
	case aggregationMedian, aggregationMean, aggregationVariance, aggregationStddev, aggregationSum, aggregationCount:
		return fmt.Sprintf("%s(value)", aggregation), nil
	default:
		return "", fmt.Errorf("unsupported aggregation function: %s", aggregation)
	}
}

// parsePercentile returns the percentile value from a string of the form "percentileX"
func parsePercentile(str string) (int, error) {
	ps := strings.TrimPrefix(str, aggregationPercentile)
	p, err := strconv.Atoi(ps)
	if err != nil || p < 0 || p > 100 {
		return -1, fmt.Errorf("invalid percentile value: %s", ps)
	}
	return p, nil
}

// extractValueFromPoint attempts to extract a float64 by asserting the point's value type
func extractValueFromPoint(point *monitoringpb.Point) (float64, error) {
	typedValue := point.GetValue()
	switch typedValue.Value.(type) {
	case *monitoringpb.TypedValue_DoubleValue:
		return typedValue.GetDoubleValue(), nil
	case *monitoringpb.TypedValue_Int64Value:
		return float64(typedValue.GetInt64Value()), nil
	}
	return -1, fmt.Errorf("could not extract value from metric of type %T", typedValue)
}

// extractValueFromPointData is similar to extractValueFromPoint, but for type *TimeSeriesData_PointData
func extractValueFromPointData(point *monitoringpb.TimeSeriesData_PointData) (float64, error) {
	typedValue := point.GetValues()[0]
	switch typedValue.Value.(type) {
	case *monitoringpb.TypedValue_DoubleValue:
		return typedValue.GetDoubleValue(), nil
	case *monitoringpb.TypedValue_Int64Value:
		return float64(typedValue.GetInt64Value()), nil
	}
	return -1, fmt.Errorf("could not extract value from metric of type %T", typedValue)
}

// GoogleApplicationCredentials is a struct representing the format of a service account
// credentials file
type GoogleApplicationCredentials struct {
	Type                string `json:"type"`
	ProjectID           string `json:"project_id"`
	PrivateKeyID        string `json:"private_key_id"`
	PrivateKey          string `json:"private_key"`
	ClientEmail         string `json:"client_email"`
	ClientID            string `json:"client_id"`
	AuthURI             string `json:"auth_uri"`
	TokenURI            string `json:"token_uri"`
	AuthProviderCertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL   string `json:"client_x509_cert_url"`
}
