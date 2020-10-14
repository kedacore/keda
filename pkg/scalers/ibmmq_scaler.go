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

// Default variables and settings
const (
	ibmMqQueueDepthMetricName = "currentQueueDepth"
	defaultTargetQueueDepth   = 20
	defaultTlsDisabled        = false
)

// Assigns IBMMQMetadata struct data pointer to metadata variable
type IBMMQScaler struct {
	metadata *IBMMQMetadata
}

// Metadata used by KEDA to query IBM MQ queue depth and scale
type IBMMQMetadata struct {
	host             string
	queueName        string
	username         string
	password         string
	targetQueueDepth int
	tlsDisabled      bool
}

// Full structured response from MQ admin REST query
type CommandResponse struct {
	CommandResponse []Response `json:"commandResponse"`
}

// The body of the response returned from the MQ admin query
type Response struct {
	Parameters Parameters `json:"parameters"`
}

// Current depth of the IBM MQ Queue
type Parameters struct {
	Curdepth int `json:"curdepth"`
}

// NewIBMMQScaler creates a new IBM MQ scaler
func NewIBMMQScaler(config *ScalerConfig) (Scaler, error) {
	meta, err := parseIBMMQMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing IBM MQ metadata: %s", err)
	}

	return &IBMMQScaler{metadata: meta}, nil
}

func (s *IBMMQScaler) Close() error {
	return nil
}

// parseIBMMQMetadata checks the existence of and validates the MQ connection data provided
func parseIBMMQMetadata(config *ScalerConfig) (*IBMMQMetadata, error) {
	meta := IBMMQMetadata{}

	if val, ok := config.TriggerMetadata["host"]; ok {
		_, err := url.ParseRequestURI(val)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %s", err)
		}
		meta.host = val
	} else {
		return nil, fmt.Errorf("no host URI given")
	}

	if val, ok := config.TriggerMetadata["queueName"]; ok {
		meta.queueName = val
	} else {
		return nil, fmt.Errorf("no queue name given")
	}

	if val, ok := config.TriggerMetadata["queueDepth"]; ok && val != "" {
		queueDepth, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid targetQueueDepth - must be an integer")
		} else {
			meta.targetQueueDepth = queueDepth
		}
	} else {
		fmt.Println("No target depth defined - setting default")
		meta.targetQueueDepth = defaultTargetQueueDepth
	}

	if val, ok := config.TriggerMetadata["tls"]; ok {
		tlsDisabled, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("invalid tls setting: %s", err)
		}
		meta.tlsDisabled = tlsDisabled
	} else {
		fmt.Println("No tls setting defined - setting default")
		meta.tlsDisabled = defaultTlsDisabled
	}

	if val, ok := config.AuthParams["username"]; ok {
		meta.username = val
	} else {
		return nil, fmt.Errorf("no username given")
	}

	if val, ok := config.AuthParams["password"]; ok {
		meta.password = val
	} else {
		return nil, fmt.Errorf("no password given")
	}

	return &meta, nil
}

// IsActive returns true if there are messages to be processed/if we need to scale from zero
func (s *IBMMQScaler) IsActive(ctx context.Context) (bool, error) {
	queueDepth, err := s.getQueueDepthViaHttp()
	if err != nil {
		return false, fmt.Errorf("error inspecting IBM MQ queue depth: %s", err)
	}
	return queueDepth > 0, nil
}

// getQueueDepthViaHttp returns the depth of the MQ Queue from the Admin endpoint
func (s *IBMMQScaler) getQueueDepthViaHttp() (int, error) {

	queue := s.metadata.queueName
	url := s.metadata.host

	var requestJson = []byte(`{"type": "runCommandJSON", "command": "display", "qualifier": "qlocal", "name": "` + queue + `", "responseParameters" : ["CURDEPTH"]}`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestJson))
	req.Header.Set("ibm-mq-rest-csrf-token", "value")
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(s.metadata.username, s.metadata.password)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: s.metadata.tlsDisabled},
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
	targetQueueLengthQty := resource.NewQuantity(int64(s.metadata.targetQueueDepth), resource.DecimalSI)
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
		MetricName: ibmMqQueueDepthMetricName,
		Value:      *resource.NewQuantity(int64(queueDepth), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
