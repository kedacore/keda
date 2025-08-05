package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"

	"github.com/kedacore/keda/v2/keda-scalers/openstack"
	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

type parseOpenstackMetricMetadataTestData struct {
	metadata map[string]string
}

type parseOpenstackMetricAuthMetadataTestData struct {
	authMetadata map[string]string
}

type openstackMetricScalerMetricIdentifier struct {
	resolvedEnv          map[string]string
	metadataTestData     *parseOpenstackMetricMetadataTestData
	authMetadataTestData *parseOpenstackMetricAuthMetadataTestData
	triggerIndex         int
	name                 string
}

var opentsackMetricMetadataTestData = []parseOpenstackMetricMetadataTestData{
	{metadata: map[string]string{"metricsURL": "http://localhost:8041/v1/metric", "metricID": "003bb589-166d-439d-8c31-cbf098d863de", "aggregationMethod": "mean", "granularity": "300", "threshold": "1250"}},
	{metadata: map[string]string{"metricsURL": "http://localhost:8041/v1/metric", "metricID": "003bb589-166d-439d-8c31-cbf098d863de", "aggregationMethod": "sum", "granularity": "300", "threshold": "1250"}},
	{metadata: map[string]string{"metricsURL": "http://localhost:8041/v1/metric", "metricID": "003bb589-166d-439d-8c31-cbf098d863de", "aggregationMethod": "max", "granularity": "300", "threshold": "1250"}},
	{metadata: map[string]string{"metricsURL": "http://localhost:8041/v1/metric", "metricID": "003bb589-166d-439d-8c31-cbf098d863de", "aggregationMethod": "mean", "granularity": "300", "threshold": "1250", "timeout": "30"}},
}

var openstackMetricAuthMetadataTestData = []parseOpenstackMetricAuthMetadataTestData{
	{authMetadata: map[string]string{"userID": "my-id", "password": "my-password", "authURL": "http://localhost:5000/v3/"}},
	{authMetadata: map[string]string{"appCredentialID": "my-app-credential-id", "appCredentialSecret": "my-app-credential-secret", "authURL": "http://localhost:5000/v3/"}},
}

var invalidOpenstackMetricMetadataTestData = []parseOpenstackMetricMetadataTestData{

	// Missing metrics url
	{metadata: map[string]string{"metricID": "003bb589-166d-439d-8c31-cbf098d863de", "aggregationMethod": "mean", "granularity": "300", "threshold": "1250"}},

	// Empty metrics url
	{metadata: map[string]string{"metricsUrl": "", "metricID": "003bb589-166d-439d-8c31-cbf098d863de", "aggregationMethod": "mean", "granularity": "300", "threshold": "1250"}},

	// Missing metricID
	{metadata: map[string]string{"metricsUrl": "http://localhost:8041/v1/metric", "aggregationMethod": "mean", "granularity": "300", "threshold": "1250", "timeout": "30"}},

	// Empty metricID
	{metadata: map[string]string{"metricsUrl": "http://localhost:8041/v1/metric", "metricID": "", "aggregationMethod": "mean", "granularity": "300", "threshold": "1250"}},

	// Missing aggregation method
	{metadata: map[string]string{"metricsUrl": "http://localhost:8041/v1/metric", "metricID": "003bb589-166d-439d-8c31-cbf098d863de", "granularity": "300", "threshold": "1250", "timeout": "30"}},

	// Missing granularity
	{metadata: map[string]string{"metricsUrl": "http://localhost:8041/v1/metric", "metricID": "003bb589-166d-439d-8c31-cbf098d863de", "aggregationMethod": "mean", "threshold": "1250", "timeout": "30"}},

	// Missing threshold
	{metadata: map[string]string{"metricsUrl": "http://localhost:8041/v1/metric", "metricID": "003bb589-166d-439d-8c31-cbf098d863de", "aggregationMethod": "mean", "granularity": "300", "timeout": "30"}},

	// granularity 0
	{metadata: map[string]string{"metricsURL": "http://localhost:8041/v1/metric", "metricID": "003bb589-166d-439d-8c31-cbf098d863de", "aggregationMethod": "mean", "granularity": "avc", "threshold": "1250"}},

	// threshold 0
	{metadata: map[string]string{"metricsURL": "http://localhost:8041/v1/metric", "metricID": "003bb589-166d-439d-8c31-cbf098d863de", "aggregationMethod": "mean", "granularity": "300", "threshold": "0z"}},

	// activation threshold invalid
	{metadata: map[string]string{"metricsURL": "http://localhost:8041/v1/metric", "metricID": "003bb589-166d-439d-8c31-cbf098d863de", "aggregationMethod": "mean", "granularity": "300", "threshold": "0", "activationThreshold": "z"}},
}

var invalidOpenstackMetricAuthMetadataTestData = []parseOpenstackMetricAuthMetadataTestData{
	// Using Password method:

	// Missing userID
	{authMetadata: map[string]string{"password": "my-password", "authURL": "http://localhost:5000/v3/"}},
	// Missing password
	{authMetadata: map[string]string{"userID": "my-id", "authURL": "http://localhost:5000/v3/"}},

	// Missing authURL
	{authMetadata: map[string]string{"userID": "my-id", "password": "my-password"}},

	// Using Application Credentials method:

	// Missing appCredentialID and appCredentialSecret
	{authMetadata: map[string]string{"authURL": "http://localhost:5000/v3/"}},
	// Missing appCredentialSecret
	{authMetadata: map[string]string{"appCredentialID": "my-app-credential-id", "authURL": "http://localhost:5000/v3/"}},
	// Missing authURL
	{authMetadata: map[string]string{"appCredentialID": "my-app-credential-id", "appCredentialSecret": "my-app-credential-secret"}},
}

