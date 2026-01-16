//go:build e2e
// +build e2e

package metrics_api_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "metrics-api-test"
)

var (
	testNamespace               = fmt.Sprintf("%s-ns", testName)
	deploymentName              = fmt.Sprintf("%s-deployment", testName)
	metricsServerDeploymentName = fmt.Sprintf("%s-metrics-server", testName)
	servciceName                = fmt.Sprintf("%s-service", testName)
	triggerAuthName             = fmt.Sprintf("%s-ta", testName)
	scaledObjectName            = fmt.Sprintf("%s-so", testName)
	secretName                  = fmt.Sprintf("%s-secret", testName)
	metricsServerEndpoint       = fmt.Sprintf("http://%s.%s.svc.cluster.local:8080/api/value", servciceName, testNamespace)
	minReplicaCount             = 0
	maxReplicaCount             = 2
)

type templateData struct {
	TestNamespace                     string
	DeploymentName                    string
	MetricsServerDeploymentName       string
	MetricsServerEndpoint             string
	ServciceName                      string
	ScaledObjectName                  string
	TriggerAuthName                   string
	SecretName                        string
	NbReplicasForMetricsServer        int
	MinReplicaCount                   string
	MaxReplicaCount                   string
	UpdateMetricURL                   string
	TargetPodName                     string
	AggregateFromKubeServiceEndpoints bool
}

