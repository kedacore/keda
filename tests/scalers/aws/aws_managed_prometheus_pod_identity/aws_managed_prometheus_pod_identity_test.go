//go:build e2e
// +build e2e

package aws_managed_prometheus_pod_identity_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/amp"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "aws-prometheus-pod-identity-test"
)

type templateData struct {
	TestNamespace      string
	DeploymentName     string
	ScaledObjectName   string
	SecretName         string
	AwsAccessKeyID     string
	AwsSecretAccessKey string
	AwsRegion          string
	WorkspaceID        string
}

const (
	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-aws-credentials
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: aws
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
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 15
  triggers:
    - type: prometheus
      authenticationRef:
        name: keda-trigger-auth-aws-credentials
      metadata:
        awsRegion: {{.AwsRegion}}
        serverAddress: "https://aps-workspaces.{{.AwsRegion}}.amazonaws.com/workspaces/{{.WorkspaceID}}"
        query: "vector(100)"
        threshold: "50.0"
        identityOwner: operator
`
)

var (
	testNamespace      = fmt.Sprintf("%s-ns", testName)
	deploymentName     = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName   = fmt.Sprintf("%s-so", testName)
	secretName         = fmt.Sprintf("%s-secret", testName)
	workspaceID        = fmt.Sprintf("workspace-%d", GetRandomNumber())
	awsAccessKeyID     = os.Getenv("TF_AWS_ACCESS_KEY")
	awsSecretAccessKey = os.Getenv("TF_AWS_SECRET_KEY")
	awsRegion          = os.Getenv("TF_AWS_REGION")
)

func TestScaler(t *testing.T) {
	require.NotEmpty(t, awsAccessKeyID, "AwsAccessKeyID env variable is required for AWS e2e test")
	require.NotEmpty(t, awsSecretAccessKey, "awsSecretAccessKey env variable is required for AWS e2e test")

	t.Log("--- setting up ---")

	ampClient := createAMPClient()
	workspaceOutput, err := ampClient.CreateWorkspace(context.Background(), nil)
	assert.NoError(t, err, "aws prometheus workspace creation has failed")
	workspaceID = *workspaceOutput.WorkspaceId

	kc := GetKubernetesClient(t)

	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	t.Log("--- assert ---")
	expectedReplicaCountNumber := 2 // as mentioned above, as the AMP returns 100 and the threshold set to 50, the expected replica count is 100 / 50 = 2
	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be %d after a minute", expectedReplicaCountNumber)

	t.Log("--- cleaning up ---")
	deleteWSInput := amp.DeleteWorkspaceInput{
		WorkspaceId: &workspaceID,
	}
	input := &deleteWSInput
	_, err = ampClient.DeleteWorkspace(context.Background(), input)
	assert.NoError(t, err, "aws prometheus workspace deletion has failed")
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:      testNamespace,
			SecretName:         secretName,
			AwsAccessKeyID:     base64.StdEncoding.EncodeToString([]byte(awsAccessKeyID)),
			AwsSecretAccessKey: base64.StdEncoding.EncodeToString([]byte(awsSecretAccessKey)),
			AwsRegion:          awsRegion,
			DeploymentName:     deploymentName,
			ScaledObjectName:   scaledObjectName,
			WorkspaceID:        workspaceID,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func createAMPClient() *amp.Client {
	configOptions := make([]func(*config.LoadOptions) error, 0)
	configOptions = append(configOptions, config.WithRegion(awsRegion))
	cfg, _ := config.LoadDefaultConfig(context.TODO(), configOptions...)
	cfg.Credentials = credentials.NewStaticCredentialsProvider(awsAccessKeyID, awsSecretAccessKey, "")
	return amp.NewFromConfig(cfg)
}
