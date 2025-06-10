//go:build e2e
// +build e2e

package forgejo_runner_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper" // For helper methods
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "forgejo-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledJobName    = fmt.Sprintf("%s-so", testName)
	configName       = fmt.Sprintf("%s-configmap", testName)
	registrationName = fmt.Sprintf("%s-registration", testName)
	newTimestamp     = time.Now().Unix()

	forgejoRunnerName  = "forgejo-runner"
	forgejoToken       = os.Getenv("FORGEJO_TOKEN")
	forgejoGlobal      = "true"
	forgejoLabel       = "ubuntu-20.04"
	forgejoAccessToken = os.Getenv("FORGEJO_ACCESS_TOKEN")
	forgejoAddress     = fmt.Sprintf("http://forgejo-service.%s.svc.cluster.local:3000", testNamespace)

	maxReplicaCount = 2
)

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	ScaledObjectName string
	ConfigName       string
	RegistrationName string
	NewTimestamp     int64

	ForgejoRunnerName  string
	ForgejoToken       string
	ForgejoGlobal      string
	ForgejoLabel       string
	ForgejoAccessToken string
	ForgejoAddress     string
}

const (
	forgejoService = `
apiVersion: v1
kind: Service
metadata:
  labels:
    app: forgejo
  name: forgejo-service
  namespace: {{.TestNamespace}}
spec:
  ports:
  - port: 3000
    protocol: TCP
    targetPort: 3000
  selector:
    app: forgejo
  type: ClusterIP
`

	forgejoDeployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: forgejo
  name: forgejo
  namespace: {{.TestNamespace}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: forgejo
  template:
    metadata:
      labels:
        app: forgejo
    spec:
      volumes:
        - name: runner-config
          emptyDir: {}
      containers:
      - image: ghcr.io/kedacore/tests-forgejo:latest
        command:
          - sh
          - -c
          - |
            sqlite3 /data/forgejo.db "UPDATE action_run_job SET created = {{.NewTimestamp}} WHERE id = 3"
            sqlite3 /data/forgejo.db "UPDATE action_run_job SET updated = {{.NewTimestamp}} WHERE id = 3"
            forgejo --config /data/app.ini forgejo-cli actions register --secret {{.ForgejoToken}} --labels ubuntu-20.04 --name keda_runner --version v6.3.1
            /usr/local/bin/gitea --config /data/app.ini web
        imagePullPolicy: IfNotPresent
        name: forgejo
        securityContext:
          allowPrivilegeEscalation: true
          runAsNonRoot: true
        ports:
        - containerPort: 3000
          name: http
          protocol: TCP
        volumeMounts:
        - name: runner-config
          mountPath: /config
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      securityContext:
        fsGroup: 1000
        runAsGroup: 1000
        runAsNonRoot: true
        runAsUser: 1000
 `

	scaledJob = `apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  labels:
    app: forgejo-runner
  name: forgejo-runner
  namespace: {{.TestNamespace}}
spec:
  jobTargetRef:
    template:
      metadata:
        labels:
          app: forgejo-job
      spec:
        volumes:
        - name: runner-data
          emptyDir: {}
        restartPolicy: Never
        shareProcessNamespace: true
        containers:
        - name: runner
          image: code.forgejo.org/forgejo/runner:6.3.1
          command:
            - sh
            - -c
            - |
              forgejo-runner create-runner-file --name keda_runner --instance {{.ForgejoAddress}} --secret {{.ForgejoToken}}
              sed -i -e "s|\"labels\": null|\"labels\": \[ \"ubuntu-20.04:host://-self-hosted\"\]|" .runner ;
              exec forgejo-runner one-job
          securityContext:
            privileged: true
            runAsUser: 0
          volumeMounts:
          - name: runner-data
            mountPath: /data
  minReplicaCount: 0
  maxReplicaCount: 1
  pollingInterval: 5
  triggers:
  - type: forgejo-runner
    metadata:
      name: "{{.ForgejoRunnerName}}"
      token: "{{.ForgejoAccessToken}}"
      address: "{{.ForgejoAddress}}"
      global: "{{.ForgejoGlobal}}"
      labels: "{{.ForgejoLabel}}"
`
)

func TestForgejoScaler(t *testing.T) {
	// setting up
	t.Log("--- setting up ---")
	require.NotEmpty(t, forgejoToken, "FORGEJO_TOKEN env variable is required")
	require.NotEmpty(t, forgejoAccessToken, "FORGEJO_ACCESS_TOKEN env variable is required")

	kc := GetKubernetesClient(t)
	dataForgejo, templatesForgejo := getForgejoData()
	t.Cleanup(func() {

	})

	CreateNamespace(t, kc, testNamespace)
	KubectlApplyMultipleWithTemplate(t, dataForgejo, templatesForgejo)

	// setup forgejo
	setupForgejo(t, kc)

	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, dataForgejo, append(templates, templatesForgejo...))
	})

	// Create kubernetes resources
	KubectlApplyMultipleWithTemplate(t, data, templates)

	// test scaling
	testScaleOut(t, kc)
	testScaleIn(t, kc)
}
func getForgejoData() (templateData, []Template) {
	return templateData{
			TestNamespace:      testNamespace,
			DeploymentName:     deploymentName,
			NewTimestamp:       newTimestamp,
			ScaledObjectName:   scaledJobName,
			ConfigName:         configName,
			RegistrationName:   registrationName,
			ForgejoRunnerName:  forgejoRunnerName,
			ForgejoToken:       forgejoToken,
			ForgejoGlobal:      forgejoGlobal,
			ForgejoLabel:       forgejoLabel,
			ForgejoAccessToken: forgejoAccessToken,
			ForgejoAddress:     forgejoAddress,
		}, []Template{
			{Name: "forgejoDeploymentTemplate", Config: forgejoDeployment},
			{Name: "forgejoServiceTemplate", Config: forgejoService},
		}
}
func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:      testNamespace,
			DeploymentName:     deploymentName,
			NewTimestamp:       newTimestamp,
			ScaledObjectName:   scaledJobName,
			ConfigName:         configName,
			RegistrationName:   registrationName,
			ForgejoRunnerName:  forgejoRunnerName,
			ForgejoToken:       forgejoToken,
			ForgejoGlobal:      forgejoGlobal,
			ForgejoLabel:       forgejoLabel,
			ForgejoAccessToken: forgejoAccessToken,
			ForgejoAddress:     forgejoAddress,
		}, []Template{
			{Name: "scaledObjectTemplate", Config: scaledJob},
		}
}

func setupForgejo(t *testing.T, kc *kubernetes.Clientset) {
	require.True(t,
		WaitForDeploymentReplicaReadyCount(t, kc, "forgejo", testNamespace, 1, 3, 20),
		"Forgejo should be running after 1 minute")
}

// pre-loaded database should scale automatically on defined label ubuntu-20.04
func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")

	assert.True(t, WaitForPodCountInNamespace(t, kc, testNamespace, maxReplicaCount, 60, 5), "pods count should be 2 after 1 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	assert.True(t, WaitForPodsCompleted(t, kc, "app=forgejo-job", testNamespace, 60, 1), "pods count should be 1 after 1 minute")
}
