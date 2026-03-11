//go:build e2e
// +build e2e

package helper

import (
	"bytes"
	"context"
	cryptoRand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/joho/godotenv"
	prommodel "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/kedacore/keda/v2/pkg/generated/clientset/versioned/typed/keda/v1alpha1"
)

const (
	ArgoRolloutsNamespace          = "argo-rollouts"
	AzureWorkloadIdentityNamespace = "azure-workload-identity-system"
	AwsIdentityNamespace           = "aws-identity-system"
	GcpIdentityNamespace           = "gcp-identity-system"
	OpentelemetryNamespace         = "open-telemetry-system"
	CertManagerNamespace           = "cert-manager"
	KEDANamespace                  = "keda"
	KEDAOperator                   = "keda-operator"
	KEDAMetricsAPIServer           = "keda-metrics-apiserver"
	KEDAAdmissionWebhooks          = "keda-admission"

	DefaultHTTPTimeOut = 3000

	StringFalse = "false"
	StringTrue  = "true"

	StrimziVersion   = "0.47.0"
	StrimziChartName = "strimzi"
	StrimziNamespace = "strimzi"
)

const (
	WaitShort     = 20 * time.Second
	IntervalShort = 1 * time.Second
)

const (
	caCrtPath = "/tmp/keda-e2e-ca.crt"
	caKeyPath = "/tmp/keda-e2e-ca.key"
)

var _ = godotenv.Load()

var random = rand.New(rand.NewSource(time.Now().UnixNano()))

// Env variables required for setup and cleanup.
var (
	AzureADMsiID                  = os.Getenv("TF_AZURE_IDENTITY_1_APP_FULL_ID")
	AzureADMsiClientID            = os.Getenv("TF_AZURE_IDENTITY_1_APP_ID")
	AzureADTenantID               = os.Getenv("TF_AZURE_SP_TENANT")
	AzureRunWorkloadIdentityTests = os.Getenv("AZURE_RUN_WORKLOAD_IDENTITY_TESTS")
	AwsIdentityTests              = os.Getenv("AWS_RUN_IDENTITY_TESTS")
	GcpIdentityTests              = os.Getenv("GCP_RUN_IDENTITY_TESTS")
	EnableOpentelemetry           = os.Getenv("ENABLE_OPENTELEMETRY")
	InstallArgoRollouts           = os.Getenv("E2E_INSTALL_ARGO_ROLLOUTS")
	InstallCertManager            = AwsIdentityTests == StringTrue || GcpIdentityTests == StringTrue
	InstallKeda                   = os.Getenv("E2E_INSTALL_KEDA")
	InstallKafka                  = os.Getenv("E2E_INSTALL_KAFKA")
)

func init() {
	if err := LoadTestConfig(); err != nil {
		fmt.Printf("Error loading test config: %v\n", err)
		os.Exit(1)
	}
}

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
	return ExecuteCommandWithDirAndEnv(cmdWithArgs, dir, nil)
}

func ExecuteCommandWithDirAndEnv(cmdWithArgs, dir string, env map[string]string) ([]byte, error) {
	cmd := ParseCommandWithDir(cmdWithArgs, dir)

	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	out, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return out, ExecutionError{StdError: exitError.Stderr}
		}
	}

	return out, err
}

func ExecCommandOnSpecificPod(t *testing.T, podName string, namespace string, command string) (string, string, error) {
	return executeCommandOnPod(t, podName, namespace, command, true)
}

func ExecCommandOnSpecificPodWithoutTTY(t *testing.T, podName string, namespace string, command string) (string, string, error) {
	return executeCommandOnPod(t, podName, namespace, command, false)
}

