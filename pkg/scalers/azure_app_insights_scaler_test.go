package scalers

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/go-logr/logr"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

type azureAppInsightsScalerTestData struct {
	name    string
	isError bool
	config  ScalerConfig
}

var azureAppInsightsScalerData = []azureAppInsightsScalerTestData{
	{name: "no target value", isError: true, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"targetValue": "", "applicationInsightsId": "1234", "metricId": "unittest/test", "metricAggregationTimespan": "01:02", "metricAggregationType": "max", "metricFilter": "", "tenantId": "1234",
		},
		AuthParams: map[string]string{
			"activeDirectoryClientId": "5678", "activeDirectoryClientPassword": "pw",
		},
	}},
	{name: "target value not a number", isError: true, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"targetValue": "a1", "applicationInsightsId": "1234", "metricId": "unittest/test", "metricAggregationTimespan": "01:02", "metricAggregationType": "max", "metricFilter": "", "tenantId": "1234",
		},
		AuthParams: map[string]string{
			"activeDirectoryClientId": "5678", "activeDirectoryClientPassword": "pw",
		},
	}},
	{name: "activation target value not a number", isError: true, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"targetValue": "1", "activationTargetValue": "a1", "applicationInsightsId": "1234", "metricId": "unittest/test", "metricAggregationTimespan": "01:02", "metricAggregationType": "max", "metricFilter": "", "tenantId": "1234",
		},
		AuthParams: map[string]string{
			"activeDirectoryClientId": "5678", "activeDirectoryClientPassword": "pw",
		},
	}},
	{name: "empty app insights id", isError: true, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"targetValue": "11", "applicationInsightsId": "", "metricId": "unittest/test", "metricAggregationTimespan": "01:02", "metricAggregationType": "max", "metricFilter": "", "tenantId": "1234",
		},
		AuthParams: map[string]string{
			"activeDirectoryClientId": "5678", "activeDirectoryClientPassword": "pw",
		},
	}},
	{name: "empty metric id", isError: true, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"targetValue": "11", "applicationInsightsId": "1234", "metricId": "", "metricAggregationTimespan": "01:02", "metricAggregationType": "max", "metricFilter": "", "tenantId": "1234",
		},
		AuthParams: map[string]string{
			"activeDirectoryClientId": "5678", "activeDirectoryClientPassword": "pw",
		},
	}},
	{name: "empty timespan", isError: true, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"targetValue": "11", "applicationInsightsId": "1234", "metricId": "unittest/test", "metricAggregationTimespan": "", "metricAggregationType": "max", "metricFilter": "", "tenantId": "1234",
		},
		AuthParams: map[string]string{
			"activeDirectoryClientId": "5678", "activeDirectoryClientPassword": "pw",
		},
	}},
	{name: "invalid timespan", isError: true, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"targetValue": "11", "applicationInsightsId": "1234", "metricId": "unittest/test", "metricAggregationTimespan": "01:02:03", "metricAggregationType": "max", "metricFilter": "", "tenantId": "1234",
		},
		AuthParams: map[string]string{
			"activeDirectoryClientId": "5678", "activeDirectoryClientPassword": "pw",
		},
	}},
	{name: "empty aggregation type", isError: true, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"targetValue": "11", "applicationInsightsId": "1234", "metricId": "unittest/test", "metricAggregationTimespan": "01:02", "metricAggregationType": "", "metricFilter": "", "tenantId": "1234",
		},
		AuthParams: map[string]string{
			"activeDirectoryClientId": "5678", "activeDirectoryClientPassword": "pw",
		},
	}},
	{name: "empty tenant id", isError: true, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"targetValue": "11", "applicationInsightsId": "1234", "metricId": "unittest/test", "metricAggregationTimespan": "01:02", "metricAggregationType": "max", "metricFilter": "", "tenantId": "",
		},
		AuthParams: map[string]string{
			"activeDirectoryClientId": "5678", "activeDirectoryClientPassword": "pw",
		},
	}},
	{name: "invalid identity", isError: true, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"targetValue": "11", "applicationInsightsId": "1234", "metricId": "unittest/test", "metricAggregationTimespan": "01:02", "metricAggregationType": "max", "metricFilter": "", "tenantId": "1234",
		},
		AuthParams: map[string]string{
			"activeDirectoryClientId": "", "activeDirectoryClientPassword": "pw",
		},
	}},
	{name: "filter empty", isError: false, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"targetValue": "11", "applicationInsightsId": "1234", "metricId": "unittest/test", "metricAggregationTimespan": "01:02", "metricAggregationType": "max", "metricFilter": "", "tenantId": "1234",
		},
		AuthParams: map[string]string{
			"activeDirectoryClientId": "5678", "activeDirectoryClientPassword": "pw",
		},
	}},
	{name: "filter given", isError: false, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"targetValue": "11", "applicationInsightsId": "1234", "metricId": "unittest/test", "metricAggregationTimespan": "01:02", "metricAggregationType": "max", "metricFilter": "cloud/roleName eq 'test'", "tenantId": "1234",
		},
		AuthParams: map[string]string{
			"activeDirectoryClientId": "5678", "activeDirectoryClientPassword": "pw",
		},
	}},
	{name: "correct pod identity", isError: false, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"targetValue": "11", "applicationInsightsId": "1234", "metricId": "unittest/test", "metricAggregationTimespan": "01:02", "metricAggregationType": "max", "metricFilter": "cloud/roleName eq 'test'", "tenantId": "1234",
		},
		PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzure},
	}},
	{name: "invalid pod Identity", isError: true, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"targetValue": "11", "applicationInsightsId": "1234", "metricId": "unittest/test", "metricAggregationTimespan": "01:02", "metricAggregationType": "max", "metricFilter": "cloud/roleName eq 'test'", "tenantId": "1234",
		},
		PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProvider("notAzure")},
	}},
	{name: "correct workload identity", isError: false, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"targetValue": "11", "applicationInsightsId": "1234", "metricId": "unittest/test", "metricAggregationTimespan": "01:02", "metricAggregationType": "max", "metricFilter": "cloud/roleName eq 'test'", "tenantId": "1234",
		},
		PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzureWorkload},
	}},
	{name: "invalid workload Identity", isError: true, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"targetValue": "11", "applicationInsightsId": "1234", "metricId": "unittest/test", "metricAggregationTimespan": "01:02", "metricAggregationType": "max", "metricFilter": "cloud/roleName eq 'test'", "tenantId": "1234",
		},
		PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProvider("notAzureWorkload")},
	}},
	{name: "app insights id in auth", isError: false, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"targetValue": "11", "metricId": "unittest/test", "metricAggregationTimespan": "01:02", "metricAggregationType": "max", "metricFilter": "cloud/roleName eq 'test'", "tenantId": "1234",
		},
		AuthParams: map[string]string{
			"activeDirectoryClientId": "5678", "activeDirectoryClientPassword": "pw", "applicationInsightsId": "1234",
		},
	}},
	{name: "tenant id in auth", isError: false, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"targetValue": "11", "applicationInsightsId": "1234", "metricId": "unittest/test", "metricAggregationTimespan": "01:02", "metricAggregationType": "max", "metricFilter": "cloud/roleName eq 'test'",
		},
		AuthParams: map[string]string{
			"activeDirectoryClientId": "5678", "activeDirectoryClientPassword": "pw", "tenantId": "1234",
		},
	}},
	{name: "from env", isError: false, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"targetValue": "11", "metricId": "unittest/test", "metricAggregationTimespan": "01:02", "metricAggregationType": "max", "metricFilter": "cloud/roleName eq 'test'",
			"activeDirectoryClientIdFromEnv": "AD_CLIENT_ID", "activeDirectoryClientPasswordFromEnv": "AD_CLIENT_PASSWORD", "applicationInsightsIdFromEnv": "APP_INSIGHTS_ID", "tenantIdFromEnv": "TENANT_ID",
		},
		AuthParams: map[string]string{},
		ResolvedEnv: map[string]string{
			"AD_CLIENT_ID": "5678", "AD_CLIENT_PASSWORD": "pw", "APP_INSIGHTS_ID": "1234", "TENANT_ID": "1234",
		},
	}},
	{name: "from env - missing environment variable", isError: true, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"targetValue": "11", "metricId": "unittest/test", "metricAggregationTimespan": "01:02", "metricAggregationType": "max", "metricFilter": "cloud/roleName eq 'test'",
			"activeDirectoryClientIdFromEnv": "MISSING_AD_CLIENT_ID", "activeDirectoryClientPasswordFromEnv": "AD_CLIENT_PASSWORD", "applicationInsightsIdFromEnv": "APP_INSIGHTS_ID", "tenantIdFromEnv": "TENANT_ID",
		},
		AuthParams: map[string]string{},
		ResolvedEnv: map[string]string{
			"AD_CLIENT_ID": "5678", "AD_CLIENT_PASSWORD": "pw", "APP_INSIGHTS_ID": "1234", "TENANT_ID": "1234",
		},
	}},
	{name: "known Azure Cloud", isError: false, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"metricAggregationTimespan": "00:01", "metricAggregationType": "count", "metricId": "unittest/test", "targetValue": "10",
			"applicationInsightsId": "appinsightid", "tenantId": "tenantid",
			"cloud": "azureChinaCloud",
		},
		AuthParams: map[string]string{
			"tenantId": "tenantId", "activeDirectoryClientId": "adClientId", "activeDirectoryClientPassword": "adClientPassword",
		},
	}},
	{name: "private cloud", isError: false, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"metricAggregationTimespan": "00:01", "metricAggregationType": "count", "metricId": "unittest/test", "targetValue": "10",
			"applicationInsightsId": "appinsightid", "tenantId": "tenantid",
			"cloud": "private", "appInsightsResourceURL": "appInsightsResourceURL", "activeDirectoryEndpoint": "adEndpoint",
		},
		AuthParams: map[string]string{
			"tenantId": "tenantId", "activeDirectoryClientId": "adClientId", "activeDirectoryClientPassword": "adClientPassword",
		},
	}},
	{name: "private cloud - missing app insights resource URL", isError: true, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"metricAggregationTimespan": "00:01", "metricAggregationType": "count", "metricId": "unittest/test", "targetValue": "10",
			"applicationInsightsId": "appinsightid", "tenantId": "tenantid",
			"cloud": "private", "activeDirectoryEndpoint": "adEndpoint",
		},
		AuthParams: map[string]string{
			"tenantId": "tenantId", "activeDirectoryClientId": "adClientId", "activeDirectoryClientPassword": "adClientPassword",
		},
	}},
	{name: "private cloud - missing active directory endpoint", isError: true, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"metricAggregationTimespan": "00:01", "metricAggregationType": "count", "metricId": "unittest/test", "targetValue": "10",
			"applicationInsightsId": "appinsightid", "tenantId": "tenantid",
			"cloud": "private", "appInsightsResourceURL": "appInsightsResourceURL",
		},
		AuthParams: map[string]string{
			"tenantId": "tenantId", "activeDirectoryClientId": "adClientId", "activeDirectoryClientPassword": "adClientPassword",
		},
	}},
	{name: "unsupported cloud", isError: true, config: ScalerConfig{
		TriggerMetadata: map[string]string{
			"metricAggregationTimespan": "00:01", "metricAggregationType": "count", "metricId": "unittest/test", "targetValue": "10",
			"applicationInsightsId": "appinsightid", "tenantId": "tenantid",
			"cloud": "azureGermanCloud",
		},
		AuthParams: map[string]string{
			"tenantId": "tenantId", "activeDirectoryClientId": "adClientId", "activeDirectoryClientPassword": "adClientPassword",
		},
	}},
}

