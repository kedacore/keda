package scalers

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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

var testCases = []parseElasticsearchMetadataTestData{
	{
		name:          "no addresses given",
		metadata:      map[string]string{},
		authParams:    map[string]string{},
		expectedError: errors.New("no addresses given"),
	},
	{
		name:          "no index given",
		metadata:      map[string]string{"addresses": "http://localhost:9200"},
		authParams:    map[string]string{"username": "admin"},
		expectedError: errors.New("no index given"),
	},
	{
		name: "no searchTemplateName given",
		metadata: map[string]string{
			"addresses": "http://localhost:9200",
			"index":     "index1",
		},
		authParams:    map[string]string{"username": "admin"},
		expectedError: errors.New("no searchTemplateName given"),
	},
	{
		name: "no valueLocation given",
		metadata: map[string]string{
			"addresses":          "http://localhost:9200",
			"index":              "index1",
			"searchTemplateName": "searchTemplateName",
		},
		authParams:    map[string]string{"username": "admin"},
		expectedError: errors.New("no valueLocation given"),
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
		expectedError: errors.New("no targetValue given"),
	},
	{
		name: "all fields ok",
		metadata: map[string]string{
			"addresses":          "http://localhost:9200",
			"unsafeSsl":          "true",
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
			addresses:          []string{"http://localhost:9200"},
			unsafeSsl:          true,
			indexes:            []string{"index1"},
			username:           "admin",
			password:           "password",
			searchTemplateName: "myAwesomeSearch",
			parameters:         []string{"param1:value1"},
			valueLocation:      "hits.hits[0]._source.value",
			targetValue:        12,
			metricName:         "s0-elasticsearch-myAwesomeSearch",
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
			addresses:          []string{"http://localhost:9200"},
			unsafeSsl:          false,
			indexes:            []string{"index1", "index2"},
			username:           "admin",
			password:           "password",
			searchTemplateName: "myAwesomeSearch",
			parameters:         []string{"param1:value1"},
			valueLocation:      "hits.hits[0]._source.value",
			targetValue:        12,
			metricName:         "s0-elasticsearch-myAwesomeSearch",
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
			addresses:          []string{"http://localhost:9200"},
			unsafeSsl:          false,
			indexes:            []string{"index1", "index2"},
			username:           "admin",
			password:           "password",
			searchTemplateName: "myAwesomeSearch",
			parameters:         []string{"param1:value1"},
			valueLocation:      "hits.hits[0]._source.value",
			targetValue:        12,
			metricName:         "s0-elasticsearch-myAwesomeSearch",
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
			addresses:          []string{"http://localhost:9200", "http://localhost:9201"},
			unsafeSsl:          false,
			indexes:            []string{"index1"},
			username:           "admin",
			password:           "password",
			searchTemplateName: "myAwesomeSearch",
			parameters:         []string{"param1:value1"},
			valueLocation:      "hits.hits[0]._source.value",
			targetValue:        12,
			metricName:         "s0-elasticsearch-myAwesomeSearch",
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
			addresses:          []string{"http://localhost:9200", "http://localhost:9201"},
			unsafeSsl:          false,
			indexes:            []string{"index1"},
			username:           "admin",
			password:           "password",
			searchTemplateName: "myAwesomeSearch",
			parameters:         []string{"param1:value1"},
			valueLocation:      "hits.hits[0]._source.value",
			targetValue:        12,
			metricName:         "s0-elasticsearch-myAwesomeSearch",
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
			addresses:          []string{"http://localhost:9200", "http://localhost:9201"},
			unsafeSsl:          false,
			indexes:            []string{"index1"},
			username:           "admin",
			password:           "password",
			searchTemplateName: "myAwesomeSearch",
			parameters:         []string{"param1:value1"},
			valueLocation:      "hits.hits[0]._source.value",
			targetValue:        12,
			metricName:         "s0-elasticsearch-myAwesomeSearch",
		},
		expectedError: nil,
	},
}

func TestParseElasticsearchMetadata(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metadata, err := parseElasticsearchMetadata(&ScalerConfig{
				TriggerMetadata: tc.metadata,
				AuthParams:      tc.authParams,
				ResolvedEnv:     tc.resolvedEnv,
			})
			if tc.expectedError != nil {
				assert.Contains(t, err.Error(), tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
				fmt.Println(tc.name)
				assert.Equal(t, tc.expectedMetadata, metadata)
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
			addresses:          []string{"http://localhost:9200"},
			unsafeSsl:          false,
			indexes:            []string{"index1"},
			username:           "admin",
			password:           "password",
			searchTemplateName: "myAwesomeSearch",
			parameters:         []string{"param1:value1"},
			valueLocation:      "hits.hits[0]._source.value",
			targetValue:        12,
			metricName:         "s0-elasticsearch-myAwesomeSearch",
		},
		expectedError: nil,
	}
	metadata, err := parseElasticsearchMetadata(&ScalerConfig{
		TriggerMetadata: tc.metadata,
		AuthParams:      tc.authParams,
	})
	assert.NoError(t, err)
	assert.Equal(t, tc.expectedMetadata, metadata)
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
			metadata, err := parseElasticsearchMetadata(&ScalerConfig{
				TriggerMetadata: tc.metadata,
				AuthParams:      tc.authParams,
			})
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedQuery, buildQuery(metadata))
		})
	}
}