func executeCommandOnPod(t *testing.T, podName string, namespace string, command string, tty bool) (string, string, error) {
	cmd := []string{
		"sh",
		"-c",
		command,
	}
	buf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	request := GetKubernetesClient(t).CoreV1().RESTClient().Post().
		Resource("pods").Name(podName).Namespace(namespace).
		SubResource("exec").Timeout(time.Second*20).
		VersionedParams(&corev1.PodExecOptions{
			Command: cmd,
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     tty,
		}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(KubeConfig, "POST", request.URL())
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	if err != nil {
		return "", "", err
	}
	err = exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
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
	DeleteNamespace(t, nsName)
	WaitForNamespaceDeletion(t, nsName)

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

func DeleteNamespace(t *testing.T, nsName string) {
	t.Logf("deleting namespace %s", nsName)
	period := int64(0)
	err := GetKubernetesClient(t).CoreV1().Namespaces().Delete(context.Background(), nsName, metav1.DeleteOptions{
		GracePeriodSeconds: &period,
	})
	if errors.IsNotFound(err) {
		err = nil
	}
	assert.NoErrorf(t, err, "cannot delete kubernetes namespace - %s", err)
	DeletePodsInNamespace(t, nsName)
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

func WaitForAllJobsSuccess(t *testing.T, kc *kubernetes.Clientset, namespace string, iterations, interval int) bool {
	for i := 0; i < iterations; i++ {
		jobs, err := kc.BatchV1().Jobs(namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			t.Logf("cannot list jobs - %s", err)
		}

		allJobsSuccess := true
		for _, job := range jobs.Items {
			if job.Status.Succeeded == 0 {
				allJobsSuccess = false
				break
			}
		}

		if allJobsSuccess {
			t.Logf("all jobs ran successfully!")
			return true // Job ran successfully
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}
	return false
}

func WaitForNamespaceDeletion(t *testing.T, nsName string) bool {
	for i := 0; i < 120; i++ {
		t.Logf("waiting for namespace %s deletion", nsName)
		_, err := GetKubernetesClient(t).CoreV1().Namespaces().Get(context.Background(), nsName, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			return true
		}
		time.Sleep(time.Second * 5)
	}
	return false
}

func WaitForScaledJobCount(t *testing.T, kc *kubernetes.Clientset, scaledJobName, namespace string, target, iterations, intervalSeconds int) bool {
	return waitForJobCount(t, kc, fmt.Sprintf("scaledjob.keda.sh/name=%s", scaledJobName), namespace, target, iterations, intervalSeconds)
}

func WaitForJobCount(t *testing.T, kc *kubernetes.Clientset, namespace string, target, iterations, intervalSeconds int) bool {
	return waitForJobCount(t, kc, "", namespace, target, iterations, intervalSeconds)
}

func waitForJobCount(t *testing.T, kc *kubernetes.Clientset, selector, namespace string, target, iterations, intervalSeconds int) bool {
	for i := 0; i < iterations; i++ {
		jobList, _ := kc.BatchV1().Jobs(namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: selector,
		})
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

func WaitForJobCountUntilIteration(t *testing.T, kc *kubernetes.Clientset, namespace string, target, iterations, intervalSeconds int) bool {
	isTargetAchieved := false

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

func WaitForJobCreation(t *testing.T, kc *kubernetes.Clientset, scaledJobName, namespace string, iterations, intervalSeconds int) (*batchv1.Job, error) {
	jobList := &batchv1.JobList{}
	var err error

	for i := 0; i < iterations; i++ {
		jobList, err = kc.BatchV1().Jobs(namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("scaledjob.keda.sh/name=%s", scaledJobName),
		})

		t.Log("Waiting for job creation")

		if len(jobList.Items) > 0 {
			return &jobList.Items[0], nil
		}

		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}

	return nil, err
}

// Waits until deployment count hits target or number of iterations are done.
func WaitForPodCountInNamespace(t *testing.T, kc *kubernetes.Clientset, namespace string, target, iterations, intervalSeconds int) bool {
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

// Waits until all the pods with a defined label are in completed status.
func WaitForPodsCompleted(t *testing.T, kc *kubernetes.Clientset, selector, namespace string, iterations, intervalSeconds int) bool {
	for i := 0; i < iterations; i++ {
		pods, err := kc.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: selector})
		if (err != nil && errors.IsNotFound(err)) || len(pods.Items) == 0 {
			t.Logf("No pods with label %s", selector)
			return true
		}

		succeededCount := 0

		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodSucceeded {
				t.Logf("Pod %s in namespace %s is in completed status", pod.Name, namespace)
				succeededCount++
			}
		}

		if succeededCount == len(pods.Items) {
			return true
		}

		t.Logf("Waiting for pods with label %s to complete", selector)

		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}
	return false
}

// Waits until all the pods in the namespace have a running status.
func WaitForAllPodRunningInNamespace(t *testing.T, kc *kubernetes.Clientset, namespace string, iterations, intervalSeconds int) bool {
	for i := 0; i < iterations; i++ {
		runningCount := 0
		pods, _ := kc.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})

		for _, pod := range pods.Items {
			if pod.Status.Phase != corev1.PodRunning {
				break
			}
			runningCount++
		}

		t.Logf("Waiting for pods in namespace to be in 'Running' status. Namespace - %s, Current - %d, Target - %d",
			namespace, runningCount, len(pods.Items))

		if runningCount == len(pods.Items) {
			return true
		}

		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}

	return false
}

// Waits until the Horizontal Pod Autoscaler for the scaledObject reports that it has metrics available
// to calculate, or until the number of iterations are done, whichever happens first.
func WaitForHPAMetricsToPopulate(t *testing.T, kc *kubernetes.Clientset, name, namespace string, iterations, intervalSeconds int) bool {
	totalWaitDuration := time.Duration(iterations) * time.Duration(intervalSeconds) * time.Second
	startedWaiting := time.Now()
	for i := 0; i < iterations; i++ {
		t.Logf("Waiting up to %s for HPA to populate metrics - %s so far", totalWaitDuration, time.Since(startedWaiting).Round(time.Second))

		hpa, _ := kc.AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if hpa.Status.CurrentMetrics != nil {
			for _, currentMetric := range hpa.Status.CurrentMetrics {
				// When testing on a kind cluster at least, an empty metricStatus object with a blank type shows up first,
				// so we need to make sure we have *actual* resource metrics before we return
				if currentMetric.Type != "" {
					j, _ := json.MarshalIndent(hpa.Status.CurrentMetrics, "  ", "    ")
					t.Logf("HPA has metrics after %s: %s", time.Since(startedWaiting), j)
					return true
				}
			}
		}

		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}
	return false
}

// Waits until deployment ready replica count hits target or number of iterations are done.
func WaitForDeploymentReplicaReadyCount(t *testing.T, kc *kubernetes.Clientset, name, namespace string, target, iterations, intervalSeconds int) bool {
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

// Waits until pod is in ready state or number of iterations are done.
func WaitForPodReady(t *testing.T, kc *kubernetes.Clientset, podName, namespace string, iterations, intervalSeconds int) bool {
	for i := 0; i < iterations; i++ {
		pod, _ := kc.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
		t.Logf("Waiting for pod to be in ready state. Pod - %s, Current Phase - %s", podName, pod.Status.Phase)

		// A pod can be in the Running phase without all containers being ready.
		// Check the Ready condition to ensure the pod is actually ready.
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
				return true
			}
		}
		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}
	return false
}

