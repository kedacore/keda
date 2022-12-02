package scalers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const loadCount = 1000 // the size of the pretend pool completed of job requests

type parseAzurePipelinesMetadataTestData struct {
	testName    string
	metadata    map[string]string
	isError     bool
	resolvedEnv map[string]string
	authParams  map[string]string
}

var testAzurePipelinesResolvedEnv = map[string]string{
	"AZP_URL":   "https://dev.azure.com/sample",
	"AZP_TOKEN": "sample",
}

var testAzurePipelinesMetadata = []parseAzurePipelinesMetadataTestData{
	// empty
	{"empty", map[string]string{}, true, testAzurePipelinesResolvedEnv, map[string]string{}},
	// all properly formed
	{"all properly formed", map[string]string{"organizationURLFromEnv": "AZP_URL", "personalAccessTokenFromEnv": "AZP_TOKEN", "poolID": "1", "targetPipelinesQueueLength": "1"}, false, testAzurePipelinesResolvedEnv, map[string]string{}},
	// using triggerAuthentication
	{"using triggerAuthentication", map[string]string{"poolID": "1", "targetPipelinesQueueLength": "1"}, false, testAzurePipelinesResolvedEnv, map[string]string{"organizationURL": "https://dev.azure.com/sample", "personalAccessToken": "sample"}},
	// using triggerAuthentication with personalAccessToken terminating in newline
	{"using triggerAuthentication with personalAccessToken terminating in newline", map[string]string{"poolID": "1", "targetPipelinesQueueLength": "1"}, false, testAzurePipelinesResolvedEnv, map[string]string{"organizationURL": "https://dev.azure.com/sample", "personalAccessToken": "sample\n"}},
	// missing organizationURL
	{"missing organizationURL", map[string]string{"organizationURLFromEnv": "", "personalAccessTokenFromEnv": "sample", "poolID": "1", "targetPipelinesQueueLength": "1"}, true, testAzurePipelinesResolvedEnv, map[string]string{}},
	// missing personalAccessToken
	{"missing personalAccessToken", map[string]string{"organizationURLFromEnv": "AZP_URL", "poolID": "1", "targetPipelinesQueueLength": "1"}, true, testAzurePipelinesResolvedEnv, map[string]string{}},
	// missing poolID
	{"missing poolID", map[string]string{"organizationURLFromEnv": "AZP_URL", "personalAccessTokenFromEnv": "AZP_TOKEN", "poolID": "", "targetPipelinesQueueLength": "1"}, true, testAzurePipelinesResolvedEnv, map[string]string{}},
	// activationTargetPipelinesQueueLength malformed
	{"all properly formed", map[string]string{"organizationURLFromEnv": "AZP_URL", "personalAccessTokenFromEnv": "AZP_TOKEN", "poolID": "1", "targetPipelinesQueueLength": "1", "activationTargetPipelinesQueueLength": "A"}, true, testAzurePipelinesResolvedEnv, map[string]string{}},
}

