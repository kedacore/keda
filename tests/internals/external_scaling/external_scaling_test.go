//go:build e2e
// +build e2e

package external_scaling_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "external-scaling-test"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

var (
	namespace                   = fmt.Sprintf("%s-ns", testName)
	deploymentName              = fmt.Sprintf("%s-deployment", testName)
	metricsServerDeploymentName = fmt.Sprintf("%s-metrics-server", testName)
	serviceName                 = fmt.Sprintf("%s-service", testName)
	triggerAuthName             = fmt.Sprintf("%s-ta", testName)
	scaledObjectName            = fmt.Sprintf("%s-so", testName)
	secretName                  = fmt.Sprintf("%s-secret", testName)
	metricsServerEndpoint       = fmt.Sprintf("http://%s.%s.svc.cluster.local:8080/api/value", serviceName, namespace)

	serverAvgName          = "server-avg"
	serverAddName          = "server-add"
	serviceExternalAvgName = fmt.Sprintf("%s-%s-service", testName, serverAvgName)
	serviceExternalAddName = fmt.Sprintf("%s-%s-service", testName, serverAddName)
	podExternalAvgName     = fmt.Sprintf("%s-pod", serverAvgName)
	podExternalAddname     = fmt.Sprintf("%s-pod", serverAddName)
	targetAvgPort          = 50051
	targetAddPort          = 50052
)

type templateData struct {
	TestNamespace               string
	DeploymentName              string
	ScaledObject                string
	TriggerAuthName             string
	SecretName                  string
	ServiceName                 string
	MetricsServerDeploymentName string
	MetricsServerEndpoint       string
	MetricValue                 int

	ServiceExternalAvgName string
	ServiceExternalAddName string
	PodExternalAvgName     string
	PodExternalAddname     string
	ExternalAvgPort        int
	ExternalAddPort        int
	ExternalAvgIP          string
	ExternalAddIP          string
	ServerAvgName          string
	ServerAddName          string
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
        image: nginxinc/nginx-unprivileged
        ports:
        - containerPort: 80
`

	// for metrics-api trigger
	metricsServerDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.MetricsServerDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.MetricsServerDeploymentName}}
spec:
  replicas: 1
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

	// for SO with 2 external scaling grpc servers
	soExternalCalculatorTwoTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObject}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  advanced:
    complexScalingLogic:
      target: '2'
      externalCalculator:
        - name: first_avg
          url: {{.ExternalAvgIP}}:{{.ExternalAvgPort}}
          timeout: '20'
        - name: second_add
          url: {{.ExternalAddIP}}:{{.ExternalAddPort}}
          timeout: '20'
  pollingInterval: 5
  cooldownPeriod: 5
  minReplicaCount: 0
  maxReplicaCount: 10
  triggers:
  - type: metrics-api
    name: metrics_api
    metadata:
      targetValue: "2"
      url: "{{.MetricsServerEndpoint}}"
      valueLocation: 'value'
      method: "query"
    authenticationRef:
      name: {{.TriggerAuthName}}
  - type: kubernetes-workload
    name: kw_trig
    metadata:
      podSelector: pod=workload-test
      value: '1'
`

	soFormulaTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObject}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  advanced:
    complexScalingLogic:
      target: '2'
      formula: metrics_api + kw_trig
  pollingInterval: 5
  cooldownPeriod: 5
  minReplicaCount: 0
  maxReplicaCount: 10
  triggers:
  - type: metrics-api
    name: metrics_api
    metadata:
      targetValue: "2"
      url: "{{.MetricsServerEndpoint}}"
      valueLocation: 'value'
      method: "query"
    authenticationRef:
      name: {{.TriggerAuthName}}
  - type: kubernetes-workload
    name: kw_trig
    metadata:
      podSelector: pod=workload-test
      value: '1'
`

	workloadDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: depl-workload-base
  namespace: {{.TestNamespace}}
  labels:
    deploy: workload-test
spec:
  replicas: 0
  selector:
    matchLabels:
      pod: workload-test
  template:
    metadata:
      labels:
        pod: workload-test
    spec:
      containers:
        - name: nginx
          image: 'nginxinc/nginx-unprivileged'`

	updateMetricsTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: update-ms-value
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 0
  backoffLimit: 4
  template:
    spec:
      containers:
      - name: job-curl
        image: curlimages/curl
        imagePullPolicy: Always
        command: ["curl", "-X", "POST", "{{.MetricsServerEndpoint}}/{{.MetricValue}}"]
      restartPolicy: OnFailure
`
	// image contains python grpc server that creates average from given metrics
	podExternalAvgTemplate = `
apiVersion: v1
kind: Pod
metadata:
  name: {{.ServerAvgName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.ServerAvgName}}
spec:
  containers:
  - name: server-avg-container
    image: docker.io/4141gauron3268/python-proto-server-avg
`

	// image contains python grpc server that adds 2 to the metric value
	podExternalAddTemplate = `
apiVersion: v1
kind: Pod
metadata:
  name: {{.ServerAddName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.ServerAddName}}
spec:
  containers:
  - name: server-add-container
    image: docker.io/4141gauron3268/python-proto-server-add
`

	serviceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.ServiceName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    app: {{.MetricsServerDeploymentName}}
  ports:
  - port: 8080
    targetPort: 8080
`

	serviceExternalAvgTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.ServiceExternalAvgName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    app: {{.ServerAvgName}}
  ports:
    - port: {{.ExternalAvgPort}}
      targetPort: {{.ExternalAvgPort}}
`

	serviceExternalAddTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.ServiceExternalAddName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    app: {{.ServerAddName}}
  ports:
  - port: {{.ExternalAddPort}}
    targetPort: {{.ExternalAddPort}}
`
)

