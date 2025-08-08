package scalers

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"testing"

	v2 "k8s.io/api/autoscaling/v2"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

// solaceSempBaseURL - host url empty or full of spaces
// each url (splitted by , is is not empty or spaces)
// matches regex http(s)://.*:/d(3)+
// username not empty
// messagevpn not empty
// Client Preffix not empty, does not contains *
// factor > 1 and < 100
// any of the 4 metrics at least > 0
type testSolaceDMConfiguration struct {
	testID             string
	configuration      map[string]string
	parseErrorExpected bool
	expectedUrls       []string
	expectedMetrics    map[string]string
}

var (
	solaceDMTestEnvUsernameKey = "SOLACE_DM_TEST_USERNAME_ENV"
	solaceDMtestEnvPasswordKey = "SOLACE_DM_TEST_PASSWORD_ENV"
	// Credential Identifiers
	solaceDMUsername        = "username"
	solaceDMPassword        = "password"
	solaceDMUsernameFromEnv = "usernameFromEnv"
	solaceDMPasswordFromEnv = "passwordFromEnv"

	//
	solaceDMSempBaseURL          = "solaceSempBaseURL"
	solaceDMMessageVpn           = "messageVpn"
	solaceDMClientNamePattern    = "clientNamePattern"
	solaceDMUnsafeSSL            = "unsafeSSL"
	solaceDMQueuedMessagesFactor = "queuedMessagesFactor"
)

// Trigger Auth - Username/Password
var testSolaceDMAuthParams = map[string]string{
	solaceDMUsername: "username_auth_param",
	solaceDMPassword: "password_auth_param",
}

// Trigger Auth - empty
var testSolaceDMEmptyAuthParams = map[string]string{}

// Environment with Username/Password
var testSolaceDMEmptyEnv = map[string]string{}

// Environment with Username/Password
var testSolaceDMNonEmptyEnv = map[string]string{
	solaceDMTestEnvUsernameKey: "username_env_value",
	solaceDMtestEnvPasswordKey: "password_env_value",
}

// Additional props to get Username/Password from Env
var testSolaceAdditionalFromEnvKeys = map[string]string{
	solaceDMUsernameFromEnv: solaceDMTestEnvUsernameKey,
	solaceDMPasswordFromEnv: solaceDMtestEnvPasswordKey,
}