var testJobRequestResponse = `{"count":2,"value":[{"requestId":890659,"queueTime":"2022-09-28T11:19:49.89Z","assignTime":"2022-09-28T11:20:29.5033333Z","receiveTime":"2022-09-28T11:20:32.0530499Z","lockedUntil":"2022-09-28T11:30:32.07Z","serviceOwner":"xxx","hostId":"xxx","scopeId":"xxx","planType":"Build","planId":"xxx","jobId":"xxx","demands":["kubectl","Agent.Version -gtVersion 2.182.1"],"reservedAgent":{"_links":{"self":{"href":"https://dev.azure.com/FOO/_apis/distributedtask/pools/44/agents/11735"},"web":{"href":"https://dev.azure.com/FOO/_settings/agentpools?view=jobs&poolId=44&agentId=11735"}},"id":11735,"name":"kube-scaledjob-5nlph-kzpgf","version":"2.210.1","osDescription":"Linux 5.4.0-1089-azure #94~18.04.1-Ubuntu SMP Fri Aug 5 12:34:50 UTC 2022","enabled":true,"status":"online","provisioningState":"Provisioned","accessPoint":"CodexAccessMapping"},"definition":{"_links":{"web":{"href":"https://dev.azure.com/FOO/1858395a-257e-4efd-bbc5-eb618128452b/_build/definition?definitionId=4869"},"self":{"href":"https://dev.azure.com/FOO/1858395a-257e-4efd-bbc5-eb618128452b/_apis/build/Definitions/4869"}},"id":4869,"name":"base - main"},"owner":{"_links":{"web":{"href":"https://dev.azure.com/FOO/1858395a-257e-4efd-bbc5-eb618128452b/_build/results?buildId=673584"},"self":{"href":"https://dev.azure.com/FOO/1858395a-257e-4efd-bbc5-eb618128452b/_apis/build/Builds/673584"}},"id":673584,"name":"20220928.2"},"data":{"ParallelismTag":"Private","IsScheduledKey":"False"},"poolId":44,"orchestrationId":"5c5c8ec9-786f-4e97-99d4-a29279befba3.build.__default","priority":0},{"requestId":890663,"queueTime":"2022-09-28T11:20:22.4633333Z","serviceOwner":"00025394-6065-48ca-87d9-7f5672854ef7","hostId":"41a18c7d-df5e-4032-a4df-d533b56bd2de","scopeId":"02696e26-a35b-424c-86b8-1f54e1b0b4b7","planType":"Build","planId":"b718cfed-493c-46be-a650-88fe762f75aa","jobId":"15b95994-59ec-5502-695d-0b93722883bd","demands":["dotnet60","java","Agent.Version -gtVersion 2.182.1"],"matchedAgents":[{"_links":{"self":{"href":"https://dev.azure.com/FOO/_apis/distributedtask/pools/44/agents/1755"},"web":{"href":"https://dev.azure.com/FOO/_settings/agentpools?view=jobs&poolId=44&agentId=1755"}},"id":1755,"name":"dotnet60-keda-template","version":"2.210.1","enabled":true,"status":"offline","provisioningState":"Provisioned"},{"_links":{"self":{"href":"https://dev.azure.com/FOO/_apis/distributedtask/pools/44/agents/11732"},"web":{"href":"https://dev.azure.com/FOO/_settings/agentpools?view=jobs&poolId=44&agentId=11732"}},"id":11732,"name":"dotnet60-scaledjob-5dsgc-pkqvm","version":"2.210.1","enabled":true,"status":"online","provisioningState":"Provisioned"},{"_links":{"self":{"href":"https://dev.azure.com/FOO/_apis/distributedtask/pools/44/agents/11733"},"web":{"href":"https://dev.azure.com/FOO/_settings/agentpools?view=jobs&poolId=44&agentId=11733"}},"id":11733,"name":"dotnet60-scaledjob-zgqnp-8h4z4","version":"2.210.1","enabled":true,"status":"online","provisioningState":"Provisioned"},{"_links":{"self":{"href":"https://dev.azure.com/FOO/_apis/distributedtask/pools/44/agents/11734"},"web":{"href":"https://dev.azure.com/FOO/_settings/agentpools?view=jobs&poolId=44&agentId=11734"}},"id":11734,"name":"dotnet60-scaledjob-wr65c-ff2cv","version":"2.210.1","enabled":true,"status":"online","provisioningState":"Provisioned"}],"definition":{"_links":{"web":{"href":"https://FOO.visualstudio.com/02696e26-a35b-424c-86b8-1f54e1b0b4b7/_build/definition?definitionId=3129"},"self":{"href":"https://FOO.visualstudio.com/02696e26-a35b-424c-86b8-1f54e1b0b4b7/_apis/build/Definitions/3129"}},"id":3129,"name":"Other Build CI"},"owner":{"_links":{"web":{"href":"https://FOO.visualstudio.com/02696e26-a35b-424c-86b8-1f54e1b0b4b7/_build/results?buildId=673585"},"self":{"href":"https://FOO.visualstudio.com/02696e26-a35b-424c-86b8-1f54e1b0b4b7/_apis/build/Builds/673585"}},"id":673585,"name":"20220928.11"},"data":{"ParallelismTag":"Private","IsScheduledKey":"False"},"poolId":44,"orchestrationId":"b718cfed-493c-46be-a650-88fe762f75aa.buildtest.build_and_test.__default","priority":0}]}`
var deadJob = `{"requestId":890659,"result":"succeeded","queueTime":"2022-09-28T11:19:49.89Z","assignTime":"2022-09-28T11:20:29.5033333Z","receiveTime":"2022-09-28T11:20:32.0530499Z","lockedUntil":"2022-09-28T11:30:32.07Z","serviceOwner":"xxx","hostId":"xxx","scopeId":"xxx","planType":"Build","planId":"xxx","jobId":"xxx","demands":["kubectl","Agent.Version -gtVersion 2.182.1"],"reservedAgent":{"_links":{"self":{"href":"https://dev.azure.com/FOO/_apis/distributedtask/pools/44/agents/11735"},"web":{"href":"https://dev.azure.com/FOO/_settings/agentpools?view=jobs&poolId=44&agentId=11735"}},"id":11735,"name":"kube-scaledjob-5nlph-kzpgf","version":"2.210.1","osDescription":"Linux 5.4.0-1089-azure #94~18.04.1-Ubuntu SMP Fri Aug 5 12:34:50 UTC 2022","enabled":true,"status":"online","provisioningState":"Provisioned","accessPoint":"CodexAccessMapping"},"definition":{"_links":{"web":{"href":"https://dev.azure.com/FOO/1858395a-257e-4efd-bbc5-eb618128452b/_build/definition?definitionId=4869"},"self":{"href":"https://dev.azure.com/FOO/1858395a-257e-4efd-bbc5-eb618128452b/_apis/build/Definitions/4869"}},"id":4869,"name":"base - main"},"owner":{"_links":{"web":{"href":"https://dev.azure.com/FOO/1858395a-257e-4efd-bbc5-eb618128452b/_build/results?buildId=673584"},"self":{"href":"https://dev.azure.com/FOO/1858395a-257e-4efd-bbc5-eb618128452b/_apis/build/Builds/673584"}},"id":673584,"name":"20220928.2"},"data":{"ParallelismTag":"Private","IsScheduledKey":"False"},"poolId":44,"orchestrationId":"5c5c8ec9-786f-4e97-99d4-a29279befba3.build.__default","priority":0}`

