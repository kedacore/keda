package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
)

var (
	testHuaweiCloudeyeIdentityEndpoint = "none"
	testHuaweiCloudeyeProjectID        = "none"
	testHuaweiCloudeyeDomainID         = "none"
	testHuaweiCloudeyeRegion           = "none"
	testHuaweiCloudeyeDomain           = "none"
	testHuaweiCloudeyeCloud            = "none"
	testHuaweiCloudeyeAccessKey        = "none"
	testHuaweiCloudeyeSecretKey        = "none"
)

type parseHuaweiCloudeyeMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
	comment    string
}

type huaweiCloudeyeMetricIdentifier struct {
	metadataTestData *parseHuaweiCloudeyeMetadataTestData
	scalerIndex      int
	name             string
}

var testHuaweiAuthenticationWithCloud = map[string]string{
	"IdentityEndpoint": testHuaweiCloudeyeIdentityEndpoint,
	"ProjectID":        testHuaweiCloudeyeProjectID,
	"DomainID":         testHuaweiCloudeyeDomainID,
	"Region":           testHuaweiCloudeyeRegion,
	"Domain":           testHuaweiCloudeyeDomain,
	"Cloud":            testHuaweiCloudeyeCloud,
	"AccessKey":        testHuaweiCloudeyeAccessKey,
	"SecretKey":        testHuaweiCloudeyeSecretKey,
}

var testHuaweiAuthenticationWithoutCloud = map[string]string{
	"IdentityEndpoint": testHuaweiCloudeyeIdentityEndpoint,
	"ProjectID":        testHuaweiCloudeyeProjectID,
	"DomainID":         testHuaweiCloudeyeDomainID,
	"Region":           testHuaweiCloudeyeRegion,
	"Domain":           testHuaweiCloudeyeDomain,
	"AccessKey":        testHuaweiCloudeyeAccessKey,
	"SecretKey":        testHuaweiCloudeyeSecretKey,
}

var testHuaweiCloudeyeMetadata = []parseHuaweiCloudeyeMetadataTestData{
	{map[string]string{
		"namespace":         "SYS.ELB",
		"dimensionName":     "lbaas_instance_id",
		"dimensionValue":    "5e052238-0346-xxb0-86ea-92d9f33e29d2",
		"metricName":        "mb_l7_qps",
		"targetMetricValue": "100",
		"minMetricValue":    "1"},
		testHuaweiAuthenticationWithCloud,
		false,
		"auth parameter with Cloud"},
	{map[string]string{
		"namespace":         "SYS.ELB",
		"dimensionName":     "lbaas_instance_id",
		"dimensionValue":    "5e052238-0346-xxb0-86ea-92d9f33e29d2",
		"metricName":        "mb_l7_qps",
		"targetMetricValue": "100",
		"minMetricValue":    "1"},
		testHuaweiAuthenticationWithoutCloud,
		false,
		"auth parameter without Cloud"},
	{map[string]string{
		"namespace":            "SYS.ELB",
		"dimensionName":        "lbaas_instance_id",
		"dimensionValue":       "5e052238-0346-xxb0-86ea-92d9f33e29d2",
		"metricName":           "mb_l7_qps",
		"targetMetricValue":    "100",
		"minMetricValue":       "1",
		"metricCollectionTime": "300",
		"metricFilter":         "average",
		"metricPeriod":         "300"},
		testHuaweiAuthenticationWithCloud,
		false,
		"all parameter"},
	{map[string]string{}, testHuaweiAuthenticationWithCloud, true, "Empty structures"},
	{map[string]string{
		"dimensionName":     "lbaas_instance_id",
		"dimensionValue":    "5e052238-0346-xxb0-86ea-92d9f33e29d2",
		"metricName":        "mb_l7_qps",
		"targetMetricValue": "100",
		"minMetricValue":    "1"},
		testHuaweiAuthenticationWithCloud,
		true,
		"metadata miss namespace"},
	{map[string]string{
		"namespace":         "SYS.ELB",
		"dimensionValue":    "5e052238-0346-xxb0-86ea-92d9f33e29d2",
		"metricName":        "mb_l7_qps",
		"targetMetricValue": "100",
		"minMetricValue":    "1"},
		testHuaweiAuthenticationWithCloud,
		true,
		"metadata miss dimensionName"},
	{map[string]string{
		"namespace":         "SYS.ELB",
		"dimensionName":     "lbaas_instance_id",
		"metricName":        "mb_l7_qps",
		"targetMetricValue": "100",
		"minMetricValue":    "1"},
		testHuaweiAuthenticationWithCloud,
		true,
		"metadata miss dimensionValue"},
	{map[string]string{
		"namespace":         "SYS.ELB",
		"dimensionName":     "lbaas_instance_id",
		"dimensionValue":    "5e052238-0346-xxb0-86ea-92d9f33e29d2",
		"targetMetricValue": "100",
		"minMetricValue":    "1"},
		testHuaweiAuthenticationWithCloud,
		true,
		"metadata miss metricName"},
	{map[string]string{
		"namespace":      "SYS.ELB",
		"dimensionName":  "lbaas_instance_id",
		"dimensionValue": "5e052238-0346-xxb0-86ea-92d9f33e29d2",
		"metricName":     "mb_l7_qps",
		"minMetricValue": "1"},
		testHuaweiAuthenticationWithCloud,
		true,
		"metadata miss targetMetricValue"},
	{map[string]string{
		"namespace":         "SYS.ELB",
		"dimensionName":     "lbaas_instance_id",
		"dimensionValue":    "5e052238-0346-xxb0-86ea-92d9f33e29d2",
		"metricName":        "mb_l7_qps",
		"targetMetricValue": "100"},
		testHuaweiAuthenticationWithCloud,
		true,
		"metadata miss minMetricValue"},
	{map[string]string{
		"namespace":                   "SYS.ELB",
		"dimensionName":               "lbaas_instance_id",
		"dimensionValue":              "5e052238-0346-xxb0-86ea-92d9f33e29d2",
		"metricName":                  "mb_l7_qps",
		"targetMetricValue":           "100",
		"activationTargetMetricValue": "aaaaa"},
		testHuaweiAuthenticationWithCloud,
		true,
		"invalid activationTargetMetricValue"},
}

var huaweiCloudeyeMetricIdentifiers = []huaweiCloudeyeMetricIdentifier{
	{&testHuaweiCloudeyeMetadata[0], 0, "s0-huawei-cloudeye-mb_l7_qps"},
	{&testHuaweiCloudeyeMetadata[0], 1, "s1-huawei-cloudeye-mb_l7_qps"},
}

func TestHuaweiCloudeyeParseMetadata(t *testing.T) {
	for _, testData := range testHuaweiCloudeyeMetadata {
		_, err := parseHuaweiCloudeyeMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams}, logr.Discard())
		if err != nil && !testData.isError {
			t.Errorf("%s: Expected success but got error %s", testData.comment, err)
		}
		if testData.isError && err == nil {
			t.Errorf("%s: Expected error but got success", testData.comment)
		}
	}
}

func TestHuaweiCloudeyeGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range huaweiCloudeyeMetricIdentifiers {
		meta, err := parseHuaweiCloudeyeMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.metadataTestData.authParams, ScalerIndex: testData.scalerIndex}, logr.Discard())
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockHuaweiCloudeyeScaler := huaweiCloudeyeScaler{"", meta, logr.Discard()}

		metricSpec := mockHuaweiCloudeyeScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName, "wanted:", testData.name)
		}
	}
}
