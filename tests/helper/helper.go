//go:build e2e

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

	// renovate: datasource=github-releases depName=strimzi/strimzi-kafka-operator
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

	ok := pollUntil(t, iterations, intervalSeconds, func(context.Context) (bool, error) {
		out, errOut, err = ExecCommandOnSpecificPod(t, podName, namespace, command)
		t.Logf("Waiting for successful execution of command on Pod; Output: %s, Error: %s", out, errOut)

		return err == nil, err
	})

	return ok, out, errOut, err
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

// pollUntil polls cond until it holds, giving up once the iterations × intervalSeconds budget is
// spent. The budget is a deadline rather than a count of attempts, so the time the API server spends
// answering comes out of the window instead of quietly extending it. Each attempt is bound to that
// deadline as well, so a read that never returns cannot outlast the wait that asked for it.
//
// The reason the wait ended is logged, because callers only see the bool and would otherwise have
// nothing to go on when a wait that should have succeeded times out.
func pollUntil(t *testing.T, iterations, intervalSeconds int, cond wait.ConditionWithContextFunc) bool {
	// KedaEventually needs a positive interval, and callers pass zero when they want a single check.
	interval := time.Duration(intervalSeconds) * time.Second
	if interval <= 0 {
		interval = IntervalShort
	}

	// One interval is the floor: with an already spent budget the first attempt would be handed an
	// expired context and fail without ever reading anything.
	budget := time.Duration(iterations*intervalSeconds) * time.Second
	if budget < interval {
		budget = interval
	}

	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()

	// A read the deadline cancelled says nothing about the resource being waited on, so it is not
	// reported as the reason the wait ended: that reads like a broken API server rather than a
	// condition that stayed false.
	bounded := func(ctx context.Context) (bool, error) {
		ok, err := cond(ctx)
		if ctx.Err() != nil {
			return ok, nil
		}
		return ok, err
	}

	if err := KedaEventually(ctx, bounded, interval); err != nil {
		t.Log(err)
		return false
	}

	return true
}

// The wait helpers below share a rule: a read that failed must not be mistaken for an
// observation. The generated clients return a non-nil zero value alongside the error, so reading
// on regardless reports 0 replicas, 0 items or an absent status, which is indistinguishable from
// the state a wait for zero is looking for. Each of them hands the error back instead, which leaves
// the wait running and names the cause if the deadline arrives with the read still failing.

func WaitForJobSuccess(t *testing.T, kc *kubernetes.Clientset, jobName, namespace string, iterations, interval int) bool {
	return pollUntil(t, iterations, interval, func(ctx context.Context) (bool, error) {
		job, err := kc.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("cannot get job %s/%s - %w", namespace, jobName, err)
		}

		if job.Status.Succeeded == 0 {
			return false, nil
		}

		t.Logf("job %s ran successfully!", jobName)
		return true, nil
	})
}

func WaitForAllJobsSuccess(t *testing.T, kc *kubernetes.Clientset, namespace string, iterations, interval int) bool {
	return pollUntil(t, iterations, interval, func(ctx context.Context) (bool, error) {
		jobs, err := kc.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, fmt.Errorf("cannot list jobs in namespace %s - %w", namespace, err)
		}

		// Without this an empty namespace reports every job successful, so a list that came
		// back before the jobs were created satisfies the wait.
		allJobsSuccess := len(jobs.Items) > 0
		for _, job := range jobs.Items {
			if job.Status.Succeeded == 0 {
				allJobsSuccess = false
				break
			}
		}

		if !allJobsSuccess {
			return false, nil
		}

		t.Logf("all jobs ran successfully!")
		return true, nil
	})
}

// DeleteAllJobsInNamespace removes all Jobs (and their pods, via background propagation) in the
// namespace, so a phase can reset the Job count for the next phase's assertions.
func DeleteAllJobsInNamespace(t *testing.T, kc *kubernetes.Clientset, namespace string) {
	policy := metav1.DeletePropagationBackground
	err := kc.BatchV1().Jobs(namespace).DeleteCollection(context.Background(),
		metav1.DeleteOptions{PropagationPolicy: &policy}, metav1.ListOptions{})
	if err != nil {
		t.Logf("cannot delete jobs in namespace %s - %s", namespace, err)
	}
}

