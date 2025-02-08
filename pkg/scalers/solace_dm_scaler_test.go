package scalers

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	v2 "k8s.io/api/autoscaling/v2"
)

// host url empty or full of spaces
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

// AUTH RECORD FOR TEST
var testSolaceDMAuthParams = map[string]string{
	"username": "username",
	"password": "password",
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
			"hostUrl":                                 "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			"messageVpn":                              "training-dmr-2",
			"clientNamePrefix":                        "direct-messaging-simple",
			"unsafeSSL":                               "true",
			"queuedMessagesFactor":                    "3",
			"aggregatedClientTxMsgRateTarget":         "600",
			"aggregatedClientTxByteRateTarget":        "0",
			"aggregatedClientAverageTxByteRateTarget": "0",
			"aggregatedClientAverageTxMsgRateTarget":  "0",
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
			"hostUrl":                                 "",
			"messageVpn":                              "training-dmr-2",
			"clientNamePrefix":                        "direct-messaging-simple",
			"unsafeSSL":                               "true",
			"queuedMessagesFactor":                    "3",
			"aggregatedClientTxMsgRateTarget":         "600",
			"aggregatedClientTxByteRateTarget":        "0",
			"aggregatedClientAverageTxByteRateTarget": "0",
			"aggregatedClientAverageTxMsgRateTarget":  "0",
		},
		true,
		[]string{},
		map[string]string{},
	},
	{
		"Invalid Host URL",
		map[string]string{
			"hostUrl":                                 "asdsdfsdhiohiosdfi, d,,this is an invalid URL?.-xD,:",
			"messageVpn":                              "training-dmr-2",
			"clientNamePrefix":                        "direct-messaging-simple",
			"unsafeSSL":                               "true",
			"queuedMessagesFactor":                    "3",
			"aggregatedClientTxMsgRateTarget":         "600",
			"aggregatedClientTxByteRateTarget":        "0",
			"aggregatedClientAverageTxByteRateTarget": "0",
			"aggregatedClientAverageTxMsgRateTarget":  "0",
		},
		true,
		[]string{},
		map[string]string{},
	},
	{
		"Empty Message VPN",
		map[string]string{
			"hostUrl":                                 "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			"messageVpn":                              "",
			"clientNamePrefix":                        "direct-messaging-simple",
			"unsafeSSL":                               "true",
			"queuedMessagesFactor":                    "3",
			"aggregatedClientTxMsgRateTarget":         "600",
			"aggregatedClientTxByteRateTarget":        "0",
			"aggregatedClientAverageTxByteRateTarget": "0",
			"aggregatedClientAverageTxMsgRateTarget":  "0",
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
			"hostUrl":                                 "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			"messageVpn":                              "default",
			"clientNamePrefix":                        "",
			"unsafeSSL":                               "true",
			"queuedMessagesFactor":                    "3",
			"aggregatedClientTxMsgRateTarget":         "600",
			"aggregatedClientTxByteRateTarget":        "0",
			"aggregatedClientAverageTxByteRateTarget": "0",
			"aggregatedClientAverageTxMsgRateTarget":  "0",
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
			"hostUrl":                                 "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			"messageVpn":                              "default",
			"clientNamePrefix":                        "direct-mess*",
			"unsafeSSL":                               "true",
			"queuedMessagesFactor":                    "3",
			"aggregatedClientTxMsgRateTarget":         "600",
			"aggregatedClientTxByteRateTarget":        "0",
			"aggregatedClientAverageTxByteRateTarget": "0",
			"aggregatedClientAverageTxMsgRateTarget":  "0",
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
			"hostUrl":                                 "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			"messageVpn":                              "default",
			"clientNamePrefix":                        "direct-mess",
			"unsafeSSL":                               "trxex",
			"queuedMessagesFactor":                    "3",
			"aggregatedClientTxMsgRateTarget":         "600",
			"aggregatedClientTxByteRateTarget":        "0",
			"aggregatedClientAverageTxByteRateTarget": "0",
			"aggregatedClientAverageTxMsgRateTarget":  "0",
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
			"hostUrl":                                 "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			"messageVpn":                              "default",
			"clientNamePrefix":                        "direct-mess",
			"unsafeSSL":                               "true",
			"queuedMessagesFactor":                    "3a",
			"aggregatedClientTxMsgRateTarget":         "600",
			"aggregatedClientTxByteRateTarget":        "0",
			"aggregatedClientAverageTxByteRateTarget": "0",
			"aggregatedClientAverageTxMsgRateTarget":  "0",
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
			"hostUrl":                                 "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			"messageVpn":                              "default",
			"clientNamePrefix":                        "direct-mess",
			"unsafeSSL":                               "true",
			"queuedMessagesFactor":                    "300",
			"aggregatedClientTxMsgRateTarget":         "600",
			"aggregatedClientTxByteRateTarget":        "0",
			"aggregatedClientAverageTxByteRateTarget": "0",
			"aggregatedClientAverageTxMsgRateTarget":  "0",
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
			"hostUrl":                                 "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			"messageVpn":                              "default",
			"clientNamePrefix":                        "direct-mess",
			"unsafeSSL":                               "true",
			"queuedMessagesFactor":                    "3",
			"aggregatedClientTxMsgRateTarget":         "0",
			"aggregatedClientTxByteRateTarget":        "0",
			"aggregatedClientAverageTxByteRateTarget": "0",
			"aggregatedClientAverageTxMsgRateTarget":  "0",
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
			"hostUrl":                                 "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			"messageVpn":                              "default",
			"clientNamePrefix":                        "direct-messaging-simple",
			"unsafeSSL":                               "true",
			"queuedMessagesFactor":                    "3",
			"aggregatedClientTxMsgRateTarget":         "1000",
			"aggregatedClientTxByteRateTarget":        "0",
			"aggregatedClientAverageTxByteRateTarget": "0",
			"aggregatedClientAverageTxMsgRateTarget":  "0",
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
			"hostUrl":                                 "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			"messageVpn":                              "default",
			"clientNamePrefix":                        "direct-messaging-simple",
			"unsafeSSL":                               "true",
			"queuedMessagesFactor":                    "3",
			"aggregatedClientTxMsgRateTarget":         "0",
			"aggregatedClientTxByteRateTarget":        "1000",
			"aggregatedClientAverageTxByteRateTarget": "0",
			"aggregatedClientAverageTxMsgRateTarget":  "0",
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
			"hostUrl":                                 "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			"messageVpn":                              "default",
			"clientNamePrefix":                        "direct-messaging-simple",
			"unsafeSSL":                               "true",
			"queuedMessagesFactor":                    "3",
			"aggregatedClientTxMsgRateTarget":         "0",
			"aggregatedClientTxByteRateTarget":        "0",
			"aggregatedClientAverageTxByteRateTarget": "10000",
			"aggregatedClientAverageTxMsgRateTarget":  "0",
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
			"hostUrl":                                 "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			"messageVpn":                              "default",
			"clientNamePrefix":                        "direct-messaging-simple",
			"unsafeSSL":                               "true",
			"queuedMessagesFactor":                    "3",
			"aggregatedClientTxMsgRateTarget":         "0",
			"aggregatedClientTxByteRateTarget":        "0",
			"aggregatedClientAverageTxByteRateTarget": "0",
			"aggregatedClientAverageTxMsgRateTarget":  "10000",
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
			"hostUrl":                                 "https://mr-connection-s2vulj70fsu.messaging.solace.cloud:944,https://mr-connection-s2vulj70fsu.messaging.solace.cloud:943",
			"messageVpn":                              "default",
			"clientNamePrefix":                        "direct-messaging-simple",
			"unsafeSSL":                               "true",
			"queuedMessagesFactor":                    "3",
			"aggregatedClientTxMsgRateTarget":         "300",
			"aggregatedClientTxByteRateTarget":        "300",
			"aggregatedClientAverageTxByteRateTarget": "10000",
			"aggregatedClientAverageTxMsgRateTarget":  "10000",
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
	for i, testData := range solaceDMParseConfigurationTestCases {
		testScenario(t, i, testData)
	}
}

func testScenario(t *testing.T, i int, testData testSolaceDMConfiguration) {
	t.Logf("Test [%d], ParseErrorExpected: '%t' - TestID: '%s'", i, testData.parseErrorExpected, testData.testID)
	config, err := parseSolaceDMConfiguration(&scalersconfig.ScalerConfig{ResolvedEnv: nil, TriggerMetadata: testData.configuration, AuthParams: testSolaceDMAuthParams, TriggerIndex: 1})
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
			if !testEq(testData.expectedUrls, config.sempUrl) {
				t.Log(" --> FAIL")
				t.Error("URLs are different from expected")
				return
			}
		}
	}

	//At this point configuration should be ok!
	testSolaceScaler := SolaceDMScaler{
		configuration: config,
		httpClient:    http.DefaultClient,
	}

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
