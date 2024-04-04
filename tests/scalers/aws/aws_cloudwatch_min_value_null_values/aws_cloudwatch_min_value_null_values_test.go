//go:build e2e
// +build e2e

package aws_cloudwatch_min_value_null_values_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"

	v1alpha1Api "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "aws-cloudwatch-min-value-null-metrics-test"
)

type templateData struct {
	TestNamespace                  string
	DeploymentName                 string
	ScaledObjectName               string
	SecretName                     string
	AwsAccessKeyID                 string
	AwsSecretAccessKey             string
	AwsRegion                      string
	CloudWatchMetricName           string
	CloudWatchMetricNamespace      string
	CloudWatchMetricDimensionName  string
	CloudWatchMetricDimensionValue string
}

const (
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  AWS_ACCESS_KEY_ID: {{.AwsAccessKeyID}}
  AWS_SECRET_ACCESS_KEY: {{.AwsSecretAccessKey}}
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-aws-credentials
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: awsAccessKeyID     # Required.
    name: {{.SecretName}}         # Required.
    key: AWS_ACCESS_KEY_ID        # Required.
  - parameter: awsSecretAccessKey # Required.
    name: {{.SecretName}}         # Required.
    key: AWS_SECRET_ACCESS_KEY    # Required.
`

	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  template:
    metadata:
      labels:
        app: {{.DeploymentName}}
    spec:
      containers:
      - name: nginx
        image: nginxinc/nginx-unprivileged
        ports:
        - containerPort: 80
`

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  maxReplicaCount: 2
  minReplicaCount: 0
  cooldownPeriod: 1
  triggers:
    - type: aws-cloudwatch
      authenticationRef:
        name: keda-trigger-auth-aws-credentials
      metadata:
        awsRegion: {{.AwsRegion}}
        namespace: {{.CloudWatchMetricNamespace}}
        dimensionName: {{.CloudWatchMetricDimensionName}}
        dimensionValue: {{.CloudWatchMetricDimensionValue}}
        metricName: {{.CloudWatchMetricName}}
        targetMetricValue: "1"
        minMetricValue: "1"
        metricCollectionTime: "120"
        metricStatPeriod: "60"
`
)

var (
	testNamespace                  = fmt.Sprintf("%s-ns", testName)
	deploymentName                 = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName               = fmt.Sprintf("%s-so", testName)
	secretName                     = fmt.Sprintf("%s-secret", testName)
	cloudwatchMetricName           = fmt.Sprintf("cw-%d", GetRandomNumber())
	awsAccessKeyID                 = os.Getenv("TF_AWS_ACCESS_KEY")
	awsSecretAccessKey             = os.Getenv("TF_AWS_SECRET_KEY")
	awsRegion                      = os.Getenv("TF_AWS_REGION")
	cloudwatchMetricNamespace      = "DoesNotExist"
	cloudwatchMetricDimensionName  = "dimensionName"
	cloudwatchMetricDimensionValue = "dimensionValue"
	minReplicaCount                = 0
	minMetricValueReplicaCount     = 1
)

func TestCloudWatchScalerWithMinValueWhenNullValues(t *testing.T) {
	// setup cloudwatch
	cloudwatchClient := createCloudWatchClient()

	// check that the metric in question is not already present, and is returning
	// an empty set of values.
	checkCloudWatchCustomMetric(t, cloudwatchClient)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	kedaClient := GetKedaKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	defer DeleteKubernetesResources(t, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 1),
		"replica count should be %d after 1 minute", minReplicaCount)

	// check that the scaledobject is in paused state
	FailIfScaledObjectStatusNotReachedWithTimeout(t, kedaClient, testNamespace, scaledObjectName, 2*time.Minute, v1alpha1Api.ConditionPaused)

	// check that the deployment scaled up to the minMetricValueReplicaCount
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minMetricValueReplicaCount, 60, 1),
		"replica count should be %d after 1 minute", minMetricValueReplicaCount)
}

func createCloudWatchClient() *cloudwatch.Client {
	configOptions := make([]func(*config.LoadOptions) error, 0)
	configOptions = append(configOptions, config.WithRegion(awsRegion))
	cfg, _ := config.LoadDefaultConfig(context.TODO(), configOptions...)
	cfg.Credentials = credentials.NewStaticCredentialsProvider(awsAccessKeyID, awsSecretAccessKey, "")
	return cloudwatch.NewFromConfig(cfg)
}

// checkCloudWatchCustomMetric will evaluate the custom metric for any metric values, if any
// values are found the test will be failed.
func checkCloudWatchCustomMetric(t *testing.T, cloudwatchClient *cloudwatch.Client) {
	metricData, err := cloudwatchClient.GetMetricData(context.Background(), &cloudwatch.GetMetricDataInput{
		MetricDataQueries: []types.MetricDataQuery{
			{
				Id:         aws.String("m1"),
				ReturnData: aws.Bool(true),
				MetricStat: &types.MetricStat{
					Metric: &types.Metric{
						Namespace:  aws.String(cloudwatchMetricNamespace),
						MetricName: aws.String(cloudwatchMetricName),
						Dimensions: []types.Dimension{
							{
								Name:  aws.String(cloudwatchMetricDimensionName),
								Value: aws.String(cloudwatchMetricDimensionValue),
							},
						},
					},
					Period: aws.Int32(60),
					Stat:   aws.String("Average"),
				},
			},
		},
		// evaluate +/- 5 minutes from now to be sure we cover the query window
		// leading into the e2e test.
		EndTime:   aws.Time(time.Now().Add(time.Minute * 5)),
		StartTime: aws.Time(time.Now().Add(-time.Minute * 5)),
	})
	if err != nil {
		t.Fatalf("error checking cloudwatch metric: %s", err)
		return
	}

	// This is a e2e preflight check for returning an error when there are no
	// metric values. If there are any metric values, then the test should fail
	// here, as the scaler will never enter an error state if there are metric
	// values in the query window.
	if len(metricData.MetricDataResults) != 1 || len(metricData.MetricDataResults[0].Values) > 0 {
		t.Fatalf("found unexpected metric data results for namespace: %s: %+v", cloudwatchMetricNamespace, metricData.MetricDataResults)
		return
	}
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:                  testNamespace,
			DeploymentName:                 deploymentName,
			ScaledObjectName:               scaledObjectName,
			SecretName:                     secretName,
			AwsAccessKeyID:                 base64.StdEncoding.EncodeToString([]byte(awsAccessKeyID)),
			AwsSecretAccessKey:             base64.StdEncoding.EncodeToString([]byte(awsSecretAccessKey)),
			AwsRegion:                      awsRegion,
			CloudWatchMetricName:           cloudwatchMetricName,
			CloudWatchMetricNamespace:      cloudwatchMetricNamespace,
			CloudWatchMetricDimensionName:  cloudwatchMetricDimensionName,
			CloudWatchMetricDimensionValue: cloudwatchMetricDimensionValue,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
