package scalers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type parseOpenstackSwiftMetadataTestData struct {
	metadata map[string]string
}

type parseOpenstackSwiftAuthMetadataTestData struct {
	authMetadata map[string]string
}

type openstackSwiftMetricIdentifier struct {
	resolvedEnv          map[string]string
	metadataTestData     *parseOpenstackSwiftMetadataTestData
	authMetadataTestData *parseOpenstackSwiftAuthMetadataTestData
	name                 string
}

var openstackSwiftMetadataTestData = []parseOpenstackSwiftMetadataTestData{
	// Only required parameters
	{metadata: map[string]string{"swiftURL": "http://localhost:8080/v1/my-account-id", "containerName": "my-container"}},
	// Adding objectCount
	{metadata: map[string]string{"swiftURL": "http://localhost:8080/v1/my-account-id", "containerName": "my-container", "objectCount": "5"}},
	// Adding objectPrefix
	{metadata: map[string]string{"swiftURL": "http://localhost:8080/v1/my-account-id", "containerName": "my-container", "objectCount": "5", "objectPrefix": "my-prefix"}},
	// Adding objectDelimiter
	{metadata: map[string]string{"swiftURL": "http://localhost:8080/v1/my-account-id", "containerName": "my-container", "objectCount": "5", "objectDelimiter": "/"}},
	// Adding objectLimit
	{metadata: map[string]string{"swiftURL": "http://localhost:8080/v1/my-account-id", "containerName": "my-container", "objectCount": "5", "objectLimit": "1000"}},
	// Adding timeout
	{metadata: map[string]string{"swiftURL": "http://localhost:8080/v1/my-account-id", "containerName": "my-container", "objectCount": "5", "timeout": "2"}},
	// Adding onlyFiles
	{metadata: map[string]string{"swiftURL": "http://localhost:8080/v1/my-account-id", "containerName": "my-container", "onlyFiles": "true"}},
}

var openstackSwiftAuthMetadataTestData = []parseOpenstackSwiftAuthMetadataTestData{
	{authMetadata: map[string]string{"userID": "my-id", "password": "my-password", "projectID": "my-project-id", "authURL": "http://localhost:5000/v3/"}},
	{authMetadata: map[string]string{"appCredentialID": "my-app-credential-id", "appCredentialSecret": "my-app-credential-secret", "authURL": "http://localhost:5000/v3/"}},
}

var invalidOpenstackSwiftMetadataTestData = []parseOpenstackSwiftMetadataTestData{
	// Missing swiftURL
	{metadata: map[string]string{"containerName": "my-container", "objectCount": "5"}},
	// Missing containerName
	{metadata: map[string]string{"swiftURL": "http://localhost:8080/v1/my-account-id", "objectCount": "5"}},
	// objectCount is not an integer value
	{metadata: map[string]string{"containerName": "my-container", "swiftURL": "http://localhost:8080/v1/my-account-id", "objectCount": "5.5"}},
	// timeout is not an integer value
	{metadata: map[string]string{"containerName": "my-container", "swiftURL": "http://localhost:8080/v1/my-account-id", "objectCount": "5", "timeout": "2.5"}},
	// onlyFiles is not a boolean value
	{metadata: map[string]string{"containerName": "my-container", "swiftURL": "http://localhost:8080/v1/my-account-id", "objectCount": "5", "onlyFiles": "yes"}},
}

var invalidOpenstackSwiftAuthMetadataTestData = []parseOpenstackSwiftAuthMetadataTestData{
	// Using Password method:

	// Missing userID
	{authMetadata: map[string]string{"password": "my-password", "projectID": "my-project-id", "authURL": "http://localhost:5000/v3/"}},
	// Missing password
	{authMetadata: map[string]string{"userID": "my-id", "projectID": "my-project-id", "authURL": "http://localhost:5000/v3/"}},
	// Missing projectID
	{authMetadata: map[string]string{"userID": "my-id", "password": "my-password", "authURL": "http://localhost:5000/v3/"}},
	// Missing authURL
	{authMetadata: map[string]string{"userID": "my-id", "password": "my-password", "projectID": "my-project-id"}},

	// Using Application Credentials method:

	// Missing appCredentialID
	{authMetadata: map[string]string{"appCredentialSecret": "my-app-credential-secret", "authURL": "http://localhost:5000/v3/"}},
	// Missing appCredentialSecret
	{authMetadata: map[string]string{"appCredentialID": "my-app-credential-id", "authURL": "http://localhost:5000/v3/"}},
	// Missing authURL
	{authMetadata: map[string]string{"appCredentialID": "my-app-credential-id", "appCredentialSecret": "my-app-credential-secret"}},
}

