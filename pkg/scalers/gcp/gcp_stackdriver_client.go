package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
	// PubSub resource types
	ResourceTypePubSubSubscription = "subscription"
	ResourceTypePubSubTopic        = "topic"

	// Default alignment period for PubSub metrics (3 minutes)
	// PubSub metrics are collected every 60 seconds
	DefaultPubSubAlignmentPeriod = int64(180)

	// Aggregation function names
	AggregationDelta        = "delta"
	AggregationMean         = "mean"
	AggregationCount        = "count"
	AggregationSum          = "sum"
	AggregationStddev       = "stddev"
	AggregationPercentile99 = "percentile_99"
	AggregationPercentile95 = "percentile_95"
	AggregationPercentile50 = "percentile_50"
	AggregationPercentile05 = "percentile_05"
)

// StackDriverClient is a generic client to fetch metrics from Stackdriver. Can be used
// for a stackdriver scaler in the future
type StackDriverClient struct {
	metricsClient *monitoring.MetricClient
	credentials   GoogleApplicationCredentials
	projectID     string
}

// NewStackDriverClient creates a new stackdriver client with the credentials that are passed
func NewStackDriverClient(ctx context.Context, credentials string) (*StackDriverClient, error) {
	var gcpCredentials GoogleApplicationCredentials

	if err := json.Unmarshal([]byte(credentials), &gcpCredentials); err != nil {
		return nil, err
	}
	clientOption := option.WithAuthCredentialsJSON(option.ServiceAccount, []byte(credentials))

	metricsClient, err := monitoring.NewMetricClient(ctx, clientOption)
	if err != nil {
		return nil, err
	}

	return &StackDriverClient{
		metricsClient: metricsClient,
		credentials:   gcpCredentials,
	}, nil
}

// NewStackDriverClientPodIdentity creates a new stackdriver client with the credentials underlying
func NewStackDriverClientPodIdentity(ctx context.Context) (*StackDriverClient, error) {
	metricsClient, err := monitoring.NewMetricClient(ctx)
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
	case AggregationDelta:
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
	case AggregationMean:
		return monitoringpb.Aggregation_ALIGN_MEAN, nil
	case AggregationCount:
		return monitoringpb.Aggregation_ALIGN_COUNT, nil
	case AggregationSum:
		return monitoringpb.Aggregation_ALIGN_SUM, nil
	case AggregationStddev:
		return monitoringpb.Aggregation_ALIGN_STDDEV, nil
	case "count_true":
		return monitoringpb.Aggregation_ALIGN_COUNT_TRUE, nil
	case "count_false":
		return monitoringpb.Aggregation_ALIGN_COUNT_FALSE, nil
	case "fraction_true":
		return monitoringpb.Aggregation_ALIGN_FRACTION_TRUE, nil
	case AggregationPercentile99:
		return monitoringpb.Aggregation_ALIGN_PERCENTILE_99, nil
	case AggregationPercentile95:
		return monitoringpb.Aggregation_ALIGN_PERCENTILE_95, nil
	case AggregationPercentile50:
		return monitoringpb.Aggregation_ALIGN_PERCENTILE_50, nil
	case AggregationPercentile05:
		return monitoringpb.Aggregation_ALIGN_PERCENTILE_05, nil
	case "percent_change":
		return monitoringpb.Aggregation_ALIGN_PERCENT_CHANGE, nil
	default:
		return monitoringpb.Aggregation_ALIGN_NONE, fmt.Errorf("unknown aligner: %s", aligner)
	}
}

