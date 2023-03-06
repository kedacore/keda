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
	personalAccessToken = os.Getenv("GITHUB_PAT")
	owner               = os.Getenv("GITHUB_OWNER")
	githubScope         = os.Getenv("GITHUB_SCOPE")
	repos               = os.Getenv("GITHUB_REPOS")
	testNamespace       = fmt.Sprintf("%s-ns", testName)
	secretName          = fmt.Sprintf("%s-secret", testName)
	deploymentName      = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName    = fmt.Sprintf("%s-so", testName)
	minReplicaCount     = 0
	maxReplicaCount     = 1
	workflowID          = os.Getenv("GITHUB_WORKFLOW_ID")
)

type templateData struct {
	TestNamespace    string
	SecretName       string
	DeploymentName   string
	ScaledObjectName string
	MinReplicaCount  string
	MaxReplicaCount  string
	Pat              string
	Owner            string
	Repos            string
	RunnerScope      string
	Labels           string
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
    app: github-runner
spec:
  replicas: 1
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
        lifecycle:
          preStop:
            exec:
              command: ["/bin/sleep","60"]
        image: myoung34/github-runner:2.302.1-ubuntu-focal
        env:
		  - name: EPHEMERAL
			value: "true"
          - name: DISABLE_RUNNER_UPDATE
		  	value: "true"
		  - name: RUNNER_SCOPE: 
		    value: {{.RunnerScope}}
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
      personalAccessTokenFromEnv: "ACCESS_TOKEN"
      activationTargetPipelinesQueueLength: "1"
	  owner: {{.Owner}}
	  repos: {{.Repos}}
	  labels: {{.Labels}}
`
	scaledJobTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
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
      personalAccessTokenFromEnv: "ACCESS_TOKEN"
      activationTargetPipelinesQueueLength: "1"
	  owner: {{.Owner}}
	  repos: {{.Repos}}
	  labels: {{.Labels}}
`
)

// getGitHub Client
func getGitHubClient(t *testing.T) *github.Client {
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
	require.NotEmpty(t, personalAccessToken, "GITHUB_PAT env variable is required for github runner test")
	require.NotEmpty(t, owner, "GITHUB_OWNER env variable is required for github runner test")
	require.NotEmpty(t, githubScope, "GITHUB_SCOPE env variable is required for github runner test")
	require.NotEmpty(t, repos, "GITHUB_REPOS env variable is required for github runner test")

	client := getGitHubClient(t)
	cancelAllRuns(t, client, repos)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	WaitForPodCountInNamespace(t, kc, testNamespace, minReplicaCount, 60, 2)

	// test scaling poolId
	testActivation(t, kc, client)
	testScaleOut(t, kc, client)
	testScaleIn(t, kc)

	// test scaling PoolName
	KubectlApplyWithTemplate(t, data, "poolNamescaledObjectTemplate", scaledObjectTemplate)
	testActivation(t, kc, client)
	testScaleOut(t, kc, client)
	testScaleIn(t, kc)

	// cleanup
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func queueRun(t *testing.T, ghClient *github.Client) {
	b := &github.CreateWorkflowDispatchEventRequest{
		Ref: "main",
	}

	wID, err := strconv.ParseInt(workflowID, 10, 64)
	if err != nil {
		t.Log(err)
	}

	_, err = ghClient.Actions.CreateWorkflowDispatchEventByID(context.Background(), owner, repos, wID, *b)
	if err != nil {
		t.Log(err)
	}

}

func cancelAllRuns(t *testing.T, ghClient *github.Client, repos string) {
	wID, err := strconv.ParseInt(workflowID, 10, 64)
	if err != nil {
		t.Log(err)
	}

	runs, _, err := ghClient.Actions.ListWorkflowRunsByID(context.Background(), owner, repos, wID, &github.ListWorkflowRunsOptions{
		Status: "queued",
	})
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
			MinReplicaCount:  fmt.Sprintf("%v", minReplicaCount),
			MaxReplicaCount:  fmt.Sprintf("%v", maxReplicaCount),
			Pat:              base64Pat,
			RunnerScope:      githubScope,
			Owner:            owner,
			Repos:            repos,
			Labels:           "e2etester",
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "poolIdscaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, ghClient *github.Client) {
	t.Log("--- testing activation ---")
	queueRun(t, ghClient)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, ghClient *github.Client) {
	t.Log("--- testing scale out ---")
	queueRun(t, ghClient)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 1),
		"replica count should be 2 after 1 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")
	assert.True(t, WaitForPodCountInNamespace(t, kc, testNamespace, minReplicaCount, 60, 5),
		"pod count should be 0 after 1 minute")
}