func WaitForNamespaceDeletion(t *testing.T, nsName string) bool {
	return pollUntil(t, 120, 5, func(ctx context.Context) (bool, error) {
		t.Logf("waiting for namespace %s deletion", nsName)
		_, err := GetKubernetesClient(t).CoreV1().Namespaces().Get(ctx, nsName, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil
		}

		// A nil error means the namespace is still there, anything else means this attempt could
		// not tell either way.
		return false, err
	})
}

func WaitForScaledJobCount(t *testing.T, kc *kubernetes.Clientset, scaledJobName, namespace string, target, iterations, intervalSeconds int) bool {
	return waitForJobCount(t, kc, fmt.Sprintf("scaledjob.keda.sh/name=%s", scaledJobName), namespace, target, iterations, intervalSeconds)
}

func WaitForJobCount(t *testing.T, kc *kubernetes.Clientset, namespace string, target, iterations, intervalSeconds int) bool {
	return waitForJobCount(t, kc, "", namespace, target, iterations, intervalSeconds)
}

func waitForJobCount(t *testing.T, kc *kubernetes.Clientset, selector, namespace string, target, iterations, intervalSeconds int) bool {
	return pollUntil(t, iterations, intervalSeconds, func(ctx context.Context) (bool, error) {
		jobList, err := kc.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: selector,
		})
		if err != nil {
			return false, fmt.Errorf("cannot list jobs in namespace %s - %w", namespace, err)
		}

		count := len(jobList.Items)

		t.Logf("Waiting for job count to hit target. Namespace - %s, Current  - %d, Target - %d",
			namespace, count, target)

		return count == target, nil
	})
}

func WaitForJobCreation(t *testing.T, kc *kubernetes.Clientset, scaledJobName, namespace string, iterations, intervalSeconds int) (*batchv1.Job, error) {
	var job *batchv1.Job

	ok := pollUntil(t, iterations, intervalSeconds, func(ctx context.Context) (bool, error) {
		jobList, err := kc.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("scaledjob.keda.sh/name=%s", scaledJobName),
		})
		if err != nil {
			return false, fmt.Errorf("cannot list jobs in namespace %s - %w", namespace, err)
		}

		t.Log("Waiting for job creation")

		if len(jobList.Items) == 0 {
			return false, nil
		}

		job = &jobList.Items[0]
		return true, nil
	})

	if !ok {
		// The caller reads fields off the job, so running out of budget has to be an error even
		// when the last read succeeded and simply found nothing.
		return nil, fmt.Errorf("no job was created for scaledjob %s/%s", namespace, scaledJobName)
	}

	return job, nil
}

// Waits until deployment count hits target or number of iterations are done.
func WaitForPodCountInNamespace(t *testing.T, kc *kubernetes.Clientset, namespace string, target, iterations, intervalSeconds int) bool {
	return pollUntil(t, iterations, intervalSeconds, func(ctx context.Context) (bool, error) {
		pods, err := kc.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, fmt.Errorf("cannot list pods in namespace %s - %w", namespace, err)
		}

		t.Logf("Waiting for pods in namespace to hit target. Namespace - %s, Current  - %d, Target - %d",
			namespace, len(pods.Items), target)

		return len(pods.Items) == target, nil
	})
}

// Waits until all the pods with a defined label are in completed status.
func WaitForPodsCompleted(t *testing.T, kc *kubernetes.Clientset, selector, namespace string, iterations, intervalSeconds int) bool {
	return pollUntil(t, iterations, intervalSeconds, func(ctx context.Context) (bool, error) {
		pods, err := kc.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
		switch {
		case err != nil && !errors.IsNotFound(err):
			return false, fmt.Errorf("cannot list pods with label %s in namespace %s - %w", selector, namespace, err)
		case errors.IsNotFound(err) || len(pods.Items) == 0:
			t.Logf("No pods with label %s", selector)
			return true, nil
		}

		succeededCount := 0
		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodSucceeded {
				t.Logf("Pod %s in namespace %s is in completed status", pod.Name, namespace)
				succeededCount++
			}
		}

		if succeededCount == len(pods.Items) {
			return true, nil
		}

		t.Logf("Waiting for pods with label %s to complete", selector)
		return false, nil
	})
}

