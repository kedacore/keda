//go:build e2e
// +build e2e

package trigger_update_so_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	cloudevent_types "github.com/kedacore/keda/v2/apis/eventing/v1alpha1"
	message "github.com/kedacore/keda/v2/pkg/common/message"
	eventreason "github.com/kedacore/keda/v2/pkg/eventreason"
	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "eventsource-test"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

var (
	namespace                       = fmt.Sprintf("%s-ns", testName)
	scaledObjectName                = fmt.Sprintf("%s-so", testName)
	deploymentName                  = fmt.Sprintf("%s-d", testName)
	clientName                      = fmt.Sprintf("%s-client", testName)
	cloudeventSourceName            = fmt.Sprintf("%s-ce", testName)
	cloudeventSourceErrName         = fmt.Sprintf("%s-ce-err", testName)
	cloudeventSourceErrName2        = fmt.Sprintf("%s-ce-err2", testName)
	clusterCloudeventSourceName     = fmt.Sprintf("%s-cce", testName)
	clusterCloudeventSourceErrName  = fmt.Sprintf("%s-cce-err", testName)
	clusterCloudeventSourceErrName2 = fmt.Sprintf("%s-cce-err2", testName)
	cloudEventHTTPReceiverName      = fmt.Sprintf("%s-cloudevent-http-receiver", testName)
	cloudEventHTTPServiceName       = fmt.Sprintf("%s-cloudevent-http-service", testName)
	cloudEventHTTPServiceURL        = fmt.Sprintf("http://%s.%s.svc.cluster.local:8899", cloudEventHTTPServiceName, namespace)
	clusterName                     = "test-cluster"
	expectedSubject                 = fmt.Sprintf("/%s/%s/scaledobject/%s", clusterName, namespace, scaledObjectName)
	expectedSource                  = fmt.Sprintf("/%s/keda/keda", clusterName)
	lastCloudEventTime              = time.Now()
)

type templateData struct {
	TestNamespace                   string
	ScaledObject                    string
	DeploymentName                  string
	ClientName                      string
	CloudEventSourceName            string
	CloudeventSourceErrName         string
	CloudeventSourceErrName2        string
	ClusterCloudEventSourceName     string
	ClusterCloudeventSourceErrName  string
	ClusterCloudeventSourceErrName2 string
	CloudEventHTTPReceiverName      string
	CloudEventHTTPServiceName       string
	CloudEventHTTPServiceURL        string
	ClusterName                     string
}

const (
	cloudEventHTTPServiceTemplate = `
  apiVersion: v1
  kind: Service
  metadata:
    name: {{.CloudEventHTTPServiceName}}
    namespace: {{.TestNamespace}}
  spec:
    type: ClusterIP
    ports:
    - protocol: TCP
      port: 8899
      targetPort: 8899
    selector:
      app: {{.CloudEventHTTPReceiverName}}
  `

	cloudEventHTTPReceiverTemplate = `
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    labels:
      deploy: {{.CloudEventHTTPReceiverName}}
    name: {{.CloudEventHTTPReceiverName}}
    namespace: {{.TestNamespace}}
  spec:
    selector:
      matchLabels:
        app: {{.CloudEventHTTPReceiverName}}
    replicas: 1
    template:
      metadata:
        labels:
          app: {{.CloudEventHTTPReceiverName}}
      spec:
        containers:
        - name: httpreceiver
          image: ghcr.io/kedacore/tests-cloudevents-http:latest
          ports:
          - containerPort: 8899
          resources:
            requests:
              cpu: "200m"
            limits:
              cpu: "500m"
  `

	scaledObjectErrTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObject}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: test
  triggers:
    - type: kubernetes-workload
      metadata:
        podSelector: 'pod=testWorkloadDeploymentName'
        value: '1'
        activationValue: '3'
`

	clientTemplate = `
apiVersion: v1
kind: Pod
metadata:
  name: {{.ClientName}}
  namespace: {{.TestNamespace}}
