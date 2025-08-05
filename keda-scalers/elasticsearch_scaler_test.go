package scalers

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

type parseElasticsearchMetadataTestData struct {
	name             string
	metadata         map[string]string
	resolvedEnv      map[string]string
	authParams       map[string]string
	expectedMetadata *elasticsearchMetadata
	expectedError    error
}

type paramsTestData struct {
	name          string
	metadata      map[string]string
	authParams    map[string]string
	expectedQuery map[string]interface{}
}

type elasticsearchMetricIdentifier struct {
	metadataTestData *parseElasticsearchMetadataTestData
	triggerIndex     int
	name             string
}

var testCases = []parseElasticsearchMetadataTestData{
	{
		name:          "must provide either endpoint addresses or cloud config",
		metadata:      map[string]string{},
		authParams:    map[string]string{},
		expectedError: fmt.Errorf("must provide either cloud config or endpoint addresses"),
	},
	{
		name:          "no apiKey given",
		metadata:      map[string]string{"cloudID": "my-cluster:xxxxxxxxxxx"},
		authParams:    map[string]string{},
		expectedError: fmt.Errorf("both cloudID and apiKey must be provided when cloudID or apiKey is used"),
	},
	{
		name:          "can't provide endpoint addresses and cloud config at the same time",
		metadata:      map[string]string{"addresses": "http://localhost:9200", "cloudID": "my-cluster:xxxxxxxxxxx"},
		authParams:    map[string]string{"username": "admin", "apiKey": "xxxxxxxxx"},
		expectedError: fmt.Errorf("can't provide both cloud config and endpoint addresses"),
	},
	{
		name: "both username and password must be provided when addresses is used",
		metadata: map[string]string{
			"addresses":             "http://localhost:9200",
			"unsafeSsl":             "true",
			"index":                 "index1",
			"searchTemplateName":    "myAwesomeSearch",
			"parameters":            "param1:value1",
			"valueLocation":         "hits.hits[0]._source.value",
			"targetValue":           "12.2",
			"activationTargetValue": "3.33",
		},
		authParams:    map[string]string{"username": "admin"},
		expectedError: fmt.Errorf("both username and password must be provided when addresses is used"),
	},
	{
		name:          "no index given",
		metadata:      map[string]string{"addresses": "http://localhost:9200"},
		authParams:    map[string]string{"username": "admin"},
		expectedError: fmt.Errorf("missing required parameter \"index\""),
	},
	{
		name: "query and searchTemplateName provided",
		metadata: map[string]string{
			"addresses":          "http://localhost:9200",
			"index":              "index1",
			"query":              `{"match": {"field": "value"}}`,
			"searchTemplateName": "myTemplate",
			"valueLocation":      "hits.total.value",
			"targetValue":        "12",
		},
		authParams: map[string]string{
			"username": "admin",
			"password": "password",
		},
		expectedError: fmt.Errorf("cannot provide both searchTemplateName and query"),
	},
	{
		name: "neither query nor searchTemplateName provided",
		metadata: map[string]string{
			"addresses":     "http://localhost:9200",
			"index":         "index1",
			"valueLocation": "hits.total.value",
			"targetValue":   "12",
		},
		authParams: map[string]string{
			"username": "admin",
			"password": "password",
		},
		expectedError: fmt.Errorf("either searchTemplateName or query must be provided"),
	},
	{
		name: "no valueLocation given",
		metadata: map[string]string{
			"addresses":          "http://localhost:9200",
			"index":              "index1",
			"searchTemplateName": "searchTemplateName",
		},
		authParams:    map[string]string{"username": "admin"},
		expectedError: fmt.Errorf("missing required parameter \"valueLocation\""),
	},
	{
		name: "no targetValue given",
		metadata: map[string]string{
			"addresses":          "http://localhost:9200",
			"index":              "index1",
			"searchTemplateName": "searchTemplateName",
			"valueLocation":      "toto",
		},
		authParams:    map[string]string{"username": "admin"},
		expectedError: fmt.Errorf("missing required parameter \"targetValue\""),
	},
	{
		name: "invalid targetValue",
		metadata: map[string]string{
			"addresses":          "http://localhost:9200",
			"index":              "index1",
			"searchTemplateName": "searchTemplateName",
			"valueLocation":      "toto",
			"targetValue":        "AA",
		},
		authParams:    map[string]string{"username": "admin"},
		expectedError: fmt.Errorf("unable to set param \"targetValue\""),
	},
	{
		name: "invalid activationTargetValue",
		metadata: map[string]string{
			"addresses":             "http://localhost:9200",
			"index":                 "index1",
			"searchTemplateName":    "searchTemplateName",
			"valueLocation":         "toto",
			"targetValue":           "12",
			"activationTargetValue": "AA",
		},
		authParams:    map[string]string{"username": "admin"},
		expectedError: fmt.Errorf("unable to set param \"activationTargetValue\""),
	},
	{
		name: "all fields ok",
		metadata: map[string]string{
			"addresses":             "http://localhost:9200",
			"unsafeSsl":             "true",
			"index":                 "index1",
			"searchTemplateName":    "myAwesomeSearch",
			"parameters":            "param1:value1",
			"valueLocation":         "hits.hits[0]._source.value",
			"targetValue":           "12.2",
			"activationTargetValue": "3.33",
		},
		authParams: map[string]string{
			"username": "admin",
			"password": "password",
		},
		expectedMetadata: &elasticsearchMetadata{
			Addresses:             []string{"http://localhost:9200"},
			UnsafeSsl:             true,
			Index:                 []string{"index1"},
			Username:              "admin",
			Password:              "password",
			SearchTemplateName:    "myAwesomeSearch",
			Parameters:            []string{"param1:value1"},
			ValueLocation:         "hits.hits[0]._source.value",
			TargetValue:           12.2,
			ActivationTargetValue: 3.33,
			MetricName:            "s0-elasticsearch-myAwesomeSearch",
		},
		expectedError: nil,
	},
	{
		name: "multi indexes",
		metadata: map[string]string{
			"addresses":          "http://localhost:9200",
			"unsafeSsl":          "false",
			"index":              "index1;index2",
			"searchTemplateName": "myAwesomeSearch",
			"parameters":         "param1:value1",
			"valueLocation":      "hits.hits[0]._source.value",
			"targetValue":        "12",
		},
		authParams: map[string]string{
			"username": "admin",
			"password": "password",
		},
		expectedMetadata: &elasticsearchMetadata{
			Addresses:          []string{"http://localhost:9200"},
			UnsafeSsl:          false,
			Index:              []string{"index1", "index2"},
			Username:           "admin",
			Password:           "password",
			SearchTemplateName: "myAwesomeSearch",
			Parameters:         []string{"param1:value1"},
			ValueLocation:      "hits.hits[0]._source.value",
			TargetValue:        12,
			MetricName:         "s0-elasticsearch-myAwesomeSearch",
		},
		expectedError: nil,
	},
	{
		name: "multi indexes trimmed",
		metadata: map[string]string{
			"addresses":          "http://localhost:9200",
			"unsafeSsl":          "false",
			"index":              "index1 ; index2",
			"searchTemplateName": "myAwesomeSearch",
			"parameters":         "param1:value1",
			"valueLocation":      "hits.hits[0]._source.value",
			"targetValue":        "12",
		},
		authParams: map[string]string{
			"username": "admin",
			"password": "password",
		},
		expectedMetadata: &elasticsearchMetadata{
			Addresses:          []string{"http://localhost:9200"},
			UnsafeSsl:          false,
			Index:              []string{"index1", "index2"},
			Username:           "admin",
			Password:           "password",
			SearchTemplateName: "myAwesomeSearch",
			Parameters:         []string{"param1:value1"},
			ValueLocation:      "hits.hits[0]._source.value",
			TargetValue:        12,
			MetricName:         "s0-elasticsearch-myAwesomeSearch",
		},
		expectedError: nil,
	},
	{
		name: "multi addresses",
		metadata: map[string]string{
			"addresses":          "http://localhost:9200,http://localhost:9201",
			"unsafeSsl":          "false",
			"index":              "index1",
			"searchTemplateName": "myAwesomeSearch",
			"parameters":         "param1:value1",
			"valueLocation":      "hits.hits[0]._source.value",
			"targetValue":        "12",
		},
		authParams: map[string]string{
			"username": "admin",
			"password": "password",
		},
		expectedMetadata: &elasticsearchMetadata{
			Addresses:          []string{"http://localhost:9200", "http://localhost:9201"},
			UnsafeSsl:          false,
			Index:              []string{"index1"},
			Username:           "admin",
			Password:           "password",
			SearchTemplateName: "myAwesomeSearch",
			Parameters:         []string{"param1:value1"},
			ValueLocation:      "hits.hits[0]._source.value",
			TargetValue:        12,
			MetricName:         "s0-elasticsearch-myAwesomeSearch",
		},
		expectedError: nil,
	},
	{
		name: "multi addresses trimmed",
		metadata: map[string]string{
			"addresses":          "http://localhost:9200 , http://localhost:9201",
			"unsafeSsl":          "false",
			"index":              "index1",
			"searchTemplateName": "myAwesomeSearch",
			"parameters":         "param1:value1",
			"valueLocation":      "hits.hits[0]._source.value",
			"targetValue":        "12",
		},
		authParams: map[string]string{
			"username": "admin",
			"password": "password",
		},
		expectedMetadata: &elasticsearchMetadata{
			Addresses:          []string{"http://localhost:9200", "http://localhost:9201"},
			UnsafeSsl:          false,
			Index:              []string{"index1"},
			Username:           "admin",
			Password:           "password",
			SearchTemplateName: "myAwesomeSearch",
			Parameters:         []string{"param1:value1"},
			ValueLocation:      "hits.hits[0]._source.value",
			TargetValue:        12,
			MetricName:         "s0-elasticsearch-myAwesomeSearch",
		},
		expectedError: nil,
	},
	{
		name: "password from env",
		metadata: map[string]string{
			"addresses":          "http://localhost:9200,http://localhost:9201",
			"unsafeSsl":          "false",
			"index":              "index1",
			"searchTemplateName": "myAwesomeSearch",
			"parameters":         "param1:value1",
			"valueLocation":      "hits.hits[0]._source.value",
			"targetValue":        "12",
			"passwordFromEnv":    "ELASTICSEARCH_PASSWORD",
		},
		authParams: map[string]string{
			"username": "admin",
		},
		resolvedEnv: map[string]string{
			"ELASTICSEARCH_PASSWORD": "password",
		},
		expectedMetadata: &elasticsearchMetadata{
			Addresses:          []string{"http://localhost:9200", "http://localhost:9201"},
			UnsafeSsl:          false,
			Index:              []string{"index1"},
			Username:           "admin",
			Password:           "password",
			SearchTemplateName: "myAwesomeSearch",
			Parameters:         []string{"param1:value1"},
			ValueLocation:      "hits.hits[0]._source.value",
			TargetValue:        12,
			MetricName:         "s0-elasticsearch-myAwesomeSearch",
		},
		expectedError: nil,
	},
	{
		name: "valid query parameter",
		metadata: map[string]string{
			"addresses":     "http://localhost:9200",
			"index":         "index1",
			"query":         `{"match": {"field": "value"}}`,
			"valueLocation": "hits.total.value",
			"targetValue":   "12",
		},
		authParams: map[string]string{
			"username": "admin",
			"password": "password",
		},
		expectedMetadata: &elasticsearchMetadata{
			Addresses:     []string{"http://localhost:9200"},
			Index:         []string{"index1"},
			Username:      "admin",
			Password:      "password",
			Query:         `{"match": {"field": "value"}}`,
			ValueLocation: "hits.total.value",
			TargetValue:   12,
			MetricName:    "s0-elasticsearch-query",
		},
		expectedError: nil,
	},
}

