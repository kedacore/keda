package scalers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	host             = "myHostSecret"
	rabbitMQUsername = "myUsernameSecret"
	rabbitMQPassword = "myPasswordSecret"
)

type parseRabbitMQMetadataTestData struct {
	metadata   map[string]string
	isError    bool
	authParams map[string]string
}

type parseRabbitMQAuthParamTestData struct {
	metadata         map[string]string
	podIdentity      v1alpha1.AuthPodIdentity
	authParams       map[string]string
	isError          bool
	enableTLS        string
	workloadIdentity bool
}

type rabbitMQMetricIdentifier struct {
	metadataTestData *parseRabbitMQMetadataTestData
	index            int
	name             string
}

var sampleRabbitMqResolvedEnv = map[string]string{
	host:             "amqp://user:sercet@somehost.com:5236/vhost",
	rabbitMQUsername: "user",
	rabbitMQPassword: "Password",
}

var testRabbitMQMetadata = []parseRabbitMQMetadataTestData{
	// nothing passed
	{map[string]string{}, true, map[string]string{}},
	// properly formed metadata
	{map[string]string{"queueLength": "10", "queueName": "sample", "hostFromEnv": host}, false, map[string]string{}},
	// malformed queueLength
	{map[string]string{"queueLength": "AA", "queueName": "sample", "hostFromEnv": host}, true, map[string]string{}},
	// missing host
	{map[string]string{"queueLength": "AA", "queueName": "sample"}, true, map[string]string{}},
	// missing queueName
	{map[string]string{"queueLength": "10", "hostFromEnv": host}, true, map[string]string{}},
	// host defined in authParams
	{map[string]string{"queueLength": "10", "hostFromEnv": host}, true, map[string]string{"host": host}},
	// properly formed metadata with http protocol
	{map[string]string{"queueLength": "10", "queueName": "sample", "host": host, "protocol": "http"}, false, map[string]string{}},
	// queue name with slashes
	{map[string]string{"queueLength": "10", "queueName": "namespace/name", "hostFromEnv": host}, false, map[string]string{}},
	// vhost passed
	{map[string]string{"vhostName": "myVhost", "queueName": "namespace/name", "hostFromEnv": host}, false, map[string]string{}},
	// vhost passed but empty
	{map[string]string{"vhostName": "", "queueName": "namespace/name", "hostFromEnv": host}, false, map[string]string{}},
	// protocol defined in authParams
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, false, map[string]string{"protocol": "http"}},
	// auto protocol and a bad URL
	{map[string]string{"queueName": "sample", "host": "something://"}, true, map[string]string{}},
	// auto protocol and an HTTP URL
	{map[string]string{"queueName": "sample", "host": "http://"}, false, map[string]string{}},
	// auto protocol and an HTTPS URL
	{map[string]string{"queueName": "sample", "host": "https://"}, false, map[string]string{}},
	// queueLength and mode
	{map[string]string{"queueLength": "10", "mode": "QueueLength", "queueName": "sample", "host": "https://"}, true, map[string]string{}},
	// queueLength and value
	{map[string]string{"queueLength": "10", "value": "20", "queueName": "sample", "host": "https://"}, true, map[string]string{}},
	// queueLength and mode and value
	{map[string]string{"queueLength": "10", "mode": "QueueLength", "value": "20", "queueName": "sample", "host": "https://"}, true, map[string]string{}},
	// only mode
	{map[string]string{"mode": "QueueLength", "queueName": "sample", "host": "https://"}, true, map[string]string{}},
	// only value
	{map[string]string{"value": "20", "queueName": "sample", "host": "https://"}, true, map[string]string{}},
	// mode and value
	{map[string]string{"mode": "QueueLength", "value": "20", "queueName": "sample", "host": "https://"}, false, map[string]string{}},
	// invalid mode
	{map[string]string{"mode": "Feelings", "value": "20", "queueName": "sample", "host": "https://"}, true, map[string]string{}},
	// invalid value
	{map[string]string{"mode": "QueueLength", "value": "lots", "queueName": "sample", "host": "https://"}, true, map[string]string{}},
	// queue length amqp
	{map[string]string{"mode": "QueueLength", "value": "20", "queueName": "sample", "host": "amqps://"}, false, map[string]string{}},
	// message rate amqp
	{map[string]string{"mode": "MessageRate", "value": "1000", "queueName": "sample", "host": "amqps://"}, true, map[string]string{}},
	// message rate amqp
	{map[string]string{"mode": "MessageRate", "value": "1000", "queueName": "sample", "host": "amqp://"}, true, map[string]string{}},
	// message rate amqp
	{map[string]string{"mode": "MessageRate", "value": "1000", "queueName": "sample", "host": "http://"}, false, map[string]string{}},
	// message rate amqp
	{map[string]string{"mode": "MessageRate", "value": "1000", "queueName": "sample", "host": "https://"}, false, map[string]string{}},
	// deliverGet rate amqps
	{map[string]string{"mode": "DeliverGetRate", "value": "1000", "queueName": "sample", "host": "amqps://"}, true, map[string]string{}},
	// deliverGet rate amqp
	{map[string]string{"mode": "DeliverGetRate", "value": "1000", "queueName": "sample", "host": "amqp://"}, true, map[string]string{}},
	// deliverGet rate http
	{map[string]string{"mode": "DeliverGetRate", "value": "1000", "queueName": "sample", "host": "http://"}, false, map[string]string{}},
	// deliverGet rate https
	{map[string]string{"mode": "DeliverGetRate", "value": "1000", "queueName": "sample", "host": "https://"}, false, map[string]string{}},
	// amqp host and useRegex
	{map[string]string{"queueName": "sample", "host": "amqps://", "useRegex": "true"}, true, map[string]string{}},
	// http host and useRegex
	{map[string]string{"queueName": "sample", "host": "http://", "useRegex": "true"}, false, map[string]string{}},
	// message rate and useRegex
	{map[string]string{"mode": "MessageRate", "value": "1000", "queueName": "sample", "host": "http://", "useRegex": "true"}, false, map[string]string{}},
	// deliverGet rate and useRegex
	{map[string]string{"mode": "DeliverGetRate", "value": "1000", "queueName": "sample", "host": "http://", "useRegex": "true"}, false, map[string]string{}},
	// queue length and useRegex
	{map[string]string{"mode": "QueueLength", "value": "1000", "queueName": "sample", "host": "http://", "useRegex": "true"}, false, map[string]string{}},
	// http valid timeout
	{map[string]string{"mode": "QueueLength", "value": "1000", "queueName": "sample", "host": "http://", "timeout": "1000"}, false, map[string]string{}},
	// http invalid timeout
	{map[string]string{"mode": "QueueLength", "value": "1000", "queueName": "sample", "host": "http://", "timeout": "-10"}, true, map[string]string{}},
	// http wrong timeout
	{map[string]string{"mode": "QueueLength", "value": "1000", "queueName": "sample", "host": "http://", "timeout": "error"}, true, map[string]string{}},
	// amqp timeout
	{map[string]string{"mode": "QueueLength", "value": "1000", "queueName": "sample", "host": "amqp://", "timeout": "10"}, true, map[string]string{}},
	// valid pageSize
	{map[string]string{"mode": "MessageRate", "value": "1000", "queueName": "sample", "host": "http://", "useRegex": "true", "pageSize": "100"}, false, map[string]string{}},
	// pageSize less than 1
	{map[string]string{"mode": "MessageRate", "value": "1000", "queueName": "sample", "host": "http://", "useRegex": "true", "pageSize": "-1"}, true, map[string]string{}},
	// invalid pageSize
	{map[string]string{"mode": "MessageRate", "value": "1000", "queueName": "sample", "host": "http://", "useRegex": "true", "pageSize": "a"}, true, map[string]string{}},
	// valid pageSize
	{map[string]string{"mode": "DeliverGetRate", "value": "1000", "queueName": "sample", "host": "http://", "useRegex": "true", "pageSize": "100"}, false, map[string]string{}},
	// pageSize less than 1
	{map[string]string{"mode": "DeliverGetRate", "value": "1000", "queueName": "sample", "host": "http://", "useRegex": "true", "pageSize": "-1"}, true, map[string]string{}},
	// invalid pageSize
	{map[string]string{"mode": "DeliverGetRate", "value": "1000", "queueName": "sample", "host": "http://", "useRegex": "true", "pageSize": "a"}, true, map[string]string{}},
	// activationValue passed
	{map[string]string{"activationValue": "10", "queueLength": "20", "queueName": "sample", "hostFromEnv": host}, false, map[string]string{}},
	// malformed activationValue
	{map[string]string{"activationValue": "AA", "queueLength": "10", "queueName": "sample", "hostFromEnv": host}, true, map[string]string{}},
	// http and excludeUnacknowledged
	{map[string]string{"mode": "QueueLength", "value": "1000", "queueName": "sample", "host": "http://", "useRegex": "true", "excludeUnacknowledged": "true"}, false, map[string]string{}},
	// amqp and excludeUnacknowledged
	{map[string]string{"mode": "QueueLength", "value": "1000", "queueName": "sample", "host": "amqp://", "useRegex": "true", "excludeUnacknowledged": "true"}, true, map[string]string{}},
	// unsafeSsl true
	{map[string]string{"queueName": "sample", "host": "https://", "unsafeSsl": "true"}, false, map[string]string{}},
	// unsafeSsl wrong input
	{map[string]string{"queueName": "sample", "host": "https://", "unsafeSsl": "random"}, true, map[string]string{}},
}

