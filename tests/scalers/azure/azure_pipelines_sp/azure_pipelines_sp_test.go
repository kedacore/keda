//go:build e2e
// +build e2e

package azure_pipelines_sp_test

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
	testName = "azure-pipelines-sp-test"
)

var (
	organizationURL     = os.Getenv("AZURE_DEVOPS_ORGANIZATION_URL")
	personalAccessToken = os.Getenv("AZURE_DEVOPS_PAT")
	project             = os.Getenv("AZURE_DEVOPS_PROJECT")
	buildID             = os.Getenv("AZURE_DEVOPS_BUILD_DEFINITION_ID")
	poolName            = os.Getenv("AZURE_DEVOPS_POOL_NAME")
	azureADClientID     = os.Getenv("TF_AZURE_SP_APP_ID")
	azureADSecret       = os.Getenv("AZURE_SP_KEY")
	azureADTenantID     = os.Getenv("TF_AZURE_SP_TENANT")
	poolID              = "0"
	testNamespace       = fmt.Sprintf("%s-ns", testName)
	agentSecretName     = fmt.Sprintf("%s-agent-secret", testName)
	scalerSecretName    = fmt.Sprintf("%s-scaler-secret", testName)
	triggerAuthName     = fmt.Sprintf("%s-ta", testName)
	deploymentName      = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName    = fmt.Sprintf("%s-so", testName)
	minReplicaCount     = 0
	maxReplicaCount     = 1
)

type templateData struct {
	TestNamespace    string
	AgentSecretName  string
	ScalerSecretName string
	TriggerAuthName  string
	DeploymentName   string
	ScaledObjectName string
	MinReplicaCount  string
	MaxReplicaCount  string
	Pat              string
	URL              string
	PoolName         string
	PoolID           string
	AzureADClientID  string
	AzureADSecret    string
	AzureADTenantID  string
}

const (
	agentSecretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.AgentSecretName}}
  namespace: {{.TestNamespace}}
data:
  personalAccessToken: {{.Pat}}
`

	scalerSecretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.ScalerSecretName}}
  namespace: {{.TestNamespace}}
data:
  clientSecret: {{.AzureADSecret}}
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
                name: {{.AgentSecretName}}
                key: personalAccessToken
          - name: AZP_POOL
            value: {{.PoolName}}
        volumeMounts:
        - mountPath: /var/run/docker.sock
          name: docker-volume
      volumes:
      - name: docker-volume
        hostPath:
          path: /var/run/docker.sock
`

	triggerAuthTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: clientSecret
      name: {{.ScalerSecretName}}
      key: clientSecret
`

	poolIDScaledObjectTemplate = `
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
      clientId: {{.AzureADClientID}}
      tenantId: {{.AzureADTenantID}}
    authenticationRef:
      name: {{.TriggerAuthName}}
`

	poolNameScaledObjectTemplate = `
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
      clientId: {{.AzureADClientID}}
      tenantId: {{.AzureADTenantID}}
    authenticationRef:
      name: {{.TriggerAuthName}}
`
)

func TestScaler(t *testing.T) {
	t.Log("--- setting up ---")
	require.NotEmpty(t, organizationURL, "AZURE_DEVOPS_ORGANIZATION_URL env variable is required for azure pipelines test")
	require.NotEmpty(t, personalAccessToken, "AZURE_DEVOPS_PAT env variable is required for azure pipelines test")
	require.NotEmpty(t, project, "AZURE_DEVOPS_PROJECT env variable is required for azure pipelines test")
	require.NotEmpty(t, buildID, "AZURE_DEVOPS_BUILD_DEFINITION_ID env variable is required for azure pipelines test")
	require.NotEmpty(t, poolName, "AZURE_DEVOPS_POOL_NAME env variable is required for azure pipelines test")
	require.NotEmpty(t, azureADClientID, "TF_AZURE_SP_APP_ID env variable is required for azure pipelines SPN test")
	require.NotEmpty(t, azureADSecret, "AZURE_SP_KEY env variable is required for azure pipelines SPN test")
	require.NotEmpty(t, azureADTenantID, "TF_AZURE_SP_TENANT env variable is required for azure pipelines SPN test")

	connection := azuredevops.NewPatConnection(organizationURL, personalAccessToken)
	clearAllBuilds(t, connection)
	poolID = fmt.Sprintf("%d", getAzDoPoolID(t, connection))

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	WaitForPodCountInNamespace(t, kc, testNamespace, minReplicaCount, 60, 2)

	testActivation(t, kc, connection)
	testScaleOut(t, kc, connection)
	testScaleIn(t, kc)

	KubectlApplyWithTemplate(t, data, "poolNameScaledObjectTemplate", poolNameScaledObjectTemplate)
	testActivation(t, kc, connection)
	testScaleOut(t, kc, connection)
	testScaleIn(t, kc)

	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getAzDoPoolID(t *testing.T, connection *azuredevops.Connection) int {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	taskClient, err := taskagent.NewClient(ctx, connection)
	if err != nil {
		t.Error(fmt.Sprintf("unable to create  task agent client: %s", err.Error()), err)
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
		t.Error(fmt.Sprintf("unable to create build client: %s", err.Error()), err)
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
		t.Error(fmt.Sprintf("unable to create build client: %s", err.Error()), err)
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
	base64ClientSecret := base64.StdEncoding.EncodeToString([]byte(azureADSecret))

	return templateData{
			TestNamespace:    testNamespace,
			AgentSecretName:  agentSecretName,
			ScalerSecretName: scalerSecretName,
			TriggerAuthName:  triggerAuthName,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			MinReplicaCount:  fmt.Sprintf("%v", minReplicaCount),
			MaxReplicaCount:  fmt.Sprintf("%v", maxReplicaCount),
			Pat:              base64Pat,
			URL:              organizationURL,
			PoolName:         poolName,
			PoolID:           poolID,
			AzureADClientID:  azureADClientID,
			AzureADSecret:    base64ClientSecret,
			AzureADTenantID:  azureADTenantID,
		}, []Template{
			{Name: "agentSecretTemplate", Config: agentSecretTemplate},
			{Name: "scalerSecretTemplate", Config: scalerSecretTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "poolIDScaledObjectTemplate", Config: poolIDScaledObjectTemplate},
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