func TestParseElasticsearchMetadata(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metadata, err := parseElasticsearchMetadata(&scalersconfig.ScalerConfig{
				TriggerMetadata: tc.metadata,
				AuthParams:      tc.authParams,
				ResolvedEnv:     tc.resolvedEnv,
			})
			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
				fmt.Println(tc.name)
				assert.Equal(t, tc.expectedMetadata, &metadata)
			}
		})
	}
}

func TestUnsafeSslDefaultValue(t *testing.T) {
	tc := &parseElasticsearchMetadataTestData{
		name: "all fields ok",
		metadata: map[string]string{
			"addresses":          "http://localhost:9200",
			"index":              "index1",
			"searchTemplateName": "myAwesomeSearch",
			"parameters":         "param1:value1",
			"valueLocation":      "hits.hits[0]._source.value",
			"targetValue":        "12",
		},
		authParams: map[string]string{
			"username": "admin",
			"password": "password",
		},
		expectedMetadata: &elasticsearchMetadata{
			Addresses:          []string{"http://localhost:9200"},
			UnsafeSsl:          false,
			Index:              []string{"index1"},
			Username:           "admin",
			Password:           "password",
			SearchTemplateName: "myAwesomeSearch",
			Parameters:         []string{"param1:value1"},
			ValueLocation:      "hits.hits[0]._source.value",
			TargetValue:        12,
			MetricName:         "s0-elasticsearch-myAwesomeSearch",
		},
		expectedError: nil,
	}
	metadata, err := parseElasticsearchMetadata(&scalersconfig.ScalerConfig{
		TriggerMetadata: tc.metadata,
		AuthParams:      tc.authParams,
	})
	assert.NoError(t, err)
	assert.Equal(t, tc.expectedMetadata, &metadata)
}

