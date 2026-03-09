//go:build e2e
// +build e2e

package akeyless_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/akeylesslabs/akeyless-go/v5"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
	redis "github.com/kedacore/keda/v2/tests/scalers/redis/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

// makes sure helper is not removed
var _ = GetRandomNumber()

// Use the same constant as defined in akeyless_handler.go
const (
	testName         = "akeyless-test"
	publicGatewayURL = "https://api.akeyless.io"
	redisPassword    = "admin"
	redisList        = "queue"
)

var (
	testNamespace             = fmt.Sprintf("%s-ns", testName)
	redisNamespace            = fmt.Sprintf("%s-redis-ns", testName)
	deploymentName            = fmt.Sprintf("%s-deployment", testName)
	jobName                   = fmt.Sprintf("%s-job", testName)
	scaledObjectName          = fmt.Sprintf("%s-so", testName)
	triggerAuthenticationName = fmt.Sprintf("%s-ta", testName)
	secretName                = fmt.Sprintf("%s-secret", testName)
	redisHost                 = fmt.Sprintf("%s.%s.svc.cluster.local", testName, redisNamespace)
	minReplicaCount           = 0
	maxReplicaCount           = 2

	akeylessGatewayURL = os.Getenv("TF_AKEYLESS_GATEWAY_URL")
	akeylessAccessID   = os.Getenv("TF_AKEYLESS_ACCESS_ID")
	akeylessAccessKey  = os.Getenv("TF_AKEYLESS_ACCESS_KEY")
	akeylessSecretPath = fmt.Sprintf("keda-test/redisPassword-%d", GetRandomNumber())
)

type templateData struct {
	TestNamespace                 string
	DeploymentName                string
	JobName                       string
	ScaledObjectName              string
	TriggerAuthenticationName     string
	SecretName                    string
	MinReplicaCount               int
	MaxReplicaCount               int
	RedisPassword                 string
	RedisPasswordBase64           string
	RedisList                     string
	RedisHost                     string
	ItemsToWrite                  int
	AkeylessGatewayURL            string
	AkeylessAccessID              string
	AkeylessAccessKey             string
	AkeylessSecretPath            string
	AkeylessCredentialsSecretName string
}