spec:
  containers:
  - name: {{.ClientName}}
    image: curlimages/curl
    command:
      - sh
      - -c
      - "exec tail -f /dev/null"`

	cloudEventSourceTemplate = `
    apiVersion: eventing.keda.sh/v1alpha1
    kind: CloudEventSource
    metadata:
      name: {{.CloudEventSourceName}}
      namespace: {{.TestNamespace}}
    spec:
      clusterName: {{.ClusterName}}
      destination:
        http:
          uri: {{.CloudEventHTTPServiceURL}}
    `

	cloudEventSourceWithExcludeTemplate = `
      apiVersion: eventing.keda.sh/v1alpha1
      kind: CloudEventSource
      metadata:
        name: {{.CloudEventSourceName}}
        namespace: {{.TestNamespace}}
      spec:
        clusterName: {{.ClusterName}}
        destination:
          http:
            uri: {{.CloudEventHTTPServiceURL}}
        eventSubscription:
          excludedEventTypes:
          - keda.scaledobject.failed.v1
      `

	cloudEventSourceWithIncludeTemplate = `
      apiVersion: eventing.keda.sh/v1alpha1
      kind: CloudEventSource
      metadata:
        name: {{.CloudEventSourceName}}
        namespace: {{.TestNamespace}}
      spec:
        clusterName: {{.ClusterName}}
        destination:
          http:
            uri: {{.CloudEventHTTPServiceURL}}
        eventSubscription:
          includedEventTypes:
          - keda.scaledobject.failed.v1
      `

	cloudEventSourceWithErrTypeTemplate = `
    apiVersion: eventing.keda.sh/v1alpha1
    kind: CloudEventSource
    metadata:
      name: {{.CloudeventSourceErrName}}
      namespace: {{.TestNamespace}}
    spec:
      clusterName: {{.ClusterName}}
      destination:
        http:
          uri: {{.CloudEventHTTPServiceURL}}
      eventSubscription:
        includedEventTypes:
        - keda.scaledobject.failed.v2
    `

	cloudEventSourceWithErrTypeTemplate2 = `
    apiVersion: eventing.keda.sh/v1alpha1
    kind: CloudEventSource
    metadata:
      name: {{.CloudeventSourceErrName2}}
      namespace: {{.TestNamespace}}
    spec:
      clusterName: {{.ClusterName}}
      destination:
        http:
          uri: {{.CloudEventHTTPServiceURL}}
      eventSubscription:
        includedEventTypes:
        - keda.scaledobject.failed.v1
        excludedEventTypes:
        - keda.scaledobject.failed.v1
    `

	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    deploy: {{.DeploymentName}}
spec:
  replicas: 1
  selector:
    matchLabels:
      pod: {{.DeploymentName}}
  template:
    metadata:
      labels:
        pod: {{.DeploymentName}}
    spec:
      containers:
        - name: nginx
          image: 'ghcr.io/nginx/nginx-unprivileged:1.26'
`

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObject}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod: 5
  minReplicaCount: 1
  maxReplicaCount: 10
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 15
  triggers:
  - type: cron
    metadata:
      timezone: Etc/UTC
      start: 3 * * * *
      end: 5 * * * *
      desiredReplicas: '4'