var testRabbitMQAuthParamData = []parseRabbitMQAuthParamTestData{
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, v1alpha1.AuthPodIdentity{}, map[string]string{"tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, false, rmqTLSEnable, false},
	// success, TLS cert/key and assumed public CA
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, v1alpha1.AuthPodIdentity{}, map[string]string{"tls": "enable", "cert": "ceert", "key": "keey"}, false, rmqTLSEnable, false},
	// success, TLS cert/key + key password and assumed public CA
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, v1alpha1.AuthPodIdentity{}, map[string]string{"tls": "enable", "cert": "ceert", "key": "keey", "keyPassword": "keeyPassword"}, false, rmqTLSEnable, false},
	// success, TLS CA only
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, v1alpha1.AuthPodIdentity{}, map[string]string{"tls": "enable", "ca": "caaa"}, false, rmqTLSEnable, false},
	// failure, TLS missing cert
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, v1alpha1.AuthPodIdentity{}, map[string]string{"tls": "enable", "ca": "caaa", "key": "kee"}, true, rmqTLSEnable, false},
	// failure, TLS missing key
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, v1alpha1.AuthPodIdentity{}, map[string]string{"tls": "enable", "ca": "caaa", "cert": "ceert"}, true, rmqTLSEnable, false},
	// failure, TLS invalid
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, v1alpha1.AuthPodIdentity{}, map[string]string{"tls": "yes", "ca": "caaa", "cert": "ceert", "key": "kee"}, true, rmqTLSEnable, false},
	// success, username and password
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, v1alpha1.AuthPodIdentity{}, map[string]string{"username": "user", "password": "PASSWORD"}, false, rmqTLSDisable, false},
	// failure, username but no password
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, v1alpha1.AuthPodIdentity{}, map[string]string{"username": "user"}, true, rmqTLSDisable, false},
	// failure, password but no username
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, v1alpha1.AuthPodIdentity{}, map[string]string{"password": "PASSWORD"}, true, rmqTLSDisable, false},
	// success, vhostName
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, v1alpha1.AuthPodIdentity{}, map[string]string{"vhostName": "myVhost"}, false, rmqTLSDisable, false},
	// success, vhostName but empty
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, v1alpha1.AuthPodIdentity{}, map[string]string{"vhostName": ""}, false, rmqTLSDisable, false},
	// success, username and password from env
	{map[string]string{"queueName": "sample", "hostFromEnv": host, "usernameFromEnv": rabbitMQUsername, "passwordFromEnv": rabbitMQPassword}, v1alpha1.AuthPodIdentity{}, map[string]string{}, false, rmqTLSDisable, false},
	// failure, username from env but not password
	{map[string]string{"queueName": "sample", "hostFromEnv": host, "usernameFromEnv": rabbitMQUsername}, v1alpha1.AuthPodIdentity{}, map[string]string{}, true, rmqTLSDisable, false},
	// failure, password from env but not username
	{map[string]string{"queueName": "sample", "hostFromEnv": host, "passwordFromEnv": rabbitMQPassword}, v1alpha1.AuthPodIdentity{}, map[string]string{}, true, rmqTLSDisable, false},
	// success, WorkloadIdentity
	{map[string]string{"queueName": "sample", "hostFromEnv": host, "protocol": "http"}, v1alpha1.AuthPodIdentity{Provider: v1alpha1.PodIdentityProviderAzureWorkload, IdentityID: kedautil.StringPointer("client-id")}, map[string]string{"workloadIdentityResource": "rabbitmq-resource-id"}, false, rmqTLSDisable, true},
	// failure, WorkloadIdentity not supported for amqp
	{map[string]string{"queueName": "sample", "hostFromEnv": host, "protocol": "amqp"}, v1alpha1.AuthPodIdentity{Provider: v1alpha1.PodIdentityProviderAzureWorkload, IdentityID: kedautil.StringPointer("client-id")}, map[string]string{"workloadIdentityResource": "rabbitmq-resource-id"}, true, rmqTLSDisable, false},
}
var rabbitMQMetricIdentifiers = []rabbitMQMetricIdentifier{
	{&testRabbitMQMetadata[1], 0, "s0-rabbitmq-sample"},
	{&testRabbitMQMetadata[7], 1, "s1-rabbitmq-namespace-2Fname"},
}