const (
	deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
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
      - name: redis-worker
        image: ghcr.io/kedacore/tests-redis-lists:latest
        imagePullPolicy: IfNotPresent
        args: ["read"]
        env:
        - name: REDIS_HOST
          value: {{.RedisHost}}
        - name: REDIS_PORT
          value: "6379"
        - name: LIST_NAME
          value: {{.RedisList}}
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: {{.SecretName}}
              key: password
        - name: READ_PROCESS_TIME
          value: "100"
`

	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  password: {{.RedisPasswordBase64}}
`
	akeylessCredentialsSecretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.AkeylessCredentialsSecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  AKEYLESS_ACCESS_KEY: {{.AkeylessAccessKey}}
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  akeyless:
    gatewayUrl: {{.AkeylessGatewayURL}}
    accessId: {{.AkeylessAccessID}}
    accessKey:
      valueFrom:
        secretKeyRef:
          name: {{.AkeylessCredentialsSecretName}}
          key: AKEYLESS_ACCESS_KEY
    secrets:
    - parameter: password
      path: {{.AkeylessSecretPath}}
`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod:  10
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  triggers:
  - type: redis
    metadata:
      hostFromEnv: REDIS_HOST
      portFromEnv: REDIS_PORT
      listName: {{.RedisList}}
      listLength: "5"
      activationListLength: "10"
    authenticationRef:
      name: {{.TriggerAuthenticationName}}
`

	insertJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: {{.JobName}}
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 0
  template:
    spec:
      containers:
      - name: redis
        image: ghcr.io/kedacore/tests-redis-lists:latest
        imagePullPolicy: IfNotPresent
        env:
        - name: REDIS_ADDRESS
          value: {{.RedisHost}}
        - name: REDIS_PASSWORD
          value: {{.RedisPassword}}
        - name: LIST_NAME
          value: {{.RedisList}}
        - name: NO_LIST_ITEMS_TO_WRITE
          value: "{{.ItemsToWrite}}"
        args: ["write"]
      restartPolicy: Never
  backoffLimit: 4
`
)

func TestAkeyless(t *testing.T) {
	err := AkeylessTest(t)
	if err != nil {
		t.Errorf("AkeylessTest failed: %v", err)
	}
}

func AkeylessTest(t *testing.T) error {
	require.NotEmpty(t, akeylessAccessID, "TF_AKEYLESS_ACCESS_ID env variable is required for Akeyless test")
	require.NotEmpty(t, akeylessAccessKey, "TF_AKEYLESS_ACCESS_KEY env variable is required for Akeyless test")

	// Resetting here since we need a unique value before each time this test function is called
	akeylessSecretPath = fmt.Sprintf("keda-test/redisPassword-%d", GetRandomNumber())
	data.AkeylessSecretPath = akeylessSecretPath

	// Create the secret in Akeyless (storing Redis password)
	err := createAkeylessSecret(t)
	assert.NoErrorf(t, err, "cannot create Akeyless secret - %s", err)

	// Create kubernetes resources for Redis server
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
		// Delete the secret in Akeyless (for cleanup purposes only - not part of the handler functionality)
		deleteErr := deleteAkeylessSecret(t)
		if deleteErr != nil {
			t.Logf("Warning: failed to delete Akeyless secret during cleanup: %v", deleteErr)
		}
		redis.RemoveStandalone(t, testName, redisNamespace)
	})

	// Create Redis Standalone
	redis.InstallStandalone(t, kc, testName, redisNamespace, redisPassword)

	// Create kubernetes resources for testing

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc)

	return nil
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	data.ItemsToWrite = 5
	KubectlReplaceWithTemplate(t, data, "insertJobTemplate", insertJobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	data.ItemsToWrite = 400
	KubectlReplaceWithTemplate(t, data, "insertJobTemplate", insertJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

var data = templateData{
	TestNamespace:                 testNamespace,
	DeploymentName:                deploymentName,
	JobName:                       jobName,
	ScaledObjectName:              scaledObjectName,
	MinReplicaCount:               minReplicaCount,
	MaxReplicaCount:               maxReplicaCount,
	TriggerAuthenticationName:     triggerAuthenticationName,
	SecretName:                    secretName,
	RedisPassword:                 redisPassword,
	RedisPasswordBase64:           base64.StdEncoding.EncodeToString([]byte(redisPassword)),
	RedisList:                     redisList,
	RedisHost:                     redisHost,
	ItemsToWrite:                  0,
	AkeylessSecretPath:            akeylessSecretPath,
	AkeylessAccessKey:             base64.StdEncoding.EncodeToString([]byte(akeylessAccessKey)),
	AkeylessAccessID:              akeylessAccessID,
	AkeylessGatewayURL:            getAkeylessGatewayURL(),
	AkeylessCredentialsSecretName: fmt.Sprintf("%s-credentials-secret", testName),
}

func getTemplateData() (templateData, []Template) {
	return data, []Template{
		{Name: "secretTemplate", Config: secretTemplate},
		{Name: "akeylessCredentialsSecretTemplate", Config: akeylessCredentialsSecretTemplate},
		{Name: "deploymentTemplate", Config: deploymentTemplate},
		{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
		{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
	}
}

func getAkeylessGatewayURL() string {
	if akeylessGatewayURL == "" {
		return publicGatewayURL
	}
	return akeylessGatewayURL
}

func createAkeylessSecret(t *testing.T) error {
	ctx := context.Background()

	// Create Akeyless API client configuration
	gatewayURL := getAkeylessGatewayURL()
	config := akeyless.NewConfiguration()
	config.Servers = []akeyless.ServerConfiguration{
		{
			URL: gatewayURL,
		},
	}
	client := akeyless.NewAPIClient(config).V2Api

	// Authenticate with Akeyless
	authRequest := akeyless.NewAuth()
	authRequest.SetAccessId(akeylessAccessID)
	authRequest.SetAccessKey(akeylessAccessKey)

	authOut, httpResponse, err := client.Auth(ctx).Body(*authRequest).Execute()
	if httpResponse != nil {
		defer httpResponse.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("failed to authenticate with Akeyless API: %w", err)
	}
	if httpResponse != nil && httpResponse.StatusCode != 200 {
		return fmt.Errorf("failed to authenticate with Akeyless API (HTTP status code: %d): %s", httpResponse.StatusCode, httpResponse.Status)
	}

	token := authOut.GetToken()
	t.Log("Authenticated with Akeyless successfully")

	// Create the secret value (Redis password)
	secretValue := redisPassword

	// Create the secret in Akeyless
	createSecretRequest := akeyless.NewCreateSecret(akeylessSecretPath, secretValue)
	createSecretRequest.SetToken(token)

	_, httpResponse, err = client.CreateSecret(ctx).Body(*createSecretRequest).Execute()
	if httpResponse != nil {
		defer httpResponse.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("failed to create Akeyless secret: %w", err)
	}
	if httpResponse != nil && httpResponse.StatusCode != 200 {
		return fmt.Errorf("failed to create Akeyless secret (HTTP status code: %d): %s", httpResponse.StatusCode, httpResponse.Status)
	}

	t.Logf("Created secret in Akeyless at path: %s", akeylessSecretPath)

	return nil
}

// deleteAkeylessSecret is used for cleanup purposes only.
// The Akeyless handler does not implement secret deletion functionality.
func deleteAkeylessSecret(t *testing.T) error {
	ctx := context.Background()

	// Create Akeyless API client configuration
	gatewayURL := getAkeylessGatewayURL()
	config := akeyless.NewConfiguration()
	config.Servers = []akeyless.ServerConfiguration{
		{
			URL: gatewayURL,
		},
	}
	client := akeyless.NewAPIClient(config).V2Api

	// Authenticate with Akeyless
	authRequest := akeyless.NewAuth()
	authRequest.SetAccessId(akeylessAccessID)
	authRequest.SetAccessKey(akeylessAccessKey)

	authOut, httpResponse, err := client.Auth(ctx).Body(*authRequest).Execute()
	if httpResponse != nil {
		defer httpResponse.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("failed to authenticate with Akeyless API: %w", err)
	}
	if httpResponse != nil && httpResponse.StatusCode != 200 {
		return fmt.Errorf("failed to authenticate with Akeyless API (HTTP status code: %d): %s", httpResponse.StatusCode, httpResponse.Status)
	}

	token := authOut.GetToken()

	// Delete the secret from Akeyless
	deleteSecretRequest := akeyless.NewDeleteItem(akeylessSecretPath)
	deleteSecretRequest.SetToken(token)
	deleteImmediately := true
	deleteSecretRequest.SetDeleteImmediately(deleteImmediately)

	_, httpResponse, err = client.DeleteItem(ctx).Body(*deleteSecretRequest).Execute()
	if httpResponse != nil {
		defer httpResponse.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("failed to delete Akeyless secret: %w", err)
	}
	if httpResponse != nil && httpResponse.StatusCode != 200 {
		return fmt.Errorf("failed to delete Akeyless secret (HTTP status code: %d): %s", httpResponse.StatusCode, httpResponse.Status)
	}

	t.Logf("Deleted secret from Akeyless at path: %s", akeylessSecretPath)

	return nil
}
