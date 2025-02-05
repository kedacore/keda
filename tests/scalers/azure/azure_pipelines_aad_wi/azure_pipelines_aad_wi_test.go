//go:build e2e
// +build e2e

package azure_pipelines_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/taskagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "azure-pipelines-aad-wi-test"
)

var (
	organizationURL     = os.Getenv("AZURE_DEVOPS_ORGANIZATION_URL")
	personalAccessToken = os.Getenv("AZURE_DEVOPS_PAT")
	project             = os.Getenv("AZURE_DEVOPS_PROJECT")
	buildID             = os.Getenv("AZURE_DEVOPS_AAD_WI_BUILD_DEFINITION_ID")
	poolName            = os.Getenv("AZURE_DEVOPS_AAD_WI_POOL_NAME")
	poolID              = "0"
	triggerAuthName     = fmt.Sprintf("%s-ta", testName)
	testNamespace       = fmt.Sprintf("%s-ns", testName)
	secretName          = fmt.Sprintf("%s-secret", testName)
	deploymentName      = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName    = fmt.Sprintf("%s-so", testName)
	minReplicaCount     = 0
	maxReplicaCount     = 1
)

type templateData struct {
	TestNamespace    string
	SecretName       string
	DeploymentName   string
	ScaledObjectName string
	MinReplicaCount  string
	MaxReplicaCount  string
	Pat              string
	URL              string
	PoolName         string
	PoolID           string
	TriggerAuthName  string
}

const (
	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  personalAccessToken: {{.Pat}}
`

	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: azdevops-agent
spec:
  replicas: 1
  selector:
    matchLabels:
      app: azdevops-agent
  template:
    metadata:
      labels:
        app: azdevops-agent
    spec:
      terminationGracePeriodSeconds: 90
      containers:
      - name: azdevops-agent
        lifecycle:
          preStop:
            exec:
              command: ["/bin/sleep","60"]
        image: ghcr.io/kedacore/tests-azure-pipelines-agent:b3a02cc
        env:
          - name: AZP_URL
            value: {{.URL}}
          - name: AZP_TOKEN
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: personalAccessToken
          - name: AZP_POOL
            value: {{.PoolName}}
`

	poolIdscaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  pollingInterval: 15
  cooldownPeriod: 5
  triggers:
  - type: azure-pipelines
    metadata:
      organizationURLFromEnv: "AZP_URL"
      activationTargetPipelinesQueueLength: "1"
      poolID: "{{.PoolID}}"
    authenticationRef:
      name: {{.TriggerAuthName}}
`
	poolNamescaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  pollingInterval: 15
  cooldownPeriod: 5
  triggers:
  - type: azure-pipelines
    metadata:
      organizationURLFromEnv: "AZP_URL"
      activationTargetPipelinesQueueLength: "1"
      poolName: "{{.PoolName}}"
    authenticationRef:
      name: {{.TriggerAuthName}}
`
	poolTriggerAuthRef = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: azure-workload
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, organizationURL, "AZURE_DEVOPS_ORGANIZATION_URL env variable is required for azure pipelines test")
	require.NotEmpty(t, personalAccessToken, "AZURE_DEVOPS_PAT env variable is required for azure pipelines test")
	require.NotEmpty(t, project, "AZURE_DEVOPS_PROJECT env variable is required for azure pipelines test")
	require.NotEmpty(t, buildID, "AZURE_DEVOPS_AAD_WI_BUILD_DEFINITION_ID env variable is required for azure pipelines test")
	require.NotEmpty(t, poolName, "AZURE_DEVOPS_AAD_WI_POOL_NAME env variable is required for azure pipelines test")
	connection := azuredevops.NewPatConnection(organizationURL, personalAccessToken)
	require.NotNil(t, connection, "unable to create azure devops connection")
	clearAllBuilds(t, connection)
	// Get pool ID
	poolID = fmt.Sprintf("%d", getAzDoPoolID(t, connection))

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	WaitForPodCountInNamespace(t, kc, testNamespace, minReplicaCount, 60, 2)

	// test scaling poolId
	testActivation(t, kc, connection)
	testScaleOut(t, kc, connection)
	testScaleIn(t, kc)

	// test scaling PoolName
	KubectlApplyWithTemplate(t, data, "poolNamescaledObjectTemplate", poolNamescaledObjectTemplate)
	testActivation(t, kc, connection)
	testScaleOut(t, kc, connection)
	testScaleIn(t, kc)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getAzDoPoolID(t *testing.T, connection *azuredevops.Connection) int {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	taskClient, err := taskagent.NewClient(ctx, connection)
	if err != nil {
		t.Errorf("unable to create task agent client")
	}
	args := taskagent.GetAgentPoolsArgs{
		PoolName: &poolName,
	}
	pools, err := taskClient.GetAgentPools(ctx, args)
	if err != nil {
		t.Errorf("unable to get the pools")
	}
	return *(*pools)[0].Id
}

