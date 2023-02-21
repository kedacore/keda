//go:build e2e
// +build e2e

package so_trigger_updates

import (
	// "context"
	"fmt"
	// "regexp"
	// "strconv"
	"testing"
	// "time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "so-trigger-update-test"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

var (
	namespace                   = fmt.Sprintf("%s-ns", testName)
	deploymentName              = fmt.Sprintf("%s-deployment", testName)
	workloadDeploymentName      = "workload-deployment"
	scaledObjectName            = fmt.Sprintf("%s-so", testName)
	secretName                  = fmt.Sprintf("%s-secret", testName)
	triggerAuthName             = fmt.Sprintf("%s-ta", testName)
	serviceName                 = fmt.Sprintf("%s-service", testName)
	metricsServerDeploymentName = fmt.Sprintf("%s-metrics-server", testName)
	metricsServerEndpoint       = fmt.Sprintf("http://%s.%s.svc.cluster.local:8080/api/value", serviceName, namespace)
	minReplicas                 = 0
	midReplicas                 = 3
	maxReplicas                 = 5
)

type templateData struct {
	TestNamespace               string
	DeploymentName              string
	ScaledObject                string
	TriggerAuthName             string
	ServiceName                 string
	SecretName                  string
	MinReplicas                 string
	MaxReplicas                 string
	MetricsServerDeploymentName string
	MetricsServerEndpoint       string
	WorkloadDeploymentName      string
	MetricValue                 int
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
    deploy: {{.DeploymentName}}
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  replicas: {{.MinReplicas}}
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
        resources:
          requests:
            cpu: "200m"
          limits:
            cpu: "500m"
`

	workloadDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.WorkloadDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    deploy: {{.WorkloadDeploymentName}}
spec:
  replicas: 0
  selector:
    matchLabels:
      pod: {{.WorkloadDeploymentName}}
  template:
    metadata:
      labels:
        pod: {{.WorkloadDeploymentName}}
    spec:
      containers:
        - name: nginx
          image: 'nginxinc/nginx-unprivileged'`

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

	scaledObjectTriggerTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObject}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 10
  pollingInterval: 10
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicas}}
  maxReplicaCount: {{.MaxReplicas}}
  cooldownPeriod: 1
  triggers:
  - type: metrics-api
    metadata:
      targetValue: "2"
      url: "{{.MetricsServerEndpoint}}"
      valueLocation: 'value'
      method: "query"
    authenticationRef:
      name: {{.TriggerAuthName}}
`

	scaledObjectTwoTriggerTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObject}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 10
  pollingInterval: 10
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicas}}
  maxReplicaCount: {{.MaxReplicas}}
  cooldownPeriod: 1
  triggers:
  - type: metrics-api
    metadata:
      targetValue: "2"
      url: "{{.MetricsServerEndpoint}}"
      valueLocation: 'value'
      method: "query"
    authenticationRef:
      name: {{.TriggerAuthName}}
  - type: kubernetes-workload
    metadata:
      podSelector: "pod={{.WorkloadDeploymentName}}"
      value: '1'
`

	scaledObjectThreeTriggerTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObject}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 10
  pollingInterval: 10
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicas}}
  maxReplicaCount: {{.MaxReplicas}}
  cooldownPeriod: 1
  triggers:
  - type: metrics-api
    metadata:
      targetValue: "2"
      url: "{{.MetricsServerEndpoint}}"
      valueLocation: 'value'
      method: "query"
    authenticationRef:
      name: {{.TriggerAuthName}}
  - type: kubernetes-workload
    metadata:
      podSelector: 'pod={{.WorkloadDeploymentName}}'
      value: '1'
  - type: cpu
    metricType: Utilization
    metadata:
      value: "50"
`

	updateMetricTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: update-ms-value
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 0
  template:
    spec:
      containers:
      - name: job-curl
        image: curlimages/curl
        imagePullPolicy: Always
        command: ["curl", "-X", "POST", "{{.MetricsServerEndpoint}}/{{.MetricValue}}"]
      restartPolicy: Never
`
)

