package scalers

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTmp(t *testing.T) {
	tmp := "12-15-2021 16:01:30"
	t.Log(len(tmp), tmp[0:2], tmp[3:5], tmp[6:10], tmp[11:13], tmp[14:16], tmp[17:19])
}

func TestNewPredictKubeScaler_doPredictRequest(t *testing.T) {
	predictScaler, err := NewPredictKubeScaler(context.Background(), &ScalerConfig{
		TriggerMetadata: map[string]string{
			"query":             `sum(irate(nginx_http_requests_total{pod=~".*bsc.*", cluster="", namespace=~"bsc"}[2m]))`,
			"predictHorizon":    "2h",
			"historyTimeWindow": "7d",
			"prometheusAddress": "https://ptmp.eu-pancakeswap.dysnix.org/",
			"threshold":         "2000",
			"metricName":        "scaledobject",
			"queryStep":         "2m",
		},
		AuthParams: map[string]string{
			"apiKey": "some_key",
		},
	})

	assert.NoError(t, err)

	response, err := predictScaler.doPredictRequest(context.Background())
	assert.NoError(t, err)

	t.Log(response)
}
