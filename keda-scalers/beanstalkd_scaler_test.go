package scalers

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

const (
	beanstalkdServer = "localhost:3000"
)

type parseBeanstalkdMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type beanstalkdMetricIdentifier struct {
	metadataTestData *parseBeanstalkdMetadataTestData
	index            int
	name             string
}

type tubeStatsTestData struct {
	response map[string]interface{}
	metadata map[string]string
	isActive bool
}

var testBeanstalkdMetadata = []parseBeanstalkdMetadataTestData{
	// nothing passed
	{map[string]string{}, true},
	// properly formed
	{map[string]string{"server": beanstalkdServer, "tube": "delayed", "value": "1", "includeDelayed": "true"}, false},
	// no includeDelayed
	{map[string]string{"server": beanstalkdServer, "tube": "no-delayed", "value": "1"}, false},
	// missing server
	{map[string]string{"tube": "stats-tube", "value": "1", "includeDelayed": "true"}, true},
	// missing tube
	{map[string]string{"server": beanstalkdServer, "value": "1", "includeDelayed": "true"}, true},
	// missing value
	{map[string]string{"server": beanstalkdServer, "tube": "stats-tube", "includeDelayed": "true"}, true},
	// invalid value
	{map[string]string{"server": beanstalkdServer, "tube": "stats-tube", "value": "lots", "includeDelayed": "true"}, true},
	// valid timeout
	{map[string]string{"server": beanstalkdServer, "tube": "stats-tube", "value": "1", "includeDelayed": "true", "timeout": "1000"}, false},
	// invalid timeout
	{map[string]string{"server": beanstalkdServer, "tube": "stats-tube", "value": "1", "includeDelayed": "true", "timeout": "-1"}, true},
	// activationValue passed
	{map[string]string{"server": beanstalkdServer, "tube": "stats-tube", "value": "1", "activationValue": "10"}, false},
	// invalid activationValue passed
	{map[string]string{"server": beanstalkdServer, "tube": "stats-tube", "value": "1", "activationValue": "AA"}, true},
}

var beanstalkdMetricIdentifiers = []beanstalkdMetricIdentifier{
	{&testBeanstalkdMetadata[2], 0, "s0-beanstalkd-no-delayed"},
	{&testBeanstalkdMetadata[1], 1, "s1-beanstalkd-delayed"},
}

var testTubeStatsTestData = []tubeStatsTestData{
	{
		response: map[string]interface{}{
			"cmd-delete":            18,
			"cmd-pause-tube":        0,
			"current-jobs-buried":   6,
			"current-jobs-delayed":  0,
			"current-jobs-ready":    10,
			"current-jobs-reserved": 0,
			"current-jobs-urgent":   0,
			"current-using":         3,
			"current-waiting":       3,
			"current-watching":      3,
			"name":                  "form-crawler-notifications",
			"pause":                 0,
			"pause-time-left":       0,
			"total-jobs":            24,
		},
		metadata: map[string]string{"server": beanstalkdServer, "tube": "no-delayed", "value": "2"},
		isActive: true,
	},
	{
		response: map[string]interface{}{
			"cmd-delete":            18,
			"cmd-pause-tube":        0,
			"current-jobs-buried":   0,
			"current-jobs-delayed":  0,
			"current-jobs-ready":    1,
			"current-jobs-reserved": 0,
			"current-jobs-urgent":   0,
			"current-using":         3,
			"current-waiting":       3,
			"current-watching":      3,
			"name":                  "form-crawler-notifications",
			"pause":                 0,
			"pause-time-left":       0,
			"total-jobs":            24,
		},
		metadata: map[string]string{"server": beanstalkdServer, "tube": "no-delayed", "value": "3", "activationValue": "2"},
		isActive: false,
	},
	{
		response: map[string]interface{}{
			"cmd-delete":            18,
			"cmd-pause-tube":        0,
			"current-jobs-buried":   0,
			"current-jobs-delayed":  10,
			"current-jobs-ready":    0,
			"current-jobs-reserved": 0,
			"current-jobs-urgent":   0,
			"current-using":         3,
			"current-waiting":       3,
			"current-watching":      3,
			"name":                  "form-crawler-notifications",
			"pause":                 0,
			"pause-time-left":       0,
			"total-jobs":            24,
		},
		metadata: map[string]string{"server": beanstalkdServer, "tube": "no-delayed", "value": "2"},
		isActive: false,
	},
	{
		response: map[string]interface{}{
			"cmd-delete":            18,
			"cmd-pause-tube":        0,
			"current-jobs-buried":   0,
			"current-jobs-delayed":  10,
			"current-jobs-ready":    0,
			"current-jobs-reserved": 0,
			"current-jobs-urgent":   0,
			"current-using":         3,
			"current-waiting":       3,
			"current-watching":      3,
			"name":                  "form-crawler-notifications",
			"pause":                 0,
			"pause-time-left":       0,
			"total-jobs":            24,
		},
		metadata: map[string]string{"server": beanstalkdServer, "tube": "no-delayed", "value": "2", "includeDelayed": "true"},
		isActive: true,
	},
}