func queueBuild(t *testing.T, connection *azuredevops.Connection) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	buildClient, err := build.NewClient(ctx, connection)
	if err != nil {
		t.Errorf("unable to create build client")
	}
	id, err := strconv.Atoi(buildID)
	if err != nil {
		t.Errorf("unable to parse buildID")
	}
	args := build.QueueBuildArgs{
		Project: &project,
		Build: &build.Build{
			Definition: &build.DefinitionReference{
				Id: &id,
			},
		},
	}
	_, err = buildClient.QueueBuild(ctx, args)
	if err != nil {
		t.Errorf("unable to get the pools")
	}
}

func clearAllBuilds(t *testing.T, connection *azuredevops.Connection) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	buildClient, err := build.NewClient(ctx, connection)
	if err != nil {
		t.Errorf("unable to create build client: %v", err)
	}
	var top = 20
	args := build.GetBuildsArgs{
		Project:      &project,
		StatusFilter: &build.BuildStatusValues.All,
		QueryOrder:   &build.BuildQueryOrderValues.QueueTimeDescending,
		Top:          &top,
	}
	azBuilds, err := buildClient.GetBuilds(ctx, args)
	if err != nil {
		t.Errorf("unable to get builds")
	}
	for _, azBuild := range azBuilds.Value {
		azBuild.Status = &build.BuildStatusValues.Cancelling
		args := build.UpdateBuildArgs{
			Build:   &azBuild,
			Project: &project,
			BuildId: azBuild.Id,
		}
		_, err = buildClient.UpdateBuild(ctx, args)
		if err != nil {
			t.Errorf("unable to cancel build")
		}
	}
}

func getTemplateData() (templateData, []Template) {
	base64Pat := base64.StdEncoding.EncodeToString([]byte(personalAccessToken))

	return templateData{
			TestNamespace:    testNamespace,
			SecretName:       secretName,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			MinReplicaCount:  fmt.Sprintf("%v", minReplicaCount),
			MaxReplicaCount:  fmt.Sprintf("%v", maxReplicaCount),
			Pat:              base64Pat,
			URL:              organizationURL,
			PoolName:         poolName,
			PoolID:           poolID,
			TriggerAuthName:  triggerAuthName,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "poolTriggerAuthRef", Config: poolTriggerAuthRef},
			{Name: "poolIdscaledObjectTemplate", Config: poolIdscaledObjectTemplate},
		}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, connection *azuredevops.Connection) {
	t.Log("--- testing activation ---")
	queueBuild(t, connection)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, connection *azuredevops.Connection) {
	t.Log("--- testing scale out ---")
	queueBuild(t, connection)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 1),
		"replica count should be 2 after 1 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")
	assert.True(t, WaitForPodCountInNamespace(t, kc, testNamespace, minReplicaCount, 60, 5),
		"pod count should be 0 after 1 minute")
}