func TestScaledObjectGeneral(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, namespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, minReplicas, 180, 3),
		"replica count should be %d after 3 minutes", minReplicas)

	KubectlApplyWithTemplate(t, data, "scaledObjectTriggerTemplate", scaledObjectTriggerTemplate)

	// single trigger
	testTargetValue(t, kc, data)          //one trigger target changes
	testTwoTriggers(t, kc, data)          //add trigger during active scaling
	testRemoveTrigger(t, kc, data)        //remove trigger during active scaling
	testThreeTriggersWithCPU(t, kc, data) //three triggers

	DeleteKubernetesResources(t, kc, namespace, data, templates)
}

// tests basic scaling with one trigger based on metrics
func testTargetValue(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- test target value 1 ---")
	KubectlApplyWithTemplate(t, data, "scaledObjectTriggerTemplate", scaledObjectTriggerTemplate)

	data.MetricValue = 1
	KubectlApplyWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 1, 180, 3),
		"replica count should be %d after 3 minutes", 1)

	t.Log("--- test target value 10 ---")
	data.MetricValue = 10
	KubectlApplyWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, maxReplicas, 180, 3),
		"replica count should be %d after 3 minutes", maxReplicas)

	t.Log("--- test target value 0 ---")
	data.MetricValue = 0
	KubectlApplyWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, minReplicas, 180, 3),
		"replica count should be %d after 3 minutes", minReplicas)
}

func testTwoTriggers(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- test two triggers ---")
	KubectlApplyWithTemplate(t, data, "scaledObjectTriggerTemplate", scaledObjectTriggerTemplate)

	data.MetricValue = 1
	KubectlApplyWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 1, 180, 3),
		"replica count should be %d after 3 minutes", 1)

	KubectlApplyWithTemplate(t, data, "scaledObjectTwoTriggerTemplate", scaledObjectTwoTriggerTemplate)
	// scale to max with k8s wl = second trigger
	KubernetesScaleDeployment(t, kc, workloadDeploymentName, int64(maxReplicas), namespace)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, maxReplicas, 180, 3),
		"replica count should be %d after 3 minutes", maxReplicas)
}

// testRemoveTrigger scales to max with kubernetes worload(second trigger),
// removes it, scales to 3 replicas based on metric value (first trigger)
func testRemoveTrigger(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- test remove trigger 2 -> 1 ---")
	KubectlApplyWithTemplate(t, data, "scaledObjectTwoTriggerTemplate", scaledObjectTwoTriggerTemplate)
	data.MetricValue = 5 // 3 replicas (midReplicas)
	KubectlApplyWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	KubernetesScaleDeployment(t, kc, workloadDeploymentName, int64(maxReplicas), namespace)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, maxReplicas, 180, 3),
		"replica count should be %d after 3 minutes", maxReplicas)

	// update SO -> remove k8s wl == second trigger
	KubectlApplyWithTemplate(t, data, "scaledObjectTriggerTemplate", scaledObjectTriggerTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, midReplicas, 180, 3),
		"replica count should be %d after 3 minutes", midReplicas)
}

// test 3 triggers scaling works including one cpu metric
func testThreeTriggersWithCPU(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- test 3 triggers (with cpu) ---")

	// update SO should scale up based on cpu
	KubectlApplyWithTemplate(t, data, "scaledObjectThreeTriggerTemplate", scaledObjectThreeTriggerTemplate)

	// scaling takes longer because of fetching of the cpu metrics (possibly increase iterations if needed)
	data.MetricValue = 10
	KubectlApplyWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, maxReplicas, 180, 3),
		"replica count should be %d after 3 minutes", maxReplicas)

	data.MetricValue = 0
	KubectlApplyWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)
	// expect min replica count to be 1 since no other load is present and cpu is given
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 1, 180, 3),
		"replica count should be %d after 3 minutes", 1)
}

