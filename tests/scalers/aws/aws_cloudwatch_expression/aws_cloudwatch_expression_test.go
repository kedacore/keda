//go:build e2e
// +build e2e

package aws_cloudwatch_expression_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "aws-cloudwatch-expression-test"
)

type templateData struct {
	TestNamespace              string
	DeploymentName             string
	ScaledObjectName           string
	SecretName                 string
	AwsAccessKeyID             string
	AwsSecretAccessKey         string
	AwsRegion                  string
	CloudwatchMetricExpression string
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
        expression: {{.CloudwatchMetricExpression}}
        targetMetricValue: "1"
        activationTargetMetricValue: "5"
        minMetricValue: "0"
        metricCollectionTime: "120"
        metricStatPeriod: "30"
`
)

var (
	testNamespace                  = fmt.Sprintf("%s-ns", testName)
	deploymentName                 = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName               = fmt.Sprintf("%s-so", testName)
	secretName                     = fmt.Sprintf("%s-secret", testName)
	cloudwatchMetricName           = fmt.Sprintf("cw-expr-%d", GetRandomNumber())
	awsAccessKeyID                 = os.Getenv("TF_AWS_ACCESS_KEY")
	awsSecretAccessKey             = os.Getenv("TF_AWS_SECRET_KEY")
	awsRegion                      = os.Getenv("TF_AWS_REGION")
	cloudwatchMetricNamespace      = "KEDA_EXPRESSION"
	cloudwatchMetricDimensionName  = "dimensionName"
	cloudwatchMetricDimensionValue = "dimensionValue"
	cloudwatchMetricExpression     = fmt.Sprintf("SELECT MAX(\"%s\") FROM \"%s\" WHERE %s = '%s'", cloudwatchMetricName, cloudwatchMetricNamespace, cloudwatchMetricDimensionName, cloudwatchMetricDimensionValue)
	maxReplicaCount                = 2
	minReplicaCount                = 0
)

func TestCloudWatchExpressionScaler(t *testing.T) {
	// setup cloudwatch
	cloudwatchClient := createCloudWatchClient()
	setCloudWatchCustomMetric(t, cloudwatchClient, 0)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 1),
		"replica count should be %d after 1 minute", minReplicaCount)

	// test scaling
	testActivation(t, kc, cloudwatchClient)
	testScaleOut(t, kc, cloudwatchClient)
	testScaleIn(t, kc, cloudwatchClient)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)

	setCloudWatchCustomMetric(t, cloudwatchClient, 0)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, cloudwatchClient *cloudwatch.Client) {
	t.Log("--- testing activation ---")
	setCloudWatchCustomMetric(t, cloudwatchClient, 3)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, cloudwatchClient *cloudwatch.Client) {
	t.Log("--- testing scale out ---")
	setCloudWatchCustomMetric(t, cloudwatchClient, 10)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, cloudwatchClient *cloudwatch.Client) {
	t.Log("--- testing scale in ---")

	setCloudWatchCustomMetric(t, cloudwatchClient, 0)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func setCloudWatchCustomMetric(t *testing.T, cloudwatchClient *cloudwatch.Client, value float64) {
	_, err := cloudwatchClient.PutMetricData(context.Background(), &cloudwatch.PutMetricDataInput{
		MetricData: []types.MetricDatum{
			{
				MetricName: aws.String(cloudwatchMetricName),
				Dimensions: []types.Dimension{
					{
						Name:  aws.String(cloudwatchMetricDimensionName),
						Value: aws.String(cloudwatchMetricDimensionValue),
					},
				},
				Unit:  types.StandardUnitNone,
				Value: aws.Float64(value),
			},
		},
		Namespace: aws.String(cloudwatchMetricNamespace),
	})
	assert.NoErrorf(t, err, "failed to set cloudwatch metric - %s", err)
}

func createCloudWatchClient() *cloudwatch.Client {
	configOptions := make([]func(*config.LoadOptions) error, 0)
	configOptions = append(configOptions, config.WithRegion(awsRegion))
	cfg, _ := config.LoadDefaultConfig(context.TODO(), configOptions...)
	cfg.Credentials = credentials.NewStaticCredentialsProvider(awsAccessKeyID, awsSecretAccessKey, "")
	return cloudwatch.NewFromConfig(cfg)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:              testNamespace,
			DeploymentName:             deploymentName,
			ScaledObjectName:           scaledObjectName,
			SecretName:                 secretName,
			AwsAccessKeyID:             base64.StdEncoding.EncodeToString([]byte(awsAccessKeyID)),
			AwsSecretAccessKey:         base64.StdEncoding.EncodeToString([]byte(awsSecretAccessKey)),
			AwsRegion:                  awsRegion,
			CloudwatchMetricExpression: cloudwatchMetricExpression,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
