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

var _ = godotenv.Load("../../../.env")

const testName = "azure-pipelines-sp-test"

var (
	organizationURL              = os.Getenv("AZURE_DEVOPS_ORGANIZATION_URL")
	personalAccessToken          = os.Getenv("AZURE_DEVOPS_PAT")
	project                      = os.Getenv("AZURE_DEVOPS_PROJECT")
	buildID                      = os.Getenv("AZURE_DEVOPS_BUILD_DEFINITION_ID")
	poolName                     = os.Getenv("AZURE_DEVOPS_POOL_NAME")
	servicePrincipalClientID     = os.Getenv("TF_AZURE_SP_APP_ID")
	servicePrincipalClientSecret = os.Getenv("AZURE_SP_KEY")
	servicePrincipalTenantID     = os.Getenv("TF_AZURE_SP_TENANT")
	poolID                       = "0"
	testNamespace                = fmt.Sprintf("%s-ns", testName)
	agentSecretName              = fmt.Sprintf("%s-agent-secret", testName)
	scalerSecretName             = fmt.Sprintf("%s-scaler-secret", testName)
	triggerAuthName              = fmt.Sprintf("%s-ta", testName)
	deploymentName               = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName             = fmt.Sprintf("%s-so", testName)
	minReplicaCount              = 0
	maxReplicaCount              = 1
)

type templateData struct {
	TestNamespace                string
	AgentSecretName              string
	ScalerSecretName             string
	TriggerAuthName              string
	DeploymentName               string
	ScaledObjectName             string
	MinReplicaCount              string
	MaxReplicaCount              string
	PersonalAccessToken          string
	OrganizationURL              string
	PoolName                     string
	PoolID                       string
	ServicePrincipalClientID     string
	ServicePrincipalClientSecret string
	ServicePrincipalTenantID     string
}

const (
	agentSecretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.AgentSecretName}}
  namespace: {{.TestNamespace}}
data:
  personalAccessToken: {{.PersonalAccessToken}}