func reducerFromString(reducer string) (monitoringpb.Aggregation_Reducer, error) {
	switch strings.ToLower(reducer) {
	case "", "none":
		return monitoringpb.Aggregation_REDUCE_NONE, nil
	case AggregationMean:
		return monitoringpb.Aggregation_REDUCE_MEAN, nil
	case "min":
		return monitoringpb.Aggregation_REDUCE_MIN, nil
	case "max":
		return monitoringpb.Aggregation_REDUCE_MAX, nil
	case AggregationSum:
		return monitoringpb.Aggregation_REDUCE_SUM, nil
	case AggregationStddev:
		return monitoringpb.Aggregation_REDUCE_STDDEV, nil
	case AggregationCount:
		return monitoringpb.Aggregation_REDUCE_COUNT, nil
	case "count_true":
		return monitoringpb.Aggregation_REDUCE_COUNT_TRUE, nil
	case "count_false":
		return monitoringpb.Aggregation_REDUCE_COUNT_FALSE, nil
	case "fraction_true":
		return monitoringpb.Aggregation_REDUCE_FRACTION_TRUE, nil
	case AggregationPercentile99:
		return monitoringpb.Aggregation_REDUCE_PERCENTILE_99, nil
	case AggregationPercentile95:
		return monitoringpb.Aggregation_REDUCE_PERCENTILE_95, nil
	case AggregationPercentile50:
		return monitoringpb.Aggregation_REDUCE_PERCENTILE_50, nil
	case AggregationPercentile05:
		return monitoringpb.Aggregation_REDUCE_PERCENTILE_05, nil
	default:
		return monitoringpb.Aggregation_REDUCE_NONE, fmt.Errorf("unknown reducer: %s", reducer)
	}
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

// GetPubSubMetrics fetches PubSub metrics from stackdriver using the filter-based API.
// This replaces the deprecated MQL-based QueryMetrics method.
func (s StackDriverClient) GetPubSubMetrics(
	ctx context.Context,
	projectID string,
	resourceType string,
	resourceName string,
	metricType string,
	aggregation string,
	timeHorizonMinutes int64,
	valueIfNull *float64,
) (float64, error) {
	// Build the filter string for PubSub metrics
	filter := fmt.Sprintf(`metric.type="%s"`, metricType)

	// Add resource label filter based on resource type
	switch resourceType {
	case ResourceTypePubSubSubscription:
		filter += fmt.Sprintf(` AND resource.labels.subscription_id="%s"`, resourceName)
	case ResourceTypePubSubTopic:
		filter += fmt.Sprintf(` AND resource.labels.topic_id="%s"`, resourceName)
	}

	// Set default time horizon
	if timeHorizonMinutes <= 0 {
		timeHorizonMinutes = 2
		if aggregation != "" {
			timeHorizonMinutes = 5
		}
	}

	// Create aggregation if specified
	var agg *monitoringpb.Aggregation
	var err error
	if aggregation != "" {
		agg, err = NewPubSubAggregator(aggregation)
		if err != nil {
			return -1, err
		}
	}

	return s.GetMetrics(ctx, filter, projectID, agg, valueIfNull, timeHorizonMinutes)
}

// NewPubSubAggregator creates an aggregation configuration for PubSub metrics
// based on the aggregation function name (e.g., "count", "mean", "sum", "percentile99").
func NewPubSubAggregator(aggregation string) (*monitoringpb.Aggregation, error) {
	agg := strings.ToLower(aggregation)

	// Map aggregation function to aligner and reducer
	var aligner string
	var reducer string

	switch {
	case agg == AggregationCount:
		aligner = AggregationDelta
		reducer = AggregationSum
	case agg == AggregationSum:
		aligner = AggregationDelta
		reducer = AggregationSum
	case agg == AggregationMean:
		aligner = AggregationMean
		reducer = AggregationMean
	case agg == "median":
		aligner = AggregationPercentile50
		reducer = AggregationPercentile50
	case agg == AggregationStddev:
		aligner = AggregationStddev
		reducer = AggregationStddev
	case agg == "variance":
		// Variance is not directly supported, use stddev as an approximation
		aligner = AggregationStddev
		reducer = AggregationStddev
	case strings.HasPrefix(agg, "percentile"):
		// Handle percentileXX format (e.g., percentile99, percentile95)
		suffix := strings.TrimPrefix(agg, "percentile")
		switch suffix {
		case "99":
			aligner = AggregationPercentile99
			reducer = AggregationPercentile99
		case "95":
			aligner = AggregationPercentile95
			reducer = AggregationPercentile95
		case "50":
			aligner = AggregationPercentile50
			reducer = AggregationPercentile50
		case "05", "5":
			aligner = AggregationPercentile05
			reducer = AggregationPercentile05
		default:
			return nil, fmt.Errorf("unsupported percentile: %s (only 99, 95, 50, 05 are supported)", suffix)
		}
	default:
		return nil, fmt.Errorf("unsupported aggregation function: %s", aggregation)
	}

	return NewStackdriverAggregator(DefaultPubSubAlignmentPeriod, aligner, reducer)
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

func (s *StackDriverClient) Close() error {
	if s.metricsClient != nil {
		return s.metricsClient.Close()
	}
	return nil
}

// extractValueFromPoint attempts to extract a float64 by asserting the point's value type
func extractValueFromPoint(point *monitoringpb.Point) (float64, error) {
	typedValue := point.GetValue()
	switch v := typedValue.Value.(type) {
	case *monitoringpb.TypedValue_DoubleValue:
		return typedValue.GetDoubleValue(), nil
	case *monitoringpb.TypedValue_Int64Value:
		return float64(typedValue.GetInt64Value()), nil
	case *monitoringpb.TypedValue_DistributionValue:
		// For distribution metrics, return the count of values
		// This is useful for metrics like message counts where the underlying
		// metric is a distribution (e.g., message sizes) but we want the count
		dist := v.DistributionValue
		if dist != nil {
			return float64(dist.GetCount()), nil
		}
		return 0, nil
	case *monitoringpb.TypedValue_BoolValue:
		if typedValue.GetBoolValue() {
			return 1, nil
		}
		return 0, nil
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