func TestBuildQuery(t *testing.T) {
	var testCases = []paramsTestData{
		{
			name: "no params",
			metadata: map[string]string{
				"addresses":          "http://localhost:9200",
				"index":              "index1",
				"searchTemplateName": "myAwesomeSearch",
				"parameters":         "",
				"valueLocation":      "hits.hits[0]._source.value",
				"targetValue":        "12",
			},
			authParams: map[string]string{
				"username": "admin",
				"password": "password",
			},
			expectedQuery: map[string]interface{}{
				"id": "myAwesomeSearch",
			},
		},
		{
			name: "one param",
			metadata: map[string]string{
				"addresses":          "http://localhost:9200",
				"index":              "index1",
				"searchTemplateName": "myAwesomeSearch",
				"parameters":         "param1:value1",
				"valueLocation":      "hits.hits[0]._source.value",
				"targetValue":        "12",
			},
			authParams: map[string]string{
				"username": "admin",
				"password": "password",
			},
			expectedQuery: map[string]interface{}{
				"id": "myAwesomeSearch",
				"params": map[string]interface{}{
					"param1": "value1",
				},
			},
		},
		{
			name: "two params",
			metadata: map[string]string{
				"addresses":          "http://localhost:9200",
				"index":              "index1",
				"searchTemplateName": "myAwesomeSearch",
				"parameters":         "param1:value1;param2:value2",
				"valueLocation":      "hits.hits[0]._source.value",
				"targetValue":        "12",
			},
			authParams: map[string]string{
				"username": "admin",
				"password": "password",
			},
			expectedQuery: map[string]interface{}{
				"id": "myAwesomeSearch",
				"params": map[string]interface{}{
					"param1": "value1",
					"param2": "value2",
				},
			},
		},
		{
			name: "params are trimmed",
			metadata: map[string]string{
				"addresses":          "http://localhost:9200",
				"index":              "index1",
				"searchTemplateName": "myAwesomeSearch",
				"parameters":         "param1 : value1   ; param2 : value2   ",
				"valueLocation":      "hits.hits[0]._source.value",
				"targetValue":        "12",
			},
			authParams: map[string]string{
				"username": "admin",
				"password": "password",
			},
			expectedQuery: map[string]interface{}{
				"id": "myAwesomeSearch",
				"params": map[string]interface{}{
					"param1": "value1",
					"param2": "value2",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metadata, err := parseElasticsearchMetadata(&scalersconfig.ScalerConfig{
				TriggerMetadata: tc.metadata,
				AuthParams:      tc.authParams,
			})
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedQuery, buildQuery(&metadata))
		})
	}
}

