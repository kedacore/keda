package scalers

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

	clientOption := option.WithCredentialsJSON([]byte(credentials))

	client, err := monitoring.NewMetricClient(ctx, clientOption)
	if err != nil {
		return nil, err
	}

	return &StackDriverClient{
		metricsClient: client,
		credentials:   gcpCredentials,
	}, nil
}

// NewStackDriverClientPodIdentity creates a new stackdriver client with the credentials underlying
func NewStackDriverClientPodIdentity(ctx context.Context) (*StackDriverClient, error) {
	client, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		return nil, err
	}
	c := metadata.NewClient(&http.Client{})

	// Running workload identity outside GKE, we can't use the metadata api and we need to use the env that it's provided from the hook
	project, found := os.LookupEnv("CLOUDSDK_CORE_PROJECT")
	if !found {
		project, err = c.ProjectID()
		if err != nil {
			return nil, err
		}
	}

	return &StackDriverClient{
		metricsClient: client,
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
	aggregation *monitoringpb.Aggregation) (float64, error) {
	// Set the start time to 1 minute ago
	startTime := time.Now().UTC().Add(time.Minute * -2)

	// Set the end time to now
	endTime := time.Now().UTC()

	// Create a request with the filter and the GCP project ID
	var req = &monitoringpb.ListTimeSeriesRequest{
		Filter: filter,
		Interval: &monitoringpb.TimeInterval{
			StartTime: &timestamppb.Timestamp{Seconds: startTime.Unix()},
			EndTime:   &timestamppb.Timestamp{Seconds: endTime.Unix()},
		},
		Aggregation: aggregation,
	}

	switch projectID {
	case "":
		if len(s.projectID) > 0 {
			req.Name = "projects/" + s.projectID
			req.Filter += ` AND resource.labels.project_id="` + s.projectID + `"`
		} else {
			req.Name = "projects/" + s.credentials.ProjectID
			req.Filter += ` AND resource.labels.project_id="` + s.credentials.ProjectID + `"`
		}
	default:
		req.Name = "projects/" + projectID
		req.Filter += ` AND resource.labels.project_id="` + projectID + `"`
	}

	// Get an iterator with the list of time series
	it := s.metricsClient.ListTimeSeries(ctx, req)

	var value float64 = -1

	// Get the value from the first metric returned
	resp, err := it.Next()

	if err == iterator.Done {
		return value, fmt.Errorf("could not find stackdriver metric with filter %s", filter)
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