func TestBeanstalkdParseMetadata(t *testing.T) {
	for idx, testData := range testBeanstalkdMetadata {
		meta, err := parseBeanstalkdMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error but got success in test case %d", idx)
		}
		if err == nil {
			if val, ok := testData.metadata["includeDelayed"]; !ok {
				assert.Equal(t, false, meta.IncludeDelayed)
			} else {
				boolVal, err := strconv.ParseBool(val)
				if err != nil {
					assert.Equal(t, boolVal, meta.IncludeDelayed)
				}
			}
		}
	}
}

func TestBeanstalkdGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range beanstalkdMetricIdentifiers {
		meta, err := parseBeanstalkdMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: nil, TriggerIndex: testData.index})
		if err != nil {
			t.Fatal("could not parse metadata", err)
		}
		mockBeanstalkdScaler := BeanstalkdScaler{
			metadata:   meta,
			connection: nil,
			tube:       nil,
		}

		metricSpec := mockBeanstalkdScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		assert.Equal(t, testData.name, metricName, "correct external source name")
	}
}

func TestGetTubeStats(t *testing.T) {
	for _, testData := range testTubeStatsTestData {
		yamlData, err := yaml.Marshal(testData.response)
		if err != nil {
			t.Fatal(err)
		}

		response := []byte(fmt.Sprintf("OK %d\r\n", len(yamlData)))
		response = append(response, yamlData...)
		response = append(response, []byte("\r\n")...)
		createTestServer(t, response)

		s, err := NewBeanstalkdScaler(
			&scalersconfig.ScalerConfig{
				TriggerMetadata:   testData.metadata,
				GlobalHTTPTimeout: 1000 * time.Millisecond,
			},
		)

		assert.NoError(t, err)

		ctx := context.Background()
		_, active, err := s.GetMetricsAndActivity(ctx, "Metric")

		assert.NoError(t, err)

		assert.Equal(t, testData.isActive, active)
	}
}

func TestGetTubeStatsNotFound(t *testing.T) {
	testData := testTubeStatsTestData[0]
	createTestServer(t, []byte("NOT_FOUND\r\n"))
	s, err := NewBeanstalkdScaler(
		&scalersconfig.ScalerConfig{
			TriggerMetadata:   testData.metadata,
			GlobalHTTPTimeout: 1000 * time.Millisecond,
		},
	)

	assert.NoError(t, err)

	ctx := context.Background()
	_, active, err := s.GetMetricsAndActivity(ctx, "Metric")

	assert.NoError(t, err)
	assert.False(t, active)
}

func createTestServer(t *testing.T, response []byte) {
	list, err := net.Listen("tcp", "localhost:3000")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		defer list.Close()
		conn, err := list.Accept()
		if err != nil {
			return
		}

		_, err = conn.Write(response)
		assert.NoError(t, err)
		conn.Close()
	}()
}
