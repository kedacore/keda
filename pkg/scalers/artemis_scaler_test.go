package scalers

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

const (
	username = "myUserName"
	password = "myPassword"
)

type parseArtemisMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type artemisMetricIdentifier struct {
	metadataTestData *parseArtemisMetadataTestData
	triggerIndex     int
	name             string
}

var sampleArtemisResolvedEnv = map[string]string{
	username: "artemis",
	password: "artemis",
}

// A complete valid authParams with username and passwd
var artemisAuthParams = map[string]string{
	"username": "admin",
	"password": "admin",
}

// An invalid authParams without username and passwd
var emptyArtemisAuthParams = map[string]string{
	"username": "",
	"password": "",
}

var testArtemisMetadata = []parseArtemisMetadataTestData{
	// nothing passed
	{map[string]string{}, true},
	// Missing missing managementEndpoint should fail
	{map[string]string{"managementEndpoint": "", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "address1", "username": "myUserName", "password": "myPassword"}, true},
	// Missing queue name, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "", "brokerName": "broker-activemq", "brokerAddress": "address1", "username": "myUserName", "password": "myPassword"}, true},
	// Missing broker name, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "", "brokerAddress": "address1", "username": "myUserName", "password": "myPassword"}, true},
	// Missing broker address, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "", "username": "myUserName", "password": "myPassword"}, true},
	// Missing username, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test", "username": "", "password": "myPassword"}, true},
	// Missing password, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test", "username": "myUserName", "password": ""}, true},
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test", "username": "myUserName", "password": "myPassword"}, false},
	{map[string]string{"restApiTemplate": "http://localhost:8161/console/jolokia/read/org.apache.activemq.artemis:broker=\"broker-activemq\",component=addresses,address=\"test\",subcomponent=queues,routing-type=\"anycast\",queue=\"queue1\"/MessageCount", "username": "myUserName", "password": "myPassword"}, false},
	// Missing brokername , should fail
	{map[string]string{"restApiTemplate": "http://localhost:8161/console/jolokia/read/org.apache.activemq.artemis:broker=\"\",component=addresses,address=\"test\",subcomponent=queues,routing-type=\"anycast\",queue=\"queue1\"/MessageCount", "username": "myUserName", "password": "myPassword"}, true},
	// HTTPS endpoint with unsafeSsl true
	{map[string]string{"managementEndpoint": "localhost:8443", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test", "username": "myUserName", "password": "myPassword", "unsafeSsl": "true"}, false},
	// HTTPS endpoint with unsafeSsl false
	{map[string]string{"managementEndpoint": "localhost:8443", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test", "username": "myUserName", "password": "myPassword", "unsafeSsl": "false"}, false},
	// HTTPS restApiTemplate with unsafeSsl true
	{map[string]string{"restApiTemplate": "https://localhost:8443/console/jolokia/read/org.apache.activemq.artemis:broker=\"broker-activemq\",component=addresses,address=\"test\",subcomponent=queues,routing-type=\"anycast\",queue=\"queue1\"/MessageCount", "username": "myUserName", "password": "myPassword", "unsafeSsl": "true"}, false},
	// HTTPS restApiTemplate with unsafeSsl false
	{map[string]string{"restApiTemplate": "https://localhost:8443/console/jolokia/read/org.apache.activemq.artemis:broker=\"broker-activemq\",component=addresses,address=\"test\",subcomponent=queues,routing-type=\"anycast\",queue=\"queue1\"/MessageCount", "username": "myUserName", "password": "myPassword", "unsafeSsl": "false"}, false},
}

var artemisMetricIdentifiers = []artemisMetricIdentifier{
	{&testArtemisMetadata[7], 0, "s0-artemis-queue1"},
	{&testArtemisMetadata[7], 1, "s1-artemis-queue1"},
}

var testArtemisMetadataWithEmptyAuthParams = []parseArtemisMetadataTestData{
	// nothing passed
	{map[string]string{}, true},
	// Missing managementEndpoint should fail
	{map[string]string{"managementEndpoint": "", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "address1"}, true},
	// Missing queue name, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "", "brokerName": "broker-activemq", "brokerAddress": "address1"}, true},
	// Missing broker name, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "", "brokerAddress": "address1"}, true},
	// Missing broker address, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": ""}, true},
	// Missing username or password, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test"}, true},
}

var testArtemisMetadataWithAuthParams = []parseArtemisMetadataTestData{
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test"}, false},
}

func TestArtemisDefaultCorsHeader(t *testing.T) {
	metadata := map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test", "username": "myUserName", "password": "myPassword"}
	meta, err := parseArtemisMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleArtemisResolvedEnv, TriggerMetadata: metadata, AuthParams: nil})

	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if meta.CorsHeader != "http://localhost:8161" {
		t.Errorf("Expected http://localhost:8161 but got %s", meta.CorsHeader)
	}
}

func TestArtemisCorsHeader(t *testing.T) {
	metadata := map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test", "username": "myUserName", "password": "myPassword", "corsHeader": "test"}
	meta, err := parseArtemisMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleArtemisResolvedEnv, TriggerMetadata: metadata, AuthParams: nil})

	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if meta.CorsHeader != "test" {
		t.Errorf("Expected test but got %s", meta.CorsHeader)
	}
}

