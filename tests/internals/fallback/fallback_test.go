//go:build e2e
// +build e2e

package fallback_test

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testMetricName = "some_metric_name"
	testSOlabel    = "test-scaled-object"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

var (
	namespace       						= fmt.Sprintf("%s-ns", testMetricName)
	deploymentName							= fmt.Sprintf("%s-deployment", testMetricName)
	scaledObjectName    				= fmt.Sprintf("%s-so", testMetricName)
	serviceName 								= fmt.Sprintf("%s-service",testMetricsName)
	metricsServerDeploymentName = fmt.Sprintf("%s-metrics-server", testMetricName)
	metricsServerEndpoint				= fmt.Sprintf("http://%s.%s.svc.cluster.local:8080/api/value",serviceName,namespace)
	minReplicas        			 		= 0
	maxReplicas        			 		= 3
	defaultFallback    			 		= 2
	defaultReplicas 						= 1
)

type templateData struct {
	Namespace       	string
	DeploymentName  	string
	ScaledObject    	string
	ServiceName  					string
	metricsServerDeploymentName string
	MetricsServerEndpoint string
	MinReplicas 					string
	MaxReplicas 					string
	DefaultReplicas     	int
	DefaultFallback     	int
	MetricValue 					int
}

const (
	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
	name: {{.DeploymentName}}
	namespace: {{.Namespace}}
	labels:
		app: {{.DeploymentName}}
spec:
	replicas: 1
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
				image: nginxinc/nginx-unprivileged
				ports:
				-	containerPort: 80
`

	metricsServerDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.MetricsServerDeploymentName}}
  namespace: {{.Namespace}}
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

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
	name: {{.ScaledObject}}
	namespace: {{.Namespace}}
	labels:
		app: {{.DeploymentName}}
spec:
	scaleTargetRef:
		name: {{.DeploymentName}}
	minReplicaCount: {{.minReplicas}}
	maxReplicaCount: {{.maxReplicas}}
	cooldownPeriod: 1
	triggers:
	- type: metrics-api
		metadata:
			targetValue: "5"
			activationTargetValue: "10"
			url: "{{.MetricsServerEndpoint}}"
`

	updateMetricsTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
	name: update-ms-value
	namespace: {{.Namespace}}
spec:
	template:
		spec:
			containers:
			- name: job-curl
				image: curlimages/curl
				imagePullPolicy: Always
				command: ["curl", "-X", "POST", "{{.MetricsServerEndpoint}}/{{.MetricValue}}"]
			restartPolicy: Never
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
)


func TestScalerDef(t *testing.T){
	t.Log("--- setting up TestScalerDef---")
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t,kc,namespace,data,templates)

	assert.True(t,WaitForDeploymentReplicaReadyCount(t,kc,deploymentName,namespace,minReplicas,180,3),
		"replica count should be %d after 3 minutes",minReplicas)

	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func TestFallback(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
}

func getTemplateData() (templateData,[]Template) {
	return templateData{
		Namespace: Namespace
		DeploymentName: deploy
		ScaledObject: scaledObjectName
		DefaultReplicas: defaultReplicas
		DefaultFallback: defaultFallback
	}, []Template{
		{Name: "deploymentTemplate", Config: deploymentTemplate},
		{Name: "metricsServerDeploymentTemplate", Config: metricsServerDeploymentTemplate},
		{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		{Name: "serviceTemplate", Config: serviceTemplate}
	}

}
