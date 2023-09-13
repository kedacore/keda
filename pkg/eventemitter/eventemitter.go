/*
Copyright 2022 The KEDA Authors

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

package eventemitter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("event_emitter")
var ch chan EventData

const MAX_RETRY_TIMES = 5

type EventEmitter struct {
	client.Client
	record.EventRecorder
	eventHandlersCache      map[string]EventDataHandler
	eventHandlersCachesLock *sync.RWMutex
	eventLoopContexts       *sync.Map
}

type EventData struct {
	object     string
	eventtype  string
	reason     string
	message    string
	handlerKey string
	retryTimes int
	err        error
}

type EventDataHandler interface {
	EmitEvent(eventData EventData, failureFunc func(eventData EventData, err error)) error
	CloseHandler()
}

const (
	AzureEventGrid = "AzureEventGrid"
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

func (e *EventEmitter) HandleCloudEvents(ctx context.Context, logger logr.Logger, cloudEvent *kedav1alpha1.CloudEvent) error {

	e.createEventHandlers(ctx, cloudEvent)

	key := cloudEvent.GenerateIdentifier()
	ctx, cancel := context.WithCancel(ctx)

	// cancel the outdated ScaleLoop for the same ScaledObject (if exists)
	value, loaded := e.eventLoopContexts.LoadOrStore(key, cancel)
	if loaded {
		cancelValue, ok := value.(context.CancelFunc)
		if ok {
			cancelValue()
		}
		e.eventLoopContexts.Store(key, cancel)
	} else {
		// h.recorder.Event(withTriggers, corev1.EventTypeNormal, eventreason.KEDAScalersStarted, "Started scalers watch")
	}

	// a mutex is used to synchronize scale requests per scalableObject
	scalingMutex := &sync.Mutex{}

	// passing deep copy of ScaledObject/ScaledJob to the scaleLoop go routines, it's a precaution to not have global objects shared between threads
	go e.startEventLoop(ctx, cloudEvent.DeepCopy(), scalingMutex)
	return nil
}

func (e *EventEmitter) DeleteCloudEvents(ctx context.Context, logger logr.Logger, cloudEvent *kedav1alpha1.CloudEvent) error {

	key := cloudEvent.GenerateIdentifier()
	result, ok := e.eventLoopContexts.Load(key)
	if ok {
		cancel, ok := result.(context.CancelFunc)
		if ok {
			cancel()
		}
		e.eventLoopContexts.Delete(key)
		err := e.clearEventHandlersCache(ctx, cloudEvent)
		if err != nil {
			log.Error(err, "error clearing cloudEvent cache", "cloudEvent", cloudEvent, "key", key)
		}
	} else {
		log.V(1).Info("ScalableObject was not found in controller cache", "key", key)
	}

	return nil
}

func (e *EventEmitter) createEventHandlers(ctx context.Context, cloudEvents *kedav1alpha1.CloudEvent) {
	e.eventHandlersCachesLock.Lock()
	defer e.eventHandlersCachesLock.Unlock()

	key := cloudEvents.GenerateIdentifier()
	if cloudEvents.Spec.AzureEventGrid != nil {
		azureEventGridHandler, err := NewAzureEventGridHandler(*cloudEvents.Spec.AzureEventGrid)
		if err != nil {
			return
		}
		e.eventHandlersCache[key+AzureEventGrid] = azureEventGridHandler
	}
}

func (e *EventEmitter) clearEventHandlersCache(ctx context.Context, cloudEvents *kedav1alpha1.CloudEvent) error {
	e.eventHandlersCachesLock.Lock()
	defer e.eventHandlersCachesLock.Unlock()

	key := cloudEvents.GenerateIdentifier()
	if cloudEvents.Spec.AzureEventGrid != nil {
		azureEventGridKey := key + AzureEventGrid
		if azureEventGridHandler, found := e.eventHandlersCache[azureEventGridKey]; found {
			azureEventGridHandler.CloseHandler()
			delete(e.eventHandlersCache, azureEventGridKey)
		}
	}

	return nil
}

func (e *EventEmitter) startEventLoop(ctx context.Context, cloudEvents *kedav1alpha1.CloudEvent, scalingMutex sync.Locker) {

	pollingInterval := 500 * time.Millisecond
	log.V(1).Info("Watching with pollingInterval", "PollingInterval", pollingInterval)

	if ch == nil {
		ch = make(chan EventData, 10)
	}

	for {
		tmr := time.NewTimer(pollingInterval)

		select {
		case <-tmr.C:
			tmr.Stop()
		case eventData := <-ch:
			fmt.Printf("\n\n\nConsuming eventing......\n\n")
			e.emitEventByHandler(ctx, eventData, scalingMutex)
		case <-ctx.Done():
			tmr.Stop()
			return

		}
	}
}

func (e *EventEmitter) Emit(ctx context.Context, object runtime.Object, namesapce types.NamespacedName, eventtype, reason, message string) {

	fmt.Printf("\n\n\nEmitEmitEmitEmitEmitEmit\n\n")
	e.EventRecorder.Event(object, eventtype, reason, message)

	eventData := EventData{object: object.GetObjectKind().GroupVersionKind().Kind, eventtype: eventtype, reason: reason, message: message}
	go e.inqueueEventData(eventData)
}

func (e *EventEmitter) inqueueEventData(eventData EventData) {
	count := 0
	for {
		if count > 5 {
			return
		}
		select {
		case ch <- eventData:
			return
		default:
			fmt.Println("channel full")
			count++
		}
		time.Sleep(time.Millisecond * 500)
	}
}

func (e *EventEmitter) emitEventByHandler(ctx context.Context, eventData EventData, emittingMutex sync.Locker) {
	e.eventHandlersCachesLock.Lock()
	defer e.eventHandlersCachesLock.Unlock()

	if eventData.handlerKey == "" {
		for key, handler := range e.eventHandlersCache {
			fmt.Printf("\n\n\nemitEventByHandler Key: %s, Value: %T\n\n\n", key, handler)
			go handler.EmitEvent(eventData, emitErrorHandle)
		}
	} else if eventData.retryTimes > MAX_RETRY_TIMES {
		log.Error(eventData.err, "Failed to Emit Event")
	} else {
		handler := e.eventHandlersCache[eventData.handlerKey]
		go handler.EmitEvent(eventData, emitErrorHandle)
	}
}

func emitErrorHandle(eventData EventData, err error) {
	requeueData := eventData
	requeueData.handlerKey = eventData.handlerKey
	requeueData.retryTimes++
	requeueData.err = err
}
