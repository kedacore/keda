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
// eventemitter package describes functions that manage different CloudEvent
// handlers and emit KEDA events to different CloudEvent destinations through
// these handlers. A loop will be launched to monitor whether there is a new
// KEDA event once a valid CloudEvent CRD is created. And then the eventemitter
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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

var log = logf.Log.WithName("event_emitter")
var ch chan EventData

const CloudEventType = "com.cloudevents.keda"
const MaxRetryTimes = 5
const MaxChannelBuffer = 10

type EventEmitter struct {
	client.Client
	record.EventRecorder
	eventHandlersCache      map[string]EventDataHandler
	eventHandlersCachesLock *sync.RWMutex
	eventLoopContexts       *sync.Map
}

// EventData will save all event info and handler info for retry.
type EventData struct {
	namespace  string
	objectName string
	eventtype  string
	reason     string
	message    string
	time       time.Time
	handlerKey string
	retryTimes int
	err        error
}

type EventDataHandler interface {
	EmitEvent(eventData EventData, failureFunc func(eventData EventData, err error))
	CloseHandler()
}

type EmitData struct {
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

const (
	CloudEventHTTP = "CloudEventHTTP"
)

func NewEventEmitter(client client.Client, recorder record.EventRecorder) *EventEmitter {
	return &EventEmitter{
		Client:                  client,
		EventRecorder:           recorder,
		eventHandlersCache:      map[string]EventDataHandler{},
		eventHandlersCachesLock: &sync.RWMutex{},
		eventLoopContexts:       &sync.Map{},
	}
}

func initializeLogger(cloudEvents *kedav1alpha1.CloudEvent, cloudEventEmitterName string) logr.Logger {
	return logf.Log.WithName(cloudEventEmitterName).WithValues("type", cloudEvents.Kind, "namespace", cloudEvents.Namespace, "name", cloudEvents.Name)
}

// HandleCloudEvents will create CloudEvent handlers that defined in spec and start an event loop once handlers
// are created successfully.
func (e *EventEmitter) HandleCloudEvents(ctx context.Context, cloudEvent *kedav1alpha1.CloudEvent) error {
	e.createEventHandlers(ctx, cloudEvent)

	if !e.checkIfEventHandlersExist(cloudEvent) {
		return fmt.Errorf("no CloudEvent handler is created for %s", cloudEvent.Name)
	}

	key := cloudEvent.GenerateIdentifier()
	ctx, cancel := context.WithCancel(ctx)

	// cancel the outdated EventLoop for the same CloudEvent (if exists)
	value, loaded := e.eventLoopContexts.LoadOrStore(key, cancel)
	if loaded {
		cancelValue, ok := value.(context.CancelFunc)
		if ok {
			cancelValue()
		}
		e.eventLoopContexts.Store(key, cancel)
	}

	// passing deep copy of CloudEvents to the eventLoop go routines, it's a precaution to not have global objects shared between threads
	log.V(1).Info("Start CloudEvent loop.")
	go e.startEventLoop(ctx)
	return nil
}

// DeleteCloudEvents will stop the event loop and clean event handlers in cache.
func (e *EventEmitter) DeleteCloudEvents(cloudEvent *kedav1alpha1.CloudEvent) error {
	key := cloudEvent.GenerateIdentifier()
	result, ok := e.eventLoopContexts.Load(key)
	if ok {
		cancel, ok := result.(context.CancelFunc)
		if ok {
			cancel()
		}
		e.eventLoopContexts.Delete(key)
		e.clearEventHandlersCache(cloudEvent)
	} else {
		log.V(1).Info("CloudEvent was not found in controller cache", "key", key)
	}

	return nil
}

// createEventHandlers will create different handler as defined in CloudEvent, and store them in cache for repeated
// use in the loop.
func (e *EventEmitter) createEventHandlers(ctx context.Context, cloudEvents *kedav1alpha1.CloudEvent) {
	e.eventHandlersCachesLock.Lock()
	defer e.eventHandlersCachesLock.Unlock()

	key := cloudEvents.GenerateIdentifier()

	clusterName := cloudEvents.Spec.ClusterName
	if clusterName == "" {
		clusterName = "default"
	}

	if cloudEvents.Spec.Destination.CloudEventHTTP != nil {
		var eventHandler EventDataHandler
		eventHandler, err := NewCloudEventHTTPHandler(ctx, clusterName, cloudEvents.Spec.Destination.CloudEventHTTP.Uri, initializeLogger(cloudEvents, "cloudevent_http"))

		if err != nil {
			log.Error(err, "create cloudevent handler failed")
		} else {
			e.eventHandlersCache[key+CloudEventHTTP] = eventHandler
		}
	}
}

// clearEventHandlersCache will clear all event handlers that created by the passing CloudEvents
func (e *EventEmitter) clearEventHandlersCache(cloudEvents *kedav1alpha1.CloudEvent) {
	e.eventHandlersCachesLock.Lock()
	defer e.eventHandlersCachesLock.Unlock()

	key := cloudEvents.GenerateIdentifier()

	if cloudEvents.Spec.Destination.CloudEventHTTP != nil {
		eventHandlerKey := key + CloudEventHTTP
		if eventHandler, found := e.eventHandlersCache[eventHandlerKey]; found {
			eventHandler.CloseHandler()
			delete(e.eventHandlersCache, key)
		}
	}
}

// clearEventHandlersCache will check if the event handlers that were created by passing CloudEvents exist
func (e *EventEmitter) checkIfEventHandlersExist(cloudEvents *kedav1alpha1.CloudEvent) bool {
	e.eventHandlersCachesLock.RLock()
	defer e.eventHandlersCachesLock.RUnlock()

	key := cloudEvents.GenerateIdentifier()

	for k := range e.eventHandlersCache {
		if strings.Contains(k, key) {
			return true
		}
	}
	return false
}

func (e *EventEmitter) startEventLoop(ctx context.Context) {
	consumingInterval := 500 * time.Millisecond

	if ch == nil {
		ch = make(chan EventData, MaxChannelBuffer)
	}

	for {
		tmr := time.NewTimer(consumingInterval)

		select {
		case <-tmr.C:
			tmr.Stop()
		case eventData := <-ch:
			log.V(1).Info("Consuming events in queue.")
			e.emitEventByHandler(eventData)
		case <-ctx.Done():
			tmr.Stop()
			return
		}
	}
}

// Emit is emitting event to both local kubernetes and custom CloudEvents handler. After emit event to local kubernetes, event will inqueue and waitng for handler's consuming.
func (e *EventEmitter) Emit(object runtime.Object, namesapce types.NamespacedName, eventtype, reason, message string) {
	e.EventRecorder.Event(object, eventtype, reason, message)
	name, _ := meta.NewAccessor().Name(object)
	eventData := EventData{
		namespace:  namesapce.Namespace,
		objectName: name,
		eventtype:  eventtype,
		reason:     reason,
		message:    message,
		time:       time.Now().UTC(),
	}
	go e.inqueueEventData(eventData)
}

func (e *EventEmitter) inqueueEventData(eventData EventData) {
	count := 0
	for {
		if count > MaxChannelBuffer {
			log.Error(nil, "CloudEvents' channel is full and need to be check if handler cannot emit events")
			return
		}
		select {
		case ch <- eventData:
			return
		default:
			log.Info("Event cannot inqueue. Wait for next round.")
			count++
		}
		time.Sleep(time.Millisecond * 500)
	}
}

// emitEventByHandler handles event emitting. It will follow these logic:
// 1. If there is a new EventData, call all handlers for emitting.
// 2. Once there is an error when emitting event, record the handler's key and reqeueu this EventData.
// 3. If the maximum number of retries has been exceeded, discard this event.
func (e *EventEmitter) emitEventByHandler(eventData EventData) {
	if eventData.retryTimes >= MaxRetryTimes {
		log.Error(eventData.err, "Failed to emit Event multiple times. Will drop this event and need to check if event endpoint works well", "Handler", eventData.handlerKey)
		return
	}

	if eventData.handlerKey == "" {
		for key, handler := range e.eventHandlersCache {
			eventData.handlerKey = key
			go handler.EmitEvent(eventData, e.emitErrorHandle)
		}
	} else {
		log.V(1).Info("Reemit failed event", "handler", eventData.handlerKey, "retry times", eventData.retryTimes)
		handler := e.eventHandlersCache[eventData.handlerKey]
		go handler.EmitEvent(eventData, e.emitErrorHandle)
	}
}

func (e *EventEmitter) emitErrorHandle(eventData EventData, err error) {
	requeueData := eventData
	requeueData.handlerKey = eventData.handlerKey
	requeueData.retryTimes++
	requeueData.err = err
	e.inqueueEventData(requeueData)
}
