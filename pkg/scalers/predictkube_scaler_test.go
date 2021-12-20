package scalers

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewPredictKubeScaler_doPredictRequest(t *testing.T) {
	//svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//	if r.RequestURI == "/api/v1/status/runtimeinfo" {
	//		w.WriteHeader(http.StatusOK)
	//		w.Header().Set("Content-Type", "application/json")
	//
	//		if err := json.NewEncoder(w).Encode(map[string]interface{}{
	//			"startTime":           "2020-05-18T15:52:53.4503113Z",
	//			"CWD":                 "/prometheus",
	//			"reloadConfigSuccess": true,
	//			"lastConfigTime":      "2020-05-18T15:52:56Z",
	//			"chunkCount":          72692,
	//			"timeSeriesCount":     18476,
	//			"corruptionCount":     0,
	//			"goroutineCount":      217,
	//			"GOMAXPROCS":          2,
	//			"GOGC":                "100",
	//			"GODEBUG":             "allocfreetrace",
	//			"storageRetention":    "1d",
	//		}); err != nil {
	//			http.Error(w, "encode response body error", fasthttp.StatusInternalServerError)
	//		}
	//
	//		return
	//	}
	//
	//	w.WriteHeader(http.StatusOK)
	//	w.Header().Set("Content-Type", "application/json")
	//
	//	type resp struct {
	//		Status    string          `json:"status"`
	//		Data      json.RawMessage `json:"data"`
	//		ErrorType v1.ErrorType    `json:"errorType"`
	//		Error     string          `json:"error"`
	//		Warnings  []string        `json:"warnings,omitempty"`
	//	}
	//
	//	raw, err := json.Marshal([]*model.SampleStream{
	//		{
	//			Values: []model.SamplePair{
	//				{
	//					Timestamp: model.Time(time.Date(2021, time.December, 15, 16, 01, 30, 0, time.UTC).Unix()),
	//					Value:     16043,
	//				},
	//				{
	//					Timestamp: model.Time(time.Date(2021, time.December, 15, 16, 02, 30, 0, time.UTC).Unix()),
	//					Value:     15000,
	//				},
	//			},
	//		},
	//	})
	//
	//	if err != nil {
	//		http.Error(w, "encode response body error", fasthttp.StatusInternalServerError)
	//	}
	//
	//	bts, err := json.Marshal(&struct {
	//		Type   model.ValueType `json:"resultType"`
	//		Result json.RawMessage `json:"result"`
	//	}{
	//		Type:   model.ValMatrix,
	//		Result: raw,
	//	})
	//
	//	if err != nil {
	//		http.Error(w, "encode response body error", fasthttp.StatusInternalServerError)
	//	}
	//
	//	if err := json.NewEncoder(w).Encode(&resp{
	//		Data:   bts,
	//		Status: "200",
	//	}); err != nil {
	//		http.Error(w, "encode response body error", fasthttp.StatusInternalServerError)
	//	}
	//
	//	//type queryResult struct {
	//	//	Type   model.ValueType
	//	//	Result []*model.SampleStream
	//	//}
	//	//
	//	//fmt.Println(r.Body)
	//}))
	//defer svr.Close()

	predictScaler, err := NewPredictKubeScaler(context.Background(), &ScalerConfig{
		TriggerMetadata: map[string]string{
			"query":             `sum(irate(nginx_http_requests_total{pod=~".*bsc.*", cluster="", namespace=~"bsc"}[2m]))`,
			"predictHorizon":    "2h",
			"historyTimeWindow": "7d",
			"prometheusAddress":/*svr.URL, //*/ "https://ptmp.eu-pancakeswap.dysnix.org/",
			"threshold":  "2000",
			"metricName": "scaledobject",
			"queryStep":  "2m",
		},
		AuthParams: map[string]string{
			"apiKey": "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJ0ZXN0LWtlZGEiLCJleHAiOjE3MDY1MjkwODQsImlzcyI6ImNkNWYxM2NjLTVmNWEtMTFlYy05MDhmLWFjZGU0ODAwMTEyMiJ9.9VHwDc2Pa4_M6SYMAh4uAWDSjXbfyID3lzr-QkJp9xpn_AUhHF2I93Wt4P-3tlXo9hNnhIBmVcp1O7wE5FKbzA",
		},
	})

	assert.NoError(t, err)

	defer predictScaler.Close(context.Background())

	response, err := predictScaler.doPredictRequest(context.Background())
	assert.NoError(t, err)

	t.Log(response)
}
