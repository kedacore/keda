package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateTriggers(t *testing.T) {
	tests := []struct {
		name           string
		triggers       []ScaleTriggers
		expectedErrMsg string
	}{
		{
			name: "valid triggers",
			triggers: []ScaleTriggers{
				{
					Name: "trigger1",
					Type: "cpu",
				},
				{
					Name: "trigger2",
					Type: "prometheus",
				},
			},
			expectedErrMsg: "",
		},
		{
			name: "duplicate trigger names",
			triggers: []ScaleTriggers{
				{
					Name: "trigger1",
					Type: "cpu",
				},
				{
					Name: "trigger1",
					Type: "prometheus",
				},
			},
			expectedErrMsg: "triggerName \"trigger1\" is defined multiple times in the ScaledObject, but it must be unique",
		},
		{
			name: "unsupported useCachedMetrics property for cpu scaler",
			triggers: []ScaleTriggers{
				{
					Name:             "trigger1",
					Type:             "cpu",
					UseCachedMetrics: true,
				},
			},
			expectedErrMsg: "property \"useCachedMetrics\" is not supported for \"cpu\" scaler",
		},
		{
			name: "unsupported useCachedMetrics property for memory scaler",
			triggers: []ScaleTriggers{
				{
					Name:             "trigger2",
					Type:             "memory",
					UseCachedMetrics: true,
				},
			},
			expectedErrMsg: "property \"useCachedMetrics\" is not supported for \"memory\" scaler",
		},
		{
			name: "unsupported useCachedMetrics property for cron scaler",
			triggers: []ScaleTriggers{
				{
					Name:             "trigger3",
					Type:             "cron",
					UseCachedMetrics: true,
				},
			},
			expectedErrMsg: "property \"useCachedMetrics\" is not supported for \"cron\" scaler",
		},
		{
			name: "supported useCachedMetrics property for kafka scaler",
			triggers: []ScaleTriggers{
				{
					Name:             "trigger4",
					Type:             "kafka",
					UseCachedMetrics: true,
				},
			},
			expectedErrMsg: "",
		},
	}

	for _, test := range tests {
		tt := test
		t.Run(test.name, func(t *testing.T) {
			err := ValidateTriggers(tt.triggers)
			if test.expectedErrMsg == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.expectedErrMsg)
			}
		})
	}
}
