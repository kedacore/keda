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

package eventemitter

import (
	"context"
	"testing"
	"time"

	"github.com/kedacore/keda/v2/pkg/eventemitter/eventdata"
	"github.com/stretchr/testify/assert"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = logf.Log.WithName("cloudeventhandler_test")

type parseCloudeventHTTPHandlerTestData struct {
	clusterName string
	uri         string
}

var testCorrectCloudeventHTTPHandlerTestData = parseCloudeventHTTPHandlerTestData{
	clusterName: "test",
	uri:         "http://fo.mo",
}

var testErrCloudeventHTTPHandlerTestData = []parseCloudeventHTTPHandlerTestData{
	{
		clusterName: "test",
		uri:         "",
	},
	{
		clusterName: "test",
		uri:         "aaa",
	},
}

var testErrEventData = eventdata.EventData{
	Namespace:  "aaa",
	ObjectName: "bbb",
	EventType:  "ccc",
	Reason:     "ddd",
	Message:    "eee",
	Time:       time.Now().UTC(),
}

func TestCorrectCloudeventHTTPHandler(t *testing.T) {
	_, err := NewCloudEventHTTPHandler(context.TODO(), testCorrectCloudeventHTTPHandlerTestData.clusterName, testCorrectCloudeventHTTPHandlerTestData.uri, logger)

	assert.NoError(t, err)
}

func TestParseActiveMQMetadata(t *testing.T) {
	for _, testData := range testErrCloudeventHTTPHandlerTestData {
		_, err := NewCloudEventHTTPHandler(context.TODO(), testData.clusterName, testData.uri, logger)

		assert.Error(t, err)
	}
}

func TestCloudeventHTTPHandlerSendData(t *testing.T) {
	h, err := NewCloudEventHTTPHandler(context.TODO(), testCorrectCloudeventHTTPHandlerTestData.clusterName, testCorrectCloudeventHTTPHandlerTestData.uri, logger)

	assert.NoError(t, err)

	h.EmitEvent(testErrEventData, func(eventData eventdata.EventData, err error) {
		assert.Error(t, err)
	})
}