func TestRabbitMQParseMetadata(t *testing.T) {
	for idx, testData := range testRabbitMQMetadata {
		meta, err := parseRabbitMQMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleRabbitMqResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error but got success in test case %d", idx)
		}
		if val, ok := testData.metadata["unsafeSsl"]; ok && err == nil {
			boolVal, err := strconv.ParseBool(val)
			if err != nil && !testData.isError {
				t.Errorf("Expect error but got success in test case %d", idx)
			}
			if boolVal != meta.UnsafeSsl {
				t.Errorf("Expect %t but got %t in test case %d", boolVal, meta.UnsafeSsl, idx)
			}
		}
	}
}

func TestRabbitMQParseAuthParamData(t *testing.T) {
	for _, testData := range testRabbitMQAuthParamData {
		metadata, err := parseRabbitMQMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleRabbitMqResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testData.authParams, PodIdentity: testData.podIdentity})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		if metadata != nil && metadata.EnableTLS != testData.enableTLS {
			t.Errorf("Expected enableTLS to be set to %v but got %v\n", testData.enableTLS, metadata.EnableTLS)
		}
		if metadata != nil && metadata.EnableTLS == rmqTLSEnable {
			if metadata.Ca != testData.authParams["ca"] {
				t.Errorf("Expected ca to be set to %v but got %v\n", testData.authParams["ca"], metadata.EnableTLS)
			}
			if metadata.Cert != testData.authParams["cert"] {
				t.Errorf("Expected cert to be set to %v but got %v\n", testData.authParams["cert"], metadata.Cert)
			}
			if metadata.Key != testData.authParams["key"] {
				t.Errorf("Expected key to be set to %v but got %v\n", testData.authParams["key"], metadata.Key)
			}
			if metadata.KeyPassword != testData.authParams["keyPassword"] {
				t.Errorf("Expected key to be set to %v but got %v\n", testData.authParams["keyPassword"], metadata.Key)
			}
		}
		if metadata != nil && metadata.workloadIdentityClientID != "" && !testData.workloadIdentity {
			t.Errorf("Expected workloadIdentity to be disabled but got %v as client ID and %v as resource\n", metadata.workloadIdentityClientID, metadata.WorkloadIdentityResource)
		}
		if metadata != nil && metadata.workloadIdentityClientID == "" && testData.workloadIdentity {
			t.Error("Expected workloadIdentity to be enabled but was not\n")
		}
	}
}

