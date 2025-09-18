//go:build e2e
// +build e2e

package github_runner_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/google/go-github/v50/github"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "github-runner-test"
)

var (
	personalAccessToken = os.Getenv("GH_AUTOMATION_PAT")
	owner               = os.Getenv("GH_OWNER")
	githubScope         = os.Getenv("GH_SCOPE")
	repos               = os.Getenv("GH_REPOS")
	testNamespace       = fmt.Sprintf("%s-ns", testName)
	secretName          = fmt.Sprintf("%s-secret", testName)
	deploymentName      = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName    = fmt.Sprintf("%s-so", testName)
	scaledJobName       = fmt.Sprintf("%s-sj", testName)
	minReplicaCount     = 0
	maxReplicaCount     = 1
	workflowID          = os.Getenv("GH_WORKFLOW_ID")
	soWorkflowID        = os.Getenv("GH_SO_WORKFLOW_ID")
	ghaWorkflowID       = os.Getenv("GH_GHA_WORKFLOW_ID")
	appID               = os.Getenv("GH_APP_ID")
	instID              = os.Getenv("GH_INST_ID")
	appKey              = os.Getenv("GH_APP_KEY")
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
	Owner            string
	Repos            string
	RunnerScope      string
	Labels           string
	ApplicationID    string
	InstallationID   string
	ApplicationKey   string
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
	triggerAuthTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: github-trigger-auth
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: personalAccessToken
      name: {{.SecretName}}
      key: personalAccessToken
`
	secretGhaTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}-gha
  namespace: {{.TestNamespace}}
data:
  appKey: {{.ApplicationKey}}
`
	triggerGhaAuthTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: github-gha-trigger-auth
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: appKey
      name: {{.SecretName}}-gha
      key: appKey
`

	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: github-runner
spec:
  replicas: 0
  selector:
    matchLabels:
      app: github-runner
  template:
    metadata:
      labels:
        app: github-runner
    spec:
      terminationGracePeriodSeconds: 90
      containers:
      - name: github-runner
        image: myoung34/github-runner
        imagePullPolicy: Always
        env:
          - name: EPHEMERAL
            value: "true"
          - name: DISABLE_RUNNER_UPDATE
            value: "true"
          - name: REPO_URL
            value: "https://github.com/{{.Owner}}/{{.Repos}}"
          - name: RUNNER_SCOPE
            value: {{.RunnerScope}}
          - name: LABELS
            value: "e2eSOtester"
          - name: ACCESS_TOKEN
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: personalAccessToken
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
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  pollingInterval: 15
  cooldownPeriod: 5
  triggers:
  - type: github-runner
    metadata:
      owner: {{.Owner}}
      repos: {{.Repos}}
      runnerScope: {{.RunnerScope}}
      labels: "e2eSOtester"
    authenticationRef:
     name: github-trigger-auth
`
	scaledJobTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.ScaledJobName}}
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
          image: myoung34/github-runner
          imagePullPolicy: IfNotPresent
          env:
          - name: EPHEMERAL
            value: "true"
          - name: DISABLE_RUNNER_UPDATE
            value: "true"
          - name: REPO_URL
            value: "https://github.com/{{.Owner}}/{{.Repos}}"
          - name: RUNNER_SCOPE
            value: {{.RunnerScope}}
          - name: LABELS
            value: "e2etester"
          - name: ACCESS_TOKEN
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: personalAccessToken
        restartPolicy: Never
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  successfulJobsHistoryLimit: 0
  pollingInterval: 15
  triggers:
  - type: github-runner
    metadata:
      owner: {{.Owner}}
      repos: {{.Repos}}
      labels: {{.Labels}}
      runnerScopeFromEnv: "RUNNER_SCOPE"
      enableEtags: "true"
      enableBackoff: "true"
    authenticationRef:
     name: github-trigger-auth
`

	scaledGhaJobTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.ScaledJobName}}-gha
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
          image: myoung34/github-runner
          imagePullPolicy: IfNotPresent
          env:
          - name: EPHEMERAL
            value: "true"
          - name: DISABLE_RUNNER_UPDATE
            value: "true"
          - name: REPO_URL
            value: "https://github.com/{{.Owner}}/{{.Repos}}"
          - name: RUNNER_SCOPE
            value: {{.RunnerScope}}
          - name: LABELS
            value: "e2e-gha-tester"
          - name: ACCESS_TOKEN
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: personalAccessToken
        restartPolicy: Never
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  successfulJobsHistoryLimit: 0
  pollingInterval: 15
  triggers:
  - type: github-runner
    metadata:
      owner: {{.Owner}}
      repos: {{.Repos}}
      labels: "e2e-gha-tester"
      runnerScopeFromEnv: "RUNNER_SCOPE"
      applicationID: "{{.ApplicationID}}"
      installationID: "{{.InstallationID}}"
    authenticationRef:
     name: github-gha-trigger-auth
`
)