// isPodReady reports whether the pod's Ready condition is true. The Running phase only means the
// pod's containers have been created, so it is reached before a readiness probe first succeeds and
// before the pod is reachable through a Service.
func isPodReady(pod corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady {
			return cond.Status == corev1.ConditionTrue
		}
	}
	return false
}

// Waits until all the pods in the namespace are ready.
func WaitForAllPodRunningInNamespace(t *testing.T, kc *kubernetes.Clientset, namespace string, iterations, intervalSeconds int) bool {
	return pollUntil(t, iterations, intervalSeconds, func(ctx context.Context) (bool, error) {
		pods, err := kc.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, fmt.Errorf("cannot list pods in namespace %s - %w", namespace, err)
		}

		readyCount := 0
		for _, pod := range pods.Items {
			if !isPodReady(pod) {
				break
			}
			readyCount++
		}

		t.Logf("Waiting for pods in namespace to be ready. Namespace - %s, Current - %d, Target - %d",
			namespace, readyCount, len(pods.Items))

		// Every caller creates the pods it is waiting on, so an empty namespace means the
		// list arrived before they were scheduled rather than that everything is ready.
		return len(pods.Items) > 0 && readyCount == len(pods.Items), nil
	})
}

func WaitForRunningPodCount(t *testing.T, kc *kubernetes.Clientset, scaledJobName, namespace string, target, iterations, interval int) bool {
	// Waiting cannot bring an overshoot back down to the target, and the callers are asserting on
	// an exact number of concurrently running pods, so it is a result rather than a state to wait
	// out. The condition reports it as met to stop the poll, and it is turned into a false below.
	overshot := false

	reached := pollUntil(t, iterations, interval, func(ctx context.Context) (bool, error) {
		pods, err := kc.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("scaledjob.keda.sh/name=%s", scaledJobName),
		})
		if err != nil {
			return false, fmt.Errorf("cannot list pods in namespace %s - %w", namespace, err)
		}

		// Phase rather than readiness on purpose: these are ScaledJob pods, and the callers are
		// counting how many the executor started, not whether each one is serving traffic.
		runningPodCount := 0
		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodRunning {
				runningPodCount++
			}
		}

		t.Logf("Waiting for running pods. Namespace - %s, Current - %d, Target - %d",
			namespace, runningPodCount, target)

		if runningPodCount > target {
			overshot = true
			return true, nil
		}

		return runningPodCount == target, nil
	})

	return reached && !overshot
}

// Waits until the Horizontal Pod Autoscaler for the scaledObject reports that it has metrics available
// to calculate, or until the number of iterations are done, whichever happens first.
func WaitForHPAMetricsToPopulate(t *testing.T, kc *kubernetes.Clientset, name, namespace string, iterations, intervalSeconds int) bool {
	totalWaitDuration := time.Duration(iterations) * time.Duration(intervalSeconds) * time.Second
	startedWaiting := time.Now()

	return pollUntil(t, iterations, intervalSeconds, func(ctx context.Context) (bool, error) {
		t.Logf("Waiting up to %s for HPA to populate metrics - %s so far", totalWaitDuration, time.Since(startedWaiting).Round(time.Second))

		hpa, err := kc.AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("cannot get hpa %s/%s - %w", namespace, name, err)
		}

		for _, currentMetric := range hpa.Status.CurrentMetrics {
			// When testing on a kind cluster at least, an empty metricStatus object with a blank type shows up first,
			// so we need to make sure we have *actual* resource metrics before we return
			if currentMetric.Type != "" {
				j, _ := json.MarshalIndent(hpa.Status.CurrentMetrics, "  ", "    ")
				t.Logf("HPA has metrics after %s: %s", time.Since(startedWaiting), j)
				return true, nil
			}
		}

		return false, nil
	})
}