func TestArtemisParseMetadata(t *testing.T) {
	for _, testData := range testArtemisMetadata {
		_, err := parseArtemisMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleArtemisResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: nil})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}

	// test with missing auth params should all fail
	for _, testData := range testArtemisMetadataWithEmptyAuthParams {
		_, err := parseArtemisMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleArtemisResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: emptyArtemisAuthParams})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}

	// test with complete auth params should not fail
	for _, testData := range testArtemisMetadataWithAuthParams {
		_, err := parseArtemisMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleArtemisResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: artemisAuthParams})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

// TypedConfig turns a panic in Validate into an error, so these cases check
// the parsed values and the error text, not only whether there is an error.
func TestArtemisRestAPITemplateParsing(t *testing.T) {
	const prefix = "http://localhost:8161/console/jolokia/read/org.apache.activemq.artemis"
	tests := []struct {
		name                               string
		template                           string
		wantBroker, wantAddress, wantQueue string
		wantErr                            string
	}{
		{
			name:        "literal quotes",
			template:    prefix + `:broker="b",component=addresses,address="a",subcomponent=queues,routing-type="anycast",queue="q"/MessageCount`,
			wantBroker:  `"b"`,
			wantAddress: `"a"`,
			wantQueue:   `"q"`,
		},
		{
			name:        "literal quotes and an encoded comma in a value",
			template:    prefix + `:broker="b",component=addresses,address="a",subcomponent=queues,routing-type="anycast",queue="q%2Cr"/MessageCount`,
			wantBroker:  `"b"`,
			wantAddress: `"a"`,
			wantQueue:   `"q,r"`,
		},
		{
			name:        "percent-encoded quotes",
			template:    prefix + ":broker=%22b%22,component=addresses,address=%22a%22,subcomponent=queues,routing-type=%22anycast%22,queue=%22q%22/MessageCount",
			wantBroker:  `"b"`,
			wantAddress: `"a"`,
			wantQueue:   `"q"`,
		},
		{
			name:        "no quotes",
			template:    prefix + ":broker=b,component=addresses,address=a,subcomponent=queues,routing-type=anycast,queue=q/MessageCount",
			wantBroker:  "b",
			wantAddress: "a",
			wantQueue:   "q",
		},
		{
			name:     "empty broker in literal quotes",
			template: prefix + `:broker="",component=addresses,address="a",subcomponent=queues,routing-type="anycast",queue="q"/MessageCount`,
			wantErr:  "no brokerName given",
		},
		{
			name:     "empty broker in percent-encoded quotes",
			template: prefix + ":broker=%22%22,component=addresses,address=%22a%22,subcomponent=queues,routing-type=%22anycast%22,queue=%22q%22/MessageCount",
			wantErr:  "no brokerName given",
		},
		{
			name:     "no queue",
			template: prefix + `:broker="b",component=addresses,address="a"/MessageCount`,
			wantErr:  "no queueName is given",
		},
		{
			name:     "no properties",
			template: prefix + "/MessageCount",
			wantErr:  "no brokerAddress given",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := parseArtemisMetadata(&scalersconfig.ScalerConfig{
				TriggerMetadata: map[string]string{"restApiTemplate": tt.template},
				AuthParams:      artemisAuthParams,
			})
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("parseArtemisMetadata() error = %v, want an error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseArtemisMetadata() error = %v, want nil", err)
			}
			if meta.BrokerName != tt.wantBroker || meta.BrokerAddress != tt.wantAddress || meta.QueueName != tt.wantQueue {
				t.Errorf("parseArtemisMetadata() broker, address, queue = %s, %s, %s, want %s, %s, %s",
					meta.BrokerName, meta.BrokerAddress, meta.QueueName, tt.wantBroker, tt.wantAddress, tt.wantQueue)
			}
		})
	}
}

func TestArtemisGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range artemisMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseArtemisMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleArtemisResolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: nil, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockArtemisScaler := artemisScaler{
			metadata:   meta,
			httpClient: http.DefaultClient,
		}

		metricSpec := mockArtemisScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestArtemisUnsafeSslDefaultValue(t *testing.T) {
	metadata := map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test", "username": "myUserName", "password": "myPassword"}
	meta, err := parseArtemisMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleArtemisResolvedEnv, TriggerMetadata: metadata, AuthParams: nil})

	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if meta.UnsafeSsl != false {
		t.Errorf("Expected UnsafeSsl to be false by default, but got %v", meta.UnsafeSsl)
	}
}

func TestArtemisUnsafeSslTrue(t *testing.T) {
	metadata := map[string]string{"managementEndpoint": "localhost:8443", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test", "username": "myUserName", "password": "myPassword", "unsafeSsl": "true"}
	meta, err := parseArtemisMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleArtemisResolvedEnv, TriggerMetadata: metadata, AuthParams: nil})

	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if meta.UnsafeSsl != true {
		t.Errorf("Expected UnsafeSsl to be true, but got %v", meta.UnsafeSsl)
	}
}

func TestArtemisUnsafeSslFalse(t *testing.T) {
	metadata := map[string]string{"managementEndpoint": "localhost:8443", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test", "username": "myUserName", "password": "myPassword", "unsafeSsl": "false"}
	meta, err := parseArtemisMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleArtemisResolvedEnv, TriggerMetadata: metadata, AuthParams: nil})

	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if meta.UnsafeSsl != false {
		t.Errorf("Expected UnsafeSsl to be false, but got %v", meta.UnsafeSsl)
	}
}

func TestArtemisTLSWithCertAndKey(t *testing.T) {
	metadata := map[string]string{"managementEndpoint": "localhost:8443", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test", "username": "myUserName", "password": "myPassword"}
	authParams := map[string]string{"ca": "caaa", "cert": "ceert", "key": "keey"}
	meta, err := parseArtemisMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleArtemisResolvedEnv, TriggerMetadata: metadata, AuthParams: authParams})

	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if meta.Ca != "caaa" {
		t.Errorf("Expected Ca to be 'caaa', but got %s", meta.Ca)
	}
	if meta.Cert != "ceert" {
		t.Errorf("Expected Cert to be 'ceert', but got %s", meta.Cert)
	}
	if meta.Key != "keey" {
		t.Errorf("Expected Key to be 'keey', but got %s", meta.Key)
	}
}

func TestArtemisTLSWithKeyPassword(t *testing.T) {
	metadata := map[string]string{"managementEndpoint": "localhost:8443", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test", "username": "myUserName", "password": "myPassword"}
	authParams := map[string]string{"ca": "caaa", "cert": "ceert", "key": "keey", "keyPassword": "secret123"}
	meta, err := parseArtemisMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleArtemisResolvedEnv, TriggerMetadata: metadata, AuthParams: authParams})

	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if meta.KeyPassword != "secret123" {
		t.Errorf("Expected KeyPassword to be 'secret123', but got %s", meta.KeyPassword)
	}
}

func TestArtemisTLSMissingCert(t *testing.T) {
	metadata := map[string]string{"managementEndpoint": "localhost:8443", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test", "username": "myUserName", "password": "myPassword"}
	authParams := map[string]string{"ca": "caaa", "key": "keey"}
	_, err := parseArtemisMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleArtemisResolvedEnv, TriggerMetadata: metadata, AuthParams: authParams})

	if err == nil {
		t.Error("Expected error for missing cert when key is provided, but got success")
	}
}

func TestArtemisTLSMissingKey(t *testing.T) {
	metadata := map[string]string{"managementEndpoint": "localhost:8443", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test", "username": "myUserName", "password": "myPassword"}
	authParams := map[string]string{"ca": "caaa", "cert": "ceert"}
	_, err := parseArtemisMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleArtemisResolvedEnv, TriggerMetadata: metadata, AuthParams: authParams})

	if err == nil {
		t.Error("Expected error for missing key when cert is provided, but got success")
	}
}

func TestArtemisTLSCaOnly(t *testing.T) {
	metadata := map[string]string{"managementEndpoint": "localhost:8443", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test", "username": "myUserName", "password": "myPassword"}
	authParams := map[string]string{"ca": "caaa"}
	meta, err := parseArtemisMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleArtemisResolvedEnv, TriggerMetadata: metadata, AuthParams: authParams})

	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if meta.Ca != "caaa" {
		t.Errorf("Expected Ca to be 'caaa', but got %s", meta.Ca)
	}
	if meta.Cert != "" {
		t.Error("Expected empty Cert")
	}
	if meta.Key != "" {
		t.Error("Expected empty Key")
	}
}
