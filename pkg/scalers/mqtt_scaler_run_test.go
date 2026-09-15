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

func (s *spyLogSink) Init(logr.RuntimeInfo)               {}
func (s *spyLogSink) Enabled(int) bool                    { return true }
func (s *spyLogSink) Info(_ int, msg string, _ ...interface{}) { s.infoCalls = append(s.infoCalls, msg) }
func (s *spyLogSink) Error(_ error, msg string, _ ...interface{}) {
	s.errorCalls = append(s.errorCalls, msg)
}
func (s *spyLogSink) WithValues(...interface{}) logr.LogSink { return s }
func (s *spyLogSink) WithName(string) logr.LogSink           { return s }

// subscribingOnConnect mirrors the real OnConnectHandler set up in
// Run(): on every (re)connect, subscribe and route retained vs. live
// messages into the scaler's window, exactly as production code does.
func subscribingOnConnect(s *mqttScaler) func(mqtt.Client) {
	return func(c mqtt.Client) {
		c.Subscribe(s.metadata.Topic, byte(s.metadata.QoS), func(_ mqtt.Client, msg mqtt.Message) {
			if msg.Retained() {
				s.window.markRetained()
				return
			}
			s.window.record()
		})
	}
}

func TestMqttScalerRunRecordsMessages(t *testing.T) {
	s := &mqttScaler{
		metadata: mqttScalerMetadata{Topic: "sensors/temp", QoS: 1},
		window:   newMessageWindow(time.Minute),
		logger:   logr.Discard(),
	}

	fake := newFakeMqttClient()
	s.newClient = func(*mqtt.ClientOptions) mqtt.Client {
		fake.onConnect = subscribingOnConnect(s)
		return fake
	}

	ctx, cancel := context.WithCancel(context.Background())
	active := make(chan bool, 1)
	go s.Run(ctx, active)

	select {
	case <-fake.readyCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Run to connect")
	}

	if !fake.connectCalled {
		t.Fatal("expected Connect to be called")
	}
	if fake.subscribedTo != "sensors/temp" {
		t.Errorf("expected subscription to sensors/temp, got %s", fake.subscribedTo)
	}

	fake.deliver(&fakeMqttMessage{topic: "sensors/temp", retained: false})
	fake.deliver(&fakeMqttMessage{topic: "sensors/temp", retained: false})
	fake.deliver(&fakeMqttMessage{topic: "sensors/temp", retained: true})

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

	cancel()
}

func TestMqttScalerReconnectResubscribesAndPreservesState(t *testing.T) {
	sink := &spyLogSink{}
	s := &mqttScaler{
		metadata: mqttScalerMetadata{Topic: "sensors/temp", QoS: 1},
		window:   newMessageWindow(time.Minute),
		logger:   logr.New(sink),
	}

	fake := newFakeMqttClient()
	var capturedOpts *mqtt.ClientOptions
	s.newClient = func(opts *mqtt.ClientOptions) mqtt.Client {
		capturedOpts = opts
		fake.onConnect = subscribingOnConnect(s)
		return fake
	}

	ctx, cancel := context.WithCancel(context.Background())
	active := make(chan bool, 1)
	go s.Run(ctx, active)

	select {
	case <-fake.readyCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for initial connect")
	}

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

	capturedOpts.OnConnectionLost(fake, errors.New("simulated broker outage"))
	if len(sink.errorCalls) != 1 {
		t.Errorf("expected connection-lost to be logged once, got %d calls", len(sink.errorCalls))
	}

	capturedOpts.OnReconnecting(fake, capturedOpts)
	if len(sink.infoCalls) == 0 {
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

	cancel()
}
