//go:build e2e
// +build e2e

package azure_pipelines_adv_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
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
	testName = "azure-pipelines-demands-test"
)

var (
	organizationURL     = os.Getenv("AZURE_DEVOPS_ORGANIZATION_URL")
	personalAccessToken = os.Getenv("AZURE_DEVOPS_PAT")
	project             = os.Getenv("AZURE_DEVOPS_PROJECT")
	demandParentBuildID = os.Getenv("AZURE_DEVOPS_DEMAND_PARENT_BUILD_DEFINITION_ID")
	poolName            = os.Getenv("AZURE_DEVOPS_DEMAND_POOL_NAME")
	poolID              = "0"
	testNamespace       = fmt.Sprintf("%s-ns", testName)
	secretName          = fmt.Sprintf("%s-secret", testName)
	deploymentName      = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName    = fmt.Sprintf("%s-so", testName)
	scaledJobName       = fmt.Sprintf("%s-sj", testName)
	minReplicaCount     = 0
	maxReplicaCount     = 1
)

type templateData struct {
	TestNamespace    string
	SecretName       string
	DeploymentName   string
	ScaledObjectName string
	ScaledJobName    string
	MinReplicaCount  string
	MaxReplicaCount  string
	Pat              string
	URL              string
	PoolName         string
	PoolID           string
	SeedType         string
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
	seedDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.SeedType}}-template
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
        image: eldarrin/azure:main
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
          - name: AZP_AGENT_NAME
            value: {{.SeedType}}-template
          - name: {{.SeedType}}
            value: "true"
`
	demandScaledJobTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.SeedType}}-agent-demand-sj
  namespace: {{.TestNamespace}}
spec:
  jobTargetRef:
    template:
      metadata:
        labels:
          app: {{.ScaledJobName}}
      spec:
        containers:
        - name: {{.ScaledJobName}}
          image: eldarrin/azure:main
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
            - name: {{.SeedType}}
              value: "true"
        restartPolicy: Never
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  pollingInterval: 15
  triggers:
  - type: azure-pipelines
    metadata:
      organizationURLFromEnv: "AZP_URL"
      personalAccessTokenFromEnv: "AZP_TOKEN"
      poolName: "{{.PoolName}}"
      demands: "{{.SeedType}}"
`
	demandRequireAllScaledJobTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.SeedType}}-alldemand-sj
  namespace: {{.TestNamespace}}
spec:
  jobTargetRef:
    template:
      metadata:
        labels:
          app: {{.ScaledJobName}}
      spec:
        containers:
        - name: {{.ScaledJobName}}
          image: eldarrin/azure:main
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
            - name: {{.SeedType}}
              value: "true"
        restartPolicy: Never
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  pollingInterval: 15
  triggers:
  - type: azure-pipelines
    metadata:
      organizationURLFromEnv: "AZP_URL"
      personalAccessTokenFromEnv: "AZP_TOKEN"
      poolName: "{{.PoolName}}"
      demands: "{{.SeedType}}"
      requireAllDemands: "true"
`

	parentScaledJobTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.SeedType}}-parent-sj
  namespace: {{.TestNamespace}}
spec:
  jobTargetRef:
    template:
      metadata:
        labels:
          app: {{.ScaledJobName}}
      spec:
        containers:
        - name: {{.ScaledJobName}}
          image: eldarrin/azure:main
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
            - name: {{.SeedType}}
              value: "true"
        restartPolicy: Never
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  pollingInterval: 15
  triggers:
  - type: azure-pipelines
    metadata:
      organizationURLFromEnv: "AZP_URL"
      personalAccessTokenFromEnv: "AZP_TOKEN"
      poolName: "{{.PoolName}}"
      parent: {{.SeedType}}-template
`
	anyScaledJobTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.SeedType}}-any-sj
  namespace: {{.TestNamespace}}
