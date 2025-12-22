package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud"
	"github.com/stretchr/testify/assert"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
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
	scaledIndex          int
	name                 string
}

var openstackSwiftMetadataTestData = []parseOpenstackSwiftMetadataTestData{
	// Only required parameters
	{metadata: map[string]string{"containerName": "my-container"}},
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
	// Missing containerName
	{metadata: map[string]string{"swiftURL": "http://localhost:8080/v1/my-account-id", "objectCount": "5"}},
	// objectCount is not an integer value
	{metadata: map[string]string{"containerName": "my-container", "swiftURL": "http://localhost:8080/v1/my-account-id", "objectCount": "5.5"}},
	// activationObjectCount is not an integer value
	{metadata: map[string]string{"containerName": "my-container", "swiftURL": "http://localhost:8080/v1/my-account-id", "objectCount": "5", "activationObjectCount": "5.5"}},
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
		{nil, &openstackSwiftMetadataTestData[0], &openstackSwiftAuthMetadataTestData[0], 0, "s0-openstack-swift-my-container"},
		{nil, &openstackSwiftMetadataTestData[1], &openstackSwiftAuthMetadataTestData[0], 1, "s1-openstack-swift-my-container"},
		{nil, &openstackSwiftMetadataTestData[2], &openstackSwiftAuthMetadataTestData[0], 2, "s2-openstack-swift-my-container-my-prefix"},
		{nil, &openstackSwiftMetadataTestData[3], &openstackSwiftAuthMetadataTestData[0], 3, "s3-openstack-swift-my-container"},
		{nil, &openstackSwiftMetadataTestData[4], &openstackSwiftAuthMetadataTestData[0], 4, "s4-openstack-swift-my-container"},
		{nil, &openstackSwiftMetadataTestData[5], &openstackSwiftAuthMetadataTestData[0], 5, "s5-openstack-swift-my-container"},
		{nil, &openstackSwiftMetadataTestData[6], &openstackSwiftAuthMetadataTestData[0], 6, "s6-openstack-swift-my-container"},

		{nil, &openstackSwiftMetadataTestData[0], &openstackSwiftAuthMetadataTestData[1], 0, "s0-openstack-swift-my-container"},
		{nil, &openstackSwiftMetadataTestData[1], &openstackSwiftAuthMetadataTestData[1], 1, "s1-openstack-swift-my-container"},
		{nil, &openstackSwiftMetadataTestData[2], &openstackSwiftAuthMetadataTestData[1], 2, "s2-openstack-swift-my-container-my-prefix"},
		{nil, &openstackSwiftMetadataTestData[3], &openstackSwiftAuthMetadataTestData[1], 3, "s3-openstack-swift-my-container"},
		{nil, &openstackSwiftMetadataTestData[4], &openstackSwiftAuthMetadataTestData[1], 4, "s4-openstack-swift-my-container"},
		{nil, &openstackSwiftMetadataTestData[5], &openstackSwiftAuthMetadataTestData[1], 5, "s5-openstack-swift-my-container"},
		{nil, &openstackSwiftMetadataTestData[6], &openstackSwiftAuthMetadataTestData[1], 6, "s6-openstack-swift-my-container"},
	}

	for _, testData := range testCases {
		meta, err := parseOpenstackSwiftMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata, TriggerIndex: testData.scaledIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}

		mockSwiftScaler := openstackSwiftScaler{"", meta, &gophercloud.ServiceClient{}, logr.Discard()}

		metricSpec := mockSwiftScaler.GetMetricSpecForScaling(context.Background())

		metricName := metricSpec[0].External.Metric.Name

		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestParseOpenstackSwiftMetadataForInvalidCases(t *testing.T) {
	testCases := []openstackSwiftMetricIdentifier{
		{nil, &invalidOpenstackSwiftMetadataTestData[0], &parseOpenstackSwiftAuthMetadataTestData{}, 0, "s0-missing containerName"},
		{nil, &invalidOpenstackSwiftMetadataTestData[1], &parseOpenstackSwiftAuthMetadataTestData{}, 1, "s1-objectCount is not an integer value"},
		{nil, &invalidOpenstackSwiftMetadataTestData[2], &parseOpenstackSwiftAuthMetadataTestData{}, 2, "s2-onlyFiles is not a boolean value"},
		{nil, &invalidOpenstackSwiftMetadataTestData[3], &parseOpenstackSwiftAuthMetadataTestData{}, 3, "s3-timeout is not an integer value"},
	}

	for _, testData := range testCases {
		t.Run(testData.name, func(pt *testing.T) {
			_, err := parseOpenstackSwiftMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata, TriggerIndex: testData.scaledIndex})
			assert.NotNil(t, err)
		})
	}
}

func TestParseOpenstackSwiftAuthenticationMetadataForInvalidCases(t *testing.T) {
	testCases := []openstackSwiftMetricIdentifier{
		{nil, &parseOpenstackSwiftMetadataTestData{}, &invalidOpenstackSwiftAuthMetadataTestData[0], 0, "s0-missing userID"},
		{nil, &parseOpenstackSwiftMetadataTestData{}, &invalidOpenstackSwiftAuthMetadataTestData[1], 1, "s1-missing password"},
		{nil, &parseOpenstackSwiftMetadataTestData{}, &invalidOpenstackSwiftAuthMetadataTestData[2], 2, "s2-missing projectID"},
		{nil, &parseOpenstackSwiftMetadataTestData{}, &invalidOpenstackSwiftAuthMetadataTestData[3], 3, "s3-missing authURL for password method"},
		{nil, &parseOpenstackSwiftMetadataTestData{}, &invalidOpenstackSwiftAuthMetadataTestData[4], 4, "s4-missing appCredentialID"},
		{nil, &parseOpenstackSwiftMetadataTestData{}, &invalidOpenstackSwiftAuthMetadataTestData[5], 5, "s5-missing appCredentialSecret"},
		{nil, &parseOpenstackSwiftMetadataTestData{}, &invalidOpenstackSwiftAuthMetadataTestData[6], 6, "s6-missing authURL for application credentials method"},
	}

	for _, testData := range testCases {
		t.Run(testData.name, func(pt *testing.T) {
			_, err := parseOpenstackSwiftMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata, TriggerIndex: testData.scaledIndex})
			assert.NotNil(t, err)
		})
	}
}
