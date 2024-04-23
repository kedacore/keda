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
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	eventingv1alpha1 "github.com/kedacore/keda/v2/apis/eventing/v1alpha1"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
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
	secretsLister            corev1listers.SecretLister
}

// EventHandler defines the behavior for EventEmitter clients
type EventHandler interface {
	DeleteCloudEventSource(cloudEventSource *eventingv1alpha1.CloudEventSource) error
	HandleCloudEventSource(ctx context.Context, cloudEventSource *eventingv1alpha1.CloudEventSource) error
	Emit(object runtime.Object, namesapce types.NamespacedName, eventType string, cloudeventType eventingv1alpha1.CloudEventType, reason string, message string)
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
func NewEventEmitter(client client.Client, recorder record.EventRecorder, clusterName string, secretsLister corev1listers.SecretLister) EventHandler {
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
		secretsLister:            secretsLister,
	}
}

func initializeLogger(cloudEventSource *eventingv1alpha1.CloudEventSource, cloudEventSourceEmitterName string) logr.Logger {
	return logf.Log.WithName(cloudEventSourceEmitterName).WithValues("type", cloudEventSource.Kind, "namespace", cloudEventSource.Namespace, "name", cloudEventSource.Name)
}

// HandleCloudEventSource will create CloudEventSource handlers that defined in spec and start an event loop once handlers
// are created successfully.
func (e *EventEmitter) HandleCloudEventSource(ctx context.Context, cloudEventSource *eventingv1alpha1.CloudEventSource) error {
	e.createEventHandlers(ctx, cloudEventSource)

	if !e.checkIfEventHandlersExist(cloudEventSource) {
		return fmt.Errorf("no CloudEventSource handler is created for %s/%s", cloudEventSource.Namespace, cloudEventSource.Name)
	}

	key := cloudEventSource.GenerateIdentifier()
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
		if updateErr := e.setCloudEventSourceStatusActive(ctx, cloudEventSource); updateErr != nil {
			e.log.Error(updateErr, "Failed to update CloudEventSource status")
			return updateErr
		}
	}

	// a mutex is used to synchronize handler per cloudEventSource
	eventingMutex := &sync.Mutex{}

	// passing deep copy of CloudEventSource to the eventLoop go routines, it's a precaution to not have global objects shared between threads
	e.log.V(1).Info("Start CloudEventSource loop.")
	go e.startEventLoop(cancelCtx, cloudEventSource.DeepCopy(), eventingMutex)
	return nil
}

// DeleteCloudEventSource will stop the event loop and clean event handlers in cache.
func (e *EventEmitter) DeleteCloudEventSource(cloudEventSource *eventingv1alpha1.CloudEventSource) error {
	key := cloudEventSource.GenerateIdentifier()
	result, ok := e.eventLoopContexts.Load(key)
	if ok {
		cancel, ok := result.(context.CancelFunc)
		if ok {
			cancel()
		}
		e.eventLoopContexts.Delete(key)
		e.clearEventHandlersCache(cloudEventSource)
	} else {
		e.log.V(1).Info("CloudEventSource was not found in controller cache", "key", key)
	}

	return nil
}

