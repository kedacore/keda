//go:build e2e
// +build e2e

package aws_cloudwatch_min_metric_value_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/kedacore/keda/v2/tests/helper"
	"github.com/kedacore/keda/v2/tests/scalers/aws/helpers/cloudwatch"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "aws-cloudwatch-min-metric-value-test"
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
        image: ghcr.io/nginx/nginx-unprivileged:1.26
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

// This test is to verify that the scaler returns the minMetricValue when the metric
// value is null and ignoreNullValues is set to true.
func TestCloudWatchScalerWithMinMetricValue(t *testing.T) {
	ctx := context.Background()

	// setup cloudwatch
	cloudwatchClient, err := cloudwatch.NewClient(ctx, awsRegion, awsAccessKeyID, awsSecretAccessKey, "")
	assert.Nil(t, err, "error creating cloudwatch client")

	// check that the metric in question is not already present, and is returning
	// an empty set of values.
	metricQuery := cloudwatch.CreateMetricDataInputForEmptyMetricValues(cloudwatchMetricNamespace, cloudwatchMetricName, cloudwatchMetricDimensionName, cloudwatchMetricDimensionValue)
	metricData, err := cloudwatch.GetMetricData(ctx, cloudwatchClient, metricQuery)
	require.Nil(t, err, "error getting metric data")
	require.Nil(t, cloudwatch.ExpectEmptyMetricDataResults(metricData), "metric data should be empty")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	defer DeleteKubernetesResources(t, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 1),
		"replica count should be %d after 1 minute", minMetricValueReplicaCount)

	// Allow a small amount of grace for stabilization, otherwise we will see the
	// minMetricValue of 1 scale up the deployment from 0 to 1, as the deployment
	// starts at a minReplicaCount of 0. The reason for this is to ensure that the
	// scaler is still functioning when the metric value is null, as opposed to
	// returning an error, and not scaling the workload at all.
	time.Sleep(5 * time.Second)

	// Then check that the deployment did not scale further, as the metric query
	// is returning null values, the scaler should evaluate the metric value as
	// the minMetricValue of 1, and not scale the deployment further beyond this
	// point.
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minMetricValueReplicaCount, 60)
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
