package sumologic

import (
	"net/http"
)

type Config struct {
	Host      string
	AccessID  string
	AccessKey string
	UnsafeSsl bool
}

type Client struct {
	config *Config
	client *http.Client
}

type SearchJobResponse struct {
	ID string `json:"id"`
}

type SearchJobStatus struct {
	State       string `json:"state"`
	RecordCount int    `json:"recordCount"`
}

type RecordsResponse struct {
	Records []struct {
		Map map[string]string `json:"map"`
	} `json:"records"`
}

type MetricsQuery struct {
	RowID        string `json:"rowId"`
	Query        string `json:"query"`
	Quantization int64  `json:"quantization"`
}

type TimeRangeBoundary struct {
	Type        string `json:"type"`
	Iso8601Time string `json:"iso8601Time"`
}

type TimeRange struct {
	Type string            `json:"type"`
	From TimeRangeBoundary `json:"from"`
	To   TimeRangeBoundary `json:"to,omitempty"`
}

type MetricsQueryRequest struct {
	Queries   []MetricsQuery `json:"queries"`
	TimeRange TimeRange      `json:"timeRange"`
}

type MetricsQueryResponse struct {
	QueryResult []QueryResult `json:"queryResult"`
	Errors      *QueryErrors  `json:"errors,omitempty"`
}

type QueryErrors struct {
	ID     string     `json:"id"`
	Errors []APIError `json:"errors"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type QueryResult struct {
	RowID          string         `json:"rowId"`
	TimeSeriesList TimeSeriesList `json:"timeSeriesList"`
}

type TimeSeriesList struct {
	TimeSeries []TimeSeries `json:"timeSeries"`
	Unit       string       `json:"unit"`
}

type TimeSeries struct {
	Points Points `json:"points"`
}

type Points struct {
	Timestamps []int64   `json:"timestamps"`
	Values     []float64 `json:"values"`
}