// Waits until rollout ready replica count hits target or number of iterations are done.
func WaitForArgoRolloutReplicaReadyCount(t *testing.T, _ *kubernetes.Clientset, name, namespace string, target, iterations, intervalSeconds int) bool {
	for i := 0; i < iterations; i++ {
		// If target==0, we check for spec replicas, since .status.readyReplicas won't be set by the controller.
		jsonPath := ".status.readyReplicas"
		if target == 0 {
			jsonPath = ".spec.replicas"
		}

		kctlGetCmd := fmt.Sprintf(`kubectl get rollouts.argoproj.io/%s -n %s -o jsonpath="{%s}"`, name, namespace, jsonPath)
		output, err := ExecuteCommand(kctlGetCmd)
		assert.NoErrorf(t, err, "cannot get rollout info - %s", err)

		unquotedOutput := strings.ReplaceAll(string(output), "\"", "")

		// Length of output can be zero, which means .status.readyReplicas is not yet set by the controller.
		// In that case, sleep and check in the next iteration. Otherwise compare.
		if len(unquotedOutput) != 0 {
			replicas, err := strconv.ParseInt(unquotedOutput, 10, 64)
			assert.NoErrorf(t, err, "cannot convert rollout count to int - %s", err)

			t.Logf("Waiting for rollout replicas to hit target. Rollout - %s, Current  - %d, Target - %d",
				name, replicas, target)

			if replicas == int64(target) {
				return true
			}
		}

		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}

	return false
}