spec:
  jobTargetRef:
    template:
      metadata:
        labels:
          app: {{.ScaledJobName}}
      spec:
        containers:
        - name: {{.ScaledJobName}}
          image: eldarrin/azure:main
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
            - name: {{.SeedType}}
              value: "true"
        restartPolicy: Never
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  pollingInterval: 15
  triggers:
  - type: azure-pipelines
    metadata:
      organizationURLFromEnv: "AZP_URL"
      personalAccessTokenFromEnv: "AZP_TOKEN"
      poolName: "{{.PoolName}}"
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, organizationURL, "AZURE_DEVOPS_ORGANIZATION_URL env variable is required for azure pipelines test")
	require.NotEmpty(t, personalAccessToken, "AZURE_DEVOPS_PAT env variable is required for azure pipelines test")
	require.NotEmpty(t, project, "AZURE_DEVOPS_PROJECT env variable is required for azure pipelines test")
	require.NotEmpty(t, demandParentBuildID, "AZURE_DEVOPS_DEMAND_PARENT_BUILD_DEFINITION_ID env variable is required for azure pipelines test")
	require.NotEmpty(t, poolName, "AZURE_DEVOPS_DEMAND_POOL_NAME env variable is required for azure pipelines test")
	connection := azuredevops.NewPatConnection(organizationURL, personalAccessToken)
	clearAllBuilds(t, connection)
	// Get pool ID
	poolID = fmt.Sprintf("%d", getAzDoPoolID(t, connection))

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// seed never runner jobs and setup Azure DevOps
	err := preSeedAgentPool(t, data)
	require.NoError(t, err)

	WaitForPodCountInNamespace(t, kc, testNamespace, minReplicaCount, 60, 2)
	// new demand tests (assumes pre-seeded template)

	KubectlApplyWithTemplate(t, data, "demandScaledJobTemplate", demandScaledJobTemplate)
	testJobScaleOut(t, kc, connection)
	testJobScaleIn(t, kc)
	KubectlDeleteWithTemplate(t, data, "demandScaledJobTemplate", demandScaledJobTemplate)

	KubectlApplyWithTemplate(t, data, "parentScaledJobTemplate", parentScaledJobTemplate)
	testJobScaleOut(t, kc, connection)
	testJobScaleIn(t, kc)
	KubectlDeleteWithTemplate(t, data, "parentScaledJobTemplate", parentScaledJobTemplate)

	KubectlApplyWithTemplate(t, data, "anyScaledJobTemplate", anyScaledJobTemplate)
	testJobScaleOut(t, kc, connection)
	testJobScaleIn(t, kc)
	KubectlDeleteWithTemplate(t, data, "anyScaledJobTemplate", anyScaledJobTemplate)

	KubectlApplyWithTemplate(t, data, "demandRequireAllScaledJobTemplate", demandRequireAllScaledJobTemplate)
	testJobScaleOut(t, kc, connection)
	testJobScaleIn(t, kc)
	KubectlDeleteWithTemplate(t, data, "demandRequireAllScaledJobTemplate", demandRequireAllScaledJobTemplate)

	DeleteKubernetesResources(t, testNamespace, data, templates)
	CleanUpAdo(t, data)
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

func queueBuild(t *testing.T, connection *azuredevops.Connection, bid int) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	buildClient, err := build.NewClient(ctx, connection)
	if err != nil {
		t.Errorf("unable to create build client")
	}
	args := build.QueueBuildArgs{
		Project: &project,
		Build: &build.Build{
			Definition: &build.DefinitionReference{
				Id: &bid,
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
		t.Errorf("unable to create build client")
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
			ScaledJobName:    scaledJobName,
			MinReplicaCount:  fmt.Sprintf("%v", minReplicaCount),
			MaxReplicaCount:  fmt.Sprintf("%v", maxReplicaCount),
			Pat:              base64Pat,
			URL:              organizationURL,
			PoolName:         poolName,
			PoolID:           poolID,
			SeedType:         "golang", // must match the pipeline's demand
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
		}
}

func testJobScaleOut(t *testing.T, kc *kubernetes.Clientset, connection *azuredevops.Connection) {
	t.Log("--- testing scale out ---")
	id, err := strconv.Atoi(demandParentBuildID)
	if err != nil {
		t.Errorf("unable to parse buildID")
	}
	queueBuild(t, connection, id)

	assert.True(t, WaitForJobCount(t, kc, testNamespace, 1, 180, 1), "replica count should be 1 after 3 minutes")
}

func testJobScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	assert.True(t, WaitForAllJobsSuccess(t, kc, testNamespace, 60, 5), "jobs should be completed after 1 minute")
	DeletePodsInNamespaceBySelector(t, kc, "app="+scaledJobName, testNamespace)
}

