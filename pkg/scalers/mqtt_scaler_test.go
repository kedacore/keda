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
	"testing"
	"time"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseMqttScalerMetadataTestData struct {
	name     string
	metadata map[string]string
	isError  bool
}

var testMqttScalerMetadata = []parseMqttScalerMetadataTestData{
	{
		name:     "missing brokerAddress",
		metadata: map[string]string{"topic": "sensors/temp", "queryValue": "10"},
		isError:  true,
	},
	{
		name:     "missing topic",
		metadata: map[string]string{"brokerAddress": "tcp://localhost:1883", "queryValue": "10"},
		isError:  true,
	},
	{
		name:     "missing queryValue",
		metadata: map[string]string{"brokerAddress": "tcp://localhost:1883", "topic": "sensors/temp"},
		isError:  true,
	},
	{
		name: "all required fields present, no optional overrides",
		metadata: map[string]string{
			"brokerAddress": "tcp://localhost:1883",
			"topic":         "sensors/temp",
			"queryValue":    "10",
		},
		isError: false,
	},
	{
		name: "all fields explicitly set, including qos=0",
		metadata: map[string]string{
			"brokerAddress":               "tcp://localhost:1883",
			"topic":                       "sensors/temp",
			"queryValue":                  "10",
			"windowSeconds":               "45",
			"qos":                         "0",
			"connectRetryIntervalSeconds": "5",
			"maxReconnectIntervalSeconds": "120",
		},
		isError: false,
	},
}

func TestMqttParseMetadata(t *testing.T) {
	for _, testData := range testMqttScalerMetadata {
		t.Run(testData.name, func(t *testing.T) {
			meta, err := parseMqttScalerMetadata(&scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadata,
				ResolvedEnv:     map[string]string{},
			})

			if testData.isError {
				if err == nil {
					t.Error("expected error but got success")
				}
				return
			}
			if err != nil {
				t.Error("expected success but got error", err)
			}
			if meta.BrokerAddress != testData.metadata["brokerAddress"] {
				t.Errorf("expected brokerAddress %s, got %s", testData.metadata["brokerAddress"], meta.BrokerAddress)
			}
			if meta.Topic != testData.metadata["topic"] {
				t.Errorf("expected topic %s, got %s", testData.metadata["topic"], meta.Topic)
			}
		})
	}
}

// TestMqttParseMetadataDefaults locks in the specific behavior a naive
// `if meta.QoS == 0 { meta.QoS = default }` pattern would get wrong: an
// omitted field should get the tag's default, while an explicitly
// provided zero value must be preserved as-is.
func TestMqttParseMetadataDefaults(t *testing.T) {
	baseMetadata := map[string]string{
		"brokerAddress": "tcp://localhost:1883",
		"topic":         "sensors/temp",
		"queryValue":    "10",
	}

	t.Run("omitted qos gets default of 1", func(t *testing.T) {
		meta, err := parseMqttScalerMetadata(&scalersconfig.ScalerConfig{
			TriggerMetadata: baseMetadata,
			ResolvedEnv:     map[string]string{},
		})
		if err != nil {
			t.Fatal("expected success but got error", err)
		}
		if meta.QoS != 1 {
			t.Errorf("expected default qos 1, got %d", meta.QoS)
		}
	})

	t.Run("explicit qos=0 is preserved, not overwritten by default", func(t *testing.T) {
		metadata := map[string]string{}
		for k, v := range baseMetadata {
			metadata[k] = v
		}
		metadata["qos"] = "0"

		meta, err := parseMqttScalerMetadata(&scalersconfig.ScalerConfig{
			TriggerMetadata: metadata,
			ResolvedEnv:     map[string]string{},
		})
		if err != nil {
			t.Fatal("expected success but got error", err)
		}
		if meta.QoS != 0 {
			t.Errorf("expected explicit qos 0 to be preserved, got %d", meta.QoS)
		}
	})

	t.Run("omitted windowSeconds gets default of 30", func(t *testing.T) {
		meta, err := parseMqttScalerMetadata(&scalersconfig.ScalerConfig{
			TriggerMetadata: baseMetadata,
			ResolvedEnv:     map[string]string{},
		})
		if err != nil {
			t.Fatal("expected success but got error", err)
		}
		if meta.WindowSeconds != 30 {
			t.Errorf("expected default windowSeconds 30, got %d", meta.WindowSeconds)
		}
	})
}

func TestMessageWindowCount(t *testing.T) {
	w := newMessageWindow(50 * time.Millisecond)

	w.record()
	w.record()
	count, retained := w.count()
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
	if retained {
		t.Error("expected retained to be false, got true")
	}

	time.Sleep(60 * time.Millisecond)
	count, _ = w.count()
	if count != 0 {
		t.Errorf("expected count 0 after window expiry, got %d", count)
	}
}

func TestMessageWindowRetainedExpiresAfterWindow(t *testing.T) {
	w := newMessageWindow(50 * time.Millisecond)

	w.markRetained()
	_, retained := w.count()
	if !retained {
		t.Error("expected retained to be true immediately after markRetained")
	}

	// A second immediate read must still see it -- retained state must
	// not be consumed by being read, since multiple independent callers
	// (KEDA's operator, the metrics adapter) poll concurrently.
	_, retained = w.count()
	if !retained {
		t.Error("expected retained to still be true on a second immediate read")
	}

	time.Sleep(60 * time.Millisecond)
	_, retained = w.count()
	if retained {
		t.Error("expected retained to expire once outside the window")
	}
}

// TestMessageWindowConcurrentAccess exercises the exact race this design
// is meant to prevent. It makes no assertions of its own -- its value
// only shows up under `go test -race`.
func TestMessageWindowConcurrentAccess(t *testing.T) {
	w := newMessageWindow(time.Second)
	done := make(chan struct{})

	go func() {
		for i := 0; i < 1000; i++ {
			w.record()
		}
		close(done)
	}()

	go func() {
		for i := 0; i < 1000; i++ {
			w.count()
		}
	}()

	<-done
}