// Waits until statefulset count hits target or number of iterations are done.
func WaitForStatefulsetReplicaReadyCount(t *testing.T, kc *kubernetes.Clientset, name, namespace string, target, iterations, intervalSeconds int) bool {
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

// WaitForReplicaSetReplicaReadyCount waits until replicaset replica count hits target or number of iterations are done.
func WaitForReplicaSetReplicaReadyCount(t *testing.T, kc *kubernetes.Clientset, name, namespace string, target, iterations, intervalSeconds int) bool {
	for i := 0; i < iterations; i++ {
		rs, _ := kc.AppsV1().ReplicaSets(namespace).Get(context.Background(), name, metav1.GetOptions{})

		// Use spec.replicas when target is 0 (status.readyReplicas won't be set)
		var replicas int32
		if target == 0 {
			if rs.Spec.Replicas != nil {
				replicas = *rs.Spec.Replicas
			}
		} else {
			replicas = rs.Status.ReadyReplicas
		}

		t.Logf("Waiting for replicaset replicas to hit target. ReplicaSet - %s, Current - %d, Target - %d",
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

// Waits some time to ensure that the replica count doesn't change.
func AssertReplicaCountNotChangeDuringTimePeriodRollout(t *testing.T, _ *kubernetes.Clientset, name, namespace string, target, intervalSeconds int) {
	t.Logf("Waiting for some time to ensure rollout replica count doesn't change from %d", target)
	var replicas int64

	for i := 0; i < intervalSeconds; i++ {
		// If target==0, we check for spec replicas, since .status.readyReplicas won't be set by the controller.
		jsonPath := ".status.replicas"
		if target == 0 {
			jsonPath = ".spec.replicas"
		}

		kctlGetCmd := fmt.Sprintf(`kubectl get rollouts.argoproj.io/%s -n %s -o jsonpath="{%s}"`, name, namespace, jsonPath)
		output, err := ExecuteCommand(kctlGetCmd)
		assert.NoErrorf(t, err, "cannot get rollout info - %s", err)

		unquotedOutput := strings.ReplaceAll(string(output), "\"", "")

		// Length of output can be zero, which means .status.replicas is not set by the controller.
		// In that case, fail the test. Otherwise, compare.
		if len(unquotedOutput) != 0 {
			replicas, err = strconv.ParseInt(unquotedOutput, 10, 64)
			assert.NoErrorf(t, err, "cannot convert rollout count to int - %s", err)

			t.Logf("Rollout - %s, Current  - %d", name, replicas)

			if replicas != int64(target) {
				assert.Fail(t, fmt.Sprintf("%s replica count has changed from %d to %d", name, target, replicas))
				return
			}
		} else {
			assert.Fail(t, fmt.Sprintf("%s replicas are not set in its status, expected %d", name, target))
		}

		time.Sleep(time.Second)
	}
}

func WaitForHpaCreation(t *testing.T, kc *kubernetes.Clientset, name, namespace string, iterations, intervalSeconds int) (*autoscalingv2.HorizontalPodAutoscaler, error) {
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

func KubectlApplyWithErrors(t *testing.T, data interface{}, templateName string, config string) error {
	t.Logf("Applying template: %s", templateName)

	tmpl, err := template.New("kubernetes resource template").Parse(config)
	assert.NoErrorf(t, err, "cannot parse template - %s", err)

	tempFile, err := os.CreateTemp("", templateName)
	assert.NoErrorf(t, err, "cannot create temp file - %s", err)
	if err != nil {
		defer tempFile.Close()
		defer os.Remove(tempFile.Name())
	}

	err = tmpl.Execute(tempFile, data)
	assert.NoErrorf(t, err, "cannot insert data into template - %s", err)

	_, err = ExecuteCommand(fmt.Sprintf("kubectl apply -f %s", tempFile.Name()))
	return err
}

// Apply templates in order of slice
func KubectlApplyMultipleWithTemplate(t *testing.T, data interface{}, templates []Template) {
	for _, tmpl := range templates {
		KubectlApplyWithTemplate(t, data, tmpl.Name, tmpl.Config)
	}
}

func KubectlReplaceWithTemplate(t *testing.T, data interface{}, templateName string, config string) {
	t.Logf("Applying template: %s", templateName)

	tmpl, err := template.New("kubernetes resource template").Parse(config)
	assert.NoErrorf(t, err, "cannot parse template - %s", err)

	tempFile, err := os.CreateTemp("", templateName)
	assert.NoErrorf(t, err, "cannot create temp file - %s", err)

	defer os.Remove(tempFile.Name())

	err = tmpl.Execute(tempFile, data)
	assert.NoErrorf(t, err, "cannot insert data into template - %s", err)

	_, err = ExecuteCommand(fmt.Sprintf("kubectl replace -f %s --force", tempFile.Name()))
	assert.NoErrorf(t, err, "cannot replace file - %s", err)

	err = tempFile.Close()
	assert.NoErrorf(t, err, "cannot close temp file - %s", err)
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

func KubectlCopyToPod(t *testing.T, content string, remotePath, pod, namespace string) {
	tempFile, err := os.CreateTemp("", "copy-to-pod")
	assert.NoErrorf(t, err, "cannot create temp file - %s", err)
	defer os.Remove(tempFile.Name())

	_, err = tempFile.WriteString(content)
	assert.NoErrorf(t, err, "cannot write temp file - %s", err)

	commnand := fmt.Sprintf("kubectl cp %s %s:/%s -n %s", tempFile.Name(), pod, remotePath, namespace)
	_, err = ExecuteCommand(commnand)
	assert.NoErrorf(t, err, "cannot copy file - %s", err)

	err = tempFile.Close()
	assert.NoErrorf(t, err, "cannot close temp file - %s", err)
}

func CreateKubernetesResources(t *testing.T, kc *kubernetes.Clientset, nsName string, data interface{}, templates []Template) {
	CreateNamespace(t, kc, nsName)
	KubectlApplyMultipleWithTemplate(t, data, templates)
}

func DeleteKubernetesResources(t *testing.T, nsName string, data interface{}, templates []Template) {
	KubectlDeleteMultipleWithTemplate(t, data, templates)
	DeleteNamespace(t, nsName)
	deleted := WaitForNamespaceDeletion(t, nsName)
	assert.Truef(t, deleted, "%s namespace not deleted", nsName)
}

func GetRandomNumber() int {
	return random.Intn(10000)
}

func RemoveANSI(input string) string {
	reg := regexp.MustCompile(`(\x9B|\x1B\[)[0-?]*[ -\/]*[@-~]`)
	return reg.ReplaceAllString(input, "")
}

func FindPodLogs(kc *kubernetes.Clientset, namespace, label string, includePrevious bool) ([]string, error) {
	pods, err := kc.CoreV1().Pods(namespace).List(context.TODO(),
		metav1.ListOptions{LabelSelector: label})
	if err != nil {
		return []string{}, err
	}
	getPodLogs := func(pod *corev1.Pod, previous bool) ([]string, error) {
		podLogRequest := kc.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
			Previous: previous,
		})
		stream, err := podLogRequest.Stream(context.TODO())
		if err != nil {
			return []string{}, err
		}
		defer stream.Close()
		logs := []string{}
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
				return []string{}, err
			}
			logs = append(logs, string(buf[:numBytes]))
		}
		return logs, nil
	}

	var outputLogs []string
	for _, pod := range pods.Items {
		getPrevious := false
		if includePrevious {
			for _, container := range pod.Status.ContainerStatuses {
				if container.RestartCount > 0 {
					getPrevious = true
				}
			}
		}

		if getPrevious {
			podLogs, err := getPodLogs(&pod, true)
			if err != nil {
				return []string{}, err
			}
			outputLogs = append(outputLogs, podLogs...)
			outputLogs = append(outputLogs, "=====================RESTART=====================\n")
		}

		podLogs, err := getPodLogs(&pod, false)
		if err != nil {
			return []string{}, err
		}
		outputLogs = append(outputLogs, podLogs...)
	}
	return outputLogs, nil
}

// Delete all pods in namespace by selector
func DeletePodsInNamespaceBySelector(t *testing.T, kc *kubernetes.Clientset, selector, namespace string) {
	t.Logf("killing all pods in %s namespace with selector %s", namespace, selector)
	err := kc.CoreV1().Pods(namespace).DeleteCollection(context.Background(), metav1.DeleteOptions{
		GracePeriodSeconds: ptr.To(int64(0)),
	}, metav1.ListOptions{
		LabelSelector: selector,
	})
	assert.NoErrorf(t, err, "cannot delete pods - %s", err)
}

// Delete all pods in namespace
func DeletePodsInNamespace(t *testing.T, namespace string) {
	err := GetKubernetesClient(t).CoreV1().Pods(namespace).DeleteCollection(context.Background(), metav1.DeleteOptions{
		GracePeriodSeconds: ptr.To(int64(0)),
	}, metav1.ListOptions{})
	if errors.IsNotFound(err) {
		err = nil
	}
	assert.NoErrorf(t, err, "cannot delete pods - %s", err)
}

// Wait for Pods identified by selector to complete termination
func WaitForPodsTerminated(t *testing.T, kc *kubernetes.Clientset, selector, namespace string, iterations, intervalSeconds int) bool {
	for i := 0; i < iterations; i++ {
		pods, err := kc.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: selector})
		if (err != nil && errors.IsNotFound(err)) || len(pods.Items) == 0 {
			t.Logf("No pods with label %s", selector)
			return true
		}

		t.Logf("Waiting for pods with label %s to terminate", selector)

		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}

	return false
}

