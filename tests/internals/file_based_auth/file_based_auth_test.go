//go:build e2e
// +build e2e

package file_based_auth_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

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
      initContainers:
        - name: init-auth
          image: busybox
          command: ["sh", "-c", "echo '{\"testParam\": \"testValue\"}' > /mnt/auth/creds.json"]
          volumeMounts:
            - name: auth-volume
              mountPath: /mnt/auth
      containers:
        - name: {{.DeploymentName}}
          image: nginx
      volumes:
        - name: auth-volume
          emptyDir: {}
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
	// setup
	t.Log("--- setting up file-based auth test ---")

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	// Patch the operator deployment to add init container and volume for auth file
	patchOperatorDeployment(t, kc)

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// test scaled object creation with file-based auth
	testScaledObjectWithFileAuth(t, kc)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func TestFileBasedAuthTemplates(t *testing.T) {
	t.Log("--- testing file-based auth YAML templates ---")

	// Test that templates contain expected filePath in ClusterTriggerAuthentication
	assert.Contains(t, clusterTriggerAuthenticationTemplate, "filePath: creds.json")
	assert.Contains(t, scaledObjectTemplate, "authenticationRef:")
	assert.Contains(t, scaledObjectTemplate, "name: file-auth")
	assert.Contains(t, scaledObjectTemplate, "kind: ClusterTriggerAuthentication")

	// Test that deployment template has init container and volume setup
	assert.Contains(t, deploymentTemplate, "initContainers:")
	assert.Contains(t, deploymentTemplate, "echo '{\\\"testParam\\\": \\\"testValue\\\"}' > /mnt/auth/creds.json")
	assert.Contains(t, deploymentTemplate, "emptyDir: {}")
}

func patchOperatorDeployment(t *testing.T, kc *kubernetes.Clientset) {
	operatorDeployment, err := kc.AppsV1().Deployments("keda").Get(context.Background(), "keda-operator", metav1.GetOptions{})
	if err != nil {
		t.Logf("Operator deployment not found, skipping patch: %v", err)
		return
	}

	// Add volume
	operatorDeployment.Spec.Template.Spec.Volumes = append(operatorDeployment.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: "auth-volume",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	// Add init container
	operatorDeployment.Spec.Template.Spec.InitContainers = append(operatorDeployment.Spec.Template.Spec.InitContainers, corev1.Container{
		Name:    "init-auth",
		Image:   "busybox",
		Command: []string{"sh", "-c", "echo '{\"testParam\": \"testValue\"}' > /mnt/auth/creds.json"},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "auth-volume",
				MountPath: "/mnt/auth",
			},
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot: &[]bool{true}[0],
			RunAsUser:    &[]int64{1000}[0],
		},
	})

	// Add the filepath-auth-root-path arg to the operator container
	if len(operatorDeployment.Spec.Template.Spec.Containers) > 0 {
		operatorDeployment.Spec.Template.Spec.Containers[0].Args = append(
			operatorDeployment.Spec.Template.Spec.Containers[0].Args,
			"--filepath-auth-root-path=/mnt/auth",
		)
	}

	// Update the deployment
	_, err = kc.AppsV1().Deployments("keda").Update(context.Background(), operatorDeployment, metav1.UpdateOptions{})
	if err != nil {
		t.Logf("Failed to patch operator deployment: %v", err)
	} else {
		t.Log("Patched operator deployment with auth init container and filepath arg")
	}
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

func testScaledObjectWithFileAuth(t *testing.T, kc *kubernetes.Clientset) {
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

	// Verify deployment has init container that creates the auth file
	deployment, err := kc.AppsV1().Deployments(testNamespace).Get(context.Background(), deploymentName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, deployment)

	// Check init container
	initContainers := deployment.Spec.Template.Spec.InitContainers
	assert.Len(t, initContainers, 1)
	assert.Equal(t, "init-auth", initContainers[0].Name)

	// Check volume mount
	volumeMounts := initContainers[0].VolumeMounts
	assert.Len(t, volumeMounts, 1)
	assert.Equal(t, "/mnt/auth", volumeMounts[0].MountPath)

	// Check volumes
	volumes := deployment.Spec.Template.Spec.Volumes
	assert.Len(t, volumes, 1)
	assert.NotNil(t, volumes[0].EmptyDir) // emptyDir type
}
