package github_runner_on_gitea_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// waitForDeployment polls the Deployment in the given namespace until all replicas are ready.
func waitForDeployment(ctx context.Context, clientset *kubernetes.Clientset, namespace, deployName string) error {
	return wait.PollUntilContextTimeout(ctx, 2*time.Second, 90*time.Second, true, func(ctx context.Context) (bool, error) {
		deploy, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deployName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		// Check rollout status (you can adjust the conditions as needed)
		if deploy.Status.UpdatedReplicas == *deploy.Spec.Replicas &&
			deploy.Status.ReadyReplicas == *deploy.Spec.Replicas &&
			deploy.Status.AvailableReplicas == *deploy.Spec.Replicas {
			return true, nil
		}
		log.Printf("Waiting for deployment %s: %d/%d ready", deployName, deploy.Status.ReadyReplicas, *deploy.Spec.Replicas)
		return false, nil
	})
}

// getFreePortForward establishes a port forwarding session from a random (ephemeral) local port to remotePort on the pod.
func getFreePortForward(ctx context.Context, clientset *kubernetes.Clientset, config *rest.Config, namespace, deploymentName string, remotePort int) (int, error) {
	// 1. List pods in the namespace that belong to the deployment.
	//    (Here we assume that the pods have a label like "app=gitea-webhook-api".)
	podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", deploymentName),
	})
	if err != nil {
		return 0, err
	}
	if len(podList.Items) == 0 {
		return 0, fmt.Errorf("no pods found for deployment %s", deploymentName)
	}

	// Choose the first pod.
	podName := podList.Items[0].Name
	log.Printf("Forwarding pod %s", podName)

	// 2. Build the URL for port forwarding for this pod.
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, podName)
	// Remove any leading scheme (https:// or http://) from the host
	host := strings.TrimPrefix(config.Host, "https://")
	host = strings.TrimPrefix(host, "http://")
	pfURL := url.URL{Scheme: "https", Host: host, Path: path}

	// 3. Create the SPDY dialer.
	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return 0, err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", &pfURL)

	// 4. Use "0:remotePort" to let the OS pick an ephemeral local port.
	ports := []string{fmt.Sprintf("0:%d", remotePort)}

	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})

	pf, err := portforward.New(dialer, ports, stopChan, readyChan, os.Stdout, os.Stderr)
	if err != nil {
		return 0, err
	}

	// 5. Start forwarding in a goroutine (this call blocks until stopChan is closed).
	go func() {
		// Port forwarder is ready
		if err := pf.ForwardPorts(); err != nil {
			fmt.Printf("Port forwarding error: %v\n", err)
		}
	}()

	// 6. Wait for the port forwarder to be ready (or time out).
	select {
	case <-readyChan:
		// ready
	case <-time.After(10 * time.Second):
		close(stopChan)
		return 0, fmt.Errorf("timeout waiting for port forward")
	}

	go func() {
		<-ctx.Done()
		close(stopChan)
	}()

	// 7. Retrieve the assigned local port.
	fwdPorts, err := pf.GetPorts()
	if err != nil {
		return 0, err
	}
	if len(fwdPorts) == 0 {
		return 0, fmt.Errorf("no forwarded ports found")
	}

	localPort := int(fwdPorts[0].Local)

	// Verify this forwarding is functional
	_, err = http.Get(fmt.Sprintf("http://localhost:%d", fwdPorts[0].Local))
	return localPort, err
}