// help function to load template data
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
			MinReplicas:                 fmt.Sprintf("%v", minReplicas),
			MaxReplicas:                 fmt.Sprintf("%v", maxReplicas),
			MetricValue:                 0,
			WorkloadDeploymentName:      workloadDeploymentName,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "workloadDeploymentTemplate", Config: workloadDeploymentTemplate},
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "metricsServerDeploymentTemplate", Config: metricsServerDeploymentTemplate},
			{Name: "scaledObjectTriggerTemplate", Config: scaledObjectTriggerTemplate},
		}
}

// // Waits until pod (that is matched via name_regex) count hits target or number of iterations are done.
// func waitForPodCountByNameRegex(t *testing.T, kc *kubernetes.Clientset, name_regex string, namespace string, targetCount int, iterations, intervalSeconds int) bool {
// 	for i := 0; i < iterations; i++ {
// 		count := 0
// 		pods, _ := kc.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})

// 		for _, pod := range pods.Items {
// 			if m, _ := regexp.MatchString(name_regex, pod.Name); m {
// 				count++
// 			}
// 		}
// 		t.Logf("Waiting for pods %s in namespace to exist. TestNamespace - %s, Current - %d, Target - %d",
// 			name_regex, namespace, count, targetCount)
// 		if count == targetCount {
// 			return true
// 		}
// 		time.Sleep(time.Duration(intervalSeconds) * time.Second)
// 	}
// 	return false
// }

///////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////////////////

// // scales deployment up, then creates kafka and updates SO with kafka trigger to
// // test for 2 triggers
// func testAddKafkaScaler(t *testing.T, kc *kubernetes.Clientset, data templateData) {
// 	t.Log("--- test - add kafka scaler ---")

// 	data.MetricValue = 5
// 	KubectlApplyWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

// 	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, midReplicas, 180, 3),
// 		"replica count should be %d after 3 minutes", midReplicas)

// 	// create kafka related stuff
// 	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, "strimzi-cluster-operator", namespace, 1, 180, 3),
// 		"replica count should be %d after 3 minutes", 1)

// 	t.Log("--- create kafka cluster ---")
// 	KubectlApplyWithTemplate(t, data, "kafkaClusterTemplate", kafkaClusterTemplate)
// 	_, err := ExecuteCommand(fmt.Sprintf("kubectl wait kafka/%s --for=condition=Ready --timeout=300s --namespace %s", kafkaName, namespace))
// 	assert.NoErrorf(t, err, "cannot execute command - %s", err)
// 	assert.True(t, waitForPodCountByNameRegex(t, kc, fmt.Sprintf("^%s-kafka", kafkaName), namespace, 3, 180, 3))
// 	assert.True(t, waitForPodCountByNameRegex(t, kc, fmt.Sprintf("^%s-zookeeper", kafkaName), namespace, 3, 180, 3))

// 	KubectlApplyWithTemplate(t, data, "kafkaTopicTemplate", kafkaTopicTemplate)

// 	// deploy consumer for kafka messages
// 	KubectlApplyWithTemplate(t, data, "kafkaDeploymentTemplate", kafkaDeploymentTemplate)
// 	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, "kafka-amqstreams-consumer", namespace, 1, 180, 3),
// 		"replica count should be %d after 3 minutes", 1)

// 	t.Log("--- add kafka trigger, send messages ---")

// 	// update ScaledObject to include kafka trigger
// 	KubectlApplyWithTemplate(t, data, "scaledObjectTwoTriggerTemplate", scaledObjectTwoTriggerTemplate)

// 	// send message load to kafka and expect deployment to go up to max
// 	KubectlApplyWithTemplate(t, data, "kafkaLoad", kafkaLoad)

// 	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, maxReplicas, 180, 3),
// 		"replica count should be %d after 3 minutes", maxReplicas)

// 	t.Log("--- remove all loads ---")

// 	// kafka load will finish and metric set to 0 -> expect 0 replicas
// 	data.MetricValue = 0
// 	KubectlApplyWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

// 	// wait for load to finish and count go down to 0 replicas
// 	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, minReplicas, 180, 3),
// 		"replica count should be %d after 3 minutes", minReplicas)
// }