const (
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  AUTH_PASSWORD: U0VDUkVUCg==
  AUTH_USERNAME: VVNFUgo=
  AUTH_MODE: YmFzaWM=
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: username
      name: {{.SecretName}}
      key: AUTH_USERNAME
    - parameter: password
      name: {{.SecretName}}
      key: AUTH_PASSWORD
    - parameter: authMode
      name: {{.SecretName}}
      key: AUTH_MODE
`

	metricsServerdeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.MetricsServerDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.MetricsServerDeploymentName}}
spec:
  replicas: {{.NbReplicasForMetricsServer}}
  selector:
    matchLabels:
      app: {{.MetricsServerDeploymentName}}
  template:
    metadata:
      labels:
        app: {{.MetricsServerDeploymentName}}
    spec:
      containers:
      - name: metrics
        image: ghcr.io/kedacore/tests-metrics-api
        ports:
        - containerPort: 8080
        envFrom:
        - secretRef:
            name: {{.SecretName}}
        imagePullPolicy: Always
`

	serviceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.ServciceName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    app: {{.MetricsServerDeploymentName}}
  ports:
  - port: 8080
    targetPort: 8080
`

	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: {{.DeploymentName}}
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  replicas: 0
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

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  cooldownPeriod:  1
  triggers:
  - type: metrics-api
    metadata:
{{ if .AggregateFromKubeServiceEndpoints }}
      aggregateFromKubeServiceEndpoints: "true"
{{ end }}
      targetValue: "5"
      activationTargetValue: "20"
      url: "{{.MetricsServerEndpoint}}"
      valueLocation: 'value'
    authenticationRef:
      name: {{.TriggerAuthName}}
`
	updateMetricTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: update-{{.TargetPodName}}
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 0
  template:
    spec:
      containers:
      - name: curl-client
        image: docker.io/curlimages/curl
        imagePullPolicy: Always
        command: ["curl", "-X", "POST", "{{.UpdateMetricURL}}"]
      restartPolicy: Never`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)

	for _, nbMetricsServerReplicas := range []int{1, 10} {
		data, templates := getTemplateData(nbMetricsServerReplicas)
		CreateKubernetesResources(t, kc, testNamespace, data, templates)

		assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 180, 3),
			"replica count should be %d after 9 minutes", minReplicaCount)

		// test scaling with single replica for metrics server
		testActivation(t, kc, data)
		testScaleOut(t, kc, data)
		testScaleIn(t, kc, data)

		// cleanup
		DeleteKubernetesResources(t, testNamespace, data, templates)
	}
}

func CryptoRandInt(minVal, maxVal int) (int, error) {
	if minVal > maxVal {
		return 0, fmt.Errorf("minVal must be <= maxVal")
	}

	diff := maxVal - minVal + 1
	n, err := rand.Int(rand.Reader, big.NewInt(int64(diff)))
	if err != nil {
		return 0, err
	}

	return int(n.Int64()) + minVal, nil
}

// getSplitArrayFromAverage creates an array of `splitInNValues` elements where:
// - The average of the elements is `average`
// - No element is exactly `average` unless splitInNValues == 1
func getSplitArrayFromAverage(t *testing.T, average, splitInNValues int) []int {
	if splitInNValues == 1 {
		return []int{average}
	}

	result := make([]int, splitInNValues)
	totalSum := average * splitInNValues

	for i := 0; i < splitInNValues-1; i++ {
		var minVal, maxVal int
		minVal = 1                                   // Ensuring no negative values
		maxVal = totalSum - (splitInNValues - i - 1) // Ensuring enough sum remains
		if maxVal < minVal {
			maxVal = minVal
		}
		val, err := CryptoRandInt(minVal, maxVal)
		if err != nil {
			t.Fatal(err)
		}
		result[i] = val
		totalSum -= val
	}
	result[splitInNValues-1] = totalSum

	return result
}

func getUpdateUrlsForAllMetricAllMetricsServerReplicas(t *testing.T, kc *kubernetes.Clientset, expectedAverageMetric int, nbReplicasForMetricsServer int, nbRetry int) map[string]string {
	nbRetriesMax := 5
	// get an array for which all elements' average would give expectedAverageMetric without any of its elements being exactly expectedAverageMetric
	individualMetrics := getSplitArrayFromAverage(t, expectedAverageMetric, nbReplicasForMetricsServer)
	// use kc to curl all metrics-server replicas
	// Get pods with the specified label selector
	pods, err := kc.CoreV1().Pods(testNamespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: "app=" + metricsServerDeploymentName,
	})
	if err != nil {
		t.Fatalf("Error listing pods: %v", err)
		return nil
	}

	retryFunc := func(message string) map[string]string {
		if nbRetry >= nbRetriesMax {
			t.Fatal(message)
			return nil
		}
		t.Logf("%s. Retry calling getUpdateUrlsForAllMetricAllMetricsServerReplicas() after 1 second", message)
		time.Sleep(5 * time.Second)
		return getUpdateUrlsForAllMetricAllMetricsServerReplicas(t, kc, expectedAverageMetric, nbReplicasForMetricsServer, nbRetry+1)
	}
	if len(pods.Items) == 0 {
		return retryFunc("No pods found with the given selector.")
	}

	if len(pods.Items) != nbReplicasForMetricsServer {
		return retryFunc(fmt.Sprintf("Number of replicas of metrics server (%d) does not match expected value (%d).", len(pods.Items), nbReplicasForMetricsServer))
	}
	postUrls := make(map[string]string, nbReplicasForMetricsServer)
	// Iterate through the pods and send HTTP requests
	for nbPod, pod := range pods.Items {
		if pod.Status.Phase != v1.PodRunning || pod.Status.PodIP == "" {
			return retryFunc(fmt.Sprintf("Pod %s was expected to be running and to have an IP.", pod.Name))
		}
		url := fmt.Sprintf("http://%s:8080/api/value/%d", pod.Status.PodIP, individualMetrics[nbPod])
		postUrls[pod.Name] = url
	}
	return postUrls
}

func updateAllMetricsServerReplicas(t *testing.T, kc *kubernetes.Clientset, data templateData, metricValue int, nbReplicasForMetricsServer int) {
	for targetPodName, urlToPost := range getUpdateUrlsForAllMetricAllMetricsServerReplicas(t, kc, metricValue, nbReplicasForMetricsServer, 0) {
		if urlToPost == "" {
			t.Fatalf("target pod %s should have non emoty url but got one", targetPodName)
			return
		}
		data.TargetPodName = targetPodName
		data.UpdateMetricURL = urlToPost
		KubectlReplaceWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)
	}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	updateAllMetricsServerReplicas(t, kc, data, 10, data.NbReplicasForMetricsServer)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	updateAllMetricsServerReplicas(t, kc, data, 50, data.NbReplicasForMetricsServer)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")
	updateAllMetricsServerReplicas(t, kc, data, 4, data.NbReplicasForMetricsServer)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func getTemplateData(nbReplicasForMetricsServer int) (templateData, []Template) {
	var aggregateFromKubeServiceEndpoints bool
	if nbReplicasForMetricsServer > 1 {
		// we need to configure metrics api scaler in Kube service aggregation mode to make sure aggregation from all service endpoints behind this service occurs
		aggregateFromKubeServiceEndpoints = true
	}
	return templateData{
			AggregateFromKubeServiceEndpoints: aggregateFromKubeServiceEndpoints,
			TestNamespace:                     testNamespace,
			DeploymentName:                    deploymentName,
			MetricsServerDeploymentName:       metricsServerDeploymentName,
			ServciceName:                      servciceName,
			TriggerAuthName:                   triggerAuthName,
			ScaledObjectName:                  scaledObjectName,
			SecretName:                        secretName,
			MetricsServerEndpoint:             metricsServerEndpoint,
			MinReplicaCount:                   fmt.Sprintf("%v", minReplicaCount),
			MaxReplicaCount:                   fmt.Sprintf("%v", maxReplicaCount),
			NbReplicasForMetricsServer:        nbReplicasForMetricsServer,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "metricsServerdeploymentTemplate", Config: metricsServerdeploymentTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