`

	scalerSecretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.ScalerSecretName}}
  namespace: {{.TestNamespace}}
data:
  clientSecret: {{.ServicePrincipalClientSecret}}
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
            value: {{.OrganizationURL}}
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
  azureServicePrincipal:
    tenantId: {{.ServicePrincipalTenantID}}
    clientId: {{.ServicePrincipalClientID}}
    clientSecret:
      valueFrom:
        secretKeyRef:
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
	require.NotEmpty(t, servicePrincipalClientID, "TF_AZURE_SP_APP_ID env variable is required for azure pipelines service principal test")
	require.NotEmpty(t, servicePrincipalClientSecret, "AZURE_SP_KEY env variable is required for azure pipelines service principal test")
	require.NotEmpty(t, servicePrincipalTenantID, "TF_AZURE_SP_TENANT env variable is required for azure pipelines service principal test")

	connection := azuredevops.NewPatConnection(organizationURL, personalAccessToken)
	clearAllBuilds(t, connection)
	poolID = fmt.Sprintf("%d", getAzureDevOpsPoolID(t, connection))

	kubernetesClient := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kubernetesClient, testNamespace, data, templates)

	WaitForPodCountInNamespace(t, kubernetesClient, testNamespace, minReplicaCount, 60, 2)

	testActivation(t, kubernetesClient, connection)
	testScaleOut(t, kubernetesClient, connection)
	testScaleIn(t, kubernetesClient)

	KubectlApplyWithTemplate(t, data, "poolNameScaledObjectTemplate", poolNameScaledObjectTemplate)
	testActivation(t, kubernetesClient, connection)
	testScaleOut(t, kubernetesClient, connection)
	testScaleIn(t, kubernetesClient)

	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getAzureDevOpsPoolID(t *testing.T, connection *azuredevops.Connection) int {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	taskClient, err := taskagent.NewClient(ctx, connection)
	require.NoError(t, err, "unable to create task agent client")

	pools, err := taskClient.GetAgentPools(ctx, taskagent.GetAgentPoolsArgs{PoolName: &poolName})
	require.NoError(t, err, "unable to get agent pools")
	require.NotEmpty(t, *pools, "no Azure DevOps agent pool found")

	return *(*pools)[0].Id
}

func queueBuild(t *testing.T, connection *azuredevops.Connection) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	buildClient, err := build.NewClient(ctx, connection)
	require.NoError(t, err, "unable to create build client")

	id, err := strconv.Atoi(buildID)
	require.NoError(t, err, "unable to parse build ID")

	_, err = buildClient.QueueBuild(ctx, build.QueueBuildArgs{
		Project: &project,
		Build: &build.Build{
			Definition: &build.DefinitionReference{Id: &id},
		},
	})
	require.NoError(t, err, "unable to queue build")
}

func clearAllBuilds(t *testing.T, connection *azuredevops.Connection) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	buildClient, err := build.NewClient(ctx, connection)
	require.NoError(t, err, "unable to create build client")

	top := 20
	builds, err := buildClient.GetBuilds(ctx, build.GetBuildsArgs{
		Project:      &project,
		StatusFilter: &build.BuildStatusValues.All,
		QueryOrder:   &build.BuildQueryOrderValues.QueueTimeDescending,
		Top:          &top,
	})
	require.NoError(t, err, "unable to get builds")

	for _, azureBuild := range builds.Value {
		azureBuild.Status = &build.BuildStatusValues.Cancelling
		_, err = buildClient.UpdateBuild(ctx, build.UpdateBuildArgs{
			Build:   &azureBuild,
			Project: &project,
			BuildId: azureBuild.Id,
		})
		require.NoError(t, err, "unable to cancel build")
	}
}

func getTemplateData() (templateData, []Template) {
	data := templateData{
		TestNamespace:                testNamespace,
		AgentSecretName:              agentSecretName,
		ScalerSecretName:             scalerSecretName,
		TriggerAuthName:              triggerAuthName,
		DeploymentName:               deploymentName,
		ScaledObjectName:             scaledObjectName,
		MinReplicaCount:              strconv.Itoa(minReplicaCount),
		MaxReplicaCount:              strconv.Itoa(maxReplicaCount),
		PersonalAccessToken:          base64.StdEncoding.EncodeToString([]byte(personalAccessToken)),
		OrganizationURL:              organizationURL,
		PoolName:                     poolName,
		PoolID:                       poolID,
		ServicePrincipalClientID:     servicePrincipalClientID,
		ServicePrincipalClientSecret: base64.StdEncoding.EncodeToString([]byte(servicePrincipalClientSecret)),
		ServicePrincipalTenantID:     servicePrincipalTenantID,
	}

	return data, []Template{
		{Name: "agentSecretTemplate", Config: agentSecretTemplate},
		{Name: "scalerSecretTemplate", Config: scalerSecretTemplate},
		{Name: "deploymentTemplate", Config: deploymentTemplate},
		{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
		{Name: "poolIDScaledObjectTemplate", Config: poolIDScaledObjectTemplate},
	}
}

func testActivation(t *testing.T, kubernetesClient *kubernetes.Clientset, connection *azuredevops.Connection) {
	t.Helper()

	t.Log("--- testing activation ---")
	queueBuild(t, connection)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kubernetesClient, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kubernetesClient *kubernetes.Clientset, connection *azuredevops.Connection) {
	t.Helper()

	t.Log("--- testing scale out ---")
	queueBuild(t, connection)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kubernetesClient, deploymentName, testNamespace, maxReplicaCount, 60, 1),
		"replica count should be 1 after 1 minute")
}

func testScaleIn(t *testing.T, kubernetesClient *kubernetes.Clientset) {
	t.Helper()

	t.Log("--- testing scale in ---")
	assert.True(t, WaitForPodCountInNamespace(t, kubernetesClient, testNamespace, minReplicaCount, 60, 5),
		"pod count should be 0 after 1 minute")
}
