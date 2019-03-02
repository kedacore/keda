package build

import (
	"fmt"
	"sync"
	"time"

	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	clientset "github.com/knative/build/pkg/client/clientset/versioned"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	done    = make(map[string]chan bool)
	doneMut = sync.Mutex{}
)

// TimeoutSet contains required k8s interfaces to handle build timeouts
type TimeoutSet struct {
	logger         *zap.SugaredLogger
	kubeclientset  kubernetes.Interface
	buildclientset clientset.Interface
	stopCh         <-chan struct{}
}

// NewTimeoutHandler returns TimeoutSet filled structure
func NewTimeoutHandler(logger *zap.SugaredLogger,
	kubeclientset kubernetes.Interface,
	buildclientset clientset.Interface,
	stopCh <-chan struct{}) *TimeoutSet {
	return &TimeoutSet{
		logger:         logger,
		kubeclientset:  kubeclientset,
		buildclientset: buildclientset,
		stopCh:         stopCh,
	}
}

// CheckTimeouts walks through all builds and creates t.wait goroutines that handles build timeout
func (t *TimeoutSet) CheckTimeouts() {
	namespaces, err := t.kubeclientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		t.logger.Errorf("Can't get namespaces list: %s", err)
	}
	for _, namespace := range namespaces.Items {
		builds, err := t.buildclientset.BuildV1alpha1().Builds(namespace.GetName()).List(metav1.ListOptions{})
		if err != nil {
			t.logger.Errorf("Can't get builds list: %s", err)
		}
		for _, build := range builds.Items {
			build := build
			if isDone(&build.Status) {
				continue
			}
			if isCancelled(build.Spec) {
				continue
			}
			go t.wait(&build)
		}
	}
}

func (t *TimeoutSet) wait(build *v1alpha1.Build) {
	key := fmt.Sprintf("%s/%s", build.Namespace, build.Name)
	timeout := defaultTimeout
	if build.Spec.Timeout != nil {
		timeout = build.Spec.Timeout.Duration
	}
	runtime := time.Duration(0)
	statusLock(build)
	if build.Status.StartTime != nil && !build.Status.StartTime.Time.IsZero() {
		runtime = time.Since(build.Status.StartTime.Time)
	}
	statusUnlock(build)
	timeout -= runtime

	finished := make(chan bool)
	doneMut.Lock()
	done[key] = finished
	doneMut.Unlock()
	defer t.release(build)

	select {
	case <-t.stopCh:
	case <-finished:
	case <-time.After(timeout):
		if err := t.stopBuild(build); err != nil {
			t.logger.Errorf("Can't stop build %q after timeout: %s", build.Name, err)
		}
	}
}

func (t *TimeoutSet) release(build *v1alpha1.Build) {
	doneMut.Lock()
	defer doneMut.Unlock()
	key := fmt.Sprintf("%s/%s", build.Namespace, build.Name)
	if finished, ok := done[key]; ok {
		delete(done, key)
		close(finished)
	}
}

func (t *TimeoutSet) stopBuild(build *v1alpha1.Build) error {
	statusLock(build)
	defer statusUnlock(build)
	if build.Status.Cluster != nil {
		if err := t.kubeclientset.CoreV1().Pods(build.Namespace).Delete(build.Status.Cluster.PodName, &metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}

	timeout := defaultTimeout
	if build.Spec.Timeout != nil {
		timeout = build.Spec.Timeout.Duration
	}
	build.Status.SetCondition(&duckv1alpha1.Condition{
		Type:    v1alpha1.BuildSucceeded,
		Status:  corev1.ConditionFalse,
		Reason:  "BuildTimeout",
		Message: fmt.Sprintf("Build %q failed to finish within %q", build.Name, timeout.String()),
	})
	build.Status.CompletionTime = &metav1.Time{time.Now()}

	newb, err := t.buildclientset.BuildV1alpha1().Builds(build.Namespace).Get(build.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	newb.Status = build.Status
	_, err = t.buildclientset.BuildV1alpha1().Builds(build.Namespace).UpdateStatus(newb)
	return err
}