func TestOpenstackMetricsGetMetricsForSpecScaling(t *testing.T) {
	// first, test cases with authentication based on password
	testCases := []openstackMetricScalerMetricIdentifier{
		{nil, &opentsackMetricMetadataTestData[0], &openstackMetricAuthMetadataTestData[0], 0, "s0-openstack-metric-003bb589-166d-439d-8c31-cbf098d863de"},
		{nil, &opentsackMetricMetadataTestData[1], &openstackMetricAuthMetadataTestData[0], 1, "s1-openstack-metric-003bb589-166d-439d-8c31-cbf098d863de"},
		{nil, &opentsackMetricMetadataTestData[2], &openstackMetricAuthMetadataTestData[0], 2, "s2-openstack-metric-003bb589-166d-439d-8c31-cbf098d863de"},
		{nil, &opentsackMetricMetadataTestData[3], &openstackMetricAuthMetadataTestData[0], 3, "s3-openstack-metric-003bb589-166d-439d-8c31-cbf098d863de"},
		{nil, &opentsackMetricMetadataTestData[0], &openstackMetricAuthMetadataTestData[1], 4, "s4-openstack-metric-003bb589-166d-439d-8c31-cbf098d863de"},
		{nil, &opentsackMetricMetadataTestData[1], &openstackMetricAuthMetadataTestData[1], 5, "s5-openstack-metric-003bb589-166d-439d-8c31-cbf098d863de"},
		{nil, &opentsackMetricMetadataTestData[2], &openstackMetricAuthMetadataTestData[1], 6, "s6-openstack-metric-003bb589-166d-439d-8c31-cbf098d863de"},
		{nil, &opentsackMetricMetadataTestData[3], &openstackMetricAuthMetadataTestData[1], 7, "s7-openstack-metric-003bb589-166d-439d-8c31-cbf098d863de"},
	}

	for _, testData := range testCases {
		meta, err := parseOpenstackMetricMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata, TriggerIndex: testData.triggerIndex})

		if err != nil {
			t.Fatal("Could not parse metadata from openstack metrics scaler")
		}

		_, err = parseOpenstackMetricAuthenticationMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata, TriggerIndex: testData.triggerIndex})

		if err != nil {
			t.Fatal("could not parse openstack metric authentication metadata")
		}

		mockMetricsScaler := openstackMetricScaler{"", meta, openstack.Client{}, logr.Discard()}
		metricsSpec := mockMetricsScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricsSpec[0].External.Metric.Name

		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestOpenstackMetricsGetMetricsForSpecScalingInvalidMetaData(t *testing.T) {
	testCases := []openstackMetricScalerMetricIdentifier{
		{nil, &invalidOpenstackMetricMetadataTestData[0], &openstackMetricAuthMetadataTestData[0], 0, "s0-Missing metrics url"},
		{nil, &invalidOpenstackMetricMetadataTestData[1], &openstackMetricAuthMetadataTestData[0], 1, "s1-Empty metrics url"},
		{nil, &invalidOpenstackMetricMetadataTestData[2], &openstackMetricAuthMetadataTestData[0], 2, "s2-Missing metricID"},
		{nil, &invalidOpenstackMetricMetadataTestData[3], &openstackMetricAuthMetadataTestData[0], 3, "s3-Empty metricID"},
		{nil, &invalidOpenstackMetricMetadataTestData[4], &openstackMetricAuthMetadataTestData[0], 4, "s4-Missing aggregation method"},
		{nil, &invalidOpenstackMetricMetadataTestData[5], &openstackMetricAuthMetadataTestData[0], 5, "s5-Missing granularity"},
		{nil, &invalidOpenstackMetricMetadataTestData[6], &openstackMetricAuthMetadataTestData[0], 6, "s6-Missing threshold"},
		{nil, &invalidOpenstackMetricMetadataTestData[7], &openstackMetricAuthMetadataTestData[0], 7, "s7-Missing threshold"},
		{nil, &invalidOpenstackMetricMetadataTestData[8], &openstackMetricAuthMetadataTestData[0], 8, "s8-Missing threshold"},
	}

	for _, testData := range testCases {
		t.Run(testData.name, func(pt *testing.T) {
			_, err := parseOpenstackMetricMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata, TriggerIndex: testData.triggerIndex})
			assert.NotNil(t, err)
		})
	}
}

func TestOpenstackMetricAuthenticationInvalidAuthMetadata(t *testing.T) {
	testCases := []openstackMetricScalerMetricIdentifier{
		{nil, &opentsackMetricMetadataTestData[0], &invalidOpenstackMetricAuthMetadataTestData[0], 0, "s0-Missing userID"},
		{nil, &opentsackMetricMetadataTestData[0], &invalidOpenstackMetricAuthMetadataTestData[1], 1, "s1-Missing password"},
		{nil, &opentsackMetricMetadataTestData[0], &invalidOpenstackMetricAuthMetadataTestData[2], 2, "s2-Missing authURL"},
		{nil, &opentsackMetricMetadataTestData[0], &invalidOpenstackMetricAuthMetadataTestData[3], 3, "s3-Missing appCredentialID and appCredentialSecret"},
		{nil, &opentsackMetricMetadataTestData[0], &invalidOpenstackMetricAuthMetadataTestData[4], 4, "s4-Missing appCredentialSecret"},
		{nil, &opentsackMetricMetadataTestData[0], &invalidOpenstackMetricAuthMetadataTestData[5], 5, "s5-Missing authURL - application credential"},
	}

	for _, testData := range testCases {
		t.Run(testData.name, func(ptr *testing.T) {
			_, err := parseOpenstackMetricAuthenticationMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata, TriggerIndex: testData.triggerIndex})
			assert.NotNil(t, err)
		})
	}
}