func TestParseAzurePipelinesMetadata(t *testing.T) {
	for _, testData := range testAzurePipelinesMetadata {
		t.Run(testData.testName, func(t *testing.T) {
			var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				personalAccessToken := strings.Split(r.Header["Authorization"][0], " ")[1]
				if personalAccessToken != "" && personalAccessToken[len(personalAccessToken)-1:] == "\n" {
					w.WriteHeader(http.StatusUnauthorized)
				} else {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"count":1,"value":[{"id":1}]}`))
				}
			}))

			// set urls into local stub only if they are already defined
			if _, ok := testData.resolvedEnv["AZP_URL"]; ok {
				testData.resolvedEnv["AZP_URL"] = apiStub.URL
			}
			if _, ok := testData.authParams["organizationURL"]; ok {
				testData.authParams["organizationURL"] = apiStub.URL
			}

			_, err := parseAzurePipelinesMetadata(context.TODO(), &ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: testData.resolvedEnv, AuthParams: testData.authParams}, http.DefaultClient)
			if err != nil && !testData.isError {
				t.Error("Expected success but got error", err)
			}
			if testData.isError && err == nil {
				t.Error("Expected error but got success")
			}
		})
	}
}

type validateAzurePipelinesPoolTestData struct {
	testName   string
	metadata   map[string]string
	isError    bool
	queryParam string
	httpCode   int
	response   string
}

var testValidateAzurePipelinesPoolData = []validateAzurePipelinesPoolTestData{
	// poolID exists and only one is returned
	{"poolID exists and only one is returned", map[string]string{"poolID": "1"}, false, "poolID", http.StatusOK, `{"count":1,"value":[{"id":1}]}`},
	// poolID doesn't exist
	{"poolID doesn't exist", map[string]string{"poolID": "1"}, true, "poolID", http.StatusNotFound, `{}`},
	// poolName exists and only one is returned
	{"poolName exists and only one is returned", map[string]string{"poolName": "sample"}, false, "poolName", http.StatusOK, `{"count":1,"value":[{"id":1}]}`},
	// poolName exists and more than one are returned
	{"poolName exists and more than one are returned", map[string]string{"poolName": "sample"}, true, "poolName", http.StatusOK, `{"count":2,"value":[{"id":1},{"id":2}]}`},
	// poolName doesn't exist
	{"poolName doesn't exist", map[string]string{"poolName": "sample"}, true, "poolName", http.StatusOK, `{"count":0,"value":[]}`},
	// poolName is used if poolName and poolID are defined
	{"poolName is used if poolName and poolID are defined", map[string]string{"poolName": "sample", "poolID": "1"}, false, "poolName", http.StatusOK, `{"count":1,"value":[{"id":1}]}`},
}

func TestValidateAzurePipelinesPool(t *testing.T) {
	for _, testData := range testValidateAzurePipelinesPoolData {
		t.Run(testData.testName, func(t *testing.T) {
			var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, ok := r.URL.Query()[testData.queryParam]
				if !ok {
					t.Error("Worng QueryParam")
				}
				w.WriteHeader(testData.httpCode)
				_, _ = w.Write([]byte(testData.response))
			}))

			authParams := map[string]string{
				"organizationURL":     apiStub.URL,
				"personalAccessToken": "PAT",
			}

			_, err := parseAzurePipelinesMetadata(context.TODO(), &ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: nil, AuthParams: authParams}, http.DefaultClient)
			if err != nil && !testData.isError {
				t.Error("Expected success but got error", err)
			}
			if testData.isError && err == nil {
				t.Error("Expected error but got success")
			}
		})
	}
}

type azurePipelinesMetricIdentifier struct {
	scalerIndex int
	name        string
}

var azurePipelinesMetricIdentifiers = []azurePipelinesMetricIdentifier{
	{0, "s0-azure-pipelines-1"},
	{1, "s1-azure-pipelines-1"},
}

func TestAzurePipelinesGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range azurePipelinesMetricIdentifiers {
		var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":1}`))
		}))

		authParams := map[string]string{
			"organizationURL":     apiStub.URL,
			"personalAccessToken": "PAT",
		}

		metadata := map[string]string{
			"poolID":                     "1",
			"targetPipelinesQueueLength": "1",
		}

		meta, err := parseAzurePipelinesMetadata(context.TODO(), &ScalerConfig{TriggerMetadata: metadata, ResolvedEnv: nil, AuthParams: authParams, ScalerIndex: testData.scalerIndex}, http.DefaultClient)
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}

		mockAzurePipelinesScaler := azurePipelinesScaler{
			metadata:   meta,
			httpClient: http.DefaultClient,
		}

		metricSpec := mockAzurePipelinesScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func getMatchedAgentMetaData(url string) *azurePipelinesMetadata {
	meta := azurePipelinesMetadata{}
	meta.organizationName = "testOrg"
	meta.organizationURL = url
	meta.parent = "dotnet60-keda-template"
	meta.personalAccessToken = "testPAT"
	meta.poolID = 1
	meta.targetPipelinesQueueLength = 1

	return &meta
}

