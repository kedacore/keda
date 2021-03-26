package scalers

import (
	"testing"

	"github.com/kedacore/keda/v2/pkg/scalers/openstack"
	"github.com/stretchr/testify/assert"
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

var invalidOpenstackMetricMetadaTestData = []parseOpenstackMetricMetadataTestData{

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
		{nil, &opentsackMetricMetadataTestData[0], &openstackMetricAuthMetadataTestData[0], "openstack-metric-003bb589-166d-439d-8c31-cbf098d863de-1250-mean"},
		{nil, &opentsackMetricMetadataTestData[1], &openstackMetricAuthMetadataTestData[0], "openstack-metric-003bb589-166d-439d-8c31-cbf098d863de-1250-sum"},
		{nil, &opentsackMetricMetadataTestData[2], &openstackMetricAuthMetadataTestData[0], "openstack-metric-003bb589-166d-439d-8c31-cbf098d863de-1250-max"},
		{nil, &opentsackMetricMetadataTestData[3], &openstackMetricAuthMetadataTestData[0], "openstack-metric-003bb589-166d-439d-8c31-cbf098d863de-1250-mean"},

		{nil, &opentsackMetricMetadataTestData[0], &openstackMetricAuthMetadataTestData[1], "openstack-metric-003bb589-166d-439d-8c31-cbf098d863de-1250-mean"},
		{nil, &opentsackMetricMetadataTestData[1], &openstackMetricAuthMetadataTestData[1], "openstack-metric-003bb589-166d-439d-8c31-cbf098d863de-1250-sum"},
		{nil, &opentsackMetricMetadataTestData[2], &openstackMetricAuthMetadataTestData[1], "openstack-metric-003bb589-166d-439d-8c31-cbf098d863de-1250-max"},
		{nil, &opentsackMetricMetadataTestData[3], &openstackMetricAuthMetadataTestData[1], "openstack-metric-003bb589-166d-439d-8c31-cbf098d863de-1250-mean"},
	}

	for _, testData := range testCases {
		testData := testData
		meta, err := parseOpenstackMetricMetadata(&ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata})

		if err != nil {
			t.Fatal("Could not parse metadata from openstack metrics scaler")
		}

		_, err = parseOpenstackMetricAuthenticationMetadata(&ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata})

		if err != nil {
			t.Fatal("could not parse openstack metric authentication metadata")
		}

		mockMetricsScaler := openstackMetricScaler{meta, openstack.Client{}}
		metricsSpec := mockMetricsScaler.GetMetricSpecForScaling()
		metricName := metricsSpec[0].External.Metric.Name

		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestOpenstackMetricsGetMetricsForSpecScalingInvalidMetaData(t *testing.T) {
	testCases := []openstackMetricScalerMetricIdentifier{
		{nil, &invalidOpenstackMetricMetadaTestData[0], &openstackMetricAuthMetadataTestData[0], "Missing metrics url"},
		{nil, &invalidOpenstackMetricMetadaTestData[1], &openstackMetricAuthMetadataTestData[0], "Empty metrics url"},
		{nil, &invalidOpenstackMetricMetadaTestData[2], &openstackMetricAuthMetadataTestData[0], "Missing metricID"},
		{nil, &invalidOpenstackMetricMetadaTestData[3], &openstackMetricAuthMetadataTestData[0], "Empty metricID"},
		{nil, &invalidOpenstackMetricMetadaTestData[4], &openstackMetricAuthMetadataTestData[0], "Missing aggregation method"},
		{nil, &invalidOpenstackMetricMetadaTestData[5], &openstackMetricAuthMetadataTestData[0], "Missing granularity"},
		{nil, &invalidOpenstackMetricMetadaTestData[6], &openstackMetricAuthMetadataTestData[0], "Missing threshold"},
		{nil, &invalidOpenstackMetricMetadaTestData[7], &openstackMetricAuthMetadataTestData[0], "Missing threshold"},
		{nil, &invalidOpenstackMetricMetadaTestData[8], &openstackMetricAuthMetadataTestData[0], "Missing threshold"},
	}

	for _, testData := range testCases {
		testData := testData
		t.Run(testData.name, func(pt *testing.T) {
			_, err := parseOpenstackMetricMetadata(&ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata})
			assert.NotNil(t, err)
		})
	}
}

func TestOpenstackMetricAuthenticationInvalidAuthMetadata(t *testing.T) {
	testCases := []openstackMetricScalerMetricIdentifier{
		{nil, &opentsackMetricMetadataTestData[0], &invalidOpenstackMetricAuthMetadataTestData[0], "Missing userID"},
		{nil, &opentsackMetricMetadataTestData[0], &invalidOpenstackMetricAuthMetadataTestData[1], "Missing password"},
		{nil, &opentsackMetricMetadataTestData[0], &invalidOpenstackMetricAuthMetadataTestData[2], "Missing authURL"},
		{nil, &opentsackMetricMetadataTestData[0], &invalidOpenstackMetricAuthMetadataTestData[3], "Missing appCredentialID and appCredentialSecret"},
		{nil, &opentsackMetricMetadataTestData[0], &invalidOpenstackMetricAuthMetadataTestData[4], "Missing appCredentialSecret"},
		{nil, &opentsackMetricMetadataTestData[0], &invalidOpenstackMetricAuthMetadataTestData[5], "Missing authURL - application credential"},
	}

	for _, testData := range testCases {
		testData := testData
		t.Run(testData.name, func(ptr *testing.T) {
			_, err := parseOpenstackMetricAuthenticationMetadata(&ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata})
			assert.NotNil(t, err)
		})
	}
}
