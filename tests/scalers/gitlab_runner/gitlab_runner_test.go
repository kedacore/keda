//go:build e2e
// +build e2e

package gitlab_runner_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName      = "gitlab-runner-test"
	gitlabBaseURL = "https://gitlab.com"

	ciFileContent = `stages: [deploy]\ndeploy-job:\n  stage: deploy\n  script: [\"sleep 15\"]`
)

var (
	personalAccessToken = os.Getenv("GITLAB_PAT")

	defaultHeaders = map[string]string{
		"PRIVATE-TOKEN": personalAccessToken,
		"Content-Type":  "application/json",
	}

	minReplicaCount = 0
	maxReplicaCount = 1

	scaledObjectName = fmt.Sprintf("%s-so", testName)
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
)

type templateData struct {
	TestNamespace                       string
	SecretName                          string
	DeploymentName                      string
	ScaledObjectName                    string
	ScaledJobName                       string
	MinReplicaCount                     string
	MaxReplicaCount                     string
	Pat                                 string
	GitlabAPIURL                        string
	ProjectID                           string
	TargetPipelineQueueLength           string
	ActivationTargetPipelineQueueLength string
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
  name: gitlab-trigger-auth
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: personalAccessToken
      name: {{.SecretName}}
      key: personalAccessToken
`
	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: gitlab-runner
spec:
  replicas: 0
  selector:
    matchLabels:
      app: gitlab-runner
  template:
    metadata:
      labels:
        app: gitlab-runner
    spec:
      terminationGracePeriodSeconds: 90
      containers:
      - name: gitlab-runner
        image: busybox
        command: ["/bin/sh", "-c", "trap 'echo SIGTERM received; exit 0' SIGTERM; while true; do sleep 30; done"]
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
  - type: gitlab-runner
    metadata:
      gitlabAPIURL: {{.GitlabAPIURL}}
      projectID: "{{.ProjectID}}"
      targetPipelineQueueLength: "{{.TargetPipelineQueueLength}}"
      activationTargetPipelineQueueLength: "{{.ActivationTargetPipelineQueueLength}}"
    authenticationRef:
     name: gitlab-trigger-auth
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	t.Log("--- deleting all projects ---")
	err := deleteAllUserProjects(gitlabBaseURL)
	require.NoError(t, err)

	t.Log("--- creating new project ---")
	projectID, err := createNewProject(gitlabBaseURL)
	require.NoError(t, err)

	defer func() {
		t.Log("--- cleanup project ---")
		err := deleteRepo(gitlabBaseURL, projectID)
		require.NoError(t, err)
	}()

	t.Log("--- add ci file ---")
	err = commitFile(gitlabBaseURL, ciFileContent, projectID, ".gitlab-ci.yml", true)
	require.NoError(t, err)

	// Create kubernetes resources
	t.Log("--- create kubernetes resources ---")
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData(projectID)
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	WaitForPodCountInNamespace(t, kc, testNamespace, minReplicaCount, 60, 2)

	// test scaling Scaled Object
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	testActivation(t, kc)

	testScaleOut(t, kc, projectID)
	testScaleIn(t, kc, projectID)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func queueRun(t *testing.T, projectID string) {
	err := commitFile(gitlabBaseURL, "dummy content hello world", projectID, "dummy"+uuid.NewString(), false)
	require.NoError(t, err)
}

func getTemplateData(projectID string) (templateData, []Template) {
	base64Pat := base64.StdEncoding.EncodeToString([]byte(personalAccessToken))

	return templateData{
			TestNamespace:                       testNamespace,
			SecretName:                          secretName,
			DeploymentName:                      deploymentName,
			ScaledObjectName:                    scaledObjectName,
			MinReplicaCount:                     fmt.Sprintf("%v", minReplicaCount),
			MaxReplicaCount:                     fmt.Sprintf("%v", maxReplicaCount),
			Pat:                                 base64Pat,
			GitlabAPIURL:                        "https://gitlab.com",
			ProjectID:                           projectID,
			TargetPipelineQueueLength:           "1",
			ActivationTargetPipelineQueueLength: "1",
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "authTemplate", Config: triggerAuthTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing none activation ---")
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, projectID string) {
	t.Log("--- testing scale out ---")
	queueRun(t, projectID)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 1),
		"replica count should be 1 after 1 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, projectID string) {
	t.Log("--- testing scale in ---")
	deleteAllPipelines(t, gitlabBaseURL, projectID)

	assert.True(t, WaitForPodCountInNamespace(t, kc, testNamespace, minReplicaCount, 60, 5),
		"pod count should be 0 after 5 minutes")
}

func createNewProject(gitlabBaseURL string) (id string, err error) {
	// Define the URL and request body
	url := gitlabBaseURL + "/api/v4/projects/"

	salt := uuid.New().String()
	data := fmt.Sprintf(`{
		"name": "new_project %s",
		"description": "New Project %s",
		"path": "new_project_%s",
		"initialize_with_readme": "false",
		"shared_runners_enabled": "true"
	}`, salt, salt, salt)

	// Create a new POST request with the required headers
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte(data)))
	if err != nil {
		return "", err
	}

	for header, value := range defaultHeaders {
		req.Header.Set(header, value)
	}

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse the response body
	var createdProject struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&createdProject); err != nil {
		return "", err
	}

	return strconv.Itoa(createdProject.ID), nil
}

func deleteRepo(gitlabBaseURL, projectID string) error {
	url := gitlabBaseURL + "/api/v4/projects/" + projectID

	// Create a new POST request with the required headers
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	for header, value := range defaultHeaders {
		req.Header.Set(header, value)
	}

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return err
}

func commitFile(gitlabBaseURL, content, projectID, filepath string, ciSkip bool) error {
	// Define the URL and request body
	url := gitlabBaseURL + "/api/v4/projects/" + projectID + "/repository/files/" + url.QueryEscape(filepath)

	ciSkipPrefix := ""
	if ciSkip {
		ciSkipPrefix = "[ci skip] "
	}

	data := fmt.Sprintf(`{
		"branch": "main",
		"author_email": "jp.sartre@example.com",
		"author_name": "JP Sartre",
		"content": "%s",
		"commit_message": "%screate a new file"
	}`, content, ciSkipPrefix)

	// Create a new POST request with the required headers
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte(data)))
	if err != nil {
		return err
	}

	for header, value := range defaultHeaders {
		req.Header.Set(header, value)
	}

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func getCurrentUser(gitlabBaseURL string) (id string, err error) {
	// Define the URL and request body
	url := gitlabBaseURL + "/api/v4/user"

	// Create a new POST request with the required headers
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	for header, value := range defaultHeaders {
		req.Header.Set(header, value)
	}

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse the response body
	var currentUser struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&currentUser); err != nil {
		return "", err
	}

	return strconv.Itoa(currentUser.ID), nil
}

type project struct {
	ID int `json:"id"`
}

func getUserProjectIDs(gitlabBaseURL, userID string) ([]project, error) {
	// Define the URL and request body
	url := gitlabBaseURL + "/api/v4/users/" + userID + "/projects"
	// Create a new POST request with the required headers
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for header, value := range defaultHeaders {
		req.Header.Set(header, value)
	}
	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	projects := make([]project, 0)
	// Parse the response body
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, err
	}

	return projects, nil
}

func deleteAllUserProjects(gitlabBaseURL string) error {
	userID, err := getCurrentUser(gitlabBaseURL)
	if err != nil {
		return err
	}

	projects, err := getUserProjectIDs(gitlabBaseURL, userID)
	if err != nil {
		return err
	}

	for _, project := range projects {
		err := deleteRepo(gitlabBaseURL, strconv.Itoa(project.ID))
		if err != nil {
			return err
		}
	}

	return nil
}

type pipeline struct {
	ID int `json:"id"`
}

func getPipelines(gitlabBaseURL, projectID string) ([]pipeline, error) {
	// Define the URL and request body
	uri := gitlabBaseURL + "/api/v4/projects/" + projectID + "/pipelines"

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	for header, value := range defaultHeaders {
		req.Header.Set(header, value)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("doing request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	gitlabPipelines := make([]pipeline, 0)
	if err := json.NewDecoder(res.Body).Decode(&gitlabPipelines); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return gitlabPipelines, nil
}

func deletePipeline(gitlabBaseURL, projectID, pipelineID string) error {
	uri := gitlabBaseURL + "/api/v4/projects/" + projectID + "/pipelines/" + pipelineID
	req, err := http.NewRequest(http.MethodDelete, uri, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	for header, value := range defaultHeaders {
		req.Header.Set(header, value)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("doing request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return nil
}

func deleteAllPipelines(t *testing.T, gitlabBaseURL, projectID string) {
	pipelines, err := getPipelines(gitlabBaseURL, projectID)
	require.NoError(t, err)

	for _, pipeline := range pipelines {
		err := deletePipeline(gitlabBaseURL, projectID, strconv.Itoa(pipeline.ID))
		require.NoError(t, err)
	}
}
