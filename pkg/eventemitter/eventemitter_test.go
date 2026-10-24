/*
Copyright 2021 The KEDA Authors

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
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"

	eventingv1alpha1 "github.com/kedacore/keda/v2/apis/eventing/v1alpha1"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/eventemitter/eventdata"
	"github.com/kedacore/keda/v2/pkg/eventreason"
	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
	"github.com/kedacore/keda/v2/pkg/mock/mock_eventemitter"
)

const testNamespaceGlobal = "testNamespace"
const testNameGlobal = "testName"

func TestEventHandler_FailedEmitEvent(t *testing.T) {
	cloudEventSourceName := testNameGlobal
	cloudEventSourceNamespace := testNamespaceGlobal

	ctrl := gomock.NewController(t)
	recorder := events.NewFakeRecorder(1)
	mockClient := mock_client.NewMockClient(ctrl)
	eventHandler := mock_eventemitter.NewMockEventDataHandler(ctrl)
	cloudEventSource := eventingv1alpha1.CloudEventSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cloudEventSourceName,
			Namespace: cloudEventSourceNamespace,
		},
		Spec: eventingv1alpha1.CloudEventSourceSpec{
			Destination: eventingv1alpha1.Destination{
				HTTP: &eventingv1alpha1.CloudEventHTTP{
					URI: "http://fo.wo",
				},
			},
		},
		Status: eventingv1alpha1.CloudEventSourceStatus{
			Conditions: kedav1alpha1.Conditions{{Type: kedav1alpha1.ConditionActive, Status: metav1.ConditionTrue}},
		},
	}

	caches := map[string]EventDataHandler{}
	key := newEventHandlerKey(cloudEventSource.GenerateIdentifier(), cloudEventHandlerTypeHTTP)
	caches[key] = eventHandler

	filtercaches := map[string]*EventFilter{}

	eventEmitter := EventEmitter{
		client:                   mockClient,
		recorder:                 recorder,
		clusterName:              "cluster-name",
		eventHandlersCache:       caches,
		eventHandlersCacheLock:   &sync.RWMutex{},
		eventFilterCache:         filtercaches,
		eventFilterCacheLock:     &sync.RWMutex{},
		eventLoopContexts:        &sync.Map{},
		cloudEventProcessingChan: make(chan eventdata.EventData, 1),
	}

	eventData := eventdata.EventData{
		Namespace:      "aaa",
		ObjectName:     "bbb",
		CloudEventType: "ccc",
		Reason:         "ddd",
		Message:        "eee",
		Time:           time.Now().UTC(),
	}

	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	eventHandler.EXPECT().GetActiveStatus().Return(metav1.ConditionTrue).AnyTimes()
	go eventEmitter.startEventLoop(context.TODO(), &cloudEventSource, &sync.Mutex{})

	time.Sleep(1 * time.Second)
	eventHandler.EXPECT().SetActiveStatus(metav1.ConditionFalse).MinTimes(1)
	eventHandler.EXPECT().EmitEvent(gomock.Any(), gomock.Any()).AnyTimes().Do(func(data eventdata.EventData, failedfunc func(eventData eventdata.EventData, err error)) {
		failedfunc(data, fmt.Errorf("testing error"))
	})
	eventEmitter.enqueueEventData(eventData)
	time.Sleep(2 * time.Second)
}

func TestEventHandler_DirectCall(t *testing.T) {
	cloudEventSourceName := testNameGlobal
	cloudEventSourceNamespace := testNamespaceGlobal

	ctrl := gomock.NewController(t)
	recorder := events.NewFakeRecorder(1)
	mockClient := mock_client.NewMockClient(ctrl)

	eventHandler := mock_eventemitter.NewMockEventDataHandler(ctrl)
	cloudEventSource := eventingv1alpha1.CloudEventSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cloudEventSourceName,
			Namespace: cloudEventSourceNamespace,
		},
		Spec: eventingv1alpha1.CloudEventSourceSpec{
			Destination: eventingv1alpha1.Destination{
				HTTP: &eventingv1alpha1.CloudEventHTTP{
					URI: "http://fo.wo",
				},
			},
		},
		Status: eventingv1alpha1.CloudEventSourceStatus{
			Conditions: kedav1alpha1.Conditions{{Type: kedav1alpha1.ConditionActive, Status: metav1.ConditionTrue}},
		},
	}

	caches := map[string]EventDataHandler{}
	key := newEventHandlerKey(cloudEventSource.GenerateIdentifier(), cloudEventHandlerTypeHTTP)
	caches[key] = eventHandler

	filtercaches := map[string]*EventFilter{}

	eventEmitter := EventEmitter{
		client:                   mockClient,
		recorder:                 recorder,
		clusterName:              "cluster-name",
		eventHandlersCache:       caches,
		eventHandlersCacheLock:   &sync.RWMutex{},
		eventFilterCache:         filtercaches,
		eventFilterCacheLock:     &sync.RWMutex{},
		eventLoopContexts:        &sync.Map{},
		cloudEventProcessingChan: make(chan eventdata.EventData, 1),
	}

	eventData := eventdata.EventData{
		Namespace:      "aaa",
		ObjectName:     "bbb",
		CloudEventType: "ccc",
		Reason:         "ddd",
		Message:        "eee",
		Time:           time.Now().UTC(),
	}

	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	eventHandler.EXPECT().GetActiveStatus().Return(metav1.ConditionTrue).AnyTimes()
	go eventEmitter.startEventLoop(context.TODO(), &cloudEventSource, &sync.Mutex{})

	time.Sleep(1 * time.Second)

	wg := sync.WaitGroup{}
	wg.Add(1)
	eventHandler.EXPECT().EmitEvent(gomock.Any(), gomock.Any()).Times(1).Do(func(arg0, arg1 any) {
		defer wg.Done()
	})
	eventEmitter.enqueueEventData(eventData)
	wg.Wait()
}

type recordedEvent struct {
	eventType, reason, action, note string
}

// actionRecorder records the action argument, which events.FakeRecorder drops.
type actionRecorder struct {
	events []recordedEvent
}

func (r *actionRecorder) Eventf(_ runtime.Object, _ runtime.Object, eventType, reason, action, note string, args ...any) {
	r.events = append(r.events, recordedEvent{eventType: eventType, reason: reason, action: action, note: fmt.Sprintf(note, args...)})
}

func TestEventEmitter_EmitUsesReasonAsAction(t *testing.T) {
	recorder := &actionRecorder{}
	eventEmitter := EventEmitter{
		recorder:               recorder,
		eventHandlersCache:     map[string]EventDataHandler{},
		eventHandlersCacheLock: &sync.RWMutex{},
	}

	scaledObject := &kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{Name: testNameGlobal, Namespace: testNamespaceGlobal},
	}
	// events.k8s.io/v1 rejects an action longer than 128 characters, so a long message must not end up there.
	message := "ScaledObject doesn't have correct Idle/Min/Max Replica Counts specification: MinReplicaCount=1 must be less than MaxReplicaCount=0"

	eventEmitter.Emit(scaledObject, testNamespaceGlobal, corev1.EventTypeWarning, eventingv1alpha1.ScaledObjectFailedType, eventreason.ScaledObjectCheckFailed, message)

	if len(recorder.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(recorder.events))
	}
	got := recorder.events[0]
	if got.action != eventreason.ScaledObjectCheckFailed {
		t.Errorf("expected action %q, got %q", eventreason.ScaledObjectCheckFailed, got.action)
	}
	if got.note != message {
		t.Errorf("expected note %q, got %q", message, got.note)
	}
}
