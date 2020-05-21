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
	host    = "myHostSecret"
	apiHost = "myApiHostSecret"
)

type parseRabbitMQMetadataTestData struct {
	metadata   map[string]string
	isError    bool
	authParams map[string]string
}

var sampleRabbitMqResolvedEnv = map[string]string{
	host:    "amqp://user:sercet@somehost.com:5236/vhost",
	apiHost: "https://user:secret@somehost.com/vhost",
}

var testRabbitMQMetadata = []parseRabbitMQMetadataTestData{
	// nothing passed
	{map[string]string{}, true, map[string]string{}},
	// properly formed metadata
	{map[string]string{"queueLength": "10", "queueName": "sample", "host": host}, false, map[string]string{}},
	// malformed queueLength
	{map[string]string{"queueLength": "AA", "queueName": "sample", "host": host}, true, map[string]string{}},
	// missing host
	{map[string]string{"queueLength": "AA", "queueName": "sample"}, true, map[string]string{}},
	// missing queueName
	{map[string]string{"queueLength": "10", "host": host}, true, map[string]string{}},
	// host defined in authParams
	{map[string]string{"queueLength": "10"}, true, map[string]string{"host": host}},
	// properly formed metadata with includeUnacked
	{map[string]string{"queueLength": "10", "queueName": "sample", "apiHost": apiHost, "includeUnacked": "true"}, false, map[string]string{}},
}

func TestRabbitMQParseMetadata(t *testing.T) {
	for _, testData := range testRabbitMQMetadata {
		_, err := parseRabbitMQMetadata(sampleRabbitMqResolvedEnv, testData.metadata, testData.authParams)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
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
	{`{"messages": 0, "messages_unacknowledged": 1, "name": "evaluate_trials"}`, http.StatusOK, true},
	{`{"messages": 1, "messages_unacknowledged": 0, "name": "evaluate_trials"}`, http.StatusOK, true},
	{`{"messages": 0, "messages_unacknowledged": 0, "name": "evaluate_trials"}`, http.StatusOK, false},
	{`Password is incorrect`, http.StatusUnauthorized, false},
}

var vhosts = []string{"myhost", "", "/", "%2F"}

func TestGetQueueInfo(t *testing.T) {
	for _, testData := range testQueueInfoTestData {
		for _, vhost := range vhosts {
			expeced_vhost := vhost

			if vhost != "myhost" {
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

			resolvedEnv := map[string]string{apiHost: fmt.Sprintf("%s/%s", apiStub.URL, vhost)}

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
