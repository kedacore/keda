package splunk

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	savedSearchPathTemplateStr = "/servicesNS/%s/search/search/jobs/export"
)

// Config contains the information required to authenticate with a Splunk instance.
type Config struct {
	Host        string
	Username    string
	Password    string
	APIToken    string
	HTTPTimeout time.Duration
	VerifyTLS   bool
}

// Client contains Splunk config information as well as an http client for requests.
type Client struct {
	*Config
	*http.Client
}

// SearchResponse is used for unmarshalling search results.
type SearchResponse struct {
	Result map[string]string `json:"result"`
}

// NewClient returns a new Splunk client.
func NewClient(c *Config) (*Client, error) {
	if c.APIToken != "" && c.Password != "" {
		return nil, errors.New("API token and Password were all set. If APIToken is set, username and password must not be used")
	}

	if c.APIToken != "" && c.Username == "" {
		return nil, errors.New("API token was set and username was not. Username is needed to determine who owns the saved search")
	}

	client := &Client{
		c,
		&http.Client{
			Timeout: c.HTTPTimeout,
		},
	}

	if !c.VerifyTLS {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	return client, nil
}

// SavedSearch fetches the results of a saved search/report in Splunk.
func (c *Client) SavedSearch(name string) (*SearchResponse, error) {
	savedSearchAPIPath := fmt.Sprintf(savedSearchPathTemplateStr, c.Username)
	endpoint := fmt.Sprintf("%s%s", c.Host, savedSearchAPIPath)

	body := strings.NewReader(fmt.Sprintf("search=savedsearch %s", name))
	req, err := http.NewRequest(http.MethodPost, endpoint, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if c.APIToken != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.APIToken))
	} else {
		req.SetBasicAuth(c.Username, c.Password)
	}

	req.URL.RawQuery = url.Values{
		"output_mode": {"json"},
	}.Encode()

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode > 399 {
		bodyText, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(string(bodyText))
	}

	result := &SearchResponse{}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return result, err
	}

	return result, nil
}

// ToMetric converts a search response to a consumable metric value.
func (s *SearchResponse) ToMetric(valueField string) (float64, error) {
	metricValueStr, ok := s.Result[valueField]
	if !ok {
		return 0, fmt.Errorf("field: %s not found in search results", valueField)
	}

	metricValueInt, err := strconv.ParseFloat(metricValueStr, 64)
	if err != nil {
		return 0, fmt.Errorf("value: %s is not a float value", valueField)
	}

	return metricValueInt, nil
}
