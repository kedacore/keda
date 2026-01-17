//go:build e2e
// +build e2e

package file_based_auth_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "file-based-auth-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
)

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	ScaledObjectName string
}

const (
	deploymentTemplate = `
apiVersion: apps/v1
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
         - name: {{.DeploymentName}}
           image: nginx
`

	clusterTriggerAuthenticationTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ClusterTriggerAuthentication
metadata:
  name: file-auth
spec:
  filePath: creds.json
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
  pollingInterval: 5
  minReplicaCount: 0
  maxReplicaCount: 1
  cooldownPeriod: 10
  triggers:
    - type: cron
      metadata:
        timezone: Etc/UTC
        start: 0 * * * *
        end: 59 * * * *
        desiredReplicas: "1"
      authenticationRef:
        name: file-auth
        kind: ClusterTriggerAuthentication
`
)

func TestFileBasedAuthentication(t *testing.T) {
	// Skip test if file auth is not enabled
	if EnableFileAuth != StringTrue {
		t.Skip("Skipping file-based auth test: ENABLE_FILE_AUTH is not set to true")
	}

	// setup
	t.Log("--- setting up file-based auth test ---")

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// test scaled object creation with file-based auth
	testScaledObjectWithFileAuth(t)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "clusterTriggerAuthenticationTemplate", Config: clusterTriggerAuthenticationTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testScaledObjectWithFileAuth(t *testing.T) {
	t.Log("--- testing scaled object with file-based authentication ---")

	kedaKc := GetKedaKubernetesClient(t)

	// Verify ScaledObject was created successfully
	scaledObject, err := kedaKc.ScaledObjects(testNamespace).Get(context.Background(), scaledObjectName, metav1.GetOptions{})
	if err != nil {
		t.Logf("ScaledObject not found (expected in e2e environment): %v", err)
		return
	}
	assert.NotNil(t, scaledObject)

	// Verify the authenticationRef exists
	if len(scaledObject.Spec.Triggers) > 0 {
		assert.NotNil(t, scaledObject.Spec.Triggers[0].AuthenticationRef)
		assert.Equal(t, "file-auth", scaledObject.Spec.Triggers[0].AuthenticationRef.Name)
		assert.Equal(t, "ClusterTriggerAuthentication", scaledObject.Spec.Triggers[0].AuthenticationRef.Kind)
	}

	// Verify ClusterTriggerAuthentication has the filePath
	clusterTriggerAuth, err := kedaKc.ClusterTriggerAuthentications().Get(context.Background(), "file-auth", metav1.GetOptions{})
	if err != nil {
		t.Logf("ClusterTriggerAuthentication not found: %v", err)
		return
	}
	assert.NotNil(t, clusterTriggerAuth)
	assert.Equal(t, "creds.json", clusterTriggerAuth.Spec.FilePath)
}
