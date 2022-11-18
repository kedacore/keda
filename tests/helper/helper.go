//go:build e2e
// +build e2e

package helper

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/kedacore/keda/v2/pkg/generated/clientset/versioned/typed/keda/v1alpha1"
)

const (
	AzureWorkloadIdentityNamespace = "azure-workload-identity-system"
	AwsIdentityNamespace           = "aws-identity-system"
	KEDANamespace                  = "keda"
	KEDAOperator                   = "keda-operator"
	KEDAMetricsAPIServer           = "keda-metrics-apiserver"

	DefaultHTTPTimeOut = 3000

	StringFalse = "false"
	StringTrue  = "true"
)

var _ = godotenv.Load()

var random = rand.New(rand.NewSource(time.Now().UnixNano()))

// Env variables required for setup and cleanup.
var (
	AzureADTenantID               = os.Getenv("TF_AZURE_SP_TENANT")
	AzureRunWorkloadIdentityTests = os.Getenv("AZURE_RUN_WORKLOAD_IDENTITY_TESTS")
	AwsIdentityTests              = os.Getenv("AWS_RUN_IDENTITY_TESTS")
)

var (
	KubeClient     *kubernetes.Clientset
	KedaKubeClient *v1alpha1.KedaV1alpha1Client
	KubeConfig     *rest.Config
)

type ExecutionError struct {
	StdError []byte
}

func (ee ExecutionError) Error() string {
	return string(ee.StdError)
}

func ParseCommand(cmdWithArgs string) *exec.Cmd {
	quoted := false
	splitCmd := strings.FieldsFunc(cmdWithArgs, func(r rune) bool {
		if r == '\'' {
			quoted = !quoted
		}
		return !quoted && r == ' '
	})
	for i, s := range splitCmd {
		if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
			splitCmd[i] = s[1 : len(s)-1]
		}
	}

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

func ExecCommandOnSpecificPod(t *testing.T, podName string, namespace string, command string) (string, string, error) {
	cmd := []string{
		"sh",
		"-c",
		command,
	}
	buf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	request := KubeClient.CoreV1().RESTClient().Post().
		Resource("pods").Name(podName).Namespace(namespace).
		SubResource("exec").Timeout(time.Second*20).
		VersionedParams(&corev1.PodExecOptions{
			Command: cmd,
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     true,
		}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(KubeConfig, "POST", request.URL())
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	if err != nil {
		return "", "", err
	}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: buf,
		Stderr: errBuf,
	})
	out := buf.String()
	errOut := errBuf.String()
	return out, errOut, err
}

func WaitForSuccessfulExecCommandOnSpecificPod(t *testing.T, podName string, namespace string, command string, iterations, intervalSeconds int) (bool, string, string, error) {
	var out, errOut string
	var err error
	for i := 0; i < iterations; i++ {
		out, errOut, err = ExecCommandOnSpecificPod(t, podName, namespace, command)
		t.Logf("Waiting for successful execution of command on Pod; Output: %s, Error: %s", out, errOut)
		if err == nil {
			return true, out, errOut, err
		}

		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}

	return false, out, errOut, err
}

func GetKubernetesClient(t *testing.T) *kubernetes.Clientset {
	if KubeClient != nil && KubeConfig != nil {
		return KubeClient
	}

	var err error
	KubeConfig, err = config.GetConfig()
	assert.NoErrorf(t, err, "cannot fetch kube config file - %s", err)

	KubeClient, err = kubernetes.NewForConfig(KubeConfig)
	assert.NoErrorf(t, err, "cannot create kubernetes client - %s", err)

	return KubeClient
}

func GetKedaKubernetesClient(t *testing.T) *v1alpha1.KedaV1alpha1Client {
	if KedaKubeClient != nil && KubeConfig != nil {
		return KedaKubeClient
	}

	var err error
	KubeConfig, err = config.GetConfig()
	assert.NoErrorf(t, err, "cannot fetch kube config file - %s", err)

	KedaKubeClient, err = v1alpha1.NewForConfig(KubeConfig)
	assert.NoErrorf(t, err, "cannot create keda kubernetes client - %s", err)

	return KedaKubeClient
}

// Creates a new namespace. If it already exists, make sure it is deleted first.
func CreateNamespace(t *testing.T, kc *kubernetes.Clientset, nsName string) {
	DeleteNamespace(t, kc, nsName)
	WaitForNamespaceDeletion(t, kc, nsName)

	t.Logf("Creating namespace - %s", nsName)
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
	t.Logf("deleting namespace %s", nsName)
	period := int64(0)
	err := KubeClient.CoreV1().Namespaces().Delete(context.Background(), nsName, metav1.DeleteOptions{
		GracePeriodSeconds: &period,
	})
	if errors.IsNotFound(err) {
		err = nil
	}
	assert.NoErrorf(t, err, "cannot delete kubernetes namespace - %s", err)
}