// TEST CASES FOR parseSolaceDMConfiguration()
var solaceDMParseConfigurationTestCases = []testSolaceDMConfiguration{
	{
		"Empty configuration",
		map[string]string{},
		true,
		[]string{},
		map[string]string{},
	},
	{
		"Correct configuration",
		map[string]string{
			solaceDMSempBaseURL:                               "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			solaceDMMessageVpn:                                "training-dmr-2",
			solaceDMClientNamePattern:                         "direct-messaging-simple",
			solaceDMUnsafeSSL:                                 "true",
			solaceDMQueuedMessagesFactor:                      "3",
			aggregatedClientTxMsgRateTargetMetricName:         "600",
			aggregatedClientTxByteRateTargetMetricName:        "0",
			aggregatedClientAverageTxByteRateTargetMetricName: "0",
			aggregatedClientAverageTxMsgRateTargetMetricName:  "0",
		},
		false,
		[]string{
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944/SEMP",
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943/SEMP",
		},
		map[string]string{
			"s1-solace-dm-direct-messaging-simple-aggregatedClientTxMsgRateTarget": "",
		},
	},
	{
		"No Host URL",
		map[string]string{
			solaceDMSempBaseURL:                               "     ",
			solaceDMMessageVpn:                                "training-dmr-2",
			solaceDMClientNamePattern:                         "direct-messaging-simple",
			solaceDMUnsafeSSL:                                 "true",
			solaceDMQueuedMessagesFactor:                      "3",
			aggregatedClientTxMsgRateTargetMetricName:         "600",
			aggregatedClientTxByteRateTargetMetricName:        "0",
			aggregatedClientAverageTxByteRateTargetMetricName: "0",
			aggregatedClientAverageTxMsgRateTargetMetricName:  "0",
		},
		true,
		[]string{},
		map[string]string{},
	},
	{
		"Invalid Host URL",
		map[string]string{
			solaceDMSempBaseURL:                               "asdsdfsdhiohiosdfi, d,,this is an invalid URL?.-xD,:",
			solaceDMMessageVpn:                                "training-dmr-2",
			solaceDMClientNamePattern:                         "direct-messaging-simple",
			solaceDMUnsafeSSL:                                 "true",
			solaceDMQueuedMessagesFactor:                      "3",
			aggregatedClientTxMsgRateTargetMetricName:         "600",
			aggregatedClientTxByteRateTargetMetricName:        "0",
			aggregatedClientAverageTxByteRateTargetMetricName: "0",
			aggregatedClientAverageTxMsgRateTargetMetricName:  "0",
		},
		true,
		[]string{},
		map[string]string{},
	},
	{
		"Empty Message VPN",
		map[string]string{
			solaceDMSempBaseURL:                               "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			solaceDMMessageVpn:                                "",
			solaceDMClientNamePattern:                         "direct-messaging-simple",
			solaceDMUnsafeSSL:                                 "true",
			solaceDMQueuedMessagesFactor:                      "3",
			aggregatedClientTxMsgRateTargetMetricName:         "600",
			aggregatedClientTxByteRateTargetMetricName:        "0",
			aggregatedClientAverageTxByteRateTargetMetricName: "0",
			aggregatedClientAverageTxMsgRateTargetMetricName:  "0",
		},
		true,
		[]string{
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944/SEMP",
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943/SEMP",
		},
		map[string]string{},
	},
	{
		"Empty Client Name Prefix",
		map[string]string{
			solaceDMSempBaseURL:                               "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			solaceDMMessageVpn:                                "default",
			solaceDMClientNamePattern:                         "",
			solaceDMUnsafeSSL:                                 "true",
			solaceDMQueuedMessagesFactor:                      "3",
			aggregatedClientTxMsgRateTargetMetricName:         "600",
			aggregatedClientTxByteRateTargetMetricName:        "0",
			aggregatedClientAverageTxByteRateTargetMetricName: "0",
			aggregatedClientAverageTxMsgRateTargetMetricName:  "0",
		},
		true,
		[]string{
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944/SEMP",
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943/SEMP",
		},
		map[string]string{},
	},
	{
		"Invalid Client Name Prefix",
		map[string]string{
			solaceDMSempBaseURL:                               "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			solaceDMMessageVpn:                                "default",
			solaceDMClientNamePattern:                         "direct-mess*",
			solaceDMUnsafeSSL:                                 "true",
			solaceDMQueuedMessagesFactor:                      "3",
			aggregatedClientTxMsgRateTargetMetricName:         "600",
			aggregatedClientTxByteRateTargetMetricName:        "0",
			aggregatedClientAverageTxByteRateTargetMetricName: "0",
			aggregatedClientAverageTxMsgRateTargetMetricName:  "0",
		},
		true,
		[]string{
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944/SEMP",
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943/SEMP",
		},
		map[string]string{},
	},
	{
		"Invalid Boolean Value in UnsafeSSL",
		map[string]string{
			solaceDMSempBaseURL:                               "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			solaceDMMessageVpn:                                "default",
			solaceDMClientNamePattern:                         "direct-mess",
			solaceDMUnsafeSSL:                                 "trxex",
			solaceDMQueuedMessagesFactor:                      "3",
			aggregatedClientTxMsgRateTargetMetricName:         "600",
			aggregatedClientTxByteRateTargetMetricName:        "0",
			aggregatedClientAverageTxByteRateTargetMetricName: "0",
			aggregatedClientAverageTxMsgRateTargetMetricName:  "0",
		},
		true,
		[]string{
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944/SEMP",
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943/SEMP",
		},
		map[string]string{},
	},
	{
		"Invalid int64 in Queued Messages Factor",
		map[string]string{
			solaceDMSempBaseURL:                               "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			solaceDMMessageVpn:                                "default",
			solaceDMClientNamePattern:                         "direct-mess",
			solaceDMUnsafeSSL:                                 "true",
			solaceDMQueuedMessagesFactor:                      "3a",
			aggregatedClientTxMsgRateTargetMetricName:         "600",
			aggregatedClientTxByteRateTargetMetricName:        "0",
			aggregatedClientAverageTxByteRateTargetMetricName: "0",
			aggregatedClientAverageTxMsgRateTargetMetricName:  "0",
		},
		true,
		[]string{
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944/SEMP",
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943/SEMP",
		},
		map[string]string{},
	},
	{
		"Invalid value (<0 and >100) in Queued Messages Factor",
		map[string]string{
			solaceDMSempBaseURL:                               "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			solaceDMMessageVpn:                                "default",
			solaceDMClientNamePattern:                         "direct-mess",
			solaceDMUnsafeSSL:                                 "true",
			solaceDMQueuedMessagesFactor:                      "300",
			aggregatedClientTxMsgRateTargetMetricName:         "600",
			aggregatedClientTxByteRateTargetMetricName:        "0",
			aggregatedClientAverageTxByteRateTargetMetricName: "0",
			aggregatedClientAverageTxMsgRateTargetMetricName:  "0",
		},
		true,
		[]string{
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944/SEMP",
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943/SEMP",
		},
		map[string]string{},
	},
	{
		"All metrics with 0 as Target",
		map[string]string{
			solaceDMSempBaseURL:                               "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			solaceDMMessageVpn:                                "default",
			solaceDMClientNamePattern:                         "direct-mess",
			solaceDMUnsafeSSL:                                 "true",
			solaceDMQueuedMessagesFactor:                      "3",
			aggregatedClientTxMsgRateTargetMetricName:         "0",
			aggregatedClientTxByteRateTargetMetricName:        "0",
			aggregatedClientAverageTxByteRateTargetMetricName: "0",
			aggregatedClientAverageTxMsgRateTargetMetricName:  "0",
		},
		true,
		[]string{
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944/SEMP",
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943/SEMP",
		},
		map[string]string{},
	},
	{
		"Correct Params - Metric 1",
		map[string]string{
			solaceDMSempBaseURL:                               "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			solaceDMMessageVpn:                                "default",
			solaceDMClientNamePattern:                         "direct-messaging-simple",
			solaceDMUnsafeSSL:                                 "true",
			solaceDMQueuedMessagesFactor:                      "3",
			aggregatedClientTxMsgRateTargetMetricName:         "1000",
			aggregatedClientTxByteRateTargetMetricName:        "0",
			aggregatedClientAverageTxByteRateTargetMetricName: "0",
			aggregatedClientAverageTxMsgRateTargetMetricName:  "0",
		},
		false,
		[]string{
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944/SEMP",
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943/SEMP",
		},
		map[string]string{
			"s1-solace-dm-direct-messaging-simple-aggregatedClientTxMsgRateTarget": "",
		},
	},
	{
		"Correct Params - Metric 2",
		map[string]string{
			solaceDMSempBaseURL:                               "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			solaceDMMessageVpn:                                "default",
			solaceDMClientNamePattern:                         "direct-messaging-simple",
			solaceDMUnsafeSSL:                                 "true",
			solaceDMQueuedMessagesFactor:                      "3",
			aggregatedClientTxMsgRateTargetMetricName:         "0",
			aggregatedClientTxByteRateTargetMetricName:        "1000",
			aggregatedClientAverageTxByteRateTargetMetricName: "0",
			aggregatedClientAverageTxMsgRateTargetMetricName:  "0",
		},
		false,
		[]string{
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944/SEMP",
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943/SEMP",
		},
		map[string]string{
			"s1-solace-dm-direct-messaging-simple-aggregatedClientTxByteRateTarget": "",
		},
	},
	{
		"Correct Params - Metric 3",
		map[string]string{
			solaceDMSempBaseURL:                               "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			solaceDMMessageVpn:                                "default",
			solaceDMClientNamePattern:                         "direct-messaging-simple",
			solaceDMUnsafeSSL:                                 "true",
			solaceDMQueuedMessagesFactor:                      "3",
			aggregatedClientTxMsgRateTargetMetricName:         "0",
			aggregatedClientTxByteRateTargetMetricName:        "0",
			aggregatedClientAverageTxByteRateTargetMetricName: "10000",
			aggregatedClientAverageTxMsgRateTargetMetricName:  "0",
		},
		false,
		[]string{
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944/SEMP",
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943/SEMP",
		},
		map[string]string{
			"s1-solace-dm-direct-messaging-simple-aggregatedClientAverageTxByteRateTarget": "",
		},
	},
	{
		"Correct Params - Metric 4",
		map[string]string{
			solaceDMSempBaseURL:                               "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			solaceDMMessageVpn:                                "default",
			solaceDMClientNamePattern:                         "direct-messaging-simple",
			solaceDMUnsafeSSL:                                 "true",
			solaceDMQueuedMessagesFactor:                      "3",
			aggregatedClientTxMsgRateTargetMetricName:         "0",
			aggregatedClientTxByteRateTargetMetricName:        "0",
			aggregatedClientAverageTxByteRateTargetMetricName: "0",
			aggregatedClientAverageTxMsgRateTargetMetricName:  "10000",
		},
		false,
		[]string{
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944/SEMP",
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943/SEMP",
		},
		map[string]string{
			"s1-solace-dm-direct-messaging-simple-aggregatedClientAverageTxMsgRateTarget": "",
		},
	},
	{
		"Correct Params - All Metrics",
		map[string]string{
			solaceDMSempBaseURL:                               "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			solaceDMMessageVpn:                                "default",
			solaceDMClientNamePattern:                         "direct-messaging-simple",
			solaceDMUnsafeSSL:                                 "true",
			solaceDMQueuedMessagesFactor:                      "3",
			aggregatedClientTxMsgRateTargetMetricName:         "300",
			aggregatedClientTxByteRateTargetMetricName:        "300",
			aggregatedClientAverageTxByteRateTargetMetricName: "10000",
			aggregatedClientAverageTxMsgRateTargetMetricName:  "10000",
		},
		false,
		[]string{
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944/SEMP",
			"https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943/SEMP",
		},
		map[string]string{
			"s1-solace-dm-direct-messaging-simple-aggregatedClientTxMsgRateTarget":         "",
			"s1-solace-dm-direct-messaging-simple-aggregatedClientTxByteRateTarget":        "",
			"s1-solace-dm-direct-messaging-simple-aggregatedClientAverageTxByteRateTarget": "",
			"s1-solace-dm-direct-messaging-simple-aggregatedClientAverageTxMsgRateTarget":  "",
		},
	},
}