func TestExternalScaling(t *testing.T) {
	// setup
	t.Log("-- setting up ---")
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, namespace, data, templates)

	// check grpc server pods are running
	assert.True(t, waitForPodsReadyInNamespace(t, kc, namespace, []string{serverAddName, serverAvgName}, 6, 10),
		fmt.Sprintf("pods '%v' should be ready after 1 minute", []string{serverAddName, serverAvgName}))

	ADDIP, err := ExecuteCommand(fmt.Sprintf("kubectl get service %s -o custom-columns=IP:spec.clusterIP -n %s", serviceExternalAddName, namespace))
	assert.NoErrorf(t, err, "cannot get service ADD - %s", err)

	AVGIP, err := ExecuteCommand(fmt.Sprintf("kubectl get service %s -o custom-columns=IP:spec.clusterIP -n %s", serviceExternalAvgName, namespace))
	assert.NoErrorf(t, err, "cannot get service AVG - %s", err)

	data.ExternalAvgIP = strings.Split(string(AVGIP), "\n")[1]
	data.ExternalAddIP = strings.Split(string(ADDIP), "\n")[1]
	testTwoExternalCalculators(t, kc, data)
	testComplexFormula(t, kc, data)
	// testFormulaAndEC()

	DeleteKubernetesResources(t, namespace, data, templates)
}

func testTwoExternalCalculators(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("-- testTwoExternalCalculators ---")
	KubectlApplyWithTemplate(t, data, "soExternalCalculatorTwoTemplate", soExternalCalculatorTwoTemplate)

	// metrics calculation: avg-> 3 + 3 = 6 / 2 = 3; add-> 3 + 2 = 5; target=2 ==> 3
	data.MetricValue = 3
	KubectlApplyWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)
	_, err := ExecuteCommand(fmt.Sprintf("kubectl scale deployment/depl-workload-base --replicas=3 -n %s", namespace))
	assert.NoErrorf(t, err, "cannot scale workload deployment - %s", err)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, "depl-workload-base", namespace, 3, 6, 10),
		"replica count should be %d after 1 minute", 3)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 3, 12, 10),
		"replica count should be %d after 2 minutes", 3)

	// metrics calculation: avg-> 5 + 5 = 10 / 2 = 5; add-> 5 + 2 = 7; target=2 ==> 4
	data.MetricValue = 5
	KubectlApplyWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)
	_, err = ExecuteCommand(fmt.Sprintf("kubectl scale deployment/depl-workload-base --replicas=5 -n %s", namespace))
	assert.NoErrorf(t, err, "cannot scale workload deployment - %s", err)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, "depl-workload-base", namespace, 5, 6, 10),
		"replica count should be %d after 1 minute", 5)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 4, 12, 10),
		"replica count should be %d after 2 minutes", 4)
}

func testComplexFormula(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testComplexFormula ---")
	// formula simply adds 2 metrics together
	data.MetricValue = 5
	KubectlApplyWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	KubectlApplyWithTemplate(t, data, "soFormulaTemplate", soFormulaTemplate)
	_, err := ExecuteCommand(fmt.Sprintf("kubectl scale deployment/depl-workload-base --replicas=5 -n %s", namespace))
	assert.NoErrorf(t, err, "cannot scale workload deployment - %s", err)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, "depl-workload-base", namespace, 5, 6, 10),
		"replica count should be %d after 1 minute", 5)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 5, 12, 10),
		"replica count should be %d after 2 minutes", 5)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:               namespace,
			DeploymentName:              deploymentName,
			MetricsServerDeploymentName: metricsServerDeploymentName,
			ServiceName:                 serviceName,
			TriggerAuthName:             triggerAuthName,
			ScaledObject:                scaledObjectName,
			SecretName:                  secretName,
			MetricsServerEndpoint:       metricsServerEndpoint,
			MetricValue:                 0,

			ServiceExternalAvgName: serviceExternalAvgName,
			ServiceExternalAddName: serviceExternalAddName,
			PodExternalAvgName:     podExternalAvgName,
			PodExternalAddname:     podExternalAddname,
			ExternalAvgPort:        targetAvgPort,
			ExternalAddPort:        targetAddPort,
			ServerAvgName:          serverAvgName,
			ServerAddName:          serverAddName,
		}, []Template{
			// basic: scaled deployment, SO, metrics-api trigger server & authentication
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "metricsServerDeploymentTemplate", Config: metricsServerDeploymentTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			// pods and services for grpc servers
			{Name: "podExternalAvgTemplate", Config: podExternalAvgTemplate},
			{Name: "podExternalAddTemplate", Config: podExternalAddTemplate},
			{Name: "serviceExternalAvgTemplate", Config: serviceExternalAvgTemplate},
			{Name: "serviceExternalAddTemplate", Config: serviceExternalAddTemplate},
			// workload base
			{Name: "workloadDeploymentTemplate", Config: workloadDeploymentTemplate},
		}
}

// Waits until deployment count hits target or number of iterations are done.
func waitForPodsReadyInNamespace(t *testing.T, kc *kubernetes.Clientset, namespace string,
	names []string, iterations, intervalSeconds int) bool {
	for i := 0; i < iterations; i++ {
		runningCount := 0
		pods, _ := kc.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
		namedPods := []corev1.Pod{}

		// count pods by name
		for _, pod := range pods.Items {
			if contains(names, pod.Name) {
				namedPods = append(namedPods, pod)
			}
		}

		for _, readyPod := range namedPods {
			if readyPod.Status.Phase != corev1.PodRunning {
				break
			}
			runningCount++
		}

		t.Logf("Waiting for pods '%v' to be ready. Namespace - %s, Current  - %d, Target - %d",
			names, namespace, runningCount, len(namedPods))

		if runningCount == len(namedPods) {
			return true
		}
		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}

	return false
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
