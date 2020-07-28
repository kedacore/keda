package scalers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	host         = "myHostSecret"
	apiHost      = "myApiHostSecret"
	localHost    = "myLocalHostSecret"
	hostURL      = "amqp://user:sercet@somehost.com:5236/vhost"
	apiHostURL   = "https://user:secret@somehost.com/vhost"
	localHostURL = "amqp://user:PASSWORD@127.0.0.1:5672"
)

type parseRabbitMQMetadataTestData struct {
	metadata     map[string]string
	isError      bool
	authParams   map[string]string
	expectedHost string
}

var sampleRabbitMqResolvedEnv = map[string]string{
	host:      hostURL,
	apiHost:   apiHostURL,
	localHost: localHostURL,
}

var testRabbitMQMetadata = []parseRabbitMQMetadataTestData{
	// nothing passed
	{map[string]string{}, true, map[string]string{}, ""},
	// properly formed metadata
	{map[string]string{"queueLength": "10", "queueName": "sample", "host": host}, false, map[string]string{}, hostURL},
	// malformed queueLength
	{map[string]string{"queueLength": "AA", "queueName": "sample", "host": host}, true, map[string]string{}, ""},
	// properly formed metadata
	{map[string]string{"queueLength": "10", "queueName": "sample", "host": host, "localhost": localHost}, false, map[string]string{}, localHostURL},
	// malformed queueLength
	{map[string]string{"queueLength": "AA", "queueName": "sample", "host": host, "localhost": localHost}, true, map[string]string{}, ""},
	// missing host
	{map[string]string{"queueLength": "AA", "queueName": "sample"}, true, map[string]string{}, ""},
	// missing queueName
	{map[string]string{"queueLength": "10", "host": host}, true, map[string]string{}, ""},
	// host defined in authParams
	{map[string]string{"queueLength": "10"}, true, map[string]string{"host": host}, ""},
	// properly formed metadata with includeUnacked
	{map[string]string{"queueLength": "10", "queueName": "sample", "apiHost": apiHost, "includeUnacked": "true"}, false, map[string]string{}, ""},
}

func TestRabbitMQParseMetadata(t *testing.T) {
	for _, testData := range testRabbitMQMetadata {
		metadata, err := parseRabbitMQMetadata(sampleRabbitMqResolvedEnv, testData.metadata, testData.authParams)
		t.Logf("%v", metadata)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		if metadata != nil && metadata.host != testData.expectedHost {
			t.Error("Unmatch Expected hostname: metadata.host: ", metadata.host, " expectedHost:", testData.expectedHost)
		}

	}
}

var testDefaultQueueLength = []parseRabbitMQMetadataTestData{
	// use default queueLength
	{map[string]string{"queueName": "sample", "host": host}, false, map[string]string{}, hostURL},
	// use default queueLength with includeUnacked
	{map[string]string{"queueName": "sample", "apiHost": apiHost, "includeUnacked": "true"}, false, map[string]string{}, ""},
}

func TestParseDefaultQueueLength(t *testing.T) {
	for _, testData := range testDefaultQueueLength {
		metadata, err := parseRabbitMQMetadata(sampleRabbitMqResolvedEnv, testData.metadata, testData.authParams)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		} else if testData.isError && err == nil {
			t.Error("Expected error but got success")
		} else if metadata.queueLength != defaultRabbitMQQueueLength {
			t.Error("Expected default queueLength =", defaultRabbitMQQueueLength, "but got", metadata.queueLength)
		}
	}
}

type getQueueInfoTestData struct {
	response       string
	responseStatus int
	isActive       bool
}

var testQueueInfoTestData = []getQueueInfoTestData{
	{`{"messages": 4, "messages_unacknowledged": 1, "name": "evaluate_trials"}`, http.StatusOK, true},
	{`{"messages": 1, "messages_unacknowledged": 1, "name": "evaluate_trials"}`, http.StatusOK, true},
	{`{"messages": 1, "messages_unacknowledged": 0, "name": "evaluate_trials"}`, http.StatusOK, true},
	{`{"messages": 0, "messages_unacknowledged": 0, "name": "evaluate_trials"}`, http.StatusOK, false},
	{`Password is incorrect`, http.StatusUnauthorized, false},
}

var vhost_pathes = []string{"/myhost", "", "/", "//", "/%2F"}

func TestGetQueueInfo(t *testing.T) {
	for _, testData := range testQueueInfoTestData {
		for _, vhost_path := range vhost_pathes {
			expeced_vhost := "myhost"

			if vhost_path != "/myhost" {
				expeced_vhost = "%2F"
			}

			var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expeced_path := "/api/queues/" + expeced_vhost + "/evaluate_trials"
				if r.RequestURI != expeced_path {
					t.Error("Expect request path to =", expeced_path, "but it is", r.RequestURI)
				}

				w.WriteHeader(testData.responseStatus)
				w.Write([]byte(testData.response))
			}))

			resolvedEnv := map[string]string{apiHost: fmt.Sprintf("%s%s", apiStub.URL, vhost_path)}

			metadata := map[string]string{
				"queueLength":    "10",
				"queueName":      "evaluate_trials",
				"apiHost":        apiHost,
				"includeUnacked": "true",
			}

			s, err := NewRabbitMQScaler(resolvedEnv, metadata, map[string]string{})

			if err != nil {
				t.Error("Expect success", err)
			}

			ctx := context.TODO()
			active, err := s.IsActive(ctx)

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
			} else {
				if !strings.Contains(err.Error(), testData.response) {
					t.Error("Expect error to be like '", testData.response, "' but it's '", err, "'")
				}
			}
		}
	}
}
