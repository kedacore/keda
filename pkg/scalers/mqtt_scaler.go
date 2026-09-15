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
	"context"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	mqttMetricType = "External"
)

// mqttScalerMetadata holds the parsed trigger configuration for a
// ScaledObject using `type: mqtt`.
type mqttScalerMetadata struct {
	triggerIndex int

	BrokerAddress string `keda:"name=brokerAddress, order=triggerMetadata"`
	Topic         string `keda:"name=topic, order=triggerMetadata"`
	WindowSeconds int    `keda:"name=windowSeconds, order=triggerMetadata, optional, default=30"`
	QueryValue    int64  `keda:"name=queryValue, order=triggerMetadata"`
	QoS           int    `keda:"name=qos, order=triggerMetadata, optional, default=1"`

	ConnectRetryIntervalSeconds int  `keda:"name=connectRetryIntervalSeconds, order=triggerMetadata, optional, default=2"`
	MaxReconnectIntervalSeconds int  `keda:"name=maxReconnectIntervalSeconds, order=triggerMetadata, optional, default=60"`
	EnableTLS                   bool `keda:"name=enableTLS, order=triggerMetadata, optional"`
	UnsafeSsl                   bool `keda:"name=unsafeSsl, order=triggerMetadata, optional"`

	Username    string `keda:"name=username, order=authParams, optional"`
	Password    string `keda:"name=password, order=authParams, optional"`
	Ca          string `keda:"name=ca, order=authParams, optional"`
	Cert        string `keda:"name=cert, order=authParams, optional"`
	Key         string `keda:"name=key, order=authParams, optional"`
	KeyPassword string `keda:"name=keyPassword, order=authParams, optional"`
}

func parseMqttScalerMetadata(config *scalersconfig.ScalerConfig) (mqttScalerMetadata, error) {
	meta := mqttScalerMetadata{triggerIndex: config.TriggerIndex}
	if err := config.TypedConfig(&meta); err != nil {
		return meta, fmt.Errorf("error parsing mqtt scaler metadata: %w", err)
	}
	return meta, nil
}
