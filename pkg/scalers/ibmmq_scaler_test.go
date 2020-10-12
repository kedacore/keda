/**
* © Copyright IBM Corporation 2020
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
**/

package scalers

import (
	"fmt"
	"testing"
)

const (
	testValidMQQueueURL   = "https://qmtest.qm2.eu-gb.mq.appdomain.cloud/ibmmq/rest/v2/admin/action/qmgr/QM1/mqsc"
	testInvalidMQQueueURL = "testInvalidURL.com"
)

type parseIBMMQMetadataTestData struct {
	metadata   map[string]string
	isError    bool
	authParams map[string]string
}

type IBMMQMetricIdentifier struct {
	metadataTestData *parseIBMMQMetadataTestData
	name             string //
}

var testIBMMQMetadata = []parseIBMMQMetadataTestData{
	// nothing passed
	{map[string]string{}, true, map[string]string{}},
	// properly formed metadata
	{map[string]string{"host": testValidMQQueueURL, "queueName": "testQueue", "queueLength": "10"}, false, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// Invalid queueLength using a string
	{map[string]string{"host": testValidMQQueueURL, "queueName": "testQueue", "queueLength": "AA"}, true, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// no host provided
	{map[string]string{"queueName": "testQueue", "queueLength": "10"}, true, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// missing queueName
	{map[string]string{"host": testValidMQQueueURL, "queueLength": "10"}, true, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// Invalid URL
	{map[string]string{"host": testInvalidMQQueueURL, "queueName": "testQueue", "queueLength": "10"}, true, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// properly formed authParams
	{map[string]string{"host": testValidMQQueueURL, "queueName": "testQueue", "queueLength": "10"}, false, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// no username provided
	{map[string]string{"host": testValidMQQueueURL, "queueName": "testQueue", "queueLength": "10"}, true, map[string]string{"password": "Pass123"}},
	// no password provided
	{map[string]string{"host": testValidMQQueueURL, "queueName": "testQueue", "queueLength": "10"}, true, map[string]string{"username": "testUsername"}},
}

var IBMMQMetricIdentifiers = []IBMMQMetricIdentifier{
	{&testIBMMQMetadata[1], "IBMMQ-testQueue"},
}

//testing if metadata is parsed correctly
func TestIBMMQParseMetadata(t *testing.T) {
	for _, testData := range testIBMMQMetadata {
		_, err := parseIBMMQMetadata(testData.metadata, testData.authParams)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
			fmt.Println(testData)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
			fmt.Println(testData)
		}
	}
}

var testDefaultQueueDepth = []parseIBMMQMetadataTestData{
	{map[string]string{"host": testValidMQQueueURL, "queueName": "testQueue"}, false, map[string]string{"username": "testUsername", "password": "Pass123"}},
}

func TestParseDefaultQueueDepth(t *testing.T) {
	for _, testData := range testDefaultQueueDepth {
		metadata, err := parseIBMMQMetadata(testData.metadata, testData.authParams)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		} else if testData.isError && err == nil {
			t.Error("Expected error but got success")
		} else if metadata.targetQueueLength != defaultTargetQueueDepth {
			t.Error("Expected default queueLength =", defaultTargetQueueDepth, "but got", metadata.targetQueueLength)
		}
	}
}

//create a scaler and check if method is available
func TestIBMMQGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range IBMMQMetricIdentifiers {
		meta, err := parseIBMMQMetadata(testData.metadataTestData.metadata, testData.metadataTestData.authParams)
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockIBMMQScaler := IBMMQScaler{meta}
		metricSpec := mockIBMMQScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name

		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