// createEventHandlers will create different handler as defined in CloudEventSource, and store them in cache for repeated
// use in the loop.
func (e *EventEmitter) createEventHandlers(ctx context.Context, cloudEventSource *eventingv1alpha1.CloudEventSource) {
	e.eventHandlersCacheLock.Lock()
	e.eventFilterCacheLock.Lock()
	defer e.eventHandlersCacheLock.Unlock()
	defer e.eventFilterCacheLock.Unlock()

	key := cloudEventSource.GenerateIdentifier()

	clusterName := cloudEventSource.Spec.ClusterName
	if clusterName == "" {
		clusterName = e.clusterName
	}

	// Resolve auth related
	authParams, podIdentity, err := resolver.ResolveAuthRefAndPodIdentity(ctx, e.client, e.log, cloudEventSource.Spec.AuthenticationRef, nil, cloudEventSource.Namespace, e.secretsLister)
	switch podIdentity.Provider {
	case kedav1alpha1.PodIdentityProviderAzure:
		// FIXME: Delete this for v2.15
		e.log.Info("WARNING: Azure AD Pod Identity has been archived (https://github.com/Azure/aad-pod-identity#-announcement) and will be removed from KEDA on v2.15")
	default:
	}

	if err != nil {
		e.log.Error(err, "error resolving auth params", "cloudEventSource", cloudEventSource)
		return
	}

	// Create EventFilter from CloudEventSource
	e.eventFilterCache[key] = NewEventFilter(cloudEventSource.Spec.EventSubscription.IncludedEventTypes, cloudEventSource.Spec.EventSubscription.ExcludedEventTypes)

	// Create different event destinations here
	if cloudEventSource.Spec.Destination.HTTP != nil {
		eventHandler, err := NewCloudEventHTTPHandler(ctx, clusterName, cloudEventSource.Spec.Destination.HTTP.URI, initializeLogger(cloudEventSource, "cloudevent_http"))
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

	if cloudEventSource.Spec.Destination.AzureEventGridTopic != nil {
		eventHandler, err := NewAzureEventGridTopicHandler(ctx, clusterName, cloudEventSource.Spec.Destination.AzureEventGridTopic, authParams, podIdentity, initializeLogger(cloudEventSource, "azure_event_grid_topic"))
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

	e.log.Info("No destionation is defined in CloudEventSource", "CloudEventSource", cloudEventSource.Name)
}

// clearEventHandlersCache will clear all event handlers that created by the passing CloudEventSource
func (e *EventEmitter) clearEventHandlersCache(cloudEventSource *eventingv1alpha1.CloudEventSource) {
	e.eventHandlersCacheLock.Lock()
	defer e.eventHandlersCacheLock.Unlock()
	e.eventFilterCacheLock.Lock()
	defer e.eventFilterCacheLock.Unlock()

	key := cloudEventSource.GenerateIdentifier()

	delete(e.eventFilterCache, key)

	// Clear different event destination here.
	if cloudEventSource.Spec.Destination.HTTP != nil {
		eventHandlerKey := newEventHandlerKey(key, cloudEventHandlerTypeHTTP)
		if eventHandler, found := e.eventHandlersCache[eventHandlerKey]; found {
			eventHandler.CloseHandler()
			delete(e.eventHandlersCache, key)
		}
	}

	if cloudEventSource.Spec.Destination.AzureEventGridTopic != nil {
		eventHandlerKey := newEventHandlerKey(key, cloudEventHandlerTypeAzureEventGridTopic)
		if eventHandler, found := e.eventHandlersCache[eventHandlerKey]; found {
			eventHandler.CloseHandler()
			delete(e.eventHandlersCache, key)
		}
	}
}

// checkIfEventHandlersExist will check if the event handlers that were created by passing CloudEventSource exist
func (e *EventEmitter) checkIfEventHandlersExist(cloudEventSource *eventingv1alpha1.CloudEventSource) bool {
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

func (e *EventEmitter) startEventLoop(ctx context.Context, cloudEventSource *eventingv1alpha1.CloudEventSource, cloudEventSourceMutex sync.Locker) {
	for {
		select {
		case eventData := <-e.cloudEventProcessingChan:
			e.log.V(1).Info("Consuming events from CloudEventSource.")
			e.emitEventByHandler(eventData)
			e.checkEventHandlers(ctx, cloudEventSource, cloudEventSourceMutex)
			metricscollector.RecordCloudEventQueueStatus(cloudEventSource.Namespace, len(e.cloudEventProcessingChan))
		case <-ctx.Done():
			e.log.V(1).Info("CloudEventSource loop has stopped.")
			metricscollector.RecordCloudEventQueueStatus(cloudEventSource.Namespace, len(e.cloudEventProcessingChan))
			return
		}
	}
}

// checkEventHandlers will check each eventhandler active status
func (e *EventEmitter) checkEventHandlers(ctx context.Context, cloudEventSource *eventingv1alpha1.CloudEventSource, cloudEventSourceMutex sync.Locker) {
	e.log.V(1).Info("Checking event handlers status.")
	cloudEventSourceMutex.Lock()
	defer cloudEventSourceMutex.Unlock()
	// Get the latest object
	err := e.client.Get(ctx, types.NamespacedName{Name: cloudEventSource.Name, Namespace: cloudEventSource.Namespace}, cloudEventSource)
	if err != nil {
		e.log.Error(err, "error getting cloudEventSource", "cloudEventSource", cloudEventSource)
		return
	}
	keyPrefix := cloudEventSource.GenerateIdentifier()
	needUpdate := false
	cloudEventSourceStatus := cloudEventSource.Status.DeepCopy()
	for k, v := range e.eventHandlersCache {
		e.log.V(1).Info("Checking event handler status.", "handler", k, "status", cloudEventSource.Status.Conditions.GetActiveCondition().Status)
		if strings.Contains(k, keyPrefix) {
			if v.GetActiveStatus() != cloudEventSource.Status.Conditions.GetActiveCondition().Status {
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
		if updateErr := e.updateCloudEventSourceStatus(ctx, cloudEventSource, cloudEventSourceStatus); updateErr != nil {
			e.log.Error(updateErr, "Failed to update CloudEventSource status")
		}
	}
}

// Emit is emitting event to both local kubernetes and custom CloudEventSource handler. After emit event to local kubernetes, event will inqueue and waitng for handler's consuming.
func (e *EventEmitter) Emit(object runtime.Object, namesapce types.NamespacedName, eventType string, cloudeventType eventingv1alpha1.CloudEventType, reason, message string) {
	e.recorder.Event(object, eventType, reason, message)

	e.eventHandlersCacheLock.RLock()
	defer e.eventHandlersCacheLock.RUnlock()
	if len(e.eventHandlersCache) == 0 {
		return
	}

	objectName, _ := meta.NewAccessor().Name(object)
	objectType, _ := meta.NewAccessor().Kind(object)
	eventData := eventdata.EventData{
		Namespace:      namesapce.Namespace,
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

func (e *EventEmitter) setCloudEventSourceStatusActive(ctx context.Context, cloudEventSource *eventingv1alpha1.CloudEventSource) error {
	cloudEventSourceStatus := cloudEventSource.Status.DeepCopy()
	cloudEventSourceStatus.Conditions.SetActiveCondition(
		metav1.ConditionTrue,
		eventingv1alpha1.CloudEventSourceConditionActiveReason,
		eventingv1alpha1.CloudEventSourceConditionActiveMessage,
	)
	return e.updateCloudEventSourceStatus(ctx, cloudEventSource, cloudEventSourceStatus)
}

func (e *EventEmitter) updateCloudEventSourceStatus(ctx context.Context, cloudEventSource *eventingv1alpha1.CloudEventSource, cloudEventSourceStatus *eventingv1alpha1.CloudEventSourceStatus) error {
	e.log.V(1).Info("Updating CloudEventSource status", "CloudEventSource", cloudEventSource.Name)
	transform := func(runtimeObj client.Object, target interface{}) error {
		status, ok := target.(*eventingv1alpha1.CloudEventSourceStatus)
		if !ok {
			return fmt.Errorf("transform target is not eventingv1alpha1.CloudEventSourceStatus type %v", target)
		}
		switch obj := runtimeObj.(type) {
		case *eventingv1alpha1.CloudEventSource:
			e.log.V(1).Info("New CloudEventSource status", "status", *status)
			obj.Status = *status
		default:
		}
		return nil
	}

	if err := kedastatus.TransformObject(ctx, e.client, e.log, cloudEventSource, cloudEventSourceStatus, transform); err != nil {
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