func WaitForJobSuccess(t *testing.T, kc *kubernetes.Clientset, jobName, namespace string, iterations, interval int) bool {
	for i := 0; i < iterations; i++ {
		job, err := kc.BatchV1().Jobs(namespace).Get(context.Background(), jobName, metav1.GetOptions{})
		if err != nil {
			t.Logf("cannot run job - %s", err)
		}

		if job.Status.Succeeded > 0 {
			t.Logf("job %s ran successfully!", jobName)
			return true // Job ran successfully
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}
	return false
}

func WaitForNamespaceDeletion(t *testing.T, kc *kubernetes.Clientset, nsName string) bool {
	for i := 0; i < 30; i++ {
		t.Logf("waiting for namespace %s deletion", nsName)
		_, err := KubeClient.CoreV1().Namespaces().Get(context.Background(), nsName, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			return true
		}
		time.Sleep(time.Second * 5)
	}
	return false
}

func WaitForJobCount(t *testing.T, kc *kubernetes.Clientset, namespace string,
	target, iterations, intervalSeconds int) bool {
	for i := 0; i < iterations; i++ {
		jobList, _ := kc.BatchV1().Jobs(namespace).List(context.Background(), metav1.ListOptions{})
		count := len(jobList.Items)

		t.Logf("Waiting for job count to hit target. Namespace - %s, Current  - %d, Target - %d",
			namespace, count, target)

		if count == target {
			return true
		}

		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}

	return false
}

func WaitForJobCountUntilIteration(t *testing.T, kc *kubernetes.Clientset, namespace string,
	target, iterations, intervalSeconds int) bool {
	var isTargetAchieved = false

	for i := 0; i < iterations; i++ {
		jobList, _ := kc.BatchV1().Jobs(namespace).List(context.Background(), metav1.ListOptions{})
		count := len(jobList.Items)

		t.Logf("Waiting for job count to hit target. Namespace - %s, Current  - %d, Target - %d",
			namespace, count, target)

		if count == target {
			isTargetAchieved = true
		} else {
			isTargetAchieved = false
		}

		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}

	return isTargetAchieved
}

// Waits until deployment count hits target or number of iterations are done.
func WaitForPodCountInNamespace(t *testing.T, kc *kubernetes.Clientset, namespace string,
	target, iterations, intervalSeconds int) bool {
	for i := 0; i < iterations; i++ {
		pods, _ := kc.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})

		t.Logf("Waiting for pods in namespace to hit target. Namespace - %s, Current  - %d, Target - %d",
			namespace, len(pods.Items), target)

		if len(pods.Items) == target {
			return true
		}

		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}

	return false
}

// Waits until deployment ready replica count hits target or number of iterations are done.
func WaitForDeploymentReplicaReadyCount(t *testing.T, kc *kubernetes.Clientset, name, namespace string,
	target, iterations, intervalSeconds int) bool {
	for i := 0; i < iterations; i++ {
		deployment, _ := kc.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
		replicas := deployment.Status.ReadyReplicas

		t.Logf("Waiting for deployment replicas to hit target. Deployment - %s, Current  - %d, Target - %d",
			name, replicas, target)

		if replicas == int32(target) {
			return true
		}

		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}

	return false
}