`

	clusterCloudEventSourceTemplate = `
    apiVersion: eventing.keda.sh/v1alpha1
    kind: ClusterCloudEventSource
    metadata:
      name: {{.ClusterCloudEventSourceName}}
    spec:
      clusterName: {{.ClusterName}}
      destination:
        http:
          uri: {{.CloudEventHTTPServiceURL}}
    `

	clusterCloudEventSourceWithExcludeTemplate = `
      apiVersion: eventing.keda.sh/v1alpha1
      kind: ClusterCloudEventSource
      metadata:
        name: {{.ClusterCloudEventSourceName}}
      spec:
        clusterName: {{.ClusterName}}
        destination:
          http:
            uri: {{.CloudEventHTTPServiceURL}}
        eventSubscription:
          excludedEventTypes:
          - keda.scaledobject.failed.v1
      `

	clusterCloudEventSourceWithIncludeTemplate = `
      apiVersion: eventing.keda.sh/v1alpha1
      kind: ClusterCloudEventSource
      metadata:
        name: {{.ClusterCloudEventSourceName}}
      spec:
        clusterName: {{.ClusterName}}
        destination:
          http:
            uri: {{.CloudEventHTTPServiceURL}}
        eventSubscription:
          includedEventTypes:
          - keda.scaledobject.failed.v1
      `

	clusterCloudEventSourceWithErrTypeTemplate = `
    apiVersion: eventing.keda.sh/v1alpha1
    kind: ClusterCloudEventSource
    metadata:
      name: {{.ClusterCloudeventSourceErrName}}
    spec:
      clusterName: {{.ClusterName}}
      destination:
        http:
          uri: {{.CloudEventHTTPServiceURL}}
      eventSubscription:
        includedEventTypes:
        - keda.scaledobject.failed.v2
    `

	clusterCloudEventSourceWithErrTypeTemplate2 = `
    apiVersion: eventing.keda.sh/v1alpha1
    kind: ClusterCloudEventSource
    metadata:
      name: {{.ClusterCloudeventSourceErrName2}}
    spec:
      clusterName: {{.ClusterName}}
      destination:
        http:
          uri: {{.CloudEventHTTPServiceURL}}
      eventSubscription:
        includedEventTypes:
        - keda.scaledobject.failed.v1
        excludedEventTypes:
        - keda.scaledobject.failed.v1
    `
)

func TestScaledObjectGeneral(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		t.Log("--- cleaning up ---")
		DeleteKubernetesResources(t, namespace, data, templates)
	})
	CreateKubernetesResources(t, kc, namespace, data, templates)

	assert.True(t, WaitForAllPodRunningInNamespace(t, kc, namespace, 5, 20), "all pods should be running")

	testErrEventSourceEmitValue(t, kc, data, true)
	testEventSourceEmitValue(t, kc, data)
	testErrEventSourceExcludeValue(t, kc, data, true)
	testErrEventSourceIncludeValue(t, kc, data, true)
	testErrEventSourceCreation(t, kc, data, true)

	testErrEventSourceEmitValue(t, kc, data, false)
	testErrEventSourceExcludeValue(t, kc, data, false)
	testErrEventSourceIncludeValue(t, kc, data, false)
	testErrEventSourceCreation(t, kc, data, false)
}

// tests error events emitted
func testErrEventSourceEmitValue(t *testing.T, kc *kubernetes.Clientset, data templateData, isClusterScope bool) {
	ceTemplate := ""
	if isClusterScope {
		ceTemplate = clusterCloudEventSourceTemplate
	} else {
		ceTemplate = cloudEventSourceTemplate
	}

	t.Logf("--- test emitting eventsource about scaledobject err --- [isClusterScope: %t]", isClusterScope)
	KubectlApplyWithTemplate(t, data, "cloudEventSourceTemplate", ceTemplate)
	defer KubectlDeleteWithTemplate(t, data, "cloudEventSourceTemplate", ceTemplate)

	WatchForEventAfterTrigger(
		t,
		kc,
		namespace,
		scaledObjectName,
		"ScaledObject",
		eventreason.ScaledObjectCheckFailed,
		corev1.EventTypeWarning,
		[]string{message.ScaleTargetErrMsg, message.ScaleTargetNotFoundMsg},
		60*time.Second,
		func() error {
			KubectlApplyWithTemplate(t, data, "scaledObjectErrTemplate", scaledObjectErrTemplate)
			return nil
		},
	)

	// in order to satisfy lastCloudEventTime, we cannot defer this deletion because the ExecCommand would then retrieve a stale list of current events
	// i.e. more ScaledObjectCheckFailed events will happen between the ExecCommand and the deletion of the ScaledObject
	// hence, we have to delete it here after an arbitrary few seconds, in order to stop all events
	// then we will calculate lastCloudEventTime which will be updated with the ACTUAL last event without any surprises
	KubectlDeleteWithTemplate(t, data, "scaledObjectErrTemplate", scaledObjectErrTemplate)

	out, outErr, err := ExecCommandOnSpecificPodWithoutTTY(t, clientName, namespace, fmt.Sprintf("curl -s -X GET %s/getCloudEvent/%s", cloudEventHTTPServiceURL, "ScaledObjectCheckFailed"))
	assert.NotEmpty(t, out)
	assert.Empty(t, outErr)
	assert.NoError(t, err, "dont expect error requesting ")

	cloudEvents := []cloudevents.Event{}
	err = json.Unmarshal([]byte(out), &cloudEvents)

	assert.NoError(t, err, "dont expect error unmarshaling the cloudEvents")
	assert.Greater(t, len(cloudEvents), 0, "cloudEvents should have at least 1 item")

	foundEvents := []cloudevents.Event{}

	for _, cloudEvent := range cloudEvents {
		if cloudEvent.Subject() == expectedSubject {
			foundEvents = append(foundEvents, cloudEvent)
			data := map[string]string{}
			err := cloudEvent.DataAs(&data)
			t.Log("--- test emitting eventsource about scaledobject err ---", "message", data["message"])

			assert.NoError(t, err)
			assert.Condition(t, func() bool {
				if strings.Contains(data["message"], message.ScaleTargetErrMsg) || strings.Contains(data["message"], message.ScaleTargetNotFoundMsg) {
					return true
				}
				return false
			}, "get filtered event")

			assert.Equal(t, cloudEvent.Type(), string(cloudevent_types.ScaledObjectFailedType))
			assert.Equal(t, cloudEvent.Source(), expectedSource)
			assert.Equal(t, cloudEvent.DataContentType(), "application/json")

			if lastCloudEventTime.Before(cloudEvent.Time()) {
				lastCloudEventTime = cloudEvent.Time()
			}
		}
	}

	assert.NotEmpty(t, foundEvents)
}

func testEventSourceEmitValue(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- test emitting eventsource about scaledobject removed ---")
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlApplyWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
	defer KubectlDeleteWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)

	WatchForEventAfterTrigger(
		t,
		kc,
		namespace,
		scaledObjectName,
		"ScaledObject",
		eventreason.ScaledObjectDeleted,
		corev1.EventTypeWarning,
		[]string{message.ScaledObjectRemoved},
		60*time.Second,
		func() error {
			KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
			return nil
		},
	)

	out, outErr, err := ExecCommandOnSpecificPodWithoutTTY(t, clientName, namespace, fmt.Sprintf("curl -s -X GET %s/getCloudEvent/%s", cloudEventHTTPServiceURL, "ScaledObjectDeleted"))
	assert.NotEmpty(t, out)
	assert.Empty(t, outErr)
	assert.NoError(t, err, "dont expect error requesting ")

	cloudEvents := []cloudevents.Event{}
	err = json.Unmarshal([]byte(out), &cloudEvents)

	assert.NoError(t, err, "dont expect error unmarshaling the cloudEvents")
	assert.Greater(t, len(cloudEvents), 0, "cloudEvents should have at least 1 item")

	foundEvents := []cloudevents.Event{}

	for _, cloudEvent := range cloudEvents {
		if cloudEvent.Subject() == expectedSubject {
			foundEvents = append(foundEvents, cloudEvent)
			data := map[string]string{}
			err := cloudEvent.DataAs(&data)

			assert.NoError(t, err)
			assert.Equal(t, data["message"], message.ScaledObjectRemoved)
			assert.Equal(t, cloudEvent.Type(), string(cloudevent_types.ScaledObjectRemovedType))
			assert.Equal(t, cloudEvent.Source(), expectedSource)
			assert.Equal(t, cloudEvent.DataContentType(), "application/json")

			if lastCloudEventTime.Before(cloudEvent.Time()) {
				lastCloudEventTime = cloudEvent.Time()
			}
		}
	}

	assert.NotEmpty(t, foundEvents)
}

// tests error events not emitted by
func testErrEventSourceExcludeValue(t *testing.T, _ *kubernetes.Clientset, data templateData, isClusterScope bool) {
	t.Logf("--- test emitting eventsource about scaledobject err with exclude filter --- [isClusterScope: %t]", isClusterScope)

	ceTemplate := ""
	if isClusterScope {
		ceTemplate = clusterCloudEventSourceWithExcludeTemplate
	} else {
		ceTemplate = cloudEventSourceWithExcludeTemplate
	}

	KubectlApplyWithTemplate(t, data, "cloudEventSourceWithExcludeTemplate", ceTemplate)
	defer KubectlDeleteWithTemplate(t, data, "cloudEventSourceWithExcludeTemplate", ceTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectErrTemplate", scaledObjectErrTemplate)
	defer KubectlDeleteWithTemplate(t, data, "scaledObjectErrTemplate", scaledObjectErrTemplate)

	consistencyDuration := 30 * time.Second
	pollingInterval := 5 * time.Second

	t.Logf("Checking consistently every %v for %v that the excluded CloudEvent does not get emitted", pollingInterval, consistencyDuration)
	conditionFunc := func(ctx context.Context) (bool, error) {
		out, outErr, err := ExecCommandOnSpecificPodWithoutTTY(t, clientName, namespace, fmt.Sprintf("curl -s -X GET %s/getCloudEvent/%s", cloudEventHTTPServiceURL, "ScaledObjectCheckFailed"))

		if err != nil {
			return false, fmt.Errorf("command execution error: %w (stderr: %s)", err, outErr)
		}
		if outErr != "" {
			return false, fmt.Errorf("command execution error: %s", outErr)
		}
		if out == "" {
			return false, fmt.Errorf("command execution returned empty output")
		}

		var cloudEvents []cloudevents.Event
		err = json.Unmarshal([]byte(out), &cloudEvents)
		if err != nil {
			return false, fmt.Errorf("cannot unmarshal cloudevents JSON '%s': %w", out, err)
		}

		for _, cloudEvent := range cloudEvents {
			if cloudEvent.Subject() == expectedSubject &&
				cloudEvent.Type() == string(cloudevent_types.ScaledObjectFailedType) &&
				cloudEvent.Time().After(lastCloudEventTime) {
				t.Logf("found excluded event: subject=%q, type=%q, time=%s, lastCloudEventTime=%s", cloudEvent.Subject(), cloudEvent.Type(), cloudEvent.Time().Format(time.RFC3339), lastCloudEventTime.Format(time.RFC3339))
				return false, nil
			}
		}

		t.Log("no excluded event found... continuing to validate consistency")
		return true, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), consistencyDuration)
	defer cancel()

	err := KedaConsistently(ctx, conditionFunc, pollingInterval)
	assert.NoError(t, err, "KedaConsistently check failed: An excluded event was likely found, or a check error occurred.")
}

// tests error events in include filter
func testErrEventSourceIncludeValue(t *testing.T, kc *kubernetes.Clientset, data templateData, isClusterScope bool) {
	t.Logf("--- test emitting eventsource about scaledobject err with include filter --- [isClusterScope: %t]", isClusterScope)

	ceTemplate := ""
	if isClusterScope {
		ceTemplate = clusterCloudEventSourceWithIncludeTemplate
	} else {
		ceTemplate = cloudEventSourceWithIncludeTemplate
	}
	KubectlApplyWithTemplate(t, data, "cloudEventSourceWithIncludeTemplate", ceTemplate)
	defer KubectlDeleteWithTemplate(t, data, "cloudEventSourceWithIncludeTemplate", ceTemplate)
	defer KubectlDeleteWithTemplate(t, data, "scaledObjectErrTemplate", scaledObjectErrTemplate)

	WatchForEventAfterTrigger(
		t,
		kc,
		namespace,
		scaledObjectName,
		"ScaledObject",
		eventreason.ScaledObjectCheckFailed,
		corev1.EventTypeWarning,
		[]string{message.ScaleTargetErrMsg, message.ScaleTargetNotFoundMsg},
		60*time.Second,
		func() error {
			KubectlApplyWithTemplate(t, data, "scaledObjectErrTemplate", scaledObjectErrTemplate)
			return nil
		},
	)

	out, outErr, err := ExecCommandOnSpecificPodWithoutTTY(t, clientName, namespace, fmt.Sprintf("curl -s -X GET %s/getCloudEvent/%s", cloudEventHTTPServiceURL, "ScaledObjectCheckFailed"))
	assert.NotEmpty(t, out)
	assert.Empty(t, outErr)
	assert.NoError(t, err, "dont expect error requesting ")

	cloudEvents := []cloudevents.Event{}
	err = json.Unmarshal([]byte(out), &cloudEvents)
	assert.NoError(t, err, "dont expect error unmarshaling the cloudEvents")

	foundCloudEvents := []cloudevents.Event{}
	for _, cloudEvent := range cloudEvents {
		if cloudEvent.Subject() == expectedSubject &&
			cloudEvent.Time().After(lastCloudEventTime) &&
			cloudEvent.Type() == string(cloudevent_types.ScaledObjectFailedType) {
			foundCloudEvents = append(foundCloudEvents, cloudEvent)
		}
	}

	assert.NotEmpty(t, foundCloudEvents)
}

// tests error event type when creation
func testErrEventSourceCreation(t *testing.T, _ *kubernetes.Clientset, data templateData, isClusterScope bool) {
	t.Logf("--- test emitting eventsource about scaledobject err with include filter --- [isClusterScope: %t]", isClusterScope)

	ceErrTemplate := ""
	ceErrTemplate2 := ""
	if isClusterScope {
		ceErrTemplate = clusterCloudEventSourceWithErrTypeTemplate
		ceErrTemplate2 = clusterCloudEventSourceWithErrTypeTemplate2
	} else {
		ceErrTemplate = cloudEventSourceWithErrTypeTemplate
		ceErrTemplate2 = cloudEventSourceWithErrTypeTemplate2
	}

	err := KubectlApplyWithErrors(t, data, "cloudEventSourceWithErrTypeTemplate", ceErrTemplate)
	if isClusterScope {
		assert.ErrorContains(t, err, `The ClusterCloudEventSource "eventsource-test-cce-err" is invalid:`)
	} else {
		assert.ErrorContains(t, err, `The CloudEventSource "eventsource-test-ce-err" is invalid:`)
	}

	err = KubectlApplyWithErrors(t, data, "cloudEventSourceWithErrTypeTemplate2", ceErrTemplate2)
	assert.ErrorContains(t, err, `setting included types and excluded types at the same time is not supported`)
}

// help function to load template data
func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:                   namespace,
			ScaledObject:                    scaledObjectName,
			DeploymentName:                  deploymentName,
			ClientName:                      clientName,
			CloudEventSourceName:            cloudeventSourceName,
			CloudeventSourceErrName:         cloudeventSourceErrName,
			CloudeventSourceErrName2:        cloudeventSourceErrName2,
			ClusterCloudEventSourceName:     clusterCloudeventSourceName,
			ClusterCloudeventSourceErrName:  clusterCloudeventSourceErrName,
			ClusterCloudeventSourceErrName2: clusterCloudeventSourceErrName2,
			CloudEventHTTPReceiverName:      cloudEventHTTPReceiverName,
			CloudEventHTTPServiceName:       cloudEventHTTPServiceName,
			CloudEventHTTPServiceURL:        cloudEventHTTPServiceURL,
			ClusterName:                     clusterName,
		}, []Template{
			{Name: "cloudEventHTTPReceiverTemplate", Config: cloudEventHTTPReceiverTemplate},
			{Name: "cloudEventHTTPServiceTemplate", Config: cloudEventHTTPServiceTemplate},
			{Name: "clientTemplate", Config: clientTemplate},
		}
}
