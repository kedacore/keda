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
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/pkg/util"
)

const (
	IBMMQQueueDepthMetricName = "currQueueDepth"
	defaultTargetQueueDepth   = 20
	IBMMQMetricType           = "External"
)

type IBMMQScaler struct {
	metadata *IBMMQMetadata
}

type IBMMQMetadata struct {
	host              string // MQ Host URI
	queueName         string // Queue Manager Name
	username          string // Username
	password          string // Password
	targetQueueLength int
}

type CommandResponse struct {
	CommandResponse []Response `json:"commandResponse"`
}

type Response struct {
	Parameters Parameters `json:"parameters"`
}

type Parameters struct {
	Curdepth int `json:"curdepth"`
}

// NewIBMMQScaler creates a new IBM MQ scaler
func NewIBMMQScaler(metadata, authParams map[string]string) (Scaler, error) {
	meta, err := parseIBMMQMetadata(metadata, authParams)
	if err != nil {
		return nil, fmt.Errorf("error parsing IBM MQ metadata: %s", err)
	}

	return &IBMMQScaler{metadata: meta}, nil
}

func (s *IBMMQScaler) Close() error {

	return nil
}

func parseIBMMQMetadata(metadata, authParams map[string]string) (*IBMMQMetadata, error) {
	meta := IBMMQMetadata{}

	if val, ok := metadata["host"]; ok {
		_, err := url.ParseRequestURI(val)
		if err != nil {
			return nil, fmt.Errorf("Invalid URL: %s", err)
		}
		meta.host = val
	} else {
		return nil, fmt.Errorf("No host URI given")
	}

	if val, ok := metadata["queueName"]; ok {
		meta.queueName = val
	} else {
		return nil, fmt.Errorf("No queue name given")
	}

	if val, ok := metadata["queueLength"]; ok && val != "" {
		queueLength, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("Invalid targetQueueLength - must be an integer")
		} else {
			meta.targetQueueLength = queueLength
		}

	} else {
		//If no target queue length is passed, use the default value
		fmt.Println("No target length defined - setting default")
		meta.targetQueueLength = defaultTargetQueueDepth
	}

	if val, ok := authParams["username"]; ok {
		meta.username = val
	} else {
		return nil, fmt.Errorf("No username given")
	}

	if val, ok := authParams["password"]; ok {
		meta.password = val
	} else {
		return nil, fmt.Errorf("No password given")
	}

	return &meta, nil
}

// IsActive returns true if there are pending messages to be processed
func (s *IBMMQScaler) IsActive(ctx context.Context) (bool, error) {
	queueDepth, err := s.getQueueDepthViaHttp()
	if err != nil {
		return false, fmt.Errorf("Error inspecting IBM MQ queue depth: %s", err)
	}

	return queueDepth > 0, nil
}

func (s *IBMMQScaler) getQueueDepthViaHttp() (int, error) {

	queue := s.metadata.queueName

	url := s.metadata.host

	var requestJson = []byte(`{"type": "runCommandJSON", "command": "display", "qualifier": "qlocal", "name": "` + queue + `", "responseParameters" : ["CURDEPTH"]}`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestJson))
	req.Header.Set("ibm-mq-rest-csrf-token", "value")
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(s.metadata.username, s.metadata.password)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var response CommandResponse

	json.Unmarshal(body, &response)

	return response.CommandResponse[0].Parameters.Curdepth, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *IBMMQScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetQueueLengthQty := resource.NewQuantity(int64(s.metadata.targetQueueLength), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s", "IBMMQ", s.metadata.queueName)),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetQueueLengthQty,
		},
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *IBMMQScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	queueDepth, err := s.getQueueDepthViaHttp()
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, fmt.Errorf("error inspecting IBM MQ queue depth: %s", err)
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: IBMMQQueueDepthMetricName,
		Value:      *resource.NewQuantity(int64(queueDepth), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
