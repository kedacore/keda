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

package scalers

import (
	"context"
	"errors"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-logr/logr"
)

// spyLogSink is a minimal logr.LogSink that records calls, so tests can
// assert a handler actually logged something without an external
// testing-logger dependency.
type spyLogSink struct {
	errorCalls []string
	infoCalls  []string
}

func (s *spyLogSink) Init(logr.RuntimeInfo) {}
func (s *spyLogSink) Enabled(int) bool      { return true }
func (s *spyLogSink) Info(_ int, msg string, _ ...interface{}) {
	s.infoCalls = append(s.infoCalls, msg)
}
func (s *spyLogSink) Error(_ error, msg string, _ ...interface{}) {
	s.errorCalls = append(s.errorCalls, msg)
}
func (s *spyLogSink) WithValues(...interface{}) logr.LogSink { return s }
func (s *spyLogSink) WithName(string) logr.LogSink           { return s }

// expectActive waits briefly for an activation event on active.
func expectActive(t *testing.T, active <-chan bool) {
	t.Helper()
	select {
	case v := <-active:
		if !v {
			t.Error("expected an activation event of true, got false")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for an activation event")
	}
}

func waitReady(t *testing.T, fake *fakeMqttClient) {
	t.Helper()
	select {
	case <-fake.readyCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Run to connect")
	}
}

func TestMqttScalerRunRecordsMessages(t *testing.T) {
	s := &mqttScaler{
		metadata: mqttScalerMetadata{Topic: "sensors/temp", QoS: 1},
		window:   newMessageWindow(time.Minute),
		logger:   logr.Discard(),
	}

	fake := newFakeMqttClient()
	s.newClient = fakeClientFactory(fake)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	active := make(chan bool, 1)
	go s.Run(ctx, active)

	waitReady(t, fake)

	if !fake.connectCalled {
		t.Fatal("expected Connect to be called")
	}
	if fake.subscribedTo != "sensors/temp" {
		t.Errorf("expected subscription to sensors/temp, got %s", fake.subscribedTo)
	}
	if fake.subscribedQoS != 1 {
		t.Errorf("expected subscription qos 1, got %d", fake.subscribedQoS)
	}

	// Live messages must push an activation event, so a burst shorter
	// than pollingInterval still activates the workload.
	fake.deliver(&fakeMqttMessage{topic: "sensors/temp", retained: false})
	expectActive(t, active)
	fake.deliver(&fakeMqttMessage{topic: "sensors/temp", retained: false})
	expectActive(t, active)

	fake.deliver(&fakeMqttMessage{topic: "sensors/temp", retained: true})
	expectActive(t, active)

	// NOTE: deliver() calls the publish handler synchronously on this
	// goroutine, not from a separate one the way a real paho callback
	// would. That's fine here since record()/markRetained() are
	// mutex-protected regardless, but it means this test does not itself
	// exercise the concurrent-access path -- see
	// TestMessageWindowConcurrentAccess for that.
	count, retained := s.window.count()
	if count != 2 {
		t.Errorf("expected 2 counted messages, got %d", count)
	}
	if !retained {
		t.Error("expected retained flag to be set")
	}
}

func TestMqttScalerPublishDoesNotBlockWhenReceiverBusy(t *testing.T) {
	s := &mqttScaler{
		metadata: mqttScalerMetadata{Topic: "sensors/temp", QoS: 1},
		window:   newMessageWindow(time.Minute),
		logger:   logr.Discard(),
	}

	fake := newFakeMqttClient()
	s.newClient = fakeClientFactory(fake)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Unbuffered and never read, like KEDA's channel while it is busy
	// scaling: the publish handler must not block paho's delivery.
	active := make(chan bool)
	go s.Run(ctx, active)
	waitReady(t, fake)

	done := make(chan struct{})
	go func() {
		for i := 0; i < 10; i++ {
			fake.deliver(&fakeMqttMessage{topic: "sensors/temp"})
		}
		fake.deliver(&fakeMqttMessage{topic: "sensors/temp", retained: true})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("publish handler blocked on the active channel")
	}

	count, retained := s.window.count()
	if count != 10 {
		t.Errorf("expected 10 counted messages, got %d", count)
	}
	if !retained {
		t.Error("expected retained flag to be set")
	}
}

func TestMqttScalerRetriesFailedSubscription(t *testing.T) {
	s := &mqttScaler{
		metadata:                  mqttScalerMetadata{Topic: "sensors/temp", QoS: 1},
		window:                    newMessageWindow(time.Minute),
		logger:                    logr.Discard(),
		subscribeRetryInterval:    time.Millisecond,
		maxSubscribeRetryInterval: 4 * time.Millisecond,
	}

	fake := newFakeMqttClient()
	fake.subscribeErrs = []error{errors.New("not authorized"), errors.New("not authorized")}
	s.newClient = fakeClientFactory(fake)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.Run(ctx, make(chan bool, 1))

	waitReady(t, fake)

	if fake.subscribeCalls != 3 {
		t.Errorf("expected 2 failed subscribe attempts followed by a success, got %d calls", fake.subscribeCalls)
	}
	if fake.subscribedTo != "sensors/temp" {
		t.Errorf("expected subscription to sensors/temp after retrying, got %q", fake.subscribedTo)
	}
}

func TestMqttScalerSubscribeRetryStopsOnCancel(t *testing.T) {
	s := &mqttScaler{
		metadata:               mqttScalerMetadata{Topic: "sensors/temp", QoS: 1},
		logger:                 logr.Discard(),
		subscribeRetryInterval: time.Hour,
	}

	fake := newFakeMqttClient()
	fake.connected = true
	fake.subscribeErrs = []error{errors.New("not authorized")}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.subscribe(ctx, fake)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("subscribe retry did not stop after context cancellation")
	}
}

