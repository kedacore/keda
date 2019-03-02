/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/knative/pkg/kmeta"
	"github.com/knative/pkg/logging"
	"github.com/knative/pkg/logging/logkey"
)

const (
	falseString = "false"
	trueString  = "true"
)

// Reconciler is the interface that controller implementations are expected
// to implement, so that the shared controller.Impl can drive work through it.
type Reconciler interface {
	Reconcile(ctx context.Context, key string) error
}

// PassNew makes it simple to create an UpdateFunc for use with
// cache.ResourceEventHandlerFuncs that can delegate the same methods
// as AddFunc/DeleteFunc but passing through only the second argument
// (which is the "new" object).
func PassNew(f func(interface{})) func(interface{}, interface{}) {
	return func(first, second interface{}) {
		f(second)
	}
}

// Filter makes it simple to create FilterFunc's for use with
// cache.FilteringResourceEventHandler that filter based on the
// schema.GroupVersionKind of the controlling resources.
func Filter(gvk schema.GroupVersionKind) func(obj interface{}) bool {
	return func(obj interface{}) bool {
		if object, ok := obj.(metav1.Object); ok {
			owner := metav1.GetControllerOf(object)
			return owner != nil &&
				owner.APIVersion == gvk.GroupVersion().String() &&
				owner.Kind == gvk.Kind
		}
		return false
	}
}

// Impl is our core controller implementation.  It handles queuing and feeding work
// from the queue to an implementation of Reconciler.
type Impl struct {
	// Reconciler is the workhorse of this controller, it is fed the keys
	// from the workqueue to process.  Public for testing.
	Reconciler Reconciler

	// WorkQueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	WorkQueue workqueue.RateLimitingInterface

	// Sugared logger is easier to use but is not as performant as the
	// raw logger. In performance critical paths, call logger.Desugar()
	// and use the returned raw logger instead. In addition to the
	// performance benefits, raw logger also preserves type-safety at
	// the expense of slightly greater verbosity.
	logger *zap.SugaredLogger

	// StatsReporter is used to send common controller metrics.
	statsReporter StatsReporter
}

// NewImpl instantiates an instance of our controller that will feed work to the
// provided Reconciler as it is enqueued.
func NewImpl(r Reconciler, logger *zap.SugaredLogger, workQueueName string, reporter StatsReporter) *Impl {
	return &Impl{
		Reconciler: r,
		WorkQueue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			workQueueName,
		),
		logger:        logger,
		statsReporter: reporter,
	}
}

// Enqueue takes a resource, converts it into a namespace/name string,
// and passes it to EnqueueKey.
func (c *Impl) Enqueue(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		c.logger.Errorw("Enqueue", zap.Error(err))
		return
	}
	c.EnqueueKey(key)
}

// EnqueueControllerOf takes a resource, identifies its controller resource,
// converts it into a namespace/name string, and passes that to EnqueueKey.
func (c *Impl) EnqueueControllerOf(obj interface{}) {
	object, err := kmeta.DeletionHandlingAccessor(obj)
	if err != nil {
		c.logger.Error(err)
		return
	}

	// If we can determine the controller ref of this object, then
	// add that object to our workqueue.
	if owner := metav1.GetControllerOf(object); owner != nil {
		c.EnqueueKey(object.GetNamespace() + "/" + owner.Name)
	}
}

// EnqueueLabelOfNamespaceScopedResource returns with an Enqueue func that
// takes a resource, identifies its controller resource through given namespace
// and name labels, converts it into a namespace/name string, and passes that
// to EnqueueKey. The controller resource must be of namespace-scoped.
func (c *Impl) EnqueueLabelOfNamespaceScopedResource(namespaceLabel, nameLabel string) func(obj interface{}) {
	return func(obj interface{}) {
		object, err := kmeta.DeletionHandlingAccessor(obj)
		if err != nil {
			c.logger.Error(err)
			return
		}

		labels := object.GetLabels()
		controllerKey, ok := labels[nameLabel]
		if !ok {
			c.logger.Debugf("Object %s/%s does not have a referring name label %s",
				object.GetNamespace(), object.GetName(), nameLabel)
			return
		}

		if namespaceLabel != "" {
			controllerNamespace, ok := labels[namespaceLabel]
			if !ok {
				c.logger.Debugf("Object %s/%s does not have a referring namespace label %s",
					object.GetNamespace(), object.GetName(), namespaceLabel)
				return
			}

			c.EnqueueKey(fmt.Sprintf("%s/%s", controllerNamespace, controllerKey))
			return
		}

		// Pass through namespace of the object itself if no namespace label specified.
		// This is for the scenario that object and the parent resource are of same namespace,
		// e.g. to enqueue the revision of an endpoint.
		c.EnqueueKey(fmt.Sprintf("%s/%s", object.GetNamespace(), controllerKey))
	}
}