func TestNewAzureAppInsightsScaler(t *testing.T) {
	for _, testData := range azureAppInsightsScalerData {
		_, err := NewAzureAppInsightsScaler(&testData.config)
		if err != nil && !testData.isError {
			t.Error(fmt.Sprintf("test %s: expected success but got error", testData.name), err)
		}
		if testData.isError && err == nil {
			t.Errorf("test %s: expected error but got success. testData: %v", testData.name, testData)
		}
	}
}

func TestAzureAppInsightsGetMetricSpecForScaling(t *testing.T) {
	scalerIndex := 0
	for _, testData := range azureAppInsightsScalerData {
		ctx := context.Background()
		if !testData.isError {
			testData.config.ScalerIndex = scalerIndex
			meta, err := parseAzureAppInsightsMetadata(&testData.config, logr.Discard())
			if err != nil {
				t.Fatal("Could not parse metadata:", err)
			}
			mockAzureAppInsightsScaler := azureAppInsightsScaler{
				metadata:    meta,
				podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzure},
			}

			metricSpec := mockAzureAppInsightsScaler.GetMetricSpecForScaling(ctx)
			metricName := metricSpec[0].External.Metric.Name
			expectedName := fmt.Sprintf("s%d-azure-app-insights-%s", scalerIndex, strings.ReplaceAll(testData.config.TriggerMetadata["metricId"], "/", "-"))
			if metricName != expectedName {
				t.Errorf("Wrong External metric name. expected: %s, actual: %s", expectedName, metricName)
			}
			scalerIndex++
		}
	}
}
