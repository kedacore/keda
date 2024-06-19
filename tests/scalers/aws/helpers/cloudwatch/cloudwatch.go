//go:build e2e
// +build e2e

package cloudwatch

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// NewClient will provision a new Cloudwatch Client.
func NewClient(ctx context.Context, region, accessKeyID, secretAccessKey, sessionToken string) (*cloudwatch.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	cfg.Credentials = credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, sessionToken)
	return cloudwatch.NewFromConfig(cfg), nil
}

// CreateMetricDataInputForEmptyMetricValues will return a GetMetricDataInput with a single metric query
// that is expected to return no metric values.
func CreateMetricDataInputForEmptyMetricValues(metricNamespace, metricName, dimensionName, dimensionValue string) *cloudwatch.GetMetricDataInput {
	return &cloudwatch.GetMetricDataInput{
		MetricDataQueries: []types.MetricDataQuery{
			{
				Id: aws.String("m1"),
				MetricStat: &types.MetricStat{
					Metric: &types.Metric{
						Namespace:  &metricNamespace,
						MetricName: &metricName,
						Dimensions: []types.Dimension{
							{
								Name:  &dimensionName,
								Value: &dimensionValue,
							},
						},
					}, Period: aws.Int32(60), Stat: aws.String("Average"),
				},
			},
		},
		// evaluate +/- 10 minutes from now to be sure we cover the query window
		// as all e2e tests use a 300 second query window.
		EndTime:   aws.Time(time.Now().Add(time.Minute * 10)),
		StartTime: aws.Time(time.Now().Add(-time.Minute * 10)),
	}
}

// GetMetricData will return the metric data for the given input.
func GetMetricData(ctx context.Context, cloudwatchClient *cloudwatch.Client, metricDataInput *cloudwatch.GetMetricDataInput) (*cloudwatch.GetMetricDataOutput, error) {
	return cloudwatchClient.GetMetricData(ctx, metricDataInput)
}

// ExpectEmptyMetricDataResults will evaluate the custom metric for any metric values, if any
// values are an error will be returned.
func ExpectEmptyMetricDataResults(metricData *cloudwatch.GetMetricDataOutput) error {
	if len(metricData.MetricDataResults) != 1 || len(metricData.MetricDataResults[0].Values) > 0 {
		return fmt.Errorf("found unexpected metric data results for metricData: %+v", metricData.MetricDataResults)
	}

	return nil
}