// getGitHub Client
func getGitHubClient() *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: personalAccessToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return client
}

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, personalAccessToken, "GH_AUTOMATION_PAT env variable is required for github runner test")
	require.NotEmpty(t, owner, "GH_OWNER env variable is required for github runner test")
	require.NotEmpty(t, githubScope, "GH_SCOPE env variable is required for github runner test")
	require.NotEmpty(t, repos, "GH_REPOS env variable is required for github runner test")
	require.NotEmpty(t, workflowID, "GH_WORKFLOW_ID env variable is required for github runner test")
	require.NotEmpty(t, soWorkflowID, "GH_SO_WORKFLOW_ID env variable is required for github runner test")
	require.NotEmpty(t, ghaWorkflowID, "GH_GHA_WORKFLOW_ID env variable is required for github runner test")
	require.NotEmpty(t, appID, "GH_APPLICATION_ID env variable is required for github runner test")
	require.NotEmpty(t, instID, "GH_INSTALLATION_ID env variable is required for github runner test")
	require.NotEmpty(t, appKey, "GH_APP_KEY env variable is required for github runner test")

	client := getGitHubClient()
	cancelAllRuns(t, client, repos, workflowID)
	cancelAllRuns(t, client, repos, soWorkflowID)
	cancelAllRuns(t, client, repos, ghaWorkflowID)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	WaitForPodCountInNamespace(t, kc, testNamespace, minReplicaCount, 60, 2)

	// test scaling Scaled Job with App
	KubectlApplyWithTemplate(t, data, "scaledGhaJobTemplate", scaledGhaJobTemplate)
	testJobScaleOut(t, kc, client, ghaWorkflowID)
	testJobScaleIn(t, kc)

	// test scaling Scaled Job
	KubectlApplyWithTemplate(t, data, "scaledJobTemplate", scaledJobTemplate)
	testJobScaleOut(t, kc, client, workflowID)
	testJobScaleIn(t, kc)

	// test scaling Scaled Object
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	testSONotActivated(t, kc)
	testSOScaleOut(t, kc, client)
	testSOScaleIn(t, kc)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func queueRun(t *testing.T, ghClient *github.Client, flowID string) {
	b := &github.CreateWorkflowDispatchEventRequest{
		Ref: "main",
	}

	wID, err := strconv.ParseInt(flowID, 10, 64)
	if err != nil {
		t.Log(err)
	}

	_, err = ghClient.Actions.CreateWorkflowDispatchEventByID(context.Background(), owner, repos, wID, *b)
	if err != nil {
		t.Log(err)
	}
}

func cancelAllRuns(t *testing.T, ghClient *github.Client, repos string, flowID string) {
	wID, err := strconv.ParseInt(flowID, 10, 64)
	if err != nil {
		t.Log(err)
	}

	runs, resp, err := ghClient.Actions.ListWorkflowRunsByID(context.Background(), owner, repos, wID, &github.ListWorkflowRunsOptions{
		Status: "queued",
	})
	t.Log(resp.Body)
	if err != nil {
		t.Log(err)
	}
	for _, run := range runs.WorkflowRuns {
		_, err := ghClient.Actions.CancelWorkflowRunByID(context.Background(), owner, repos, *run.ID)
		if err != nil {
			t.Log(err)
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
			RunnerScope:      githubScope,
			Owner:            owner,
			Repos:            repos,
			Labels:           "e2etester",
			ApplicationID:    appID,
			InstallationID:   instID,
			ApplicationKey:   appKey,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "authTemplate", Config: triggerAuthTemplate},
			{Name: "secretGhaTemplate", Config: secretGhaTemplate},
			{Name: "authGhaTemplate", Config: triggerGhaAuthTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
			{Name: "scaledJobTemplate", Config: scaledJobTemplate},
			{Name: "scaledGhaJobTemplate", Config: scaledGhaJobTemplate},
		}
}

func testSONotActivated(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing none activation ---")
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testSOScaleOut(t *testing.T, kc *kubernetes.Clientset, ghClient *github.Client) {
	t.Log("--- testing scale out ---")
	queueRun(t, ghClient, soWorkflowID)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 1),
		"replica count should be 2 after 1 minute")
}

func testSOScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	assert.True(t, WaitForPodCountInNamespace(t, kc, testNamespace, minReplicaCount, 60, 5),
		"pod count should be 0 after 1 minute")
}

func testJobScaleOut(t *testing.T, kc *kubernetes.Clientset, ghClient *github.Client, wfID string) {
	t.Log("--- testing scale out ---")
	queueRun(t, ghClient, wfID)

	assert.True(t, WaitForJobCount(t, kc, testNamespace, 1, 60, 1), "replica count should be 1 after 1 minute")
}

func testJobScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	assert.True(t, WaitForAllJobsSuccess(t, kc, testNamespace, 60, 5), "jobs should be completed after 1 minute")
	DeletePodsInNamespaceBySelector(t, kc, "app="+scaledJobName, testNamespace)
}
