//go:build e2e
// +build e2e

package gitlab_runner_test

import (
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

var _ = godotenv.Load("../../.env")

const (
	testName = "gitlab-runner-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	triggerAuthName  = fmt.Sprintf("%s-ta", testName)
	mockServerName   = fmt.Sprintf("%s-mock-gitlab", testName)
	mockServiceName  = fmt.Sprintf("%s-mock-service", testName)
	configMapName    = fmt.Sprintf("%s-mock-config", testName)
	minReplicaCount  = 0
	maxReplicaCount  = 2

	gitlabAPIURL = fmt.Sprintf("http://%s.%s.svc.cluster.local:8080", mockServiceName, testNamespace)

	// These can be overridden via env vars for testing against a real GitLab instance.
	// When empty, the test uses the in-cluster mock GitLab API server.
	realGitLabURL       = os.Getenv("TF_GITLAB_API_URL")
	realGitLabToken     = os.Getenv("TF_GITLAB_TOKEN")
	realGitLabProjectID = os.Getenv("TF_GITLAB_PROJECT_ID")
)

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	ScaledObjectName string
	SecretName       string
	TriggerAuthName  string
	MockServerName   string
	MockServiceName  string
	ConfigMapName    string
	MinReplicaCount  string
	MaxReplicaCount  string
	GitLabAPIURL     string
	PersonalToken    string
	ProjectID        string
}

const (
	// Mock GitLab API server using nginx with a lua-like approach via ConfigMap.
	// Returns canned JSON for the jobs endpoint based on a ConfigMap-mounted file.
	mockConfigMapTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.ConfigMapName}}
  namespace: {{.TestNamespace}}
data:
  nginx.conf: |
    server {
      listen 8080;

      location ~ ^/api/v4/projects/.*/jobs {
        default_type application/json;
        add_header x-total $arg_x_total;
        alias /data/jobs.json;
      }

      location ~ ^/api/v4/groups/.*/projects {
        default_type application/json;
        alias /data/projects.json;
      }

      location /health {
        return 200 'ok';
      }
    }
  jobs.json: '[]'
  projects.json: '[]'
`

	mockConfigMapWithJobsTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.ConfigMapName}}
  namespace: {{.TestNamespace}}
data:
  nginx.conf: |
    server {
      listen 8080;

      location ~ ^/api/v4/projects/.*/jobs {
        default_type application/json;
        add_header x-total 5;
        alias /data/jobs.json;
      }

      location ~ ^/api/v4/groups/.*/projects {
        default_type application/json;
        alias /data/projects.json;
      }

      location /health {
        return 200 'ok';
      }
    }
  jobs.json: |
    [
      {"id":1,"status":"pending","tag_list":[]},
      {"id":2,"status":"pending","tag_list":[]},
      {"id":3,"status":"pending","tag_list":[]},
      {"id":4,"status":"pending","tag_list":[]},
      {"id":5,"status":"pending","tag_list":[]}
    ]
  projects.json: '[]'
`

	mockServerDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.MockServerName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.MockServerName}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.MockServerName}}
  template:
    metadata:
      labels:
        app: {{.MockServerName}}
    spec:
      containers:
      - name: nginx
        image: ghcr.io/nginx/nginx-unprivileged:1.26
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: config
          mountPath: /etc/nginx/conf.d/default.conf
          subPath: nginx.conf
        - name: config
          mountPath: /data/jobs.json
          subPath: jobs.json
        - name: config
          mountPath: /data/projects.json
          subPath: projects.json
      volumes:
      - name: config
        configMap:
          name: {{.ConfigMapName}}
`

	mockServiceTemplate = `apiVersion: v1
kind: Service
metadata:
  name: {{.MockServiceName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    app: {{.MockServerName}}
  ports:
  - port: 8080
    targetPort: 8080
`

	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  personalAccessToken: {{.PersonalToken}}
`

	triggerAuthTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: personalAccessToken
      name: {{.SecretName}}
      key: personalAccessToken
`

	deploymentTemplate = `apiVersion: apps/v1
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

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  pollingInterval: 5
  cooldownPeriod: 5
  triggers:
  - type: gitlab-runner
    metadata:
      gitlabAPIURL: "{{.GitLabAPIURL}}"
      projectID: "{{.ProjectID}}"
      targetQueueLength: "1"
    authenticationRef:
      name: {{.TriggerAuthName}}
`
)

func TestScaler(t *testing.T) {
	t.Log("--- setting up ---")
	kc := GetKubernetesClient(t)

	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// Wait for mock server to be ready
	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, mockServerName, testNamespace, 1, 60, 1),
		"mock GitLab API server should be running after 1 minute")

	// Test that scaler stays inactive with empty queue
	testActivation(t, kc)

	// Test scale out by adding pending jobs
	testScaleOut(t, kc, data)

	// Test scale in by clearing the queue
	testScaleIn(t, kc, data)
}

func getTemplateData() (templateData, []Template) {
	projectID := "12345"
	apiURL := gitlabAPIURL
	token := "glpat-mock-token"

	if realGitLabURL != "" && realGitLabToken != "" && realGitLabProjectID != "" {
		apiURL = realGitLabURL
		token = realGitLabToken
		projectID = realGitLabProjectID
	}

	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			SecretName:       secretName,
			TriggerAuthName:  triggerAuthName,
			MockServerName:   mockServerName,
			MockServiceName:  mockServiceName,
			ConfigMapName:    configMapName,
			MinReplicaCount:  fmt.Sprintf("%d", minReplicaCount),
			MaxReplicaCount:  fmt.Sprintf("%d", maxReplicaCount),
			GitLabAPIURL:     apiURL,
			PersonalToken:    base64.StdEncoding.EncodeToString([]byte(token)),
			ProjectID:        projectID,
		}, []Template{
			{Name: "mockConfigMapTemplate", Config: mockConfigMapTemplate},
			{Name: "mockServerDeploymentTemplate", Config: mockServerDeploymentTemplate},
			{Name: "mockServiceTemplate", Config: mockServiceTemplate},
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation ---")
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")

	KubectlApplyWithTemplate(t, data, "mockConfigMapWithJobsTemplate", mockConfigMapWithJobsTemplate)

	// Kill mock server pods so new ones mount the updated ConfigMap
	DeletePodsInNamespaceBySelector(t, kc, "app="+mockServerName, testNamespace)
	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, mockServerName, testNamespace, 1, 60, 1),
		"mock GitLab API server should be running after restart")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")

	KubectlApplyWithTemplate(t, data, "mockConfigMapTemplate", mockConfigMapTemplate)

	// Kill mock server pods so new ones mount the updated ConfigMap
	DeletePodsInNamespaceBySelector(t, kc, "app="+mockServerName, testNamespace)
	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, mockServerName, testNamespace, 1, 60, 1),
		"mock GitLab API server should be running after restart")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 5),
		"replica count should be %d after 5 minutes", minReplicaCount)
}