func GetTestCA(t *testing.T) ([]byte, []byte) {
	generateCA(t)
	caCrt, err := os.ReadFile(caCrtPath)
	require.NoErrorf(t, err, "error reading custom CA crt - %s", err)
	caKey, err := os.ReadFile(caKeyPath)
	require.NoErrorf(t, err, "error reading custom CA key - %s", err)
	return caCrt, caKey
}

func GenerateServerCert(t *testing.T, domain string) (string, string) {
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Company, INC."},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"Golden Gate Bridge"},
			PostalCode:    []string{"94016"},
		},
		DNSNames: []string{
			domain,
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certPrivKey, err := rsa.GenerateKey(cryptoRand.Reader, 4096)
	require.NoErrorf(t, err, "error generating tls key - %s", err)

	caCrtBytes, caKeyBytes := GetTestCA(t)
	block, _ := pem.Decode(caCrtBytes)
	if block == nil {
		t.Fail()
		return "", ""
	}
	ca, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fail()
		return "", ""
	}
	blockKey, _ := pem.Decode(caKeyBytes)
	if blockKey == nil {
		t.Fail()
		return "", ""
	}
	caKey, err := x509.ParsePKCS1PrivateKey(blockKey.Bytes)
	require.NoErrorf(t, err, "error reading custom CA key - %s", err)
	certBytes, err := x509.CreateCertificate(cryptoRand.Reader, cert, ca, &certPrivKey.PublicKey, caKey)
	require.NoErrorf(t, err, "error creating tls cert - %s", err)

	certPEM := new(bytes.Buffer)
	err = pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	require.NoErrorf(t, err, "error encoding cert - %s", err)

	certPrivKeyPEM := new(bytes.Buffer)
	err = pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})
	require.NoErrorf(t, err, "error encoding key - %s", err)

	return certPEM.String(), certPrivKeyPEM.String()
}

