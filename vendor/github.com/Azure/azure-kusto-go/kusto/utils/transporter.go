package utils

import "net/http"

type Transporter struct {
	Http *http.Client
}

func (t Transporter) Do(req *http.Request) (*http.Response, error) {
	return t.Http.Do(req)
}
