//go:build e2e
// +build e2e

package github_runner_on_gitea_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

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
	testName = "gitea-runner-test"
)

var (
	personalAccessToken = os.Getenv("GT_AUTOMATION_PAT")
	owner               = os.Getenv("GT_OWNER")
	githubScope         = os.Getenv("GT_SCOPE")
	repos               = os.Getenv("GT_REPOS")
	testNamespace       = fmt.Sprintf("%s-ns", testName)
	testGiteaNamespace  = fmt.Sprintf("%s-gt-ns", testName)
	secretName          = fmt.Sprintf("%s-secret", testName)
	deploymentName      = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName    = fmt.Sprintf("%s-so", testName)
	scaledJobName       = fmt.Sprintf("%s-sj", testName)
	minReplicaCount     = 0
	maxReplicaCount     = 1
	workflowID          = os.Getenv("GT_WORKFLOW_ID")
	soWorkflowID        = os.Getenv("GT_SO_WORKFLOW_ID")
	giteaURL            = os.Getenv("GT_URL")
	clusterGiteaURL     = os.Getenv("GT_CLUSTER_URL")
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
	GiteaURL         string
	GiteaAPIURL      string
	GiteaNamespace   string
}

const (
	giteaServer = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gitea-webhook-api
  namespace: {{ .GiteaNamespace }}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gitea-webhook-api
  template:
    metadata:
      labels:
        app: gitea-webhook-api
    spec:
      containers:
        - name: gitea-webhook-api
          image: "ghcr.io/christopherhx/gitea-workflow-webhook-api:nightly@sha256:7cb4eaaf0f4fb6db6a4a3fb8b4d79c588b32012ded2bde296fd61e055c8ee7d7"
          ports:
            - containerPort: 3000
          command:
            - sh
            - -c
            - |
              mkdir -p /data/gitea/conf
              chmod -R 755 /data
              cat > /data/gitea/conf/app.ini << 'EOF'
              I_AM_BEING_UNSAFE_RUNNING_AS_ROOT = true
              [security]
              INSTALL_LOCK   = true
              PASSWORD_COMPLEXITY = off
              [database]
              DB_TYPE = sqlite3
              PATH = "/data/gitea.db"
              [repository]
              ROOT = "/data/"
              [server]
              ROOT_URL = http://gitea-webhook-api.{{ .GiteaNamespace }}.svc.cluster.local:3000
              EOF
              gitea migrate -c /data/gitea/conf/app.ini
              gitea admin user create --username=test01 --password=test01 --email=test01@gitea.io --admin=true --must-change-password=false --access-token
              gitea web
          volumeMounts:
            - name: data-volume
              mountPath: /data
      volumes:
        - name: data-volume
          emptyDir: {}
        - name: app-config
          configMap:
            name: gitea-app-config
            items:
              - key: app.ini
                path: app.ini
`
	giteaService = `
apiVersion: v1
kind: Service
metadata:
  namespace: {{ .GiteaNamespace }}
  name: gitea-webhook-api
  labels:
    app: gitea-webhook-api
spec:
  selector:
    app: gitea-webhook-api
  type: NodePort
  ports:
    - protocol: TCP
      port: 3000        # Port exposed inside the cluster
      targetPort: 3000  # Port on the container
`

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
        image: ghcr.io/christopherhx/gitea-keda-runner:nightly@sha256:a1519c1908eb8aa018a0ae35232b01e07e1ef93a3ff127d889419009c0645ad2
        imagePullPolicy: Always
        env:
          - name: GITEA_RUNNER_EPHEMERAL
            value: "true"
          - name: GITEA_INSTANCE_URL
            value: {{ .GiteaURL }}
          - name: GITEA_RUNNER_OWNER
            value: "{{.Owner}}"
          - name: GITEA_RUNNER_REPO
            value: "{{.Repos}}"
          - name: GITEA_RUNNER_LABELS
            value: "e2eSOtester"
          - name: GITEA_RUNNER_PAT
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
      githubApiURL: {{ .GiteaAPIURL }}
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
          image: ghcr.io/christopherhx/gitea-keda-runner:nightly@sha256:a1519c1908eb8aa018a0ae35232b01e07e1ef93a3ff127d889419009c0645ad2
          imagePullPolicy: IfNotPresent
          env:
          - name: GITEA_RUNNER_EPHEMERAL
            value: "true"
          - name: GITEA_INSTANCE_URL
            value: {{ .GiteaURL }}
          - name: GITEA_RUNNER_OWNER
            value: "{{.Owner}}"
          - name: GITEA_RUNNER_REPO
            value: "{{.Repos}}"
          - name: GITEA_RUNNER_LABELS
            value: "e2etester"
          - name: GITEA_RUNNER_PAT
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
      githubApiURL: {{ .GiteaAPIURL }}
      owner: {{.Owner}}
      repos: {{.Repos}}
      labels: {{.Labels}}
      runnerScope: {{.RunnerScope}}
    authenticationRef:
     name: github-trigger-auth
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
	client.BaseURL, _ = url.Parse(giteaURL + "/api/v1/")
	return client
}

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	if giteaURL != "" {
		require.NotEmpty(t, personalAccessToken, "GT_AUTOMATION_PAT env variable is required for github runner test")
		require.NotEmpty(t, owner, "GT_OWNER env variable is required for github runner test")
		require.NotEmpty(t, githubScope, "GT_SCOPE env variable is required for github runner test")
		require.NotEmpty(t, repos, "GT_REPOS env variable is required for github runner test")
		require.NotEmpty(t, workflowID, "GT_WORKFLOW_ID env variable is required for github runner test")
		require.NotEmpty(t, soWorkflowID, "GT_SO_WORKFLOW_ID env variable is required for github runner test")
		clusterGiteaURL = giteaURL
	}

	// Create kubernetes resources
	kc := GetKubernetesClient(t)

	// Start our own Gitea instance if not provided
	if giteaURL == "" {
		giteaData, giteaTemplates := getTemplateGiteaData()
		defer DeleteKubernetesResources(t, testGiteaNamespace, giteaData, giteaTemplates)
		CreateKubernetesResources(t, kc, testGiteaNamespace, giteaData, giteaTemplates)
		WaitForPodCountInNamespace(t, kc, testGiteaNamespace, 1, 60, 2)

		// 1. Wait for the deployment to be ready.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		deploymentName := "gitea-webhook-api"

		if err := waitForDeployment(ctx, kc, testGiteaNamespace, deploymentName); err != nil {
			assert.NoError(t, fmt.Errorf("deployment %s did not become ready: %w", deploymentName, err))
			return
		}
		log.Printf("Deployment %q is ready", deploymentName)

		httpClient := &http.Client{}

		for i := 0; i < 5; i++ {

			// 2. Port-forward: use remote port 3000 and get a random free local port.
			localPort, err := getFreePortForward(ctx, kc, KubeConfig, testGiteaNamespace, deploymentName, 3000)
			if err != nil {
				// assert.NoError(t, fmt.Errorf("Error setting up port forwarding: %w", err))
				log.Printf("Error setting up port forwarding: %v", err)
				if i == 4 {
					assert.NoError(t, fmt.Errorf("Error setting up port forwarding: %w", err))
					return
				}
				log.Printf("Retrying port forwarding in 5 seconds...")
				time.Sleep(5 * time.Second)
				continue
			}
			log.Printf("Port forwarding established: localhost:%d -> remote:3000", localPort)

			giteaURL = fmt.Sprintf("http://%s:%d", "localhost", localPort)
			clusterGiteaURL = fmt.Sprintf("http://%s:%d", "gitea-webhook-api."+testGiteaNamespace+".svc.cluster.local", 3000)
			break
		}

		var resp *http.Response
		req, err := http.NewRequest("POST", giteaURL+"/api/v1/users/test01/tokens", bytes.NewBufferString(`{"name":"test01","scopes":["all"]}`))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.SetBasicAuth("test01", "test01")

		resp, err = httpClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		body := json.NewDecoder(resp.Body)
		var tokenResponse struct {
			Token string `json:"sha1"`
		}
		err = body.Decode(&tokenResponse)
		assert.NoError(t, err)
		fmt.Printf("Token: %s\n", tokenResponse.Token)

		personalAccessToken = tokenResponse.Token

		client := getGitHubClient()

		repoName := "test-repo"
		repoRequest := &github.Repository{
			Name:          &repoName,
			Private:       github.Bool(true),
			AutoInit:      github.Bool(true),
			DefaultBranch: github.String("main"),
		}
		// Passing an empty string as the organization creates the repo in the user's namespace.
		repo, _, err := client.Repositories.Create(ctx, "", repoRequest)
		if err != nil {
			assert.NoError(t, fmt.Errorf("Error creating repository: %w", err))
			return
		}
		fmt.Printf("Repository created: %s\n", repo.GetFullName())

		// In our script, the repository owner is "test01".
		owner = "test01"
		repos = "test-repo"
		githubScope = "repo"
		workflowID = "main.yml"
		soWorkflowID = "main-so.yml"

		// Step 2: Create a new file (the workflow) in the repository.
		workflowPath := ".github/workflows/main.yml"
		workflowContent := `on: workflow_dispatch
jobs:
  test:
    runs-on: e2etester
    steps:
    - run: echo ok
`
		commitMessage := "Add test workflow"
		createResponse := createWorkflow(ctx, t, commitMessage, workflowContent, client, owner, repoName, workflowPath)
		fmt.Printf("Workflow file created at %s: Commit SHA %s\n", workflowPath, createResponse.Commit.GetSHA())

		createResponse = createWorkflow(ctx, t, commitMessage, `on: workflow_dispatch
jobs:
  test:
    runs-on: e2eSOtester
    steps:
    - run: echo ok
`, client, owner, repoName, ".github/workflows/main-so.yml")
		fmt.Printf("Workflow file created at %s: Commit SHA %s\n", ".github/workflows/main-so.yml", createResponse.Commit.GetSHA())
	}

	client := getGitHubClient()

	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	WaitForPodCountInNamespace(t, kc, testNamespace, minReplicaCount, 60, 2)

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

func createWorkflow(ctx context.Context, t *testing.T, commitMessage string, workflowContent string, client *github.Client, owner string, repoName string, workflowPath string) *github.RepositoryContentResponse {
	fileOpts := &github.RepositoryContentFileOptions{
		Message: &commitMessage,
		Content: []byte(workflowContent),
		Branch:  github.String("main"),
	}
	req, err := client.NewRequest(http.MethodPost, fmt.Sprintf("repos/%s/%s/contents/%s", owner, repoName, workflowPath), fileOpts)
	assert.NoError(t, err)
	createResponse := new(github.RepositoryContentResponse)
	_, err = client.Do(ctx, req, createResponse)
	assert.NoError(t, err)
	return createResponse
}

func queueRun(t *testing.T, ghClient *github.Client, flowID string) {
	b := &github.CreateWorkflowDispatchEventRequest{
		Ref: "main",
	}

	_, err := ghClient.Actions.CreateWorkflowDispatchEventByFileName(context.Background(), owner, repos, flowID, *b)
	if err != nil {
		t.Log(err)
	}
}

func getTemplateGiteaData() (templateData, []Template) {
	return templateData{
			GiteaNamespace: testGiteaNamespace,
		}, []Template{
			{Name: "giteaServerTemplate", Config: giteaServer},
			{Name: "giteaServiceTemplate", Config: giteaService},
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
			GiteaURL:         clusterGiteaURL,
			GiteaAPIURL:      clusterGiteaURL + "/api/v1",
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "authTemplate", Config: triggerAuthTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
			{Name: "scaledJobTemplate", Config: scaledJobTemplate},
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
