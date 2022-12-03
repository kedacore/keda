//go:build e2e
// +build e2e

package gcp_pubsub_workload_identity_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

var now = time.Now().UnixNano()

const (
	testName = "gcp-pubsub-workload-identity-test"
)

var (
	gcpKey              = "{\"type\":\"service_account\",\"project_id\":\"cncf-keda-testing\",\"private_key_id\":\"af00fb2db9490b41cc06917b23b573c8fb5b1859\",\"private_key\":\"-----BEGIN PRIVATE KEY-----\\nMIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQCShKXOpGBV4NIn\\nklaaY+xodpai0fsASpr5lYk1BwnTmjj0huuVgsCSZuhUSbXkKNBdeX2QN9V0zuM7\\nLws0BmunMrMw1ho5cO7VQtPYF0fw4PcyGm4zIFhChB5MD4SlqI7L8Q3X5jZLnHv7\\nof9FEeuyTZo2NP7VK9vRrupdcsOspIhtCSOz+tvIbARvtgwjUZILETrEDpSjGVto\\nodwetdmeeJGbmdv4EesN8Bu6i5YlU/qB6I9eBOq5ezIOKp5GW57+GYEqZe+6zAtb\\nhecZfTiTI0EGoR6JDmOB3fTswTDmYIQqz/9nx3OteBDQ6m94fdyoJqvu1FPAaIM0\\nK1hMlJgpAgMBAAECggEAEe9JBQUHskdrbhLahTpFREm82Wga4X+gXfv1ED/A7w0O\\nGvd/c50OUbVlS7j8mfW5gLGoAizldO+ErthMtqTxDUW2W7xfeCfH4mS0XfuGj7iX\\n5aMI2XsEdrrpov2UyvrZpOLoQwTf4UxBvGzpXHTrvRcE8Qz2YxVj6lQvBbqQM2v+\\nf8qfGZRrQREIt4OacJSeORMdRoG95hfGOiBna+KOJcKBNOFgvPTSB14ANxyFAoIK\\n2MkAyM5CbP/hlvlVXhUkzRbf7o83q/YOg6wZjzadPTkrI8gGDJO2z1eaegnn5Rva\\nYB8yIxUbzgboTxh3ZUc+c9ie6RGPYHJMY7PCdzHd4QKBgQDJpBa4X3o5CSDKpBqt\\nWf+03X/Jmydf2gWJOirs/ek759XvCyAg3/PfPSciM2fW7LESShkr3dN7X1QUqHog\\n31PqLIWgcriNC6J6YcQ9SeVIUWK/cjfFKT0OjtNoxHrkUkr7CbDfLttwnX4RfKdn\\nkmCvSmKd1qcWgS+2ak9sFM/8SQKBgQC6BFnOB9qWmOko77mLVloVYmc5PRIhaIju\\nT5W1G3qO1ykUjH3oUYVPHY1C35p8l7Y01OaEuyDy+9wYb4I7GbtE+fz+kaxx1zE9\\n6wFxil8TyKZc5AvKDzZS+yQ+AkeJr1tUxV9/4odr+NBWfO22CpvLr39ZTvEz+fSx\\njUQQufD84QKBgQCQiuqqge64Yf26pUZmS6SMf1cyKuFfyYa8ZxEMT7tYcQkfUSdX\\nyZIkzc52qsjd/U+1X56JnnsR7jT0lgzt8YlSzVWAvZvjp5pyBhFJKeaNH6IcwICP\\n+c7F18ZeTLIXZ5JOQBUk947gPFV5rZTHHtvl6/mjUZL3A+Yy6iRCwuyQ2QKBgDmn\\nyZIDeyv8XyBSFUdrz2YbZvUlya3TMcXzoupMhxMo+1GkLg5I3jHkbflRRxfhCheb\\n+YsgWRkXGWP1g/7/fbzmYxUgX7u1QEz5vyvLAKcoJPBbuo+5YVQdBWG24Sd606sV\\ntgD0XJcJusFj3WX0Kc/bKHSs9DPxAHfb2kH48AnhAoGBAJ6lbFEjuFV8z0RTVURk\\npfcwrfZ9epgjkils8KyDDeMOyoK/Ay+g/B3xkPvfdeczQV6FZVmJICtKfJa5CV5Q\\nDRHXQdGpYTiBKv/pu/mlZHTTX0/qVhLudLXv+EN1D6CdaX3YdaK2vULCXPnWxcIB\\nkZee5bzQnA783QVZ9Ok6WBLF\\n-----END PRIVATE KEY-----\\n\",\"client_email\":\"e2e-test-user@cncf-keda-testing.iam.gserviceaccount.com\",\"client_id\":\"107657704857365440469\",\"auth_uri\":\"https://accounts.google.com/o/oauth2/auth\",\"token_uri\":\"https://oauth2.googleapis.com/token\",\"auth_provider_x509_cert_url\":\"https://www.googleapis.com/oauth2/v1/certs\",\"client_x509_cert_url\":\"https://www.googleapis.com/robot/v1/metadata/x509/e2e-test-user%40cncf-keda-testing.iam.gserviceaccount.com\"}"
	creds               = make(map[string]interface{})
	errGcpKey           = json.Unmarshal([]byte(gcpKey), &creds)
	testNamespace       = fmt.Sprintf("%s-ns", testName)
	secretName          = fmt.Sprintf("%s-secret", testName)
	deploymentName      = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName    = fmt.Sprintf("%s-so", testName)
	projectID           = creds["project_id"]
	topicID             = fmt.Sprintf("projects/%s/topics/keda-test-topic-%d", projectID, now)
	subscriptionName    = fmt.Sprintf("keda-test-topic-sub-%d", now)
	subscriptionID      = fmt.Sprintf("projects/%s/subscriptions/%s", projectID, subscriptionName)
	maxReplicaCount     = 4
	activationThreshold = 5
	gsPrefix            = fmt.Sprintf("kubectl exec --namespace %s deploy/gcp-sdk -- ", testNamespace)
)