func TestOpenstackSwiftGetMetricSpecForScaling(t *testing.T) {
	testCases := []openstackSwiftMetricIdentifier{
		{nil, &openstackSwiftMetadataTestData[0], &openstackSwiftAuthMetadataTestData[0], "openstack-swift-my-container"},
		{nil, &openstackSwiftMetadataTestData[1], &openstackSwiftAuthMetadataTestData[0], "openstack-swift-my-container"},
		{nil, &openstackSwiftMetadataTestData[2], &openstackSwiftAuthMetadataTestData[0], "openstack-swift-my-container-my-prefix"},
		{nil, &openstackSwiftMetadataTestData[3], &openstackSwiftAuthMetadataTestData[0], "openstack-swift-my-container"},
		{nil, &openstackSwiftMetadataTestData[4], &openstackSwiftAuthMetadataTestData[0], "openstack-swift-my-container"},
		{nil, &openstackSwiftMetadataTestData[5], &openstackSwiftAuthMetadataTestData[0], "openstack-swift-my-container"},
		{nil, &openstackSwiftMetadataTestData[6], &openstackSwiftAuthMetadataTestData[0], "openstack-swift-my-container"},

		{nil, &openstackSwiftMetadataTestData[0], &openstackSwiftAuthMetadataTestData[1], "openstack-swift-my-container"},
		{nil, &openstackSwiftMetadataTestData[1], &openstackSwiftAuthMetadataTestData[1], "openstack-swift-my-container"},
		{nil, &openstackSwiftMetadataTestData[2], &openstackSwiftAuthMetadataTestData[1], "openstack-swift-my-container-my-prefix"},
		{nil, &openstackSwiftMetadataTestData[3], &openstackSwiftAuthMetadataTestData[1], "openstack-swift-my-container"},
		{nil, &openstackSwiftMetadataTestData[4], &openstackSwiftAuthMetadataTestData[1], "openstack-swift-my-container"},
		{nil, &openstackSwiftMetadataTestData[5], &openstackSwiftAuthMetadataTestData[1], "openstack-swift-my-container"},
		{nil, &openstackSwiftMetadataTestData[6], &openstackSwiftAuthMetadataTestData[1], "openstack-swift-my-container"},
	}

	for _, testData := range testCases {
		testData := testData
		meta, err := parseOpenstackSwiftMetadata(&ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		_, err = parseOpenstackSwiftAuthenticationMetadata(&ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata})
		if err != nil {
			t.Fatal("Could not parse auth metadata:", err)
		}

		mockSwiftScaler := openstackSwiftScaler{meta, nil}

		metricSpec := mockSwiftScaler.GetMetricSpecForScaling()

		metricName := metricSpec[0].External.Metric.Name

		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestParseOpenstackSwiftMetadataForInvalidCases(t *testing.T) {
	testCases := []openstackSwiftMetricIdentifier{
		{nil, &invalidOpenstackSwiftMetadataTestData[0], &parseOpenstackSwiftAuthMetadataTestData{}, "missing swiftURL"},
		{nil, &invalidOpenstackSwiftMetadataTestData[1], &parseOpenstackSwiftAuthMetadataTestData{}, "missing containerName"},
		{nil, &invalidOpenstackSwiftMetadataTestData[2], &parseOpenstackSwiftAuthMetadataTestData{}, "objectCount is not an integer value"},
		{nil, &invalidOpenstackSwiftMetadataTestData[3], &parseOpenstackSwiftAuthMetadataTestData{}, "onlyFiles is not a boolean value"},
		{nil, &invalidOpenstackSwiftMetadataTestData[4], &parseOpenstackSwiftAuthMetadataTestData{}, "timeout is not an integer value"},
	}

	for _, testData := range testCases {
		testData := testData
		t.Run(testData.name, func(pt *testing.T) {
			_, err := parseOpenstackSwiftMetadata(&ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata})
			assert.NotNil(t, err)
		})
	}
}

func TestParseOpenstackSwiftAuthenticationMetadataForInvalidCases(t *testing.T) {
	testCases := []openstackSwiftMetricIdentifier{
		{nil, &parseOpenstackSwiftMetadataTestData{}, &invalidOpenstackSwiftAuthMetadataTestData[0], "missing userID"},
		{nil, &parseOpenstackSwiftMetadataTestData{}, &invalidOpenstackSwiftAuthMetadataTestData[1], "missing password"},
		{nil, &parseOpenstackSwiftMetadataTestData{}, &invalidOpenstackSwiftAuthMetadataTestData[2], "missing projectID"},
		{nil, &parseOpenstackSwiftMetadataTestData{}, &invalidOpenstackSwiftAuthMetadataTestData[3], "missing authURL for password method"},
		{nil, &parseOpenstackSwiftMetadataTestData{}, &invalidOpenstackSwiftAuthMetadataTestData[4], "missing appCredentialID"},
		{nil, &parseOpenstackSwiftMetadataTestData{}, &invalidOpenstackSwiftAuthMetadataTestData[5], "missing appCredentialSecret"},
		{nil, &parseOpenstackSwiftMetadataTestData{}, &invalidOpenstackSwiftAuthMetadataTestData[6], "missing authURL for application credentials method"},
	}

	for _, testData := range testCases {
		testData := testData
		t.Run(testData.name, func(pt *testing.T) {
			_, err := parseOpenstackSwiftAuthenticationMetadata(&ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata})
			assert.NotNil(t, err)
		})
	}
}