func TestAzurePipelinesMatchedAgent(t *testing.T) {
	var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buildLoadJSON())
	}))

	meta := getMatchedAgentMetaData(apiStub.URL)

	mockAzurePipelinesScaler := azurePipelinesScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queuelen, err := mockAzurePipelinesScaler.GetAzurePipelinesQueueLength(context.TODO())

	if err != nil {
		t.Fail()
	}

	if queuelen < 1 {
		t.Fail()
	}
}

func getDemandJobMetaData(url string) *azurePipelinesMetadata {
	meta := getMatchedAgentMetaData(url)
	meta.parent = ""
	meta.demands = "dotnet60,java"

	return meta
}

func getMismatchDemandJobMetaData(url string) *azurePipelinesMetadata {
	meta := getMatchedAgentMetaData(url)
	meta.parent = ""
	meta.demands = "testDemand,iamnotademand"

	return meta
}

func TestAzurePipelinesMatchedDemandAgent(t *testing.T) {
	var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buildLoadJSON())
	}))

	meta := getDemandJobMetaData(apiStub.URL)

	mockAzurePipelinesScaler := azurePipelinesScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queuelen, err := mockAzurePipelinesScaler.GetAzurePipelinesQueueLength(context.TODO())

	if err != nil {
		t.Fail()
	}

	if queuelen < 1 {
		t.Fail()
	}
}

func TestAzurePipelinesNonMatchedDemandAgent(t *testing.T) {
	var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buildLoadJSON())
	}))

	meta := getMismatchDemandJobMetaData(apiStub.URL)

	mockAzurePipelinesScaler := azurePipelinesScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queuelen, err := mockAzurePipelinesScaler.GetAzurePipelinesQueueLength(context.TODO())

	if err != nil {
		t.Fail()
	}

	if queuelen > 0 {
		t.Fail()
	}
}

func buildLoadJSON() []byte {
	output := testJobRequestResponse[0 : len(testJobRequestResponse)-2]
	for i := 1; i < loadCount; i++ {
		output = output + "," + deadJob
	}

	output += "]}"

	return []byte(output)
}