// EnqueueLabelOfClusterScopedResource returns with an Enqueue func
// that takes a resource, identifies its controller resource through
// given name label, and passes it to EnqueueKey.
// The controller resource must be of cluster-scoped.
func (c *Impl) EnqueueLabelOfClusterScopedResource(nameLabel string) func(obj interface{}) {
	return func(obj interface{}) {
		object, err := kmeta.DeletionHandlingAccessor(obj)
		if err != nil {
			c.logger.Error(err)
			return
		}

		labels := object.GetLabels()
		controllerKey, ok := labels[nameLabel]
		if !ok {
			c.logger.Debugf("Object %s/%s does not have a referring name label %s",
				object.GetNamespace(), object.GetName(), nameLabel)
			return
		}

		c.EnqueueKey(controllerKey)
	}
}

// EnqueueKey takes a namespace/name string and puts it onto the work queue.
func (c *Impl) EnqueueKey(key string) {
	c.WorkQueue.Add(key)
}

// Run starts the controller's worker threads, the number of which is threadiness.
// It then blocks until stopCh is closed, at which point it shuts down its internal
// work queue and waits for workers to finish processing their current work items.
func (c *Impl) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	sg := sync.WaitGroup{}
	defer sg.Wait()
	defer c.WorkQueue.ShutDown()

	// Launch workers to process resources that get enqueued to our workqueue.
	logger := c.logger
	logger.Info("Starting controller and workers")
	for i := 0; i < threadiness; i++ {
		sg.Add(1)
		go func() {
			defer sg.Done()
			for c.processNextWorkItem() {
			}
		}()
	}

	logger.Info("Started workers")
	<-stopCh
	logger.Info("Shutting down workers")

	return nil
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling Reconcile on our Reconciler.
func (c *Impl) processNextWorkItem() bool {
	obj, shutdown := c.WorkQueue.Get()
	if shutdown {
		return false
	}
	key := obj.(string)

	startTime := time.Now()
	// Send the metrics for the current queue depth
	c.statsReporter.ReportQueueDepth(int64(c.WorkQueue.Len()))

	// We call Done here so the workqueue knows we have finished
	// processing this item. We also must remember to call Forget if
	// reconcile succeeds. If a transient error occurs, we do not call
	// Forget and put the item back to the queue with an increased
	// delay.
	defer c.WorkQueue.Done(key)

	var err error
	defer func() {
		status := trueString
		if err != nil {
			status = falseString
		}
		c.statsReporter.ReportReconcile(time.Since(startTime), key, status)
	}()

	// Embed the key into the logger and attach that to the context we pass
	// to the Reconciler.
	logger := c.logger.With(zap.String(logkey.Key, key))
	ctx := logging.WithLogger(context.TODO(), logger)

	// Run Reconcile, passing it the namespace/name string of the
	// resource to be synced.
	if err = c.Reconciler.Reconcile(ctx, key); err != nil {
		c.handleErr(err, key)
		logger.Infof("Reconcile failed. Time taken: %v.", time.Since(startTime))
		return true
	}

	// Finally, if no error occurs we Forget this item so it does not
	// have any delay when another change happens.
	c.WorkQueue.Forget(key)
	logger.Infof("Reconcile succeeded. Time taken: %v.", time.Since(startTime))

	return true
}

func (c *Impl) handleErr(err error, key string) {
	c.logger.Errorw("Reconcile error", zap.Error(err))

	// Re-queue the key if it's an transient error.
	if !IsPermanentError(err) {
		c.WorkQueue.AddRateLimited(key)
		return
	}

	c.WorkQueue.Forget(key)
}

// GlobalResync enqueues all objects from the passed SharedInformer
func (c *Impl) GlobalResync(si cache.SharedInformer) {
	for _, key := range si.GetStore().ListKeys() {
		c.EnqueueKey(key)
	}
}

// NewPermanentError returns a new instance of permanentError.
// Users can wrap an error as permanentError with this in reconcile,
// when he does not expect the key to get re-queued.
func NewPermanentError(err error) error {
	return permanentError{e: err}
}

// permanentError is an error that is considered not transient.
// We should not re-queue keys when it returns with thus error in reconcile.
type permanentError struct {
	e error
}

// IsPermanentError returns true if given error is permanentError
func IsPermanentError(err error) bool {
	switch err.(type) {
	case permanentError:
		return true
	default:
		return false
	}
}

// Error implements the Error() interface of error.
func (err permanentError) Error() string {
	if err.e == nil {
		return ""
	}

	return err.e.Error()
}