type templateData struct {
	TestNamespace       string
	SecretName          string
	GcpCreds            string
	DeploymentName      string
	ScaledObjectName    string
	SubscriptionName    string
	SubscriptionID      string
	MaxReplicaCount     int
	ActivationThreshold int
}

const (
	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  creds.json: {{.GcpCreds}}
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
        - name: {{.DeploymentName}}-processor
          image: google/cloud-sdk:slim
          # Consume a message
          command: [ "/bin/bash", "-c", "--" ]
          args: [ "gcloud auth activate-service-account --key-file /etc/secret-volume/creds.json && \
          while true; do gcloud pubsub subscriptions pull {{.SubscriptionID}} --auto-ack; sleep 20; done" ]
          env:
            - name: GOOGLE_APPLICATION_CREDENTIALS_JSON
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key: creds.json
          volumeMounts:
            - name: secret-volume
              mountPath: /etc/secret-volume
      volumes:
        - name: secret-volume
          secret:
            secretName: {{.SecretName}}
`
	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-gcp-credentials
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: gcp
`

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  minReplicaCount: 0
  maxReplicaCount: {{.MaxReplicaCount}}
  cooldownPeriod: 10
  triggers:
    - type: gcp-pubsub
      authenticationRef:
        name: keda-trigger-auth-gcp-credentials
      metadata:
        subscriptionName: {{.SubscriptionName}}
        mode: SubscriptionSize
        value: "5"
        activationValue: "{{.ActivationThreshold}}"
`

	gcpSdkTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gcp-sdk
  namespace: {{.TestNamespace}}
  labels:
    app: gcp-sdk
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gcp-sdk
  template:
    metadata:
      labels:
        app: gcp-sdk
    spec:
      containers:
        - name: gcp-sdk-container
          image: google/cloud-sdk:slim
          # Just spin & wait forever
          command: [ "/bin/bash", "-c", "--" ]
          args: [ "ls /tmp && while true; do sleep 30; done;" ]
          volumeMounts:
            - name: secret-volume
              mountPath: /etc/secret-volume
      volumes:
        - name: secret-volume
          secret:
            secretName: {{.SecretName}}
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, gcpKey, "TF_GCP_SA_CREDENTIALS env variable is required for GCP storage test")
	assert.NoErrorf(t, errGcpKey, "Failed to load credentials from gcpKey - %s", errGcpKey)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after a minute")

	sdkReady := WaitForDeploymentReplicaReadyCount(t, kc, "gcp-sdk", testNamespace, 1, 60, 1)
	assert.True(t, sdkReady, "gcp-sdk deployment should be ready after a minute")

	if sdkReady {
		if createPubsub(t) == nil {
			// test scaling
			testActivation(t, kc)
			testScaleOut(t, kc)
			testScaleIn(t, kc)

			// cleanup
			t.Log("--- cleanup ---")
			cleanupPubsub(t)
		}
	}

	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func createPubsub(t *testing.T) error {
	// Authenticate to GCP
	t.Log("--- authenticate to GCP ---")
	cmd := fmt.Sprintf("%sgcloud auth activate-service-account %s --key-file /etc/secret-volume/creds.json --project=%s", gsPrefix, creds["client_email"], projectID)
	_, err := ExecuteCommand(cmd)
	assert.NoErrorf(t, err, "Failed to set GCP authentication on gcp-sdk - %s", err)
	if err != nil {
		return err
	}

	// Create topic
	t.Log("--- create topic ---")
	cmd = fmt.Sprintf("%sgcloud pubsub topics create %s", gsPrefix, topicID)
	_, err = ExecuteCommand(cmd)
	assert.NoErrorf(t, err, "Failed to create Pubsub topic %s: %s", topicID, err)
	if err != nil {
		return err
	}

	// Create subscription
	t.Log("--- create subscription ---")
	cmd = fmt.Sprintf("%sgcloud pubsub subscriptions create %s --topic=%s", gsPrefix, subscriptionID, topicID)
	_, err = ExecuteCommand(cmd)
	assert.NoErrorf(t, err, "Failed to create Pubsub subscription %s: %s", subscriptionID, err)

	return err
}

func cleanupPubsub(t *testing.T) {
	// Delete the topic and subscription
	t.Log("--- cleaning up the subscription and topic ---")
	_, _ = ExecuteCommand(fmt.Sprintf("%sgcloud pubsub subscriptions delete %s", gsPrefix, subscriptionID))
	_, _ = ExecuteCommand(fmt.Sprintf("%sgcloud pubsub topics delete %s", gsPrefix, topicID))
}

func getTemplateData() (templateData, []Template) {
	base64GcpCreds := base64.StdEncoding.EncodeToString([]byte(gcpKey))

	return templateData{
			TestNamespace:       testNamespace,
			SecretName:          secretName,
			GcpCreds:            base64GcpCreds,
			DeploymentName:      deploymentName,
			ScaledObjectName:    scaledObjectName,
			SubscriptionID:      subscriptionID,
			SubscriptionName:    subscriptionName,
			MaxReplicaCount:     maxReplicaCount,
			ActivationThreshold: activationThreshold,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
			{Name: "gcpSdkTemplate", Config: gcpSdkTemplate},
		}
}

func publishMessages(t *testing.T, count int) {
	t.Logf("--- publishing %d messages ---", count)
	publish := fmt.Sprintf(
		"%s/bin/bash -c -- 'for i in {1..%d}; do gcloud pubsub topics publish %s --message=AAAAAAAAAA;done'",
		gsPrefix,
		count,
		topicID)
	_, err := ExecuteCommand(publish)
	assert.NoErrorf(t, err, "cannot publish messages to pubsub topic - %s", err)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing not scaling if below threshold ---")

	publishMessages(t, activationThreshold)

	t.Log("--- waiting to see replicas are not scaled up ---")
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 240)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")

	publishMessages(t, 20-activationThreshold)

	t.Log("--- waiting for replicas to scale out ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 30, 10),
		fmt.Sprintf("replica count should be %d after five minutes", maxReplicaCount))
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")
	cmd := fmt.Sprintf("%sgcloud pubsub subscriptions seek %s --time=-P1S", gsPrefix, subscriptionID)
	_, err := ExecuteCommand(cmd)
	assert.NoErrorf(t, err, "cannot reset subscription position - %s", err)

	t.Log("--- waiting for replicas to scale in to zero ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 30, 10),
		"replica count should be 0 after five minute")
}