// Waits until statefulset count hits target or number of iterations are done.
func WaitForStatefulsetReplicaReadyCount(t *testing.T, kc *kubernetes.Clientset, name, namespace string,
	target, iterations, intervalSeconds int) bool {
	for i := 0; i < iterations; i++ {
		statefulset, _ := kc.AppsV1().StatefulSets(namespace).Get(context.Background(), name, metav1.GetOptions{})
		replicas := statefulset.Status.ReadyReplicas

		t.Logf("Waiting for statefulset replicas to hit target. Statefulset - %s, Current  - %d, Target - %d",
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

// Waits some time to ensure that the replica count doesn't change.
func AssertReplicaCountNotChangeDuringTimePeriod(t *testing.T, kc *kubernetes.Clientset, name, namespace string, target, intervalSeconds int) {
	t.Logf("Waiting for some time to ensure deployment replica count doesn't change from %d", target)
	var replicas int32

	for i := 0; i < intervalSeconds; i++ {
		deployment, _ := kc.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
		replicas = deployment.Status.Replicas

		t.Logf("Deployment - %s, Current  - %d", name, replicas)

		if replicas != int32(target) {
			assert.Fail(t, fmt.Sprintf("%s replica count has changed from %d to %d", name, target, replicas))
			return
		}

		time.Sleep(time.Second)
	}
}

func WaitForHpaCreation(t *testing.T, kc *kubernetes.Clientset, name, namespace string,
	iterations, intervalSeconds int) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	var err error
	for i := 0; i < iterations; i++ {
		hpa, err = kc.AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(context.Background(), name, metav1.GetOptions{})
		t.Log("Waiting for hpa creation")
		if err == nil {
			return hpa, err
		}
		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}
	return hpa, err
}

func KubernetesScaleDeployment(t *testing.T, kc *kubernetes.Clientset, name string, desiredReplica int64, namespace string) {
	scaleObject, _ := kc.AppsV1().Deployments(namespace).GetScale(context.TODO(), name, metav1.GetOptions{})
	sc := *scaleObject
	sc.Spec.Replicas = int32(desiredReplica)
	us, err := kc.AppsV1().Deployments(namespace).UpdateScale(context.TODO(), name, &sc, metav1.UpdateOptions{})
	if err != nil {
		assert.NoErrorf(t, err, "couldn't scale the deployment: %v: %v", us.Name, err.Error())
	}
}

type Template struct {
	Name, Config string
}

func KubectlApplyWithTemplate(t *testing.T, data interface{}, templateName string, config string) {
	t.Logf("Applying template: %s", templateName)

	tmpl, err := template.New("kubernetes resource template").Parse(config)
	assert.NoErrorf(t, err, "cannot parse template - %s", err)

	tempFile, err := os.CreateTemp("", templateName)
	assert.NoErrorf(t, err, "cannot create temp file - %s", err)

	defer os.Remove(tempFile.Name())

	err = tmpl.Execute(tempFile, data)
	assert.NoErrorf(t, err, "cannot insert data into template - %s", err)

	_, err = ExecuteCommand(fmt.Sprintf("kubectl apply -f %s", tempFile.Name()))
	assert.NoErrorf(t, err, "cannot apply file - %s", err)

	err = tempFile.Close()
	assert.NoErrorf(t, err, "cannot close temp file - %s", err)
}

// Apply templates in order of slice
func KubectlApplyMultipleWithTemplate(t *testing.T, data interface{}, templates []Template) {
	for _, tmpl := range templates {
		KubectlApplyWithTemplate(t, data, tmpl.Name, tmpl.Config)
	}
}

func KubectlDeleteWithTemplate(t *testing.T, data interface{}, templateName, config string) {
	t.Logf("Deleting template: %s", templateName)

	tmpl, err := template.New("kubernetes resource template").Parse(config)
	assert.NoErrorf(t, err, "cannot parse template - %s", err)

	tempFile, err := os.CreateTemp("", templateName)
	assert.NoErrorf(t, err, "cannot delete temp file - %s", err)

	defer os.Remove(tempFile.Name())

	err = tmpl.Execute(tempFile, data)
	assert.NoErrorf(t, err, "cannot insert data into template - %s", err)

	_, err = ExecuteCommand(fmt.Sprintf("kubectl delete -f %s", tempFile.Name()))
	assert.NoErrorf(t, err, "cannot apply file - %s", err)

	err = tempFile.Close()
	assert.NoErrorf(t, err, "cannot close temp file - %s", err)
}

// Delete templates in reverse order of slice
func KubectlDeleteMultipleWithTemplate(t *testing.T, data interface{}, templates []Template) {
	for idx := len(templates) - 1; idx >= 0; idx-- {
		tmpl := templates[idx]
		KubectlDeleteWithTemplate(t, data, tmpl.Name, tmpl.Config)
	}
}

func CreateKubernetesResources(t *testing.T, kc *kubernetes.Clientset, nsName string, data interface{}, templates []Template) {
	CreateNamespace(t, kc, nsName)
	KubectlApplyMultipleWithTemplate(t, data, templates)
}

func DeleteKubernetesResources(t *testing.T, kc *kubernetes.Clientset, nsName string, data interface{}, templates []Template) {
	KubectlDeleteMultipleWithTemplate(t, data, templates)
	DeleteNamespace(t, kc, nsName)
	deleted := WaitForNamespaceDeletion(t, kc, nsName)
	assert.Truef(t, deleted, "%s namespace not deleted", nsName)
}

func GetRandomNumber() int {
	return random.Intn(10000)
}

func RemoveANSI(input string) string {
	reg := regexp.MustCompile(`(\x9B|\x1B\[)[0-?]*[ -\/]*[@-~]`)
	return reg.ReplaceAllString(input, "")
}

func FindPodLogs(t *testing.T, kc *kubernetes.Clientset, namespace, label string) []string {
	var podLogs []string
	t.Logf("Searching for pod logs.........")
	pods, err := kc.CoreV1().Pods(namespace).List(context.TODO(),
		metav1.ListOptions{LabelSelector: label})
	if err != nil {
		assert.NoErrorf(t, err, "no pod in the list - %s", err)
	}
	var podLogRequest *rest.Request
	for _, v := range pods.Items {
		podLogRequest = kc.CoreV1().Pods(namespace).GetLogs(v.Name, &corev1.PodLogOptions{})
		stream, err := podLogRequest.Stream(context.TODO())
		if err != nil {
			assert.NoErrorf(t, err, "cannot open the stream - %s", err)
		}
		defer stream.Close()
		for {
			buf := make([]byte, 2000)
			numBytes, err := stream.Read(buf)
			if err == io.EOF {
				break
			}
			if numBytes == 0 {
				continue
			}
			if err != nil {
				assert.NoErrorf(t, err, "cannot read log stream - %s", err)
			}
			podLogs = append(podLogs, string(buf[:numBytes]))
		}
	}
	return podLogs
}
