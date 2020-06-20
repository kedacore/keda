package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/api/iterator"
	option "google.golang.org/api/option"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

// StackDriverClient is a generic client to fetch metrics from Stackdriver. Can be used
// for a stackdriver scaler in the future
type StackDriverClient struct {
	metricsClient *monitoring.MetricClient
	credentials   GoogleApplicationCredentials
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

// GetMetrics fetches metrics from stackdriver for a specific filter for the last minute
func (s StackDriverClient) GetMetrics(ctx context.Context, filter string) (int64, error) {

	// Set the start time to 1 minute ago
	startTime := time.Now().UTC().Add(time.Minute * -2)

	// Set the end time to now
	endTime := time.Now().UTC()

	// Create a request with the filter and the GCP project ID
	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   "projects/" + s.credentials.ProjectID,
		Filter: filter,
		Interval: &monitoringpb.TimeInterval{
			StartTime: &timestamp.Timestamp{
				Seconds: startTime.Unix(),
			},
			EndTime: &timestamp.Timestamp{
				Seconds: endTime.Unix(),
			},
		},
	}

	// Get an iterator with the list of time series
	it := s.metricsClient.ListTimeSeries(ctx, req)

	var value int64 = -1

	// Get the value from the first metric returned
	resp, err := it.Next()

	if err == iterator.Done {
		return value, fmt.Errorf("Could not find stackdriver metric with filter %s", filter)
	}

	if err != nil {
		return value, err
	}

	if len(resp.GetPoints()) > 0 {
		point := resp.GetPoints()[0]
		value = point.GetValue().GetInt64Value()
	}

	return value, nil
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