func TestParseSolaceDMConfiguration(t *testing.T) {
	// initial round - empty environment, auth params
	t.Log(" --> *** Round 1: Environment: empty, AuthParams: true ***")
	for i, testData := range solaceDMParseConfigurationTestCases {
		testScenario(t, i, testSolaceDMEmptyEnv, testSolaceDMAuthParams, testData)
	}

	// Second round - environment with usr/pwd, empty params, reference to env keys
	t.Log(" --> *** Round 2: Environment: true, AuthParams: empty ***")
	for i, testData := range solaceDMParseConfigurationTestCases {
		testDatawithEnvKeys := map[string]string{}
		maps.Insert(testDatawithEnvKeys, maps.All(testData.configuration))
		maps.Insert(testDatawithEnvKeys, maps.All(testSolaceAdditionalFromEnvKeys))

		testData.configuration = testDatawithEnvKeys

		testScenario(t, i, testSolaceDMNonEmptyEnv, testSolaceDMEmptyAuthParams, testData)
	}
}

func testScenario(t *testing.T, i int, resolvedEnv map[string]string, authParams map[string]string, testData testSolaceDMConfiguration) {
	t.Logf("Test [%d], ParseErrorExpected: '%t' - TestID: '%s'", i, testData.parseErrorExpected, testData.testID)
	config, err := parseSolaceDMConfiguration(&scalersconfig.ScalerConfig{ResolvedEnv: resolvedEnv, TriggerMetadata: testData.configuration, AuthParams: authParams, TriggerIndex: 1})

	switch {
	case testData.parseErrorExpected && err == nil:
		t.Log(" --> FAIL")
		t.Error("expected error but got success")
		return

	case testData.parseErrorExpected && err != nil:
		t.Logf(" --> PASS - Error (expected): '%s'", err.Error())
		return

	case !testData.parseErrorExpected && err != nil:
		t.Log(" --> FAIL")
		t.Error("expected success but got error: ", err)
		return

	case !testData.parseErrorExpected && err == nil:
		if len(testData.expectedUrls) > 0 {
			if !testEq(testData.expectedUrls, config.sempURL) {
				t.Log(" --> FAIL")
				t.Error("URLs are different from expected")
				return
			}
		}
	}

	// At this point configuration should be ok!
	testSolaceScaler := SolaceDMScaler{
		configuration: config,
		httpClient:    http.DefaultClient,
	}

	// Test username/password resolution from triggerAuth and Env
	if testSolaceScaler.configuration.Username == "" {
		t.Log(" --> FAIL")
		t.Errorf("Username not populated")
		return
	}

	if testSolaceScaler.configuration.Password == "" {
		t.Log(" --> FAIL")
		t.Errorf("Username not populated")
		return
	}

	fmt.Printf("Username: '%s', Password: '%s'", testSolaceScaler.configuration.Username, testSolaceScaler.configuration.Password)

	var metrics []v2.MetricSpec
	if metrics = testSolaceScaler.GetMetricSpecForScaling(context.Background()); len(metrics) == 0 {
		t.Log(" --> FAIL")
		t.Errorf("metric value not found")
		return
	}

	for _, metric := range metrics {
		metricName := metric.External.Metric.Name

		if _, ok := testData.expectedMetrics[metricName]; ok == false {
			t.Log(" --> FAIL")
			t.Errorf("unexpected metric: '%s'", metricName)
			return
		}
	}

	t.Log(" --> PASS - No Error expected")
}

func testEq(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		fmt.Printf("'%s'<>'%s' - Diff: '%t'\n", a[i], b[i], (a[i] != b[i]))
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