// Waits until deployment ready replica count hits target or number of iterations are done.
func WaitForDeploymentReplicaReadyCount(t *testing.T, kc *kubernetes.Clientset, name, namespace string, target, iterations, intervalSeconds int) bool {
	return pollUntil(t, iterations, intervalSeconds, func(ctx context.Context) (bool, error) {
		deployment, err := kc.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("cannot get deployment %s/%s - %w", namespace, name, err)
		}

		status := deployment.Status

		t.Logf("Waiting for deployment replicas to hit target. Deployment - %s, ObservedGeneration - %d/%d, ReadyReplicas - %d, UpdatedReplicas - %d, Target - %d",
			name, status.ObservedGeneration, deployment.Generation, status.ReadyReplicas, status.UpdatedReplicas, target)

		// ObservedGeneration rejects a status written before the current spec, and UpdatedReplicas
		// rejects pods built from a previous template. Without both, a wait that follows a template
		// change can be satisfied by the very pods that change is replacing.
		return status.ObservedGeneration >= deployment.Generation &&
			status.ReadyReplicas == int32(target) &&
			status.UpdatedReplicas == int32(target), nil
	})
}

// Waits until pod is in ready state or number of iterations are done.
func WaitForPodReady(t *testing.T, kc *kubernetes.Clientset, podName, namespace string, iterations, intervalSeconds int) bool {
	return pollUntil(t, iterations, intervalSeconds, func(ctx context.Context) (bool, error) {
		pod, err := kc.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("cannot get pod %s/%s - %w", namespace, podName, err)
		}

		t.Logf("Waiting for pod to be in ready state. Pod - %s, Current Phase - %s", podName, pod.Status.Phase)

		return isPodReady(*pod), nil
	})
}

// Waits until rollout ready replica count hits target or number of iterations are done.
func WaitForArgoRolloutReplicaReadyCount(t *testing.T, _ *kubernetes.Clientset, name, namespace string, target, iterations, intervalSeconds int) bool {
	return pollUntil(t, iterations, intervalSeconds, func(context.Context) (bool, error) {
		// If target==0, we check for spec replicas, since .status.readyReplicas won't be set by the controller.
		jsonPath := ".status.readyReplicas"
		if target == 0 {
			jsonPath = ".spec.replicas"
		}

		kctlGetCmd := fmt.Sprintf(`kubectl get rollouts.argoproj.io/%s -n %s -o jsonpath="{%s}"`, name, namespace, jsonPath)
		output, err := ExecuteCommand(kctlGetCmd)
		if err != nil {
			// One failed invocation is a lost observation, not a failed test: the rollout can still
			// reach the target within the remaining budget.
			return false, fmt.Errorf("cannot get rollout %s/%s - %w", namespace, name, err)
		}

		unquotedOutput := strings.ReplaceAll(string(output), "\"", "")

		// Length of output can be zero, which means .status.readyReplicas is not yet set by the controller.
		if len(unquotedOutput) == 0 {
			return false, nil
		}

		replicas, err := strconv.ParseInt(unquotedOutput, 10, 64)
		if err != nil {
			return false, fmt.Errorf("cannot convert rollout count %q to int - %w", unquotedOutput, err)
		}

		t.Logf("Waiting for rollout replicas to hit target. Rollout - %s, Current  - %d, Target - %d",
			name, replicas, target)

		return replicas == int64(target), nil
	})
}

// Waits until statefulset count hits target or number of iterations are done.
func WaitForStatefulsetReplicaReadyCount(t *testing.T, kc *kubernetes.Clientset, name, namespace string, target, iterations, intervalSeconds int) bool {
	return pollUntil(t, iterations, intervalSeconds, func(ctx context.Context) (bool, error) {
		statefulset, err := kc.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("cannot get statefulset %s/%s - %w", namespace, name, err)
		}

		status := statefulset.Status

		t.Logf("Waiting for statefulset replicas to hit target. Statefulset - %s, ObservedGeneration - %d/%d, ReadyReplicas - %d, UpdatedReplicas - %d, Target - %d",
			name, status.ObservedGeneration, statefulset.Generation, status.ReadyReplicas, status.UpdatedReplicas, target)

		// Same reasoning as the deployment helper above: the generation rules out a stale status,
		// and UpdatedReplicas rules out pods still running the previous revision.
		return status.ObservedGeneration >= statefulset.Generation &&
			status.ReadyReplicas == int32(target) &&
			status.UpdatedReplicas == int32(target), nil
	})
}