func TestMqttScalerCloseDisconnectsClient(t *testing.T) {
	s := &mqttScaler{
		metadata: mqttScalerMetadata{Topic: "sensors/temp", QoS: 1},
		window:   newMessageWindow(time.Minute),
		logger:   logr.Discard(),
	}

	fake := newFakeMqttClient()
	s.newClient = fakeClientFactory(fake)

	ctx, cancel := context.WithCancel(context.Background())
	active := make(chan bool, 1)
	go s.Run(ctx, active)
	waitReady(t, fake)

	if err := s.Close(context.Background()); err != nil {
		t.Fatal("unexpected error from Close", err)
	}
	if !fake.disconnectCalled {
		t.Error("expected Close to disconnect the client")
	}

	cancel()
	// drain until Run closes the channel
	for {
		if _, open := <-active; !open {
			break
		}
	}

	// A message arriving after Run has returned must not panic by
	// sending on the closed active channel.
	fake.deliver(&fakeMqttMessage{topic: "sensors/temp"})
}

func TestMqttScalerCloseBeforeRunPreventsConnect(t *testing.T) {
	s := &mqttScaler{
		metadata: mqttScalerMetadata{Topic: "sensors/temp", QoS: 1},
		window:   newMessageWindow(time.Minute),
		logger:   logr.Discard(),
	}

	created := false
	s.newClient = func(*mqtt.ClientOptions) mqtt.Client {
		created = true
		return newFakeMqttClient()
	}

	if err := s.Close(context.Background()); err != nil {
		t.Fatal("unexpected error from Close", err)
	}

	active := make(chan bool, 1)
	done := make(chan struct{})
	go func() {
		s.Run(context.Background(), active)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after the scaler was closed")
	}
	if created {
		t.Error("expected no client to be created after Close")
	}
	if _, open := <-active; open {
		t.Error("expected the active channel to be closed")
	}
}

func TestMqttScalerReconnectResubscribesAndPreservesState(t *testing.T) {
	sink := &spyLogSink{}
	s := &mqttScaler{
		metadata: mqttScalerMetadata{Topic: "sensors/temp", QoS: 1},
		window:   newMessageWindow(time.Minute),
		logger:   logr.New(sink),
	}

	fake := newFakeMqttClient()
	s.newClient = fakeClientFactory(fake)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	active := make(chan bool, 1)
	go s.Run(ctx, active)

	waitReady(t, fake)
	capturedOpts := fake.opts

	if capturedOpts.OnConnectionLost == nil {
		t.Fatal("expected OnConnectionLost handler to be set")
	}
	if capturedOpts.OnReconnecting == nil {
		t.Fatal("expected OnReconnecting handler to be set")
	}

	// record a message before the "outage" -- this state must survive
	// the reconnect, since it lives on the scaler, not the client.
	fake.deliver(&fakeMqttMessage{topic: "sensors/temp", retained: false})
	countBefore, _ := s.window.count()
	if countBefore != 1 {
		t.Fatalf("expected 1 message recorded before outage, got %d", countBefore)
	}

	errorsBefore := len(sink.errorCalls)
	capturedOpts.OnConnectionLost(fake, errors.New("simulated broker outage"))
	if len(sink.errorCalls) != errorsBefore+1 {
		t.Errorf("expected connection-lost to be logged once, got %d calls", len(sink.errorCalls)-errorsBefore)
	}

	infoBefore := len(sink.infoCalls)
	capturedOpts.OnReconnecting(fake, capturedOpts)
	if len(sink.infoCalls) == infoBefore {
		t.Error("expected reconnect attempt to be logged")
	}

	// simulate the reconnect succeeding -- paho calls OnConnect again,
	// which is exactly what fake.Connect() does.
	fake.subscribedTo = "" // reset to prove resubscription actually happens again
	fake.Connect()

	if fake.subscribedTo != "sensors/temp" {
		t.Error("expected resubscription to sensors/temp after reconnect")
	}

	// the critical assertion: state from before the outage must not be
	// wiped just because the connection dropped and came back.
	countAfter, _ := s.window.count()
	if countAfter != 1 {
		t.Errorf("expected pre-outage message count to survive reconnect, got %d", countAfter)
	}
}