func TestElasticsearchGetMetricSpecForScaling(t *testing.T) {
	var elasticsearchMetricIdentifiers = []elasticsearchMetricIdentifier{
		{&testCases[7], 0, "s0-elasticsearch-myAwesomeSearch"},
		{&testCases[8], 1, "s1-elasticsearch-myAwesomeSearch"},
	}

	for _, testData := range elasticsearchMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseElasticsearchMetadata(&scalersconfig.ScalerConfig{
			TriggerMetadata: testData.metadataTestData.metadata,
			AuthParams:      testData.metadataTestData.authParams,
			TriggerIndex:    testData.triggerIndex,
		})
		if testData.metadataTestData.expectedError != nil {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), testData.metadataTestData.expectedError.Error())
			continue
		}
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		elasticsearchScaler := elasticsearchScaler{metadata: meta, esClient: nil}
		metricSpec := elasticsearchScaler.GetMetricSpecForScaling(ctx)
		assert.Equal(t, metricSpec[0].External.Metric.Name, testData.name)
	}
}

func TestIgnoreNullValues(t *testing.T) {
	// Test getValueFromSearch function
	t.Run("getValueFromSearch handles null values based on ignoreNullValues", func(t *testing.T) {
		jsonWithNull := []byte(`{
			"hits": {
				"total": {
					"value": null
				}
			}
		}`)

		// Test ignoreNullValues = false
		_, err := getValueFromSearch(jsonWithNull, "hits.total.value", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "valueLocation must point to value of type number but got: 'Null'")

		// Test ignoreNullValues = true
		val, err := getValueFromSearch(jsonWithNull, "hits.total.value", true)
		assert.NoError(t, err)
		assert.Equal(t, float64(0), val)
	})

	// Test parsing ignoreNullValues parameter
	t.Run("parseElasticsearchMetadata parses ignoreNullValues", func(t *testing.T) {
		metadata, err := parseElasticsearchMetadata(&scalersconfig.ScalerConfig{
			TriggerMetadata: map[string]string{
				"addresses":        "http://localhost:9200",
				"index":            "index1",
				"query":            `{"match": {"field": "value"}}`,
				"valueLocation":    "hits.total.value",
				"targetValue":      "12",
				"ignoreNullValues": "true",
			},
			AuthParams: map[string]string{
				"username": "admin",
				"password": "password",
			},
		})

		assert.NoError(t, err)
		assert.True(t, metadata.IgnoreNullValues)
	})
}
