package scalers

import (
	"fmt"
	"reflect"
	"testing"
)

type parseCoralogixMetadataTestData struct {
	metadata    map[string]string
	metadateRes CoralogixMetadata
	raisesError bool
}

type overloadPrometheusConfig struct {
	metadata          map[string]string
	coralogixMetadata CoralogixMetadata
	metadateRes       map[string]string
}

var testCoralogixMetadata = []parseCoralogixMetadataTestData{
	// No metadata
	{
		metadata:    map[string]string{},
		metadateRes: CoralogixMetadata{},
		raisesError: true,
	},
	// Missing token
	{
		metadata:    map[string]string{domainNameKey: "coralogix.com"},
		metadateRes: CoralogixMetadata{},
		raisesError: true,
	},
	// Missing domain
	{
		metadata:    map[string]string{tokenKey: "abcdefghijklmnop123123456"},
		metadateRes: CoralogixMetadata{},
		raisesError: true,
	},
	// Missing scaler type
	{
		metadata:    map[string]string{domainNameKey: "coralogix.com", tokenKey: "abcdefghijklmnop123123456"},
		metadateRes: CoralogixMetadata{},
		raisesError: true,
	},
	// Basic configurations
	{
		metadata:    map[string]string{domainNameKey: "coralogix.com", tokenKey: "abcdefghijklmnop123123456", scalerTypeKey: promScaler, "query": "some dummy query"},
		metadateRes: CoralogixMetadata{domain: "coralogix.com", token: "token=abcdefghijklmnop123123456", scalerType: promScaler, promServerAddress: "https://prom-api.coralogix.com"},
		raisesError: false,
	},
	// Custom prometheus api endpoint
	{
		metadata:    map[string]string{domainNameKey: "coralogix.com", tokenKey: "abcdefghijklmnop123123456", scalerTypeKey: promScaler, "query": "some dummy query", promServerAddressFmt: "https://blabla.%s"},
		metadateRes: CoralogixMetadata{domain: "coralogix.com", token: "token=abcdefghijklmnop123123456", scalerType: promScaler, promServerAddress: "https://blabla.coralogix.com"},
		raisesError: false,
	},
	// Custom prometheus api endpoint - missing format placeholder
	{
		metadata:    map[string]string{domainNameKey: "coralogix.com", tokenKey: "abcdefghijklmnop123123456", scalerTypeKey: promScaler, "query": "some dummy query", promServerAddressFmt: "https://blabla."},
		metadateRes: CoralogixMetadata{},
		raisesError: true,
	},
	// Custom token header formatter
	{
		metadata:    map[string]string{domainNameKey: "coralogix.com", tokenKey: "abcdefghijklmnop123123456", scalerTypeKey: promScaler, "query": "some dummy query", tokenHeaderStringFmtKey: "token_api=%s"},
		metadateRes: CoralogixMetadata{domain: "coralogix.com", token: "token_api=abcdefghijklmnop123123456", scalerType: promScaler, promServerAddress: "https://prom-api.coralogix.com"},
		raisesError: false,
	},
	// Custom token header formatter - missing format placeholder
	{
		metadata:    map[string]string{domainNameKey: "coralogix.com", tokenKey: "abcdefghijklmnop123123456", scalerTypeKey: promScaler, "query": "some dummy query", tokenHeaderStringFmtKey: "token_api:"},
		metadateRes: CoralogixMetadata{},
		raisesError: true,
	},
}

var testOverloadMetadata = []overloadPrometheusConfig{
	// Basic configurations
	{
		metadata:          map[string]string{domainNameKey: "coralogix.com", tokenKey: "abcdefghijklmnop123123456", scalerTypeKey: promScaler, "query": "some dummy query"},
		coralogixMetadata: CoralogixMetadata{domain: "coralogix.com", token: "token=abcdefghijklmnop123123456", scalerType: promScaler, promServerAddress: "https://prom-api.coralogix.com"},
		metadateRes: map[string]string{
			domainNameKey: "coralogix.com", tokenKey: "abcdefghijklmnop123123456", scalerTypeKey: promScaler, "query": "some dummy query",
			promCustomHeaders: "token=abcdefghijklmnop123123456", promServerAddress: "https://prom-api.coralogix.com",
		},
	},
	// Append headers
	{
		metadata:          map[string]string{promCustomHeaders: "some_header=bla_bla", domainNameKey: "coralogix.com", tokenKey: "abcdefghijklmnop123123456", scalerTypeKey: promScaler, "query": "some dummy query"},
		coralogixMetadata: CoralogixMetadata{domain: "coralogix.com", token: "token=abcdefghijklmnop123123456", scalerType: promScaler, promServerAddress: "https://prom-api.coralogix.com"},
		metadateRes: map[string]string{
			domainNameKey: "coralogix.com", tokenKey: "abcdefghijklmnop123123456", scalerTypeKey: promScaler, "query": "some dummy query",
			promCustomHeaders: "some_header=bla_bla,token=abcdefghijklmnop123123456", promServerAddress: "https://prom-api.coralogix.com",
		},
	},
	// Append headers if not exists
	{
		metadata:          map[string]string{promCustomHeaders: fmt.Sprintf("some_header=bla_bla,%s=some_token", tokenKey), domainNameKey: "coralogix.com", tokenKey: "abcdefghijklmnop123123456", scalerTypeKey: promScaler, "query": "some dummy query"},
		coralogixMetadata: CoralogixMetadata{domain: "coralogix.com", token: "token=abcdefghijklmnop123123456", scalerType: promScaler, promServerAddress: "https://prom-api.coralogix.com"},
		metadateRes: map[string]string{
			domainNameKey: "coralogix.com", tokenKey: "abcdefghijklmnop123123456", scalerTypeKey: promScaler, "query": "some dummy query",
			promCustomHeaders: fmt.Sprintf("some_header=bla_bla,%s=some_token", tokenKey), promServerAddress: "https://prom-api.coralogix.com",
		},
	},
}

func TestParseCoralogixMetadata(t *testing.T) {
	for _, testData := range testCoralogixMetadata {
		meta, err := parseCoralogixMetadata(&ScalerConfig{TriggerMetadata: testData.metadata})
		if err != nil && !testData.raisesError {
			t.Error("Expected success but got error:", err)
		} else if !testData.raisesError && !reflect.DeepEqual(meta, &testData.metadateRes) {
			t.Errorf("Expected parsed metadata to be equal to test values:\nParsed metadate: %+v \nTest metadata: %+v", meta, testData.metadateRes)
		}
	}
}

func TestOverloadPrometheusMetadata(t *testing.T) {
	for _, testData := range testOverloadMetadata {
		scalerConfig := OverloadPrometheusMetadata(&ScalerConfig{TriggerMetadata: testData.metadata}, &testData.coralogixMetadata)
		if !reflect.DeepEqual(&scalerConfig.TriggerMetadata, &testData.metadateRes) {
			t.Errorf("Expected parsed metadata to be equal to test values:\nParsed metadate: %+v \nTest metadata: %+v", scalerConfig.TriggerMetadata, testData.metadateRes)
		}
	}
}
