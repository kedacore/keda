package e2e

import (
	"testing"
	"time"

	rnames "github.com/knative/serving/pkg/reconciler/v1alpha1/revision/resources/names"
	"github.com/knative/serving/test"

	v1types "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DiagnoseMeEvery queries the k8s controller and reports the pod stats to the logger,
// every `duration` period.
func DiagnoseMeEvery(t *testing.T, duration time.Duration, clients *test.Clients) chan struct{} {
	stopChan := make(chan struct{})
	go func() {
		c := time.NewTicker(duration)
		for {
			select {
			case <-c.C:
				diagnoseMe(t, clients)
				continue
			case <-stopChan:
				c.Stop()
				return
			}
		}
	}()
	return stopChan
}

func diagnoseMe(t *testing.T, clients *test.Clients) {
	if clients == nil || clients.KubeClient == nil || clients.KubeClient.Kube == nil {
		t.Log("Could not diagnose: nil kube client")
		return
	}

	for _, check := range []func(*testing.T, *test.Clients){
		checkCurrentPodCount,
		checkUnschedulablePods,
	} {
		check(t, clients)
	}
}

func checkCurrentPodCount(t *testing.T, clients *test.Clients) {
	revs, err := clients.ServingClient.Revisions.List(metav1.ListOptions{})
	if err != nil {
		t.Logf("Could not check current pod count: %v", err)
		return
	}
	for _, r := range revs.Items {
		deploymentName := rnames.Deployment(&r)
		dep, err := clients.KubeClient.Kube.AppsV1().Deployments(test.ServingNamespace).Get(deploymentName, metav1.GetOptions{})
		if err != nil {
			t.Logf("Could not get deployment %v", deploymentName)
			continue
		}
		t.Logf("Deployment %s has %d pods. wants %d.", deploymentName, dep.Status.Replicas, dep.Status.ReadyReplicas)
	}
}

func checkUnschedulablePods(t *testing.T, clients *test.Clients) {
	kube := clients.KubeClient.Kube
	pods, err := kube.CoreV1().Pods(test.ServingNamespace).List(metav1.ListOptions{})
	if err != nil {
		t.Logf("Could not check unschedulable pods: %v", err)
		return
	}

	totalPods := len(pods.Items)
	unschedulablePods := 0
	for _, p := range pods.Items {
		for _, c := range p.Status.Conditions {
			if c.Type == v1types.PodScheduled && c.Status == v1types.ConditionFalse && c.Reason == v1types.PodReasonUnschedulable {
				unschedulablePods++
				break
			}
		}
	}
	if unschedulablePods != 0 {
		t.Logf("%v out of %v pods are unschedulable. Insufficient cluster capacity?", unschedulablePods, totalPods)
	}
}