// WaitForReplicaSetReplicaReadyCount waits until replicaset replica count hits target or number of iterations are done.
func WaitForReplicaSetReplicaReadyCount(t *testing.T, kc *kubernetes.Clientset, name, namespace string, target, iterations, intervalSeconds int) bool {
	return pollUntil(t, iterations, intervalSeconds, func(ctx context.Context) (bool, error) {
		rs, err := kc.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("cannot get replicaset %s/%s - %w", namespace, name, err)
		}

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

		return replicas == int32(target), nil
	})
}

// Waits for number of iterations and returns replica count.
func WaitForDeploymentReplicaCountChange(t *testing.T, kc *kubernetes.Clientset, name, namespace string, iterations, intervalSeconds int) int {
	t.Log("Waiting for some time to see if deployment replica count changes")
	var replicas, prevReplicas int32
	prevReplicas = -1

	for i := 0; i < iterations; i++ {
		deployment, err := kc.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			// Reading on would report 0 replicas, which the caller cannot tell apart from a
			// genuine scale to zero.
			t.Logf("cannot get deployment %s/%s - %s", namespace, name, err)
		} else {
			replicas = deployment.Status.Replicas

			t.Logf("Deployment - %s, Current  - %d", name, replicas)

			if replicas != prevReplicas && prevReplicas != -1 {
				break
			}

			prevReplicas = replicas
		}

		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}

	return int(replicas)
}

// Waits some time to ensure that the replica count doesn't change.
func AssertReplicaCountNotChangeDuringTimePeriod(t *testing.T, kc *kubernetes.Clientset, name, namespace string, target, intervalSeconds int) {
	t.Logf("Waiting for some time to ensure deployment replica count doesn't change from %d", target)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(intervalSeconds)*time.Second)
	defer cancel()

	var replicas int32
	err := KedaConsistently(ctx, func(ctx context.Context) (bool, error) {
		deployment, err := kc.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			// Reading on would report 0 replicas and fail the assertion with a count the deployment
			// never had. KedaConsistently fails on a condition error, so a read that could not be
			// made is reported as an attempt that saw nothing to object to.
			t.Logf("cannot get deployment %s/%s - %s", namespace, name, err)
			return true, nil
		}

		replicas = deployment.Status.Replicas

		t.Logf("Deployment - %s, Current  - %d", name, replicas)

		return replicas == int32(target), nil
	}, IntervalShort)

	if err != nil {
		assert.Fail(t, fmt.Sprintf("%s replica count has changed from %d to %d", name, target, replicas))
	}
}

// Waits some time to ensure that the replica count doesn't change.
func AssertReplicaCountNotChangeDuringTimePeriodRollout(t *testing.T, _ *kubernetes.Clientset, name, namespace string, target, intervalSeconds int) {
	t.Logf("Waiting for some time to ensure rollout replica count doesn't change from %d", target)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(intervalSeconds)*time.Second)
	defer cancel()

	// Held rather than asserted inside the condition, so that whichever attempt objects reports
	// once instead of once per second for the rest of the period.
	var failure string

	err := KedaConsistently(ctx, func(context.Context) (bool, error) {
		// If target==0, we check for spec replicas, since .status.replicas won't be set by the controller.
		jsonPath := ".status.replicas"
		if target == 0 {
			jsonPath = ".spec.replicas"
		}

		kctlGetCmd := fmt.Sprintf(`kubectl get rollouts.argoproj.io/%s -n %s -o jsonpath="{%s}"`, name, namespace, jsonPath)
		output, err := ExecuteCommand(kctlGetCmd)
		if err != nil {
			// One failed invocation is no evidence that the count changed, so it is not the
			// assertion's business.
			t.Logf("cannot get rollout %s/%s - %s", namespace, name, err)
			return true, nil
		}

		unquotedOutput := strings.ReplaceAll(string(output), "\"", "")

		// Length of output can be zero, which means .status.replicas is not set by the controller.
		if len(unquotedOutput) == 0 {
			failure = fmt.Sprintf("%s replicas are not set in its status, expected %d", name, target)
			return false, nil
		}

		replicas, err := strconv.ParseInt(unquotedOutput, 10, 64)
		if err != nil {
			failure = fmt.Sprintf("cannot convert rollout count %q to int - %s", unquotedOutput, err)
			return false, nil
		}

		t.Logf("Rollout - %s, Current  - %d", name, replicas)

		if replicas != int64(target) {
			failure = fmt.Sprintf("%s replica count has changed from %d to %d", name, target, replicas)
			return false, nil
		}

		return true, nil
	}, IntervalShort)

	if err != nil {
		assert.Fail(t, failure)
	}
}

