package scalers

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
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
		expectedError: ErrElasticsearchMissingAddressesOrCloudConfig,
	},
	{
		name:          "no apiKey given",
		metadata:      map[string]string{"cloudID": "my-cluster:xxxxxxxxxxx"},
		authParams:    map[string]string{},
		expectedError: ErrScalerConfigMissingField,
	},
	{
		name:          "can't provide endpoint addresses and cloud config at the same time",
		metadata:      map[string]string{"addresses": "http://localhost:9200", "cloudID": "my-cluster:xxxxxxxxxxx"},
		authParams:    map[string]string{"username": "admin", "apiKey": "xxxxxxxxx"},
		expectedError: ErrElasticsearchConfigConflict,
	},
	{
		name:          "no index given",
		metadata:      map[string]string{"addresses": "http://localhost:9200"},
		authParams:    map[string]string{"username": "admin"},
		expectedError: ErrScalerConfigMissingField,
	},
	{
		name: "no searchTemplateName given",
		metadata: map[string]string{
			"addresses": "http://localhost:9200",
			"index":     "index1",
		},
		authParams:    map[string]string{"username": "admin"},
		expectedError: ErrScalerConfigMissingField,
	},
	{
		name: "no valueLocation given",
		metadata: map[string]string{
			"addresses":          "http://localhost:9200",
			"index":              "index1",
			"searchTemplateName": "searchTemplateName",
		},
		authParams:    map[string]string{"username": "admin"},
		expectedError: ErrScalerConfigMissingField,
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
		expectedError: ErrScalerConfigMissingField,
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
		expectedError: strconv.ErrSyntax,
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
		expectedError: strconv.ErrSyntax,
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
			addresses:             []string{"http://localhost:9200"},
			unsafeSsl:             true,
			indexes:               []string{"index1"},
			username:              "admin",
			password:              "password",
			searchTemplateName:    "myAwesomeSearch",
			parameters:            []string{"param1:value1"},
			valueLocation:         "hits.hits[0]._source.value",
			targetValue:           12.2,
			activationTargetValue: 3.33,
			metricName:            "s0-elasticsearch-myAwesomeSearch",
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
			metadata, err := parseElasticsearchMetadata(&scalersconfig.ScalerConfig{
				TriggerMetadata: tc.metadata,
				AuthParams:      tc.authParams,
				ResolvedEnv:     tc.resolvedEnv,
			})
			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
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
	metadata, err := parseElasticsearchMetadata(&scalersconfig.ScalerConfig{
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
			metadata, err := parseElasticsearchMetadata(&scalersconfig.ScalerConfig{
				TriggerMetadata: tc.metadata,
				AuthParams:      tc.authParams,
			})
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedQuery, buildQuery(metadata))
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
			assert.ErrorIs(t, err, testData.metadataTestData.expectedError)
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