var testDefaultQueueLength = []parseRabbitMQMetadataTestData{
	// use default queueLength
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, false, map[string]string{}},
	// use default queueLength with includeUnacked
	{map[string]string{"queueName": "sample", "hostFromEnv": host, "protocol": "http"}, false, map[string]string{}},
}

func TestParseDefaultQueueLength(t *testing.T) {
	for _, testData := range testDefaultQueueLength {
		metadata, err := parseRabbitMQMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleRabbitMqResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		switch {
		case err != nil && !testData.isError:
			t.Error("Expected success but got error", err)
		case testData.isError && err == nil:
			t.Error("Expected error but got success")
		case metadata.Value != defaultRabbitMQQueueLength:
			t.Error("Expected default queueLength =", defaultRabbitMQQueueLength, "but got", metadata.Value)
		}
	}
}

type getQueueInfoTestData struct {
	response       string
	responseStatus int
	isActive       bool
	extraMetadata  map[string]string
	vhostPath      string
	urlPath        string
}

var testQueueInfoTestData = []getQueueInfoTestData{
	// queueLength
	{response: `{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10"}},
	{response: `{"messages": 1, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10"}},
	{response: `{"messages": 1, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10"}},
	{response: `{"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"queueLength": "10"}},
	{response: `{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 1.4}, "deliver_get_details": {"rate": 0.8}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10"}},
	{response: `{"messages": 1, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 1.4}, "deliver_get_details": {"rate": 1.2}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10"}},
	{response: `{"messages": 1, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 1.4}, "deliver_get_details": {"rate": 1.6}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10"}},
	{response: `{"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 1.4}, "deliver_get_details": {"rate": 2}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"queueLength": "10"}},
	// mode QueueLength
	{response: `{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 1.3}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"value": "100", "mode": "QueueLength"}},
	{response: `{"messages": 1, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 1}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"value": "100", "mode": "QueueLength"}},
	{response: `{"messages": 1, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 1}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"value": "100", "mode": "QueueLength"}},
	{response: `{"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"value": "100", "mode": "QueueLength"}},
	{response: `{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 1.4}, "deliver_get_details": {"rate": 1.3}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"value": "100", "mode": "QueueLength"}},
	{response: `{"messages": 1, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 1.4}, "deliver_get_details": {"rate": 1}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"value": "100", "mode": "QueueLength"}},
	{response: `{"messages": 1, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 1.4}, "deliver_get_details": {"rate": 1}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"value": "100", "mode": "QueueLength"}},
	{response: `{"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 1.4}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"value": "100", "mode": "QueueLength"}},
	// mode MessageRate
	{response: `{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 1.3}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"value": "100", "mode": "MessageRate"}},
	{response: `{"messages": 1, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 1}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"value": "100", "mode": "MessageRate"}},
	{response: `{"messages": 1, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 1}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"value": "100", "mode": "MessageRate"}},
	{response: `{"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"value": "100", "mode": "MessageRate"}},
	{response: `{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 1.4}, "deliver_get_details": {"rate": 1.3}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"value": "100", "mode": "MessageRate"}},
	{response: `{"messages": 1, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 1.4}, "deliver_get_details": {"rate": 1}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"value": "100", "mode": "MessageRate"}},
	{response: `{"messages": 1, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 1.4}, "deliver_get_details": {"rate": 1}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"value": "100", "mode": "MessageRate"}},
	{response: `{"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 1.4}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"value": "100", "mode": "MessageRate"}},
	// mode DeliverGetRate
	{response: `{"messages": 30, "messages_unacknowledged": 10, "message_stats": {"publish_details": {"rate": 5}, "deliver_get_details": {"rate": 1}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"value": "100", "mode": "DeliverGetRate"}},
	{response: `{"messages": 20, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 10}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"value": "100", "mode": "DeliverGetRate"}},
	{response: `{"messages": 10, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 12.5}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"value": "100", "mode": "DeliverGetRate"}},
	{response: `{"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: false, extraMetadata: map[string]string{"value": "100", "mode": "DeliverGetRate"}},
	{response: `{"messages": 45, "messages_unacknowledged": 5, "message_stats": {"publish_details": {"rate": 5.5}, "deliver_get_details": {"rate": 1.5}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"value": "100", "mode": "DeliverGetRate"}},
	{response: `{"messages": 22, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 1.5}, "deliver_get_details": {"rate": 23}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"value": "100", "mode": "DeliverGetRate"}},
	{response: `{"messages": 3, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 18.5}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"value": "100", "mode": "DeliverGetRate"}},
	{response: `{"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: false, extraMetadata: map[string]string{"value": "100", "mode": "DeliverGetRate"}},

	// error response
	{response: `Password is incorrect`, responseStatus: http.StatusUnauthorized},
}

var testQueueInfoTestDataSingleVhost = []getQueueInfoTestData{
	{response: `{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 1.4}, "deliver_get_details": {"rate": 1.6}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"hostFromEnv": "plainHost", "vhostName": "myhost"}, vhostPath: "/myhost"},
	{response: `{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 1.4}, "deliver_get_details": {"rate": 2}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"hostFromEnv": "plainHost", "vhostName": "/"}, vhostPath: "//"},
	{response: `{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 1.4}, "deliver_get_details": {"rate": 2.5}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"hostFromEnv": "plainHost", "vhostName": ""}},
	{response: `{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"hostFromEnv": "plainHost", "vhostName": "myhost"}, vhostPath: "/myhost"},
	{response: `{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"hostFromEnv": "plainHost", "vhostName": "/"}, vhostPath: rabbitRootVhostPath},
	{response: `{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"hostFromEnv": "plainHost", "vhostName": ""}, vhostPath: "/"},
}

type resolvedVhostAndPathTestData struct {
	rawPath           string
	overrideVhost     string
	resolvedVhostPath string
	resolvedPath      string
}

var getVhostAndPathFromURLTestData = []resolvedVhostAndPathTestData{
	// level 0 + vhost
	{"myVhost", "", "/myVhost", ""},

	// level 0 + vhost as /, // and empty
	{"/", "", rabbitRootVhostPath, ""},
	{"//", "", rabbitRootVhostPath, ""},
	{"", "", rabbitRootVhostPath, ""},

	// level 1 + vhost
	{"sub1/myVhost", "", "/myVhost", "/sub1"},
	{"sub1/", "overridenVhost", "/overridenVhost", "/sub1"},
	{"myVhost", "overridenVhost", "/overridenVhost", ""},

	// level 1 + vhost as / and //
	{"sub1/", "", rabbitRootVhostPath, "/sub1"},
	{"sub1/", "myVhost", "/myVhost", "/sub1"},
	{"myVhost", "overridenVhost", "/overridenVhost", ""},
	{"sub1//", "", rabbitRootVhostPath, "/sub1"},

	// level 2 + vhost
	{"sub1/sub2/myVhost", "", "/myVhost", "/sub1/sub2"},
	{"sub1/sub2/myVhost", "overridenVhost", "/overridenVhost", "/sub1/sub2"},
	{"myVhost", "overridenVhost", "/overridenVhost", ""},

	// level 2 + vhost as / and //
	{"sub1/sub2/", "", rabbitRootVhostPath, "/sub1/sub2"},
	{"sub1/sub2/", "myVhost", "/myVhost", "/sub1/sub2"},
	{"sub1/myVhost", "overridenVhost", "/overridenVhost", "/sub1"},
	{"sub1/sub2//", "", rabbitRootVhostPath, "/sub1/sub2"},
}

func Test_getVhostAndPathFromURL(t *testing.T) {
	for _, data := range getVhostAndPathFromURLTestData {
		resolvedVhostPath, resolvedPath := getVhostAndPathFromURL(data.rawPath, data.overrideVhost)
		assert.Equal(t, data.resolvedVhostPath, resolvedVhostPath, "expect resolvedVhostPath to = %s, but it is %s", data.resolvedVhostPath, resolvedVhostPath)
		assert.Equal(t, data.resolvedPath, resolvedPath, "expect resolvedPath to = %s, but it is %s", data.resolvedPath, resolvedPath)
	}
}

func TestGetQueueInfo(t *testing.T) {
	var allTestData []getQueueInfoTestData
	allTestData = append(allTestData, testQueueInfoTestDataSingleVhost...)
	for _, testData := range testQueueInfoTestData {
		for _, vhostAnsSubpathsData := range getVhostAndPathFromURLTestData {
			testData := testData
			if testData.extraMetadata == nil {
				testData.extraMetadata = make(map[string]string)
			}
			testData.urlPath = vhostAnsSubpathsData.rawPath
			testData.extraMetadata["vhostName"] = vhostAnsSubpathsData.overrideVhost
			allTestData = append(allTestData, testData)
		}
	}

	for _, testData := range allTestData {
		vhost, path := getVhostAndPathFromURL(testData.urlPath, testData.extraMetadata["vhostName"])
		var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedPath := fmt.Sprintf("%s/api/queues%s/evaluate_trials", path, vhost)
			if r.RequestURI != expectedPath {
				t.Error("Expect request path to =", expectedPath, "but it is", r.RequestURI)
			}

			w.WriteHeader(testData.responseStatus)
			_, err := w.Write([]byte(testData.response))
			if err != nil {
				t.Error("Expect request path to =", testData.response, "but it is", err)
			}
		}))

		resolvedEnv := map[string]string{host: fmt.Sprintf("%s%s%s", apiStub.URL, path, vhost), "plainHost": apiStub.URL}

		metadata := map[string]string{
			"queueName":   "evaluate_trials",
			"hostFromEnv": host,
			"protocol":    "http",
		}
		for k, v := range testData.extraMetadata {
			metadata[k] = v
		}

		s, err := NewRabbitMQScaler(
			&scalersconfig.ScalerConfig{
				ResolvedEnv:       resolvedEnv,
				TriggerMetadata:   metadata,
				AuthParams:        map[string]string{},
				GlobalHTTPTimeout: 1000 * time.Millisecond,
			},
		)

		if err != nil {
			t.Error("Expect success", err)
		}

		ctx := context.TODO()
		_, active, err := s.GetMetricsAndActivity(ctx, "Metric")

		if testData.responseStatus == http.StatusOK {
			if err != nil {
				t.Error("Expect success", err)
			}

			if active != testData.isActive {
				if testData.isActive {
					t.Error("Expect to be active")
				} else {
					t.Error("Expect to not be active")
				}
			}
		} else if !strings.Contains(err.Error(), testData.response) {
			t.Error("Expect error to be like '", testData.response, "' but it's '", err, "'")
		}
	}
}

var testRegexQueueInfoTestData = []getQueueInfoTestData{
	// sum queue length
	{response: `{"items":[{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum"}},
	{response: `{"items":[{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum"}},
	{response: `{"items":[{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum"}},
	{response: `{"items":[]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum"}},
	// sum queue length + ignoreUnacknowledged
	{response: `{"items":[{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum", "excludeUnacknowledged": "true"}},
	{response: `{"items":[{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum", "excludeUnacknowledged": "true"}},
	{response: `{"items":[{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum", "excludeUnacknowledged": "true"}},
	{response: `{"items":[{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum"}},
	{response: `{"items":[{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum", "excludeUnacknowledged": "true"}},
	{response: `{"items":[]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum"}},
	// max queue length
	{response: `{"items":[{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max"}},
	{response: `{"items":[{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max"}},
	{response: `{"items":[{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max"}},
	{response: `{"items":[]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max"}},
	// max queue length + excludeUnacknowledged
	{response: `{"items":[{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max", "excludeUnacknowledged": "true"}},
	{response: `{"items":[{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max", "excludeUnacknowledged": "true"}},
	{response: `{"items":[{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max", "excludeUnacknowledged": "true"}},
	{response: `{"items":[{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max"}},
	{response: `{"items":[{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max", "excludeUnacknowledged": "true"}},
	{response: `{"items":[]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max"}},
	// avg queue length
	{response: `{"items":[{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg"}},
	{response: `{"items":[{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg"}},
	{response: `{"items":[{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg"}},
	{response: `{"items":[]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg"}},
	// avg queue length + excludeUnacknowledged
	{response: `{"items":[{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg", "excludeUnacknowledged": "true"}},
	{response: `{"items":[{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg", "excludeUnacknowledged": "true"}},
	{response: `{"items":[{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg", "excludeUnacknowledged": "true"}},
	{response: `{"items":[{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg"}},
	{response: `{"items":[{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg", "excludeUnacknowledged": "true"}},
	{response: `{"items":[]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg"}},
	// sum message rate
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "sum"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "sum"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "sum"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "sum"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "sum"}},
	{response: `{"items":[]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "sum"}},
	// max message rate
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "max"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "max"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "max"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "max"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "max"}},
	{response: `{"items":[]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "max"}},
	// avg message rate
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "avg"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "avg"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "avg"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "avg"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "avg"}},
	{response: `{"items":[]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "avg"}},
	// sum deliverGet rate
	{response: `{"items":[{"messages": 4, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 2}}, "name": "evaluate_trials"}, {"messages": 2, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 2}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "DeliverGetRate", "value": "1000", "useRegex": "true", "operation": "sum"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"}, {"messages": 4, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 4}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "DeliverGetRate", "value": "1000", "useRegex": "true", "operation": "sum"}},
	{response: `{"items":[{"messages": 4, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 4}}, "name": "evaluate_trials"}, {"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "DeliverGetRate", "value": "1000", "useRegex": "true", "operation": "sum"}},
	{response: `{"items":[{"messages": 4, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 4}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "DeliverGetRate", "value": "1000", "useRegex": "true", "operation": "sum"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"}, {"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"mode": "DeliverGetRate", "value": "1000", "useRegex": "true", "operation": "sum"}},
	{response: `{"items":[]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"mode": "DeliverGetRate", "value": "1000", "useRegex": "true", "operation": "sum"}},
	// max deliverGet rate
	{response: `{"items":[{"messages": 8, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 2}}, "name": "evaluate_trials"}, {"messages": 4, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 4}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "DeliverGetRate", "value": "1000", "useRegex": "true", "operation": "max"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"}, {"messages": 4, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 4}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "DeliverGetRate", "value": "1000", "useRegex": "true", "operation": "max"}},
	{response: `{"items":[{"messages": 4, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 4}}, "name": "evaluate_trials"}, {"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "DeliverGetRate", "value": "1000", "useRegex": "true", "operation": "max"}},
	{response: `{"items":[{"messages": 4, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 4}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "DeliverGetRate", "value": "1000", "useRegex": "true", "operation": "max"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"}, {"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"mode": "DeliverGetRate", "value": "1000", "useRegex": "true", "operation": "max"}},
	{response: `{"items":[]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"mode": "DeliverGetRate", "value": "1000", "useRegex": "true", "operation": "max"}},
	// avg deliverGet rate
	{response: `{"items":[{"messages": 8, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 4}}, "name": "evaluate_trials"}, {"messages": 4, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 4}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "DeliverGetRate", "value": "1000", "useRegex": "true", "operation": "avg"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"}, {"messages": 4, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 4}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "DeliverGetRate", "value": "1000", "useRegex": "true", "operation": "avg"}},
	{response: `{"items":[{"messages": 4, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 4}}, "name": "evaluate_trials"}, {"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "DeliverGetRate", "value": "1000", "useRegex": "true", "operation": "avg"}},
	{response: `{"items":[{"messages": 4, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 4}, "deliver_get_details": {"rate": 4}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, isActive: true, extraMetadata: map[string]string{"mode": "DeliverGetRate", "value": "1000", "useRegex": "true", "operation": "avg"}},
	{response: `{"items":[{"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trials"}, {"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}, "deliver_get_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"mode": "DeliverGetRate", "value": "1000", "useRegex": "true", "operation": "avg"}},
	{response: `{"items":[]}`, responseStatus: http.StatusOK, extraMetadata: map[string]string{"mode": "DeliverGetRate", "value": "1000", "useRegex": "true", "operation": "avg"}},
}

func TestGetQueueInfoWithRegex(t *testing.T) {
	var allTestData []getQueueInfoTestData
	for _, testData := range testRegexQueueInfoTestData {
		for _, vhostAndSubpathsData := range getVhostAndPathFromURLTestData {
			testData := testData
			if testData.extraMetadata == nil {
				testData.extraMetadata = make(map[string]string)
			}
			testData.extraMetadata["vhostName"] = vhostAndSubpathsData.overrideVhost
			testData.urlPath = vhostAndSubpathsData.rawPath
			allTestData = append(allTestData, testData)
		}
	}

	for _, testData := range allTestData {
		vhost, path := getVhostAndPathFromURL(testData.urlPath, testData.extraMetadata["vhostName"])
		var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedPath := fmt.Sprintf("%s/api/queues%s?page=1&use_regex=true&pagination=false&name=%%5Eevaluate_trials%%24&page_size=100", path, vhost)
			if r.RequestURI != expectedPath {
				t.Error("Expect request path to =", expectedPath, "but it is", r.RequestURI)
			}

			w.WriteHeader(testData.responseStatus)
			_, err := w.Write([]byte(testData.response))
			if err != nil {
				t.Error("Expect request path to =", testData.response, "but it is", err)
			}
		}))

		resolvedEnv := map[string]string{host: fmt.Sprintf("%s%s%s", apiStub.URL, path, vhost), "plainHost": apiStub.URL}

		metadata := map[string]string{
			"queueName":   "^evaluate_trials$",
			"hostFromEnv": host,
			"protocol":    "http",
		}
		for k, v := range testData.extraMetadata {
			metadata[k] = v
		}

		s, err := NewRabbitMQScaler(
			&scalersconfig.ScalerConfig{
				ResolvedEnv:       resolvedEnv,
				TriggerMetadata:   metadata,
				AuthParams:        map[string]string{},
				GlobalHTTPTimeout: 1000 * time.Millisecond,
			},
		)

		if err != nil {
			t.Error("Expect success", err)
		}

		ctx := context.TODO()
		_, active, err := s.GetMetricsAndActivity(ctx, "Metric")

		if testData.responseStatus == http.StatusOK {
			if err != nil {
				t.Error("Expect success", err)
			}

			if active != testData.isActive {
				if testData.isActive {
					t.Error("Expect to be active")
				} else {
					t.Error("Expect to not be active")
				}
			}
		} else if !strings.Contains(err.Error(), testData.response) {
			t.Error("Expect error to be like '", testData.response, "' but it's '", err, "'")
		}
	}
}

type getRegexPageSizeTestData struct {
	queueInfo getQueueInfoTestData
	pageSize  int
}

var testRegexPageSizeTestData = []getRegexPageSizeTestData{
	{testRegexQueueInfoTestData[0], 100},
	{testRegexQueueInfoTestData[0], 200},
	{testRegexQueueInfoTestData[0], 500},
}

func TestGetPageSizeWithRegex(t *testing.T) {
	var allTestData []getRegexPageSizeTestData
	for _, testData := range testRegexPageSizeTestData {
		for _, vhostAndSubpathsData := range getVhostAndPathFromURLTestData {
			testData := testData
			if testData.queueInfo.extraMetadata == nil {
				testData.queueInfo.extraMetadata = make(map[string]string)
			}
			testData.queueInfo.extraMetadata["vhostName"] = vhostAndSubpathsData.overrideVhost
			testData.queueInfo.urlPath = vhostAndSubpathsData.rawPath
			allTestData = append(allTestData, testData)
		}
	}

	for _, testData := range allTestData {
		vhost, path := getVhostAndPathFromURL(testData.queueInfo.urlPath, testData.queueInfo.extraMetadata["vhostName"])
		var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedPath := fmt.Sprintf("%s/api/queues%s?page=1&use_regex=true&pagination=false&name=%%5Eevaluate_trials%%24&page_size=%d", path, vhost, testData.pageSize)
			if r.RequestURI != expectedPath {
				t.Error("Expect request path to =", expectedPath, "but it is", r.RequestURI)
			}

			w.WriteHeader(testData.queueInfo.responseStatus)
			_, err := w.Write([]byte(testData.queueInfo.response))
			if err != nil {
				t.Error("Expect request path to =", testData.queueInfo.response, "but it is", err)
			}
		}))

		resolvedEnv := map[string]string{host: fmt.Sprintf("%s%s%s", apiStub.URL, path, vhost), "plainHost": apiStub.URL}

		metadata := map[string]string{
			"queueName":   "^evaluate_trials$",
			"hostFromEnv": host,
			"protocol":    "http",
			"useRegex":    "true",
			"pageSize":    fmt.Sprint(testData.pageSize),
		}

		s, err := NewRabbitMQScaler(
			&scalersconfig.ScalerConfig{
				ResolvedEnv:       resolvedEnv,
				TriggerMetadata:   metadata,
				AuthParams:        map[string]string{},
				GlobalHTTPTimeout: 1000 * time.Millisecond,
			},
		)

		if err != nil {
			t.Error("Expect success", err)
		}

		ctx := context.TODO()
		_, active, err := s.GetMetricsAndActivity(ctx, "Metric")

		if err != nil {
			t.Error("Expect success", err)
		}

		if !active {
			t.Error("Expect to be active")
		}
	}
}

func TestRabbitMQGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range rabbitMQMetricIdentifiers {
		meta, err := parseRabbitMQMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleRabbitMqResolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: nil, TriggerIndex: testData.index})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockRabbitMQScaler := rabbitMQScaler{
			metadata:   meta,
			connection: nil,
			channel:    nil,
			httpClient: http.DefaultClient,
		}

		metricSpec := mockRabbitMQScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName, "wanted:", testData.name)
		}
	}
}

type rabbitMQErrorTestData struct {
	err     error
	message string
}

var anonymizeRabbitMQErrorTestData = []rabbitMQErrorTestData{
	{fmt.Errorf("https://user1:password1@domain.com"), "error inspecting RabbitMQ: https://user:password@domain.com"},
	{fmt.Errorf("https://fdasr345_-:password1@domain.com"), "error inspecting RabbitMQ: https://user:password@domain.com"},
	{fmt.Errorf("https://user1:fdasr345_-@domain.com"), "error inspecting RabbitMQ: https://user:password@domain.com"},
	{fmt.Errorf("https://fdakls_dsa:password1@domain.com"), "error inspecting RabbitMQ: https://user:password@domain.com"},
	{fmt.Errorf("fdasr345_-:password1@domain.com"), "error inspecting RabbitMQ: user:password@domain.com"},
	{fmt.Errorf("this user1:password1@domain.com fails"), "error inspecting RabbitMQ: this user:password@domain.com fails"},
	{fmt.Errorf("this https://user1:password1@domain.com fails also"), "error inspecting RabbitMQ: this https://user:password@domain.com fails also"},
	{fmt.Errorf("nothing to replace here"), "error inspecting RabbitMQ: nothing to replace here"},
	{fmt.Errorf("the queue https://user1:fdasr345_-@domain.com/api/virtual is unavailable"), "error inspecting RabbitMQ: the queue https://user:password@domain.com/api/virtual is unavailable"},
}

func TestRabbitMQAnonymizeRabbitMQError(t *testing.T) {
	metadata := map[string]string{
		"queueName":   "evaluate_trials",
		"hostFromEnv": host,
		"protocol":    "http",
	}
	meta, err := parseRabbitMQMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleRabbitMqResolvedEnv, TriggerMetadata: metadata, AuthParams: nil})

	if err != nil {
		t.Fatalf("Error parsing metadata (%s)", err)
	}

	s := &rabbitMQScaler{
		metadata:   meta,
		httpClient: nil,
	}
	for _, testData := range anonymizeRabbitMQErrorTestData {
		err := s.anonymizeRabbitMQError(testData.err)
		assert.Equal(t, fmt.Sprint(err), testData.message)
	}
}

type getQueueInfoNavigationTestData struct {
	response string
	isError  bool
}

var testRegexQueueInfoNavigationTestData = []getQueueInfoNavigationTestData{
	// sum queue length
	{`{"items":[], "filtered_count": 250, "page": 1, "page_count": 3}`, true},
	{`{"items":[], "filtered_count": 250, "page": 1, "page_count": 1}`, false},
}

func TestRegexQueueMissingError(t *testing.T) {
	for _, testData := range testRegexQueueInfoNavigationTestData {
		var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedPath := "/api/queues/%2F?page=1&use_regex=true&pagination=false&name=evaluate_trials&page_size=100"
			if r.RequestURI != expectedPath {
				t.Error("Expect request path to =", expectedPath, "but it is", r.RequestURI)
			}

			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(testData.response))
			if err != nil {
				t.Error("Expect request path to =", testData.response, "but it is", err)
			}
		}))

		resolvedEnv := map[string]string{host: apiStub.URL, "plainHost": apiStub.URL}

		metadata := map[string]string{
			"queueName":   "evaluate_trials",
			"hostFromEnv": host,
			"protocol":    "http",
			"useRegex":    "true",
		}

		s, err := NewRabbitMQScaler(
			&scalersconfig.ScalerConfig{
				ResolvedEnv:       resolvedEnv,
				TriggerMetadata:   metadata,
				AuthParams:        map[string]string{},
				GlobalHTTPTimeout: 1000 * time.Millisecond,
			},
		)

		if err != nil {
			t.Error("Expect success", err)
		}

		ctx := context.TODO()
		_, _, err = s.GetMetricsAndActivity(ctx, "Metric")
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestConnectionName(t *testing.T) {
	c := scalersconfig.ScalerConfig{
		ScalableObjectNamespace: "test-namespace",
		ScalableObjectName:      "test-name",
	}

	connectionName := connectionName(&c)

	if connectionName != "keda-test-namespace-test-name" {
		t.Error("Expected connection name to be keda-test-namespace-test-name but got", connectionName)
	}
}