// preSeed Agent Pool to stop AzDO auto failing unfulfillable jobs
func preSeedAgentPool(t *testing.T, data templateData) error {
	naData := data
	naData.SeedType = "never"
	naData.ScaledJobName = "never-agent-demand-scaledjob"
	KubectlApplyWithTemplate(t, naData, "demandScaledJobTemplate", demandScaledJobTemplate)

	naData.ScaledJobName = "never-agent-parent-scaledjob"
	KubectlApplyWithTemplate(t, naData, "parentScaledJobTemplate", parentScaledJobTemplate)

	err := KubectlApplyWithErrors(t, naData, "deploymentTemplateSeed", seedDeploymentTemplate)
	if err != nil {
		return err
	}

	err = KubectlApplyWithErrors(t, data, "deploymentTemplateSeed", seedDeploymentTemplate)
	if err != nil {
		return err
	}
	// wait for deployment to be ready in AzDO
	for !checkAgentState(t, data, "online") {
		time.Sleep(10 * time.Second)
	}
	for !checkAgentState(t, naData, "online") {
		time.Sleep(10 * time.Second)
	}
	// delete the deployment
	KubectlDeleteWithTemplate(t, naData, "deploymentTemplateSeed", seedDeploymentTemplate)
	KubectlDeleteWithTemplate(t, data, "deploymentTemplateSeed", seedDeploymentTemplate)
	for !checkAgentState(t, data, "offline") {
		time.Sleep(10 * time.Second)
	}
	for !checkAgentState(t, naData, "offline") {
		time.Sleep(10 * time.Second)
	}
	return nil
}

// isAgentPoolReady checks if the agent pool is ready
func checkAgentState(t *testing.T, data templateData, state string) bool {
	// get the agent pool id
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connection := azuredevops.NewPatConnection(data.URL, personalAccessToken)
	taskClient, err := taskagent.NewClient(ctx, connection)
	if err != nil {
		t.Errorf("unable to create task agent client, %s", err)
	}

	args := taskagent.GetAgentPoolsArgs{
		PoolName: &data.PoolName,
	}
	pools, err := taskClient.GetAgentPools(ctx, args)
	if err != nil {
		t.Errorf("unable to get the pools, %s", err)
		return false
	}

	poolID := *(*pools)[0].Id

	agents, err := taskClient.GetAgents(ctx, taskagent.GetAgentsArgs{PoolId: &poolID})
	if err != nil {
		t.Errorf("unable to get the agent, %s", err)
		return false
	}

	tState := taskagent.TaskAgentStatus(state)

	for _, agent := range *agents {
		if *agent.Enabled && *agent.Status == tState && strings.HasPrefix(*agent.Name, data.SeedType+"-template") {
			return true
		}
	}

	t.Logf("not got %s, %s agent yet", data.SeedType+"-template", state)

	return false
}

func removeAgentFromAdo(t *testing.T, data templateData) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connection := azuredevops.NewPatConnection(data.URL, personalAccessToken)
	taskClient, err := taskagent.NewClient(ctx, connection)
	if err != nil {
		t.Errorf("unable to create task agent client, %s", err)
	}

	args := taskagent.GetAgentPoolsArgs{
		PoolName: &data.PoolName,
	}
	pools, err := taskClient.GetAgentPools(ctx, args)
	if err != nil {
		t.Errorf("unable to get the pools, %s", err)
	}

	poolID := *(*pools)[0].Id

	agents, err := taskClient.GetAgents(ctx, taskagent.GetAgentsArgs{PoolId: &poolID})
	if err != nil {
		t.Errorf("unable to get the agent, %s", err)
	}

	for _, agent := range *agents {
		if *agent.Enabled && strings.HasPrefix(*agent.Name, data.SeedType+"-template") {
			err := taskClient.DeleteAgent(ctx, taskagent.DeleteAgentArgs{PoolId: &poolID, AgentId: agent.Id})
			if err != nil {
				t.Errorf("unable to delete the agent, %s", err)
			}
		}
	}
}

func CleanUpAdo(t *testing.T, data templateData) {
	// cleanup
	removeAgentFromAdo(t, data)
}
