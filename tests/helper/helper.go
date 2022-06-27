//go:build e2e
// +build e2e

package helper

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	AzureWorkloadIdentityNamespace = "azure-workload-identity-system"
	KEDANamespace                  = "keda"
	KEDAOperator                   = "keda-operator"
	KEDAMetricsAPIServer           = "keda-metrics-apiserver"

	DefaultHTTPTimeOut = 3000
)

var _ = godotenv.Load()

// Env variables required for setup and cleanup.
var (
	AzureADTenantID               = os.Getenv("AZURE_SP_TENANT")
	AzureRunWorkloadIdentityTests = os.Getenv("AZURE_RUN_WORKLOAD_IDENTITY_TESTS")
)

var (
	Kc *kubernetes.Clientset
)

type ExecutionError struct {
	StdError []byte
}

func (ee ExecutionError) Error() string {
	return string(ee.StdError)
}

func ParseCommand(cmdWithArgs string) *exec.Cmd {
	splitCmd := strings.Fields(cmdWithArgs)

	return exec.Command(splitCmd[0], splitCmd[1:]...)
}

func ParseCommandWithDir(cmdWithArgs, dir string) *exec.Cmd {
	cmd := ParseCommand(cmdWithArgs)
	cmd.Dir = dir

	return cmd
}

func ExecuteCommand(cmdWithArgs string) ([]byte, error) {
	out, err := ParseCommand(cmdWithArgs).Output()
	if err != nil {
		exitError, ok := err.(*exec.ExitError)
		if ok {
			return out, ExecutionError{StdError: exitError.Stderr}
		}
	}

	return out, err
}

func ExecuteCommandWithDir(cmdWithArgs, dir string) ([]byte, error) {
	out, err := ParseCommandWithDir(cmdWithArgs, dir).Output()
	if err != nil {
		exitError, ok := err.(*exec.ExitError)
		if ok {
			return out, ExecutionError{StdError: exitError.Stderr}
		}
	}

	return out, err
}

func GetKubernetesClient(t *testing.T) *kubernetes.Clientset {
	if Kc != nil {
		return Kc
	}

	kubeConfig, err := config.GetConfig()
	assert.NoErrorf(t, err, "cannot fetch kube config file - %s", err)

	Kc, err = kubernetes.NewForConfig(kubeConfig)
	assert.NoErrorf(t, err, "cannot create kubernetes client - %s", err)

	return Kc
}

func CreateNamespace(t *testing.T, kc *kubernetes.Clientset, nsName string) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   nsName,
			Labels: map[string]string{"type": "e2e"},
		},
	}

	_, err := kc.CoreV1().Namespaces().Create(context.Background(), namespace, metav1.CreateOptions{})
	assert.NoErrorf(t, err, "cannot create kubernetes namespace - %s", err)
}

func DeleteNamespace(t *testing.T, kc *kubernetes.Clientset, nsName string) {
	err := Kc.CoreV1().Namespaces().Delete(context.Background(), nsName, metav1.DeleteOptions{})
	assert.NoErrorf(t, err, "cannot delete kubernetes namespace - %s", err)
}

// Waits until deployment count hits target or number of iterations are done.
func WaitForDeploymentReplicaCount(t *testing.T, kc *kubernetes.Clientset, name, namespace string,
	target, iterations, intervalSeconds int) bool {
	for i := 0; i < iterations; i++ {
		deployment, _ := kc.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
		replicas := deployment.Status.Replicas

		t.Logf("Waiting for deployment replicas to hit target. Deployment - %s, Current  - %d, Target - %d",
			name, replicas, target)

		if replicas == int32(target) {
			return true
		}

		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}

	return false
}

// Waits for number of iterations and returns replica count.
func WaitForDeploymentReplicaCountChange(t *testing.T, kc *kubernetes.Clientset, name, namespace string, iterations, intervalSeconds int) int {
	t.Log("Waiting for some time to see if deployment replica count changes")
	var replicas, prevReplicas int32
	prevReplicas = -1

	for i := 0; i < iterations; i++ {
		deployment, _ := kc.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
		replicas = deployment.Status.Replicas

		t.Logf("Deployment - %s, Current  - %d", name, replicas)

		if replicas != prevReplicas && prevReplicas != -1 {
			break
		}

		prevReplicas = replicas
		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}

	return int(replicas)
}

func WaitForHpaCreation(t *testing.T, kc *kubernetes.Clientset, name, namespace string,
	iterations, intervalSeconds int) (*autoscalingv2beta2.HorizontalPodAutoscaler, error) {
	hpa := &autoscalingv2beta2.HorizontalPodAutoscaler{}
	var err error
	for i := 0; i < iterations; i++ {
		hpa, err = kc.AutoscalingV2beta2().HorizontalPodAutoscalers(namespace).Get(context.Background(), name, metav1.GetOptions{})
		t.Log("Waiting for hpa creation")
		if err == nil {
			return hpa, err
		}
		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}
	return hpa, err
}

func KubectlApplyWithTemplate(t *testing.T, data interface{}, config string) {
	tmpl, err := template.New("kubernetes resource template").Parse(config)
	assert.NoErrorf(t, err, "cannot parse template - %s", err)

	tempFile, err := ioutil.TempFile("", "tempTemplateFile")
	assert.NoErrorf(t, err, "cannot create temp file - %s", err)

	defer os.Remove(tempFile.Name())

	err = tmpl.Execute(tempFile, data)
	assert.NoErrorf(t, err, "cannot insert data into template - %s", err)

	_, err = ExecuteCommand(fmt.Sprintf("kubectl apply -f %s", tempFile.Name()))
	assert.NoErrorf(t, err, "cannot apply file - %s", err)

	err = tempFile.Close()
	assert.NoErrorf(t, err, "cannot close temp file - %s", err)
}

func KubectlApplyMultipleWithTemplate(t *testing.T, data interface{}, configs ...string) {
	for _, config := range configs {
		KubectlApplyWithTemplate(t, data, config)
	}
}

func KubectlDeleteWithTemplate(t *testing.T, data interface{}, config string) {
	tmpl, err := template.New("kubernetes resource template").Parse(config)
	assert.NoErrorf(t, err, "cannot parse template - %s", err)

	tempFile, err := ioutil.TempFile("", "tempTemplateFile")
	assert.NoErrorf(t, err, "cannot create temp file - %s", err)

	defer os.Remove(tempFile.Name())

	err = tmpl.Execute(tempFile, data)
	assert.NoErrorf(t, err, "cannot insert data into template - %s", err)

	_, err = ExecuteCommand(fmt.Sprintf("kubectl delete -f %s", tempFile.Name()))
	assert.NoErrorf(t, err, "cannot apply file - %s", err)

	err = tempFile.Close()
	assert.NoErrorf(t, err, "cannot close temp file - %s", err)
}

func KubectlDeleteMultipleWithTemplate(t *testing.T, data interface{}, configs ...string) {
	for _, config := range configs {
		KubectlDeleteWithTemplate(t, data, config)
	}
}

func CreateKubernetesResources(t *testing.T, kc *kubernetes.Clientset, nsName string, data interface{}, configs ...string) {
	CreateNamespace(t, kc, nsName)
	KubectlApplyMultipleWithTemplate(t, data, configs...)
}

func DeleteKubernetesResources(t *testing.T, kc *kubernetes.Clientset, nsName string, data interface{}, configs ...string) {
	DeleteNamespace(t, kc, nsName)
	KubectlDeleteMultipleWithTemplate(t, data, configs...)
}