func generateCA(t *testing.T) {
	_, err := os.Stat(caCrtPath)
	if err == nil {
		return
	}
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Company, INC."},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"Golden Gate Bridge"},
			PostalCode:    []string{"94016"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create our private and public key
	caPrivKey, err := rsa.GenerateKey(cryptoRand.Reader, 4096)
	require.NoErrorf(t, err, "error generating custom CA key - %s", err)

	// create the CA
	caBytes, err := x509.CreateCertificate(cryptoRand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	require.NoErrorf(t, err, "error generating custom CA - %s", err)

	// pem encode
	crtFile, err := os.OpenFile(caCrtPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	require.NoErrorf(t, err, "error opening custom CA file - %s", err)
	err = pem.Encode(crtFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	require.NoErrorf(t, err, "error encoding ca - %s", err)
	if err := crtFile.Close(); err != nil {
		require.NoErrorf(t, err, "error closing custom CA file - %s", err)
	}

	keyFile, err := os.OpenFile(caKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		require.NoErrorf(t, err, "error opening custom CA key file- %s", err)
	}
	err = pem.Encode(keyFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})
	require.NoErrorf(t, err, "error encoding CA key - %s", err)
	if err := keyFile.Close(); err != nil {
		require.NoErrorf(t, err, "error closing custom CA key file- %s", err)
	}
}

// CheckKubectlGetResult runs `kubectl get` with parameters and compares output with expected value
func CheckKubectlGetResult(t *testing.T, kind string, name string, namespace string, otherparameter string, expected string) {
	time.Sleep(1 * time.Second) // wait a second for recource deployment finished
	kctlGetCmd := fmt.Sprintf(`kubectl get %s/%s -n %s %s"`, kind, name, namespace, otherparameter)
	t.Log("Running kubectl cmd:", kctlGetCmd)
	output, err := ExecuteCommand(kctlGetCmd)
	assert.NoErrorf(t, err, "cannot get rollout info - %s", err)

	unqoutedOutput := strings.ReplaceAll(string(output), "\"", "")
	assert.Equal(t, expected, unqoutedOutput)
}

// KedaEventually checks if the provided conditionFunc eventually returns true
// (and no error) within the context's deadline. It polls the conditionFunc
// at the given interval until the condition is met or the context times out.
func KedaEventually(ctx context.Context, conditionFunc wait.ConditionWithContextFunc, interval time.Duration) error {
	if interval <= 0 {
		return fmt.Errorf("polling interval must be positive, got %v", interval)
	}

	ok, err := conditionFunc(ctx)
	if err != nil {
		return fmt.Errorf("eventually check failed on initial check: %w", err)
	}
	if ok {
		return nil
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("eventually check failed: context deadline exceeded before condition was met")

		case <-ticker.C:
			ok, err := conditionFunc(ctx)
			if err != nil {
				return fmt.Errorf("eventually check failed during polling: %w", err)
			}
			if ok {
				return nil
			}
		}
	}
}

// KedaConsistently checks if the provided conditionFunc consistently returns true
// (and no error) for the entire duration specified by the context's deadline.
// It polls the conditionFunc at the given interval.
func KedaConsistently(ctx context.Context, conditionFunc wait.ConditionWithContextFunc, interval time.Duration) error {
	if interval <= 0 {
		return fmt.Errorf("polling interval must be positive, got %v", interval)
	}

	ok, err := conditionFunc(ctx)
	if err != nil {
		return fmt.Errorf("consistency check failed on initial check: %w", err)
	}
	if !ok {
		return fmt.Errorf("consistency check failed: condition was false on initial check")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			ok, err := conditionFunc(ctx)
			if err != nil {
				return fmt.Errorf("consistency check failed during polling: %w", err)
			}
			if !ok {
				return fmt.Errorf("consistency check failed: condition returned false during polling period")
			}
		}
	}
}

// EventWatchResult holds the outcome of the event watch.
type EventWatchResult struct {
	Event corev1.Event
	Err   error
}

// StartEventWatch starts a watch for Kubernetes events matching the specified criteria.
// It returns a channel that will receive the first matching event or an error if the watch fails.
func StartEventWatch(
	ctx context.Context,
	t *testing.T,
	clientset *kubernetes.Clientset,
	namespace string,
	involvedObjectName string,
	involvedObjectKind string,
	expectedReason string,
	expectedType string,
	expectedMessages []string,
	resourceVersion string,
) <-chan EventWatchResult {
	t.Helper()
	resultChan := make(chan EventWatchResult, 1)

	fieldSelectorMap := fields.Set{
		"involvedObject.name": involvedObjectName,
		"involvedObject.kind": involvedObjectKind,
		"reason":              expectedReason,
		"type":                expectedType,
	}
	fieldSelector := fieldSelectorMap.String()

	listOptions := metav1.ListOptions{
		FieldSelector:   fieldSelector,
		ResourceVersion: resourceVersion,
	}

	t.Logf("Starting event watch goroutine: namespace=%s, selector='%s', messages=%v, resourceVersion=%s",
		namespace, fieldSelector, expectedMessages, resourceVersion)

	watcher, err := clientset.CoreV1().Events(namespace).Watch(ctx, listOptions)
	assert.NoErrorf(t, err, "failed to start watch for events with selector '%s' and rv %s: %s", fieldSelector, resourceVersion, err)
	if err != nil {
		resultChan <- EventWatchResult{Err: err}
		close(resultChan)
		return resultChan
	}

	go func() {
		defer close(resultChan)
		defer watcher.Stop()

		for {
			select {
			case <-ctx.Done():
				err := fmt.Errorf("watch context cancelled or timed out while waiting for event: %w", ctx.Err())
				resultChan <- EventWatchResult{Err: err}
				return

			case watchEvent, ok := <-watcher.ResultChan():
				if !ok {
					err := fmt.Errorf("watcher channel closed unexpectedly for selector '%s'", fieldSelector)
					if ctx.Err() != nil {
						err = fmt.Errorf("watcher channel closed, likely due to context timeout: %w", ctx.Err())
					}
					resultChan <- EventWatchResult{Err: err}
					return
				}

				switch watchEvent.Type {
				case watch.Added, watch.Modified:
					event, ok := watchEvent.Object.(*corev1.Event)
					if !ok {
						t.Logf("Received non-Event object type (%T) from watch, skipping", watchEvent.Object)
						continue
					}

					t.Logf("Watch goroutine received event: %s/%s - %s %s (ts: %s): %s",
						event.InvolvedObject.Kind, event.InvolvedObject.Name, event.Type, event.Reason,
						event.LastTimestamp.Format(time.RFC3339), event.Message)

					messageMatched := false
					for _, expectedMsg := range expectedMessages {
						if strings.Contains(event.Message, expectedMsg) {
							messageMatched = true
							break
						}
					}
					if !messageMatched {
						t.Logf(" -> Skipping event: Message '%s' did not match any of expected %v", event.Message, expectedMessages)
						continue
					}

					t.Logf("Watch goroutine found matching event: %s/%s - %s %s: %s",
						event.InvolvedObject.Kind, event.InvolvedObject.Name, event.Type, event.Reason, event.Message)
					resultChan <- EventWatchResult{Event: *event}
					return

				case watch.Error:
					status, ok := watchEvent.Object.(*metav1.Status)
					var errMsg string
					if ok {
						errMsg = fmt.Sprintf("received watch.Error status: %s", status.Message)
					} else {
						errMsg = fmt.Sprintf("received watch.Error with unexpected object type %T: %v", watchEvent.Object, watchEvent.Object)
					}
					resultChan <- EventWatchResult{Err: fmt.Errorf("%s", errMsg)}
					return

				case watch.Deleted, watch.Bookmark:
					continue
				}
			}
		}
	}()

	return resultChan
}

// WatchForEventAfterTrigger starts watching for a specific Kubernetes event,
// executes a trigger function, and then waits for the event to appear.
func WatchForEventAfterTrigger(
	t *testing.T,
	kc *kubernetes.Clientset,
	namespace string,
	involvedObjectName string,
	involvedObjectKind string,
	expectedReason string,
	expectedType string,
	expectedMessages []string, // matches any of the messages
	watchTimeout time.Duration,
	triggerAction func() error,
) {
	t.Helper()

	watchCtx, watchCancel := context.WithTimeout(context.Background(), watchTimeout)
	defer watchCancel()

	initialListOptions := metav1.ListOptions{
		FieldSelector: fields.Set{
			"involvedObject.name": involvedObjectName,
			"involvedObject.kind": involvedObjectKind,
			"reason":              expectedReason,
			"type":                expectedType,
		}.String(),
		Limit: 1,
	}

	eventsList, err := kc.CoreV1().Events(namespace).List(watchCtx, initialListOptions)
	require.NoError(t, err, "failed to get initial resource version for watch")

	// Use the resource version from the initial list to start the watch
	// This ensures we only get events that occur after the initial list
	initialResourceVersion := eventsList.ResourceVersion

	t.Logf("Waiting up to %s for Kubernetes event '%s' via watch...", watchTimeout, expectedReason)

	resultChan := StartEventWatch(
		watchCtx,
		t,
		kc,
		namespace,
		involvedObjectName,
		involvedObjectKind,
		expectedReason,
		expectedType,
		expectedMessages,
		initialResourceVersion,
	)

	err = triggerAction()
	require.NoError(t, err, "Trigger action failed")

	select {
	case result := <-resultChan:
		assert.NoError(t, result.Err, "Event watch failed for %s", expectedReason)
		if result.Err == nil {
			t.Logf("Received expected Kubernetes event via watch: %s/%s - %s %s: %s",
				result.Event.InvolvedObject.Kind, result.Event.InvolvedObject.Name, result.Event.Type, result.Event.Reason, result.Event.Message)
		}

	case <-time.After(watchTimeout + 5*time.Second):
		assert.Fail(t, "Timed out waiting for result from Kubernetes event watch channel")
	}
}

func ExtractPrometheusLabelValue(key string, labels []*prommodel.LabelPair) string {
	for _, label := range labels {
		if label.Name != nil && *label.Name == key {
			return *label.Value
		}
	}
	return ""
}
