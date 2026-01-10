/*
Copyright 2025 The KEDA Authors

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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

func TestMockScaler_GetMetricsAndActivity_Success(t *testing.T) {
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{
			"mockMetricValue": "15",
			"mockIsActive":    "true",
		},
		TriggerIndex: 0,
	}

	scaler, err := NewMockScaler(config)
	assert.NoError(t, err)

	metrics, isActive, err := scaler.GetMetricsAndActivity(context.Background(), "s0-mock")

	assert.NoError(t, err)
	assert.True(t, isActive)
	assert.Len(t, metrics, 1)
	assert.Equal(t, int64(15000), metrics[0].Value.MilliValue())
}

func TestMockScaler_GetMetricsAndActivity_Failure(t *testing.T) {
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{
			"mockShouldFail":  "true",
			"mockFailureType": "timeout",
		},
		TriggerIndex: 0,
	}

	scaler, err := NewMockScaler(config)
	assert.NoError(t, err)

	_, _, err = scaler.GetMetricsAndActivity(context.Background(), "s0-mock")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestMockScaler_GetMetricSpecForScaling(t *testing.T) {
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{
			"mockTargetValue": "20",
		},
		TriggerIndex: 1,
	}

	scaler, err := NewMockScaler(config)
	assert.NoError(t, err)

	specs := scaler.GetMetricSpecForScaling(context.Background())

	assert.Len(t, specs, 1)
	assert.Equal(t, "s1-mock", specs[0].External.Metric.Name)
	assert.Equal(t, int64(20000), specs[0].External.Target.AverageValue.MilliValue())
}
