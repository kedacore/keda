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