func WaitForHpaCreation(t *testing.T, kc *kubernetes.Clientset, name, namespace string, iterations, intervalSeconds int) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	// Callers read fields off the result whether or not the wait succeeded, and one of them expects
	// the wait to fail, so an empty HPA stands in until a read succeeds.
	hpa := &autoscalingv2.HorizontalPodAutoscaler{}

	// The API error is handed back rather than described, because custom_hpa_name waits for an HPA
	// it expects to be gone and asserts errors.IsNotFound on the result. A read the deadline
	// cancelled is not kept: it says nothing about the HPA and would mask that NotFound.
	var lastErr error

	ok := pollUntil(t, iterations, intervalSeconds, func(ctx context.Context) (bool, error) {
		t.Log("Waiting for hpa creation")

		found, err := kc.AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if ctx.Err() == nil {
				lastErr = err
			}
			return false, err
		}

		hpa = found
		return true, nil
	})

	if !ok {
		if lastErr != nil {
			return hpa, lastErr
		}
		return hpa, fmt.Errorf("hpa %s/%s was not created", namespace, name)
	}

	return hpa, nil
}

func KubernetesScaleDeployment(t *testing.T, kc *kubernetes.Clientset, name string, desiredReplica int64, namespace string) {
	// Scaling an empty Scale read back from a failed request would ask the API server to set
	// replicas on a deployment with no name, so there is nothing useful to do but report it.
	scaleObject, err := kc.AppsV1().Deployments(namespace).GetScale(context.Background(), name, metav1.GetOptions{})
	if !assert.NoErrorf(t, err, "couldn't read the scale of deployment %s/%s - %s", namespace, name, err) {
		return
	}

	sc := *scaleObject
	sc.Spec.Replicas = int32(desiredReplica)
	_, err = kc.AppsV1().Deployments(namespace).UpdateScale(context.Background(), name, &sc, metav1.UpdateOptions{})
	assert.NoErrorf(t, err, "couldn't scale deployment %s/%s - %s", namespace, name, err)
}

func SetDeploymentContainerArg(t *testing.T, kc *kubernetes.Clientset, name, namespace, containerName, flagName, flagValue string) {
	t.Helper()

	deployment, err := kc.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	require.NoErrorf(t, err, "cannot get deployment %s/%s - %s", namespace, name, err)

	newArg := fmt.Sprintf("%s=%s", flagName, flagValue)
	var updated bool
	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name != containerName {
			continue
		}

		replaced := false
		for j, arg := range container.Args {
			if arg == flagName || strings.HasPrefix(arg, flagName+"=") {
				container.Args[j] = newArg
				replaced = true
				break
			}
		}
		if !replaced {
			container.Args = append(container.Args, newArg)
		}

		deployment.Spec.Template.Spec.Containers[i].Args = container.Args
		updated = true
		break
	}

	require.Truef(t, updated, "container %s not found in deployment %s/%s", containerName, namespace, name)

	updatedDeployment, err := kc.AppsV1().Deployments(namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{})
	require.NoErrorf(t, err, "cannot update deployment %s/%s - %s", namespace, name, err)

	targetReplicas := int32(1)
	if updatedDeployment.Spec.Replicas != nil {
		targetReplicas = *updatedDeployment.Spec.Replicas
	}

	assert.True(t, WaitForDeploymentRollout(t, kc, name, namespace, updatedDeployment.Generation, targetReplicas, 60, 2),
		"deployment %s/%s should finish rollout after updating %s", namespace, name, flagName)
}

