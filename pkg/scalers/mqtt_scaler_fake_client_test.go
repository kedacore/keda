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
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// fakeToken is a Token that resolves immediately -- every interaction
// in these tests is synchronous, so there is nothing to actually wait on.
type fakeToken struct {
	err error
}

func (t *fakeToken) Wait() bool                    { return true }
func (t *fakeToken) WaitTimeout(time.Duration) bool { return true }
func (t *fakeToken) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}
func (t *fakeToken) Error() error { return t.err }

// fakeMqttMessage is a minimal mqtt.Message for driving the publish
// handler directly in tests, without a real broker.
type fakeMqttMessage struct {
	topic    string
	payload  []byte
	retained bool
	qos      byte
}

func (m *fakeMqttMessage) Duplicate() bool   { return false }
func (m *fakeMqttMessage) Qos() byte         { return m.qos }
func (m *fakeMqttMessage) Retained() bool    { return m.retained }
func (m *fakeMqttMessage) Topic() string     { return m.topic }
func (m *fakeMqttMessage) MessageID() uint16 { return 0 }
func (m *fakeMqttMessage) Payload() []byte   { return m.payload }
func (m *fakeMqttMessage) Ack()              {}

// fakeMqttClient implements mqtt.Client (already an interface in paho,
// not a concrete struct, which is what makes this fake possible without
// any extra abstraction layer of our own). It records what was called
// on it and lets a test manually fire the connect/publish handlers
// instead of going over a real network connection.
type fakeMqttClient struct {
	connected     bool
	connectCalled bool
	subscribedTo  string
	subscribedQoS byte
	connectErr    error

	onConnect      func(mqtt.Client)
	publishHandler mqtt.MessageHandler

	// readyCh is closed once Connect() finishes running onConnect (i.e.
	// once subscription has happened). Tests wait on this instead of
	// sleeping, so there is no arbitrary timing window to get wrong.
	readyCh chan struct{}
}

func newFakeMqttClient() *fakeMqttClient {
	return &fakeMqttClient{readyCh: make(chan struct{})}
}

func (c *fakeMqttClient) Connect() mqtt.Token {
	c.connectCalled = true
	if c.connectErr != nil {
		close(c.readyCh)
		return &fakeToken{err: c.connectErr}
	}
	c.connected = true
	if c.onConnect != nil {
		c.onConnect(c)
	}
	close(c.readyCh)
	return &fakeToken{}
}

func (c *fakeMqttClient) Disconnect(uint) {
	c.connected = false
}

func (c *fakeMqttClient) IsConnected() bool      { return c.connected }
func (c *fakeMqttClient) IsConnectionOpen() bool { return c.connected }

func (c *fakeMqttClient) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token {
	c.subscribedTo = topic
	c.subscribedQoS = qos
	if callback != nil {
		c.publishHandler = callback
	}
	return &fakeToken{}
}

func (c *fakeMqttClient) SubscribeMultiple(map[string]byte, mqtt.MessageHandler) mqtt.Token {
	return &fakeToken{}
}
func (c *fakeMqttClient) Unsubscribe(...string) mqtt.Token { return &fakeToken{} }
func (c *fakeMqttClient) Publish(string, byte, bool, interface{}) mqtt.Token {
	return &fakeToken{}
}
func (c *fakeMqttClient) AddRoute(string, mqtt.MessageHandler) {}
func (c *fakeMqttClient) DeleteRoute(string)                   {}
func (c *fakeMqttClient) OptionsReader() mqtt.ClientOptionsReader {
	return mqtt.ClientOptionsReader{} // unused by mqttScaler; stubbed to satisfy the interface
}

// deliver simulates an incoming message, exactly as paho would invoke
// the registered publish handler when a real message arrives.
func (c *fakeMqttClient) deliver(msg *fakeMqttMessage) {
	if c.publishHandler != nil {
		c.publishHandler(c, msg)
	}
}
