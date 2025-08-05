/*
Copyright 2023 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// ******************************* DESCRIPTION ****************************** \\
// eventemitter package describes functions that manage different CloudEventSource
// handlers and emit KEDA events to different CloudEventSource destinations through
// these handlers. A loop will be launched to monitor whether there is a new
// KEDA event once a valid CloudEventSource CRD is created. And then the eventemitter
// will send the event data to all event handlers when a new KEDA event reached.
// ************************************************************************** \\

package eventemitter

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	eventingv1alpha1 "github.com/kedacore/keda/v2/apis/eventing/v1alpha1"
	"github.com/kedacore/keda/v2/keda-scalers/authentication"
	"github.com/kedacore/keda/v2/pkg/eventemitter/eventdata"
	"github.com/kedacore/keda/v2/pkg/metricscollector"
	"github.com/kedacore/keda/v2/pkg/scaling/resolver"
	kedastatus "github.com/kedacore/keda/v2/pkg/status"
)

const (
	maxRetryTimes         = 5
	maxChannelBuffer      = 1024
	maxWaitingEnqueueTime = 10
)

// EventEmitter is the main struct for eventemitter package
type EventEmitter struct {
	log                      logr.Logger
	client                   client.Client
	recorder                 record.EventRecorder
	clusterName              string
	eventHandlersCache       map[string]EventDataHandler
	eventFilterCache         map[string]*EventFilter
	eventHandlersCacheLock   *sync.RWMutex
	eventFilterCacheLock     *sync.RWMutex
	eventLoopContexts        *sync.Map
	cloudEventProcessingChan chan eventdata.EventData
	authClientSet            *authentication.AuthClientSet
}

// EventHandler defines the behavior for EventEmitter clients
type EventHandler interface {
	DeleteCloudEventSource(cloudEventSource eventingv1alpha1.CloudEventSourceInterface) error
	HandleCloudEventSource(ctx context.Context, cloudEventSource eventingv1alpha1.CloudEventSourceInterface) error
	Emit(object runtime.Object, namespace string, eventType string, cloudeventType eventingv1alpha1.CloudEventType, reason string, message string)
}

// EventDataHandler defines the behavior for different event handlers
type EventDataHandler interface {
	EmitEvent(eventData eventdata.EventData, failureFunc func(eventData eventdata.EventData, err error))
	SetActiveStatus(status metav1.ConditionStatus)
	GetActiveStatus() metav1.ConditionStatus
	CloseHandler()
}

// EmitData defines the data structure for emitting event
type EmitData struct {
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

const (
	cloudEventHandlerTypeHTTP                = "http"
	cloudEventHandlerTypeAzureEventGridTopic = "azureEventGridTopic"
)

// NewEventEmitter creates a new EventEmitter
func NewEventEmitter(client client.Client, recorder record.EventRecorder, clusterName string, authClientSet *authentication.AuthClientSet) EventHandler {
	return &EventEmitter{
		log:                      logf.Log.WithName("event_emitter"),
		client:                   client,
		recorder:                 recorder,
		clusterName:              clusterName,
		eventHandlersCache:       map[string]EventDataHandler{},
		eventFilterCache:         map[string]*EventFilter{},
		eventHandlersCacheLock:   &sync.RWMutex{},
		eventFilterCacheLock:     &sync.RWMutex{},
		eventLoopContexts:        &sync.Map{},
		cloudEventProcessingChan: make(chan eventdata.EventData, maxChannelBuffer),
		authClientSet:            authClientSet,
	}
}

func initializeLogger(cloudEventSourceI eventingv1alpha1.CloudEventSourceInterface, cloudEventSourceEmitterName string) logr.Logger {
	return logf.Log.WithName(cloudEventSourceEmitterName).WithValues("type", cloudEventSourceI.GetObjectKind(), "namespace", cloudEventSourceI.GetNamespace(), "name", cloudEventSourceI.GetName())
}

// HandleCloudEventSource will create CloudEventSource handlers that defined in spec and start an event loop once handlers
// are created successfully.
func (e *EventEmitter) HandleCloudEventSource(ctx context.Context, cloudEventSourceI eventingv1alpha1.CloudEventSourceInterface) error {
	e.createEventHandlers(ctx, cloudEventSourceI)

	if !e.checkIfEventHandlersExist(cloudEventSourceI) {
		return fmt.Errorf("no CloudEventSource handler is created for %s/%s", cloudEventSourceI.GetNamespace(), cloudEventSourceI.GetName())
	}

	key := cloudEventSourceI.GenerateIdentifier()
	cancelCtx, cancel := context.WithCancel(ctx)

	// cancel the outdated EventLoop for the same CloudEventSource (if exists)
	value, loaded := e.eventLoopContexts.LoadOrStore(key, cancel)
	if loaded {
		cancelValue, ok := value.(context.CancelFunc)
		if ok {
			cancelValue()
		}
		e.eventLoopContexts.Store(key, cancel)
	} else {
		if updateErr := e.setCloudEventSourceStatusActive(ctx, cloudEventSourceI); updateErr != nil {
			e.log.Error(updateErr, "Failed to update CloudEventSource status")
			return updateErr
		}
	}

	// a mutex is used to synchronize handler per cloudEventSource
	eventingMutex := &sync.Mutex{}

	// passing deep copy of CloudEventSource to the eventLoop go routines, it's a precaution to not have global objects shared between threads
	e.log.V(1).Info("Start CloudEventSource loop.")
	go e.startEventLoop(cancelCtx, cloudEventSourceI.DeepCopyObject().(eventingv1alpha1.CloudEventSourceInterface), eventingMutex)
	return nil
}

// DeleteCloudEventSource will stop the event loop and clean event handlers in cache.
func (e *EventEmitter) DeleteCloudEventSource(cloudEventSource eventingv1alpha1.CloudEventSourceInterface) error {
	key := cloudEventSource.GenerateIdentifier()
	result, ok := e.eventLoopContexts.Load(key)
	e.log.V(1).Info("successfully DeleteCloudEventSourceDeleteCloudEventSourceDeleteCloudEventSource", "key", key)
	if ok {
		cancel, ok := result.(context.CancelFunc)
		if ok {
			cancel()
		}
		e.eventLoopContexts.Delete(key)
		e.clearEventHandlersCache(cloudEventSource)
	} else {
		e.log.V(1).Info("successfully CloudEventSource was not found in controller cache", "key", key)
	}

	return nil
}

// createEventHandlers will create different handler as defined in CloudEventSource, and store them in cache for repeated
// use in the loop.
func (e *EventEmitter) createEventHandlers(ctx context.Context, cloudEventSourceI eventingv1alpha1.CloudEventSourceInterface) {
	e.eventHandlersCacheLock.Lock()
	e.eventFilterCacheLock.Lock()
	defer e.eventHandlersCacheLock.Unlock()
	defer e.eventFilterCacheLock.Unlock()

	key := cloudEventSourceI.GenerateIdentifier()
	spec := cloudEventSourceI.GetSpec()

	clusterName := spec.ClusterName
	if clusterName == "" {
		clusterName = e.clusterName
	}

	// Resolve auth related
	authParams, podIdentity, err := resolver.ResolveAuthRefAndPodIdentity(ctx, e.client, e.log, spec.AuthenticationRef, nil, cloudEventSourceI.GetNamespace(), e.authClientSet)
	if err != nil {
		e.log.Error(err, "error resolving auth params", "cloudEventSource", cloudEventSourceI)
		return
	}

	// Create EventFilter from CloudEventSource
	e.eventFilterCache[key] = NewEventFilter(spec.EventSubscription.IncludedEventTypes, spec.EventSubscription.ExcludedEventTypes)

	// Create different event destinations here
	if spec.Destination.HTTP != nil {
		eventHandler, err := NewCloudEventHTTPHandler(ctx, clusterName, spec.Destination.HTTP.URI, initializeLogger(cloudEventSourceI, "cloudevent_http"))
		if err != nil {
			e.log.Error(err, "create CloudEvent HTTP handler failed")
			return
		}

		eventHandlerKey := newEventHandlerKey(key, cloudEventHandlerTypeHTTP)
		if h, ok := e.eventHandlersCache[eventHandlerKey]; ok {
			h.CloseHandler()
		}
		e.eventHandlersCache[eventHandlerKey] = eventHandler
		return
	}

	if spec.Destination.AzureEventGridTopic != nil {
		eventHandler, err := NewAzureEventGridTopicHandler(ctx, clusterName, spec.Destination.AzureEventGridTopic, authParams, podIdentity, initializeLogger(cloudEventSourceI, "azure_event_grid_topic"))
		if err != nil {
			e.log.Error(err, "create Azure Event Grid handler failed")
			return
		}

		eventHandlerKey := newEventHandlerKey(key, cloudEventHandlerTypeAzureEventGridTopic)
		if h, ok := e.eventHandlersCache[eventHandlerKey]; ok {
			h.CloseHandler()
		}
		e.eventHandlersCache[eventHandlerKey] = eventHandler
		return
	}

	e.log.Info("No destionation is defined in CloudEventSource", "CloudEventSource", cloudEventSourceI.GetName())
}

// clearEventHandlersCache will clear all event handlers that created by the passing CloudEventSource
func (e *EventEmitter) clearEventHandlersCache(cloudEventSource eventingv1alpha1.CloudEventSourceInterface) {
	e.eventHandlersCacheLock.Lock()
	defer e.eventHandlersCacheLock.Unlock()
	e.eventFilterCacheLock.Lock()
	defer e.eventFilterCacheLock.Unlock()

	spec := cloudEventSource.GetSpec()
	key := cloudEventSource.GenerateIdentifier()

	delete(e.eventFilterCache, key)

	// Clear different event destination here.
	if spec.Destination.HTTP != nil {
		eventHandlerKey := newEventHandlerKey(key, cloudEventHandlerTypeHTTP)
		if eventHandler, found := e.eventHandlersCache[eventHandlerKey]; found {
			eventHandler.CloseHandler()
			delete(e.eventHandlersCache, eventHandlerKey)
		}
	}

	if spec.Destination.AzureEventGridTopic != nil {
		eventHandlerKey := newEventHandlerKey(key, cloudEventHandlerTypeAzureEventGridTopic)
		if eventHandler, found := e.eventHandlersCache[eventHandlerKey]; found {
			eventHandler.CloseHandler()
			delete(e.eventHandlersCache, eventHandlerKey)
		}
	}
}

// checkIfEventHandlersExist will check if the event handlers that were created by passing CloudEventSource exist
func (e *EventEmitter) checkIfEventHandlersExist(cloudEventSource eventingv1alpha1.CloudEventSourceInterface) bool {
	e.eventHandlersCacheLock.RLock()
	defer e.eventHandlersCacheLock.RUnlock()

	key := cloudEventSource.GenerateIdentifier()

	for k := range e.eventHandlersCache {
		if strings.Contains(k, key) {
			return true
		}
	}
	return false
}

func (e *EventEmitter) startEventLoop(ctx context.Context, cloudEventSourceI eventingv1alpha1.CloudEventSourceInterface, cloudEventSourceMutex sync.Locker) {
	e.log.V(1).Info("Start CloudEventSource loop.", "name", cloudEventSourceI.GetName())
	for {
		select {
		case eventData := <-e.cloudEventProcessingChan:
			e.log.V(1).Info("Consuming events from CloudEventSource.", "name", cloudEventSourceI.GetName())
			e.emitEventByHandler(eventData)
			e.checkEventHandlers(ctx, cloudEventSourceI, cloudEventSourceMutex)
			metricscollector.RecordCloudEventQueueStatus(cloudEventSourceI.GetNamespace(), len(e.cloudEventProcessingChan))
		case <-ctx.Done():
			e.log.V(1).Info("CloudEventSource loop has stopped.")
			metricscollector.RecordCloudEventQueueStatus(cloudEventSourceI.GetNamespace(), len(e.cloudEventProcessingChan))
			return
		}
	}
}

// checkEventHandlers will check each eventhandler active status
func (e *EventEmitter) checkEventHandlers(ctx context.Context, cloudEventSourceI eventingv1alpha1.CloudEventSourceInterface, cloudEventSourceMutex sync.Locker) {
	e.log.V(1).Info("Checking event handlers status.")
	cloudEventSourceMutex.Lock()
	defer cloudEventSourceMutex.Unlock()
	// Get the latest object
	err := e.client.Get(ctx, types.NamespacedName{Name: cloudEventSourceI.GetName(), Namespace: cloudEventSourceI.GetNamespace()}, cloudEventSourceI)
	if err != nil {
		e.log.Error(err, "error getting cloudEventSource", "cloudEventSource", cloudEventSourceI)
		return
	}
	keyPrefix := cloudEventSourceI.GenerateIdentifier()
	needUpdate := false
	cloudEventSourceStatus := cloudEventSourceI.GetStatus().DeepCopy()
	for k, v := range e.eventHandlersCache {
		e.log.V(1).Info("Checking event handler status.", "handler", k, "status", cloudEventSourceI.GetStatus().Conditions.GetActiveCondition().Status)
		if strings.Contains(k, keyPrefix) {
			if v.GetActiveStatus() != cloudEventSourceI.GetStatus().Conditions.GetActiveCondition().Status {
				needUpdate = true
				cloudEventSourceStatus.Conditions.SetActiveCondition(
					metav1.ConditionFalse,
					eventingv1alpha1.CloudEventSourceConditionFailedReason,
					eventingv1alpha1.CloudEventSourceConditionFailedMessage,
				)
			}
		}
	}
	if needUpdate {
		if updateErr := e.updateCloudEventSourceStatus(ctx, cloudEventSourceI, cloudEventSourceStatus); updateErr != nil {
			e.log.Error(updateErr, "Failed to update CloudEventSource status")
		}
	}
}

// Emit is emitting event to both local kubernetes and custom CloudEventSource handler. After emit event to local kubernetes, event will inqueue and waitng for handler's consuming.
func (e *EventEmitter) Emit(object runtime.Object, namespace string, eventType string, cloudeventType eventingv1alpha1.CloudEventType, reason, message string) {
	e.recorder.Event(object, eventType, reason, message)

	e.eventHandlersCacheLock.RLock()
	defer e.eventHandlersCacheLock.RUnlock()
	if len(e.eventHandlersCache) == 0 {
		return
	}

	objectName, _ := meta.NewAccessor().Name(object)
	objectType, _ := meta.NewAccessor().Kind(object)
	eventData := eventdata.EventData{
		Namespace:      namespace,
		CloudEventType: cloudeventType,
		ObjectName:     strings.ToLower(objectName),
		ObjectType:     strings.ToLower(objectType),
		Reason:         reason,
		Message:        message,
		Time:           time.Now().UTC(),
	}
	go e.enqueueEventData(eventData)
}

func (e *EventEmitter) enqueueEventData(eventData eventdata.EventData) {
	metricscollector.RecordCloudEventQueueStatus(eventData.Namespace, len(e.cloudEventProcessingChan))
	select {
	case e.cloudEventProcessingChan <- eventData:
		e.log.V(1).Info("Event enqueued successfully.")
	case <-time.After(maxWaitingEnqueueTime * time.Second):
		e.log.Error(nil, "Failed to enqueue CloudEvent. Need to be check if handler can emit events.")
	}
}

// emitEventByHandler handles event emitting. It will follow these logic:
// 1. If there is a new EventData, call all handlers for emitting.
// 2. Once there is an error when emitting event, record the handler's key and reqeueu this EventData.
// 3. If the maximum number of retries has been exceeded, discard this event.
func (e *EventEmitter) emitEventByHandler(eventData eventdata.EventData) {
	if eventData.RetryTimes >= maxRetryTimes {
		e.log.Error(eventData.Err, "Failed to emit Event multiple times. Will drop this event and need to check if event endpoint works well", "CloudEventSource", eventData.ObjectName)
		handler, found := e.eventHandlersCache[eventData.HandlerKey]
		if found {
			e.log.V(1).Info("Set handler failure status. 1", "handler", eventData.HandlerKey)
			handler.SetActiveStatus(metav1.ConditionFalse)
		}
		return
	}

	if eventData.HandlerKey == "" {
		for key, handler := range e.eventHandlersCache {
			e.eventFilterCacheLock.RLock()
			defer e.eventFilterCacheLock.RUnlock()
			// Filter Event
			identifierKey := getPrefixIdentifierFromKey(key)

			if e.eventFilterCache[identifierKey] != nil {
				isFiltered := e.eventFilterCache[identifierKey].FilterEvent(eventData.CloudEventType)
				if isFiltered {
					e.log.V(1).Info("Event is filtered", "cloudeventType", eventData.CloudEventType, "event identifier", identifierKey)
					return
				}
			}
			eventData.HandlerKey = key
			if handler.GetActiveStatus() == metav1.ConditionTrue {
				go handler.EmitEvent(eventData, e.emitErrorHandle)

				metricscollector.RecordCloudEventEmitted(eventData.Namespace, getSourceNameFromKey(eventData.HandlerKey), getHandlerTypeFromKey(key))
			} else {
				e.log.V(1).Info("EventHandler's status is not active. Please check if event endpoint works well", "CloudEventSource", eventData.ObjectName)
			}
		}
	} else {
		e.log.Info("Failed to emit event", "handler", eventData.HandlerKey, "retry times", fmt.Sprintf("%d/%d", eventData.RetryTimes, maxRetryTimes), "error", eventData.Err)
		handler, found := e.eventHandlersCache[eventData.HandlerKey]
		if found && handler.GetActiveStatus() == metav1.ConditionTrue {
			go handler.EmitEvent(eventData, e.emitErrorHandle)
		}
	}
}

func (e *EventEmitter) emitErrorHandle(eventData eventdata.EventData, err error) {
	metricscollector.RecordCloudEventEmittedError(eventData.Namespace, getSourceNameFromKey(eventData.HandlerKey), getHandlerTypeFromKey(eventData.HandlerKey))

	if eventData.RetryTimes >= maxRetryTimes {
		e.log.V(1).Info("Failed to emit Event multiple times. Will set handler failure status.", "handler", eventData.HandlerKey, "retry times", eventData.RetryTimes)
		handler, found := e.eventHandlersCache[eventData.HandlerKey]
		if found {
			handler.SetActiveStatus(metav1.ConditionFalse)
		}
		return
	}

	requeueData := eventData
	requeueData.HandlerKey = eventData.HandlerKey
	requeueData.RetryTimes++
	requeueData.Err = err
	e.enqueueEventData(requeueData)
}

func (e *EventEmitter) setCloudEventSourceStatusActive(ctx context.Context, cloudEventSourceI eventingv1alpha1.CloudEventSourceInterface) error {
	cloudEventSourceStatus := cloudEventSourceI.GetStatus()
	cloudEventSourceStatus.Conditions.SetActiveCondition(
		metav1.ConditionTrue,
		eventingv1alpha1.CloudEventSourceConditionActiveReason,
		eventingv1alpha1.CloudEventSourceConditionActiveMessage,
	)
	return e.updateCloudEventSourceStatus(ctx, cloudEventSourceI, cloudEventSourceStatus)
}

func (e *EventEmitter) updateCloudEventSourceStatus(ctx context.Context, cloudEventSourceI eventingv1alpha1.CloudEventSourceInterface, cloudEventSourceStatus *eventingv1alpha1.CloudEventSourceStatus) error {
	e.log.V(1).Info("Updating CloudEventSource status", "CloudEventSource", cloudEventSourceI.GetName())
	transform := func(runtimeObj client.Object, target interface{}) error {
		status, ok := target.(eventingv1alpha1.CloudEventSourceStatus)
		if !ok {
			return fmt.Errorf("transform target is not eventingv1alpha1.CloudEventSourceStatus type %v", target)
		}
		switch obj := runtimeObj.(type) {
		case *eventingv1alpha1.CloudEventSource:
			e.log.V(1).Info("New CloudEventSource status", "status", status)
			obj.Status = status
		case *eventingv1alpha1.ClusterCloudEventSource:
			e.log.V(1).Info("New ClusterCloudEventSource status", "status", status)
			obj.Status = status
		default:
		}
		return nil
	}

	if err := kedastatus.TransformObject(ctx, e.client, e.log, cloudEventSourceI, *cloudEventSourceStatus, transform); err != nil {
		e.log.Error(err, "Failed to update CloudEventSourceStatus")
		return err
	}

	return nil
}

func newEventHandlerKey(kindNamespaceName string, handlerType string) string {
	return fmt.Sprintf("%s.%s", kindNamespaceName, handlerType)
}

// getPrefixIdentifierFromKey will return the prefix identifier from the handler key. Handler key is generated by the format of "CloudEventSource.Namespace.Name.HandlerType" and the prefix identifier is "CloudEventSource.Namespace.Name"
func getPrefixIdentifierFromKey(handlerKey string) string {
	keys := strings.Split(handlerKey, ".")
	if len(keys) >= 3 {
		return keys[0] + "." + keys[1] + "." + keys[2]
	}
	return ""
}

// getHandlerTypeFromKey will return the handler type from the handler key. Handler key is generated by the format of "CloudEventSource.Namespace.Name.HandlerType" and the handler type is "HandlerType"
func getHandlerTypeFromKey(handlerKey string) string {
	keys := strings.Split(handlerKey, ".")
	if len(keys) >= 4 {
		return keys[3]
	}
	return ""
}

// getSourceNameFromKey will return the handler type from the source name. Source name is generated by the format of "CloudEventSource.Namespace.Name.HandlerType" and the source name is "Name"
func getSourceNameFromKey(handlerKey string) string {
	keys := strings.Split(handlerKey, ".")
	if len(keys) >= 4 {
		return keys[2]
	}
	return ""
}