func WaitForDeploymentRollout(t *testing.T, kc *kubernetes.Clientset, name, namespace string, generation int64, targetReplicas int32, iterations, intervalSeconds int) bool {
	t.Helper()

	return pollUntil(t, iterations, intervalSeconds, func(ctx context.Context) (bool, error) {
		deployment, err := kc.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("cannot get deployment %s/%s - %w", namespace, name, err)
		}

		t.Logf(
			"Waiting for deployment rollout. Deployment - %s, ObservedGeneration - %d/%d, Replicas - %d, UpdatedReplicas - %d, ReadyReplicas - %d, AvailableReplicas - %d, Target - %d",
			name,
			deployment.Status.ObservedGeneration,
			generation,
			deployment.Status.Replicas,
			deployment.Status.UpdatedReplicas,
			deployment.Status.ReadyReplicas,
			deployment.Status.AvailableReplicas,
			targetReplicas,
		)

		return deployment.Status.ObservedGeneration >= generation &&
			deployment.Status.Replicas == targetReplicas &&
			deployment.Status.UpdatedReplicas == targetReplicas &&
			deployment.Status.ReadyReplicas == targetReplicas &&
			deployment.Status.AvailableReplicas == targetReplicas, nil
	})
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
	return pollUntil(t, iterations, intervalSeconds, func(ctx context.Context) (bool, error) {
		pods, err := kc.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
		switch {
		case err != nil && !errors.IsNotFound(err):
			return false, fmt.Errorf("cannot list pods with label %s in namespace %s - %w", selector, namespace, err)
		case errors.IsNotFound(err) || len(pods.Items) == 0:
			t.Logf("No pods with label %s", selector)
			return true, nil
		}

		t.Logf("Waiting for pods with label %s to terminate", selector)
		return false, nil
	})
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

// The fields this reads are written by the operator as it reconciles the resource that references
// the authentication, so they lag the apply or delete that triggered the change. A minute is many
// times a reconcile and is only reached when the operator has genuinely stopped updating them.
const kubectlGetResultTimeout = time.Minute

// CheckKubectlGetResult runs `kubectl get` with parameters and waits for the output to match the
// expected value.
func CheckKubectlGetResult(t *testing.T, kind string, name string, namespace string, otherparameter string, expected string) {
	kctlGetCmd := fmt.Sprintf(`kubectl get %s/%s -n %s %s"`, kind, name, namespace, otherparameter)
	t.Log("Running kubectl cmd:", kctlGetCmd)

	ctx, cancel := context.WithTimeout(context.Background(), kubectlGetResultTimeout)
	defer cancel()

	lastRead := "<nothing read>"
	err := KedaEventually(ctx, func(_ context.Context) (bool, error) {
		output, cmdErr := ExecuteCommand(kctlGetCmd)
		if cmdErr != nil {
			// Retried rather than reported, so that a command that fails while the resource is
			// still being created does not end the wait.
			lastRead = fmt.Sprintf("<command failed: %s>", cmdErr)
			return false, nil
		}

		lastRead = strings.ReplaceAll(string(output), "\"", "")
		return lastRead == expected, nil
	}, IntervalShort)

	assert.NoErrorf(t, err, "%s/%s %s: expected %q, last read %q", kind, name, otherparameter, expected, lastRead)
}

// KedaEventually checks if the provided conditionFunc eventually returns true
// (and no error) within the context's deadline. It polls the conditionFunc
// at the given interval until the condition is met or the context times out.
// An error from conditionFunc is treated as an attempt that could not observe
// the condition, so polling continues; the error is only reported if the
// deadline arrives while the most recent attempt is still failing. Callers may
// therefore return errors directly instead of hiding them behind a false.
func KedaEventually(ctx context.Context, conditionFunc wait.ConditionWithContextFunc, interval time.Duration) error {
	if interval <= 0 {
		return fmt.Errorf("polling interval must be positive, got %v", interval)
	}

	// A condition that returns an error has not been shown to be false, only that it could not be
	// observed this time, so the error is remembered and the wait carries on. It is reported if the
	// deadline arrives while the most recent attempt is still failing, which is when it becomes the
	// likely explanation. A later attempt that observes the condition clears it, so a transient
	// failure early in a long wait does not end up blamed for a condition that simply stayed false.
	var lastErr error

	ok, err := conditionFunc(ctx)
	switch {
	case err != nil:
		lastErr = err
	case ok:
		return nil
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("eventually check failed: context deadline exceeded, last attempt errored: %w", lastErr)
			}
			return fmt.Errorf("eventually check failed: context deadline exceeded before condition was met")

		case <-ticker.C:
			ok, err := conditionFunc(ctx)
			lastErr = err
			if err == nil && ok {
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
