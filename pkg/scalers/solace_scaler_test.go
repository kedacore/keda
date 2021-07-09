package scalers

import (
	"fmt"
	"net/http"
	"testing"

	"k8s.io/api/autoscaling/v2beta2"
)

type testSolaceMetadata struct {
	testId   string
	metadata map[string]string
	isError  bool
}

type testSolaceMetricIdentifier struct {
	metadataTestData *testSolaceMetadata
	name             string
}

var (
	soltest_VALID_BASEURL  = "http://localhost:8080"
	soltest_VALID_PROTOCOL = "http"
	soltest_VALID_HOSTNAME = "localhost"
	soltest_VALID_PORT     = "8080"
	// ##### TODO -- RESET THIS
	//	soltest_VALID_USERNAME                = "solace_user"
	//	soltest_VALID_PASSWORD                = "solace_pass"
	//	soltest_VALID_VPN                     = "solace_keda_vpn"
	//	soltest_VALID_QUEUE_NAME              = "KEDA_Q1"
	soltest_VALID_USERNAME         = "admin"
	soltest_VALID_PASSWORD         = "admin"
	soltest_VALID_VPN              = "dennis_vpn"
	soltest_VALID_QUEUE_NAME       = "queue3"
	soltest_VALID_MSG_COUNT_TARGET = "10"
	soltest_VALID_MSG_SPOOL_TARGET = "20"

	soltest_ENVVAR_USERNAME = "SOLTEST_USERNAME"
	soltest_ENVVAR_PASSWORD = "SOLTEST_PASSWORD"
)

// AUTH RECORD FOR TEST
var testDataSolaceAuthParamsVALID = map[string]string{
	solace_META_username: soltest_VALID_USERNAME,
	solace_META_password: soltest_VALID_PASSWORD,
}

// ENV VARS FOR TEST -- VALID USER / PWD
var testDataSolaceResolvedEnvVALID = map[string]string{
	soltest_ENVVAR_USERNAME: soltest_VALID_USERNAME, // Sets the environment variables to the correct values
	soltest_ENVVAR_PASSWORD: soltest_VALID_PASSWORD,
}

// ENV VARS FOR TEST -- INVALID USER / PWD
var testDataSolaceResolvedEnvINVALID = map[string]string{
	soltest_ENVVAR_USERNAME: "NOT_CORRECT_USERID", // Sets the environment variables to incorrect values
	soltest_ENVVAR_PASSWORD: "NOT_A_PASSWORD",
}

// TEST CASES FOR SolaceParseMetadata()
var testParseSolaceMetadata = []testSolaceMetadata{
	/*
		IF brokerBaseUrl is present, use it without interpretation as the base URL: http://my.host.domain:1234
		IF brokerBaseUrl in not present, Use protocol, host, and port
	*/

	// Empty
	{
		"#001 - EMPTY", map[string]string{},
		true,
	},

	// +Case - brokerBaseUrl
	{
		"#002 - brokerBaseUrl",
		map[string]string{
			"":                         "",
			solace_META_brokerBaseUrl:  soltest_VALID_BASEURL,
			solace_META_brokerProtocol: "",
			solace_META_brokerHostname: "",
			solace_META_brokerPort:     "",
			solace_META_msgVpn:         soltest_VALID_VPN,
			solace_META_usernameEnv:    "",
			solace_META_passwordEnv:    "",
			solace_META_username:       soltest_VALID_USERNAME,
			solace_META_password:       soltest_VALID_PASSWORD,
			solace_META_queueName:      soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget: soltest_VALID_MSG_COUNT_TARGET,
		},
		false,
	},
	// +Case - protocol + host + port
	{
		"#003 - protocol + host + port",
		map[string]string{
			solace_META_brokerBaseUrl:  "",
			solace_META_brokerProtocol: soltest_VALID_PROTOCOL,
			solace_META_brokerHostname: soltest_VALID_HOSTNAME,
			solace_META_brokerPort:     soltest_VALID_PORT,
			solace_META_msgVpn:         soltest_VALID_VPN,
			solace_META_usernameEnv:    "",
			solace_META_passwordEnv:    "",
			solace_META_username:       soltest_VALID_USERNAME,
			solace_META_password:       soltest_VALID_PASSWORD,
			solace_META_queueName:      soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget: soltest_VALID_MSG_COUNT_TARGET,
		},
		false,
	},

	// -Case - missing protocol
	{
		"#004 - missing protocol",
		map[string]string{
			solace_META_brokerBaseUrl:  "",
			solace_META_brokerProtocol: "",
			solace_META_brokerHostname: soltest_VALID_HOSTNAME,
			solace_META_brokerPort:     soltest_VALID_PORT,
			solace_META_msgVpn:         soltest_VALID_VPN,
			solace_META_usernameEnv:    "",
			solace_META_passwordEnv:    "",
			solace_META_username:       soltest_VALID_USERNAME,
			solace_META_password:       soltest_VALID_PASSWORD,
			solace_META_queueName:      soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget: soltest_VALID_MSG_COUNT_TARGET,
		},
		true,
	},
	// -Case - missing hostname
	{
		"#005 - missing hostname",
		map[string]string{
			solace_META_brokerBaseUrl:  "",
			solace_META_brokerProtocol: soltest_VALID_PROTOCOL,
			solace_META_brokerHostname: "",
			solace_META_brokerPort:     soltest_VALID_PORT,
			solace_META_msgVpn:         soltest_VALID_VPN,
			solace_META_usernameEnv:    "",
			solace_META_passwordEnv:    "",
			solace_META_username:       soltest_VALID_USERNAME,
			solace_META_password:       soltest_VALID_PASSWORD,
			solace_META_queueName:      soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget: soltest_VALID_MSG_COUNT_TARGET,
		},
		true,
	},
	// -Case - missing port
	{
		"#006 - missing port",
		map[string]string{
			solace_META_brokerBaseUrl:  "",
			solace_META_brokerProtocol: soltest_VALID_PROTOCOL,
			solace_META_brokerHostname: soltest_VALID_HOSTNAME,
			solace_META_brokerPort:     "",
			solace_META_msgVpn:         soltest_VALID_VPN,
			solace_META_usernameEnv:    "",
			solace_META_passwordEnv:    "",
			solace_META_username:       soltest_VALID_USERNAME,
			solace_META_password:       soltest_VALID_PASSWORD,
			solace_META_queueName:      soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget: soltest_VALID_MSG_COUNT_TARGET,
		},
		true,
	},
	// -Case - missing username (clear)
	{
		"#007 - missing username (clear)",
		map[string]string{
			solace_META_brokerBaseUrl:  "",
			solace_META_brokerProtocol: soltest_VALID_PROTOCOL,
			solace_META_brokerHostname: soltest_VALID_HOSTNAME,
			solace_META_brokerPort:     soltest_VALID_PORT,
			solace_META_msgVpn:         soltest_VALID_VPN,
			solace_META_usernameEnv:    "",
			solace_META_passwordEnv:    "",
			solace_META_username:       "",
			solace_META_password:       soltest_VALID_PASSWORD,
			solace_META_queueName:      soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget: soltest_VALID_MSG_COUNT_TARGET,
		},
		true,
	},
	// -Case - missing password (clear)
	{
		"#008 - missing password (clear)",
		map[string]string{
			solace_META_brokerBaseUrl:  "",
			solace_META_brokerProtocol: soltest_VALID_PROTOCOL,
			solace_META_brokerHostname: soltest_VALID_HOSTNAME,
			solace_META_brokerPort:     soltest_VALID_PORT,
			solace_META_msgVpn:         soltest_VALID_VPN,
			solace_META_usernameEnv:    "",
			solace_META_passwordEnv:    "",
			solace_META_username:       soltest_VALID_USERNAME,
			solace_META_password:       "",
			solace_META_queueName:      soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget: soltest_VALID_MSG_COUNT_TARGET,
		},
		true,
	},
	// -Case - missing queue
	{
		"#009 - missing queueName",
		map[string]string{
			solace_META_brokerBaseUrl:  "",
			solace_META_brokerProtocol: soltest_VALID_PROTOCOL,
			solace_META_brokerHostname: soltest_VALID_HOSTNAME,
			solace_META_brokerPort:     soltest_VALID_PORT,
			solace_META_msgVpn:         soltest_VALID_VPN,
			solace_META_usernameEnv:    "",
			solace_META_passwordEnv:    "",
			solace_META_username:       soltest_VALID_USERNAME,
			solace_META_password:       soltest_VALID_PASSWORD,
			solace_META_queueName:      "",
			solace_META_msgCountTarget: soltest_VALID_MSG_COUNT_TARGET,
		},
		true,
	},
	// -Case - missing msgCountTarget
	{
		"#010 - missing msgCountTarget",
		map[string]string{
			solace_META_brokerBaseUrl:       "",
			solace_META_brokerProtocol:      soltest_VALID_PROTOCOL,
			solace_META_brokerHostname:      soltest_VALID_HOSTNAME,
			solace_META_brokerPort:          soltest_VALID_PORT,
			solace_META_msgVpn:              soltest_VALID_VPN,
			solace_META_usernameEnv:         "",
			solace_META_passwordEnv:         "",
			solace_META_username:            soltest_VALID_USERNAME,
			solace_META_password:            soltest_VALID_PASSWORD,
			solace_META_queueName:           soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget:      "",
			solace_META_msgSpoolUsageTarget: "",
		},
		true,
	},
	// -Case - msgSpoolUsageTarget non-numeric
	{
		"#011 - msgSpoolUsageTarget non-numeric",
		map[string]string{
			solace_META_brokerBaseUrl:  "",
			solace_META_brokerProtocol: soltest_VALID_PROTOCOL,
			solace_META_brokerHostname: soltest_VALID_HOSTNAME,
			solace_META_brokerPort:     soltest_VALID_PORT,
			solace_META_msgVpn:         soltest_VALID_VPN,
			solace_META_usernameEnv:    "",
			solace_META_passwordEnv:    "",
			solace_META_username:       soltest_VALID_USERNAME,
			solace_META_password:       soltest_VALID_PASSWORD,
			solace_META_queueName:      soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget: "NOT_AN_INTEGER",
		},
		true,
	},
	// -Case - msgSpoolUsage non-numeric
	{
		"#012 - msgSpoolUsage non-numeric",
		map[string]string{
			solace_META_brokerBaseUrl:       "",
			solace_META_brokerProtocol:      soltest_VALID_PROTOCOL,
			solace_META_brokerHostname:      soltest_VALID_HOSTNAME,
			solace_META_brokerPort:          soltest_VALID_PORT,
			solace_META_msgVpn:              soltest_VALID_VPN,
			solace_META_usernameEnv:         "",
			solace_META_passwordEnv:         "",
			solace_META_username:            soltest_VALID_USERNAME,
			solace_META_password:            soltest_VALID_PASSWORD,
			solace_META_queueName:           soltest_VALID_QUEUE_NAME,
			solace_META_msgSpoolUsageTarget: "NOT_AN_INTEGER",
		},
		true,
	},
	// +Case - Pass with msgSpoolUsageTarget and not msgCountTarget
	{
		"#013 - brokerBaseUrl",
		map[string]string{
			"":                              "",
			solace_META_brokerBaseUrl:       soltest_VALID_BASEURL,
			solace_META_brokerProtocol:      "",
			solace_META_brokerHostname:      "",
			solace_META_brokerPort:          "",
			solace_META_msgVpn:              soltest_VALID_VPN,
			solace_META_usernameEnv:         "",
			solace_META_passwordEnv:         "",
			solace_META_username:            soltest_VALID_USERNAME,
			solace_META_password:            soltest_VALID_PASSWORD,
			solace_META_queueName:           soltest_VALID_QUEUE_NAME,
			solace_META_msgSpoolUsageTarget: soltest_VALID_MSG_SPOOL_TARGET,
		},
		false,
	},
}

var testSolaceEnvCreds = []testSolaceMetadata{
	// +Case - Should find ENV vars
	// Environment user/pass should be set:
	// - SOLTEST_USERNAME
	// - SOLTEST_PASSWORD
	{
		"#101 - Connect with Credentials in env",
		map[string]string{
			//		"":               "",
			solace_META_brokerBaseUrl:  soltest_VALID_BASEURL,
			solace_META_brokerProtocol: "",
			solace_META_brokerHostname: "",
			solace_META_brokerPort:     "",
			solace_META_msgVpn:         soltest_VALID_VPN,
			solace_META_usernameEnv:    soltest_ENVVAR_USERNAME,
			solace_META_passwordEnv:    soltest_ENVVAR_PASSWORD,
			//		solace_META_username:              "",
			//		solace_META_password:              "",
			solace_META_queueName:      soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget: soltest_VALID_MSG_COUNT_TARGET,
		},
		false,
	},
	// -Case - Should fail with ENV var not found
	{
		"#102 - Environment vars referenced but not found",
		map[string]string{
			//		"":               "",
			solace_META_brokerBaseUrl:  soltest_VALID_BASEURL,
			solace_META_brokerProtocol: "",
			solace_META_brokerHostname: "",
			solace_META_brokerPort:     "",
			solace_META_msgVpn:         soltest_VALID_VPN,
			solace_META_usernameEnv:    "SOLTEST_DNE",
			solace_META_passwordEnv:    "SOLTEST_DNE",
			//		solace_META_username:              "",
			//		solace_META_password:              "",
			solace_META_queueName:      soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget: soltest_VALID_MSG_COUNT_TARGET,
		},
		true,
	},
}

var testSolaceK8sSecretCreds = []testSolaceMetadata{
	// Records require Auth Record to be passed

	// +Case - Should find
	{
		"#201 - Connect with credentials from Auth Record (ENV VAR Present)",
		map[string]string{
			//		"":               "",
			solace_META_brokerBaseUrl:  soltest_VALID_BASEURL,
			solace_META_brokerProtocol: "",
			solace_META_brokerHostname: "",
			solace_META_brokerPort:     "",
			solace_META_msgVpn:         soltest_VALID_VPN,
			solace_META_usernameEnv:    soltest_ENVVAR_USERNAME,
			solace_META_passwordEnv:    soltest_ENVVAR_PASSWORD,
			//		solace_META_username:              "",
			//		solace_META_password:              "",
			solace_META_queueName:      soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget: soltest_VALID_MSG_COUNT_TARGET,
		},
		false,
	},
	// +Case - should find creds
	{
		"#202 - Connect with credentials from Auth Record (ENV VAR and Clear Auth not present)",
		map[string]string{
			//		"":               "",
			//		solace_META_brokerBaseUrl:  soltest_VALID_BASEURL,
			solace_META_brokerProtocol: soltest_VALID_PROTOCOL,
			solace_META_brokerHostname: soltest_VALID_HOSTNAME,
			solace_META_brokerPort:     soltest_VALID_PORT,
			solace_META_msgVpn:         soltest_VALID_VPN,
			//		solace_META_usernameEnv:    soltest_ENVVAR_USERNAME,
			//		solace_META_passwordEnv:    soltest_ENVVAR_PASSWORD,
			//		solace_META_username:              "",
			//		solace_META_password:              "",
			solace_META_queueName:      soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget: soltest_VALID_MSG_COUNT_TARGET,
		},
		false,
	},
	// +Case - Should find with creds
	{
		"#203 - Connect with credentials from Auth Record (ENV VAR Present, Clear Auth not present)",
		map[string]string{
			//		"":               "",
			solace_META_brokerBaseUrl:  soltest_VALID_BASEURL,
			solace_META_brokerProtocol: "",
			solace_META_brokerHostname: "",
			solace_META_brokerPort:     "",
			solace_META_msgVpn:         soltest_VALID_VPN,
			solace_META_usernameEnv:    "SOLTEST_DNE",
			solace_META_passwordEnv:    "SOLTEST_DNE",
			//		solace_META_username:              "",
			//		solace_META_password:              "",
			solace_META_queueName:      soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget: soltest_VALID_MSG_COUNT_TARGET,
		},
		false,
	},
}

var testSolaceGetMetricSpecData = []testSolaceMetadata{
	{
		"#401 - Get Metric Spec - msgCountTarget",
		map[string]string{
			"":                         "",
			solace_META_brokerBaseUrl:  soltest_VALID_BASEURL,
			solace_META_brokerProtocol: "",
			solace_META_brokerHostname: "",
			solace_META_brokerPort:     "",
			solace_META_msgVpn:         soltest_VALID_VPN,
			solace_META_usernameEnv:    "",
			solace_META_passwordEnv:    "",
			solace_META_username:       soltest_VALID_USERNAME,
			solace_META_password:       soltest_VALID_PASSWORD,
			solace_META_queueName:      soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget: soltest_VALID_MSG_COUNT_TARGET,
			//			solace_META_msgSpoolUsageTarget: soltest_VALID_MSG_SPOOL_TARGET,
		},
		false,
	},
	{
		"#402 - Get Metric Spec - msgSpoolUsageTarget",
		map[string]string{
			"":                         "",
			solace_META_brokerBaseUrl:  soltest_VALID_BASEURL,
			solace_META_brokerProtocol: "",
			solace_META_brokerHostname: "",
			solace_META_brokerPort:     "",
			solace_META_msgVpn:         soltest_VALID_VPN,
			solace_META_usernameEnv:    "",
			solace_META_passwordEnv:    "",
			solace_META_username:       soltest_VALID_USERNAME,
			solace_META_password:       soltest_VALID_PASSWORD,
			solace_META_queueName:      soltest_VALID_QUEUE_NAME,
			//			solace_META_msgCountTarget:      soltest_VALID_MSG_COUNT_TARGET,
			solace_META_msgSpoolUsageTarget: soltest_VALID_MSG_SPOOL_TARGET,
		},
		false,
	},
	{
		"#403 - Get Metric Spec - BOTH msgSpoolUsage and msgCountTarget",
		map[string]string{
			"":                              "",
			solace_META_brokerBaseUrl:       soltest_VALID_BASEURL,
			solace_META_brokerProtocol:      "",
			solace_META_brokerHostname:      "",
			solace_META_brokerPort:          "",
			solace_META_msgVpn:              soltest_VALID_VPN,
			solace_META_usernameEnv:         "",
			solace_META_passwordEnv:         "",
			solace_META_username:            soltest_VALID_USERNAME,
			solace_META_password:            soltest_VALID_PASSWORD,
			solace_META_queueName:           soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget:      soltest_VALID_MSG_COUNT_TARGET,
			solace_META_msgSpoolUsageTarget: soltest_VALID_MSG_SPOOL_TARGET,
		},
		false,
	},
	{
		"#404 - Get Metric Spec - BOTH MISSING",
		map[string]string{
			"":                         "",
			solace_META_brokerBaseUrl:  soltest_VALID_BASEURL,
			solace_META_brokerProtocol: "",
			solace_META_brokerHostname: "",
			solace_META_brokerPort:     "",
			solace_META_msgVpn:         soltest_VALID_VPN,
			solace_META_usernameEnv:    "",
			solace_META_passwordEnv:    "",
			solace_META_username:       soltest_VALID_USERNAME,
			solace_META_password:       soltest_VALID_PASSWORD,
			solace_META_queueName:      soltest_VALID_QUEUE_NAME,
			//			solace_META_msgCountTarget:      soltest_VALID_MSG_COUNT_TARGET,
			//			solace_META_msgSpoolUsageTarget: soltest_VALID_MSG_SPOOL_TARGET,
		},
		true,
	},
	{
		"#405 - Get Metric Spec - BOTH ZERO",
		map[string]string{
			"":                              "",
			solace_META_brokerBaseUrl:       soltest_VALID_BASEURL,
			solace_META_brokerProtocol:      "",
			solace_META_brokerHostname:      "",
			solace_META_brokerPort:          "",
			solace_META_msgVpn:              soltest_VALID_VPN,
			solace_META_usernameEnv:         "",
			solace_META_passwordEnv:         "",
			solace_META_username:            soltest_VALID_USERNAME,
			solace_META_password:            soltest_VALID_PASSWORD,
			solace_META_queueName:           soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget:      "0",
			solace_META_msgSpoolUsageTarget: "0",
		},
		true,
	},
	{
		"#406 - Get Metric Spec - ONE ZERO; OTHER VALID",
		map[string]string{
			"":                              "",
			solace_META_brokerBaseUrl:       soltest_VALID_BASEURL,
			solace_META_brokerProtocol:      "",
			solace_META_brokerHostname:      "",
			solace_META_brokerPort:          "",
			solace_META_msgVpn:              soltest_VALID_VPN,
			solace_META_usernameEnv:         "",
			solace_META_passwordEnv:         "",
			solace_META_username:            soltest_VALID_USERNAME,
			solace_META_password:            soltest_VALID_PASSWORD,
			solace_META_queueName:           soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget:      "0",
			solace_META_msgSpoolUsageTarget: soltest_VALID_MSG_SPOOL_TARGET,
		},
		false,
	},
}

var testSolaceExpectedMetricNames = map[string]string{
	solace_SCALER_ID + "-" + soltest_VALID_VPN + "-" + soltest_VALID_QUEUE_NAME + "-" + "msgCount":      "",
	solace_SCALER_ID + "-" + soltest_VALID_VPN + "-" + soltest_VALID_QUEUE_NAME + "-" + "msgSpoolUsage": "",
}

var testSolaceSEMPConnectionMetadata = []testSolaceMetadata{
	/*
		IF brokerBaseUrl is present, use it without interpretation as the base URL: http://my.host.domain:1234
		IF brokerBaseUrl in not present, Use protocol, host, and port
	*/

	// +Case - Should Connect (Base URL Provided)
	{
		"#301 - Connect w/ baseUrl + Clear Auth",
		map[string]string{
			"":                         "",
			solace_META_brokerBaseUrl:  soltest_VALID_BASEURL,
			solace_META_brokerProtocol: "",
			solace_META_brokerHostname: "",
			solace_META_brokerPort:     "",
			solace_META_msgVpn:         soltest_VALID_VPN,
			solace_META_usernameEnv:    "",
			solace_META_passwordEnv:    "",
			solace_META_username:       soltest_VALID_USERNAME,
			solace_META_password:       soltest_VALID_PASSWORD,
			solace_META_queueName:      soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget: soltest_VALID_MSG_COUNT_TARGET,
		},
		false,
	},
	{
		"#302 - Connect with invalid username",
		map[string]string{
			solace_META_brokerBaseUrl:       "",
			solace_META_brokerProtocol:      soltest_VALID_PROTOCOL,
			solace_META_brokerHostname:      soltest_VALID_HOSTNAME,
			solace_META_brokerPort:          soltest_VALID_PORT,
			solace_META_username:            "NOT_A_REAL_USER",
			solace_META_password:            soltest_VALID_PASSWORD,
			solace_META_msgVpn:              soltest_VALID_VPN,
			solace_META_queueName:           soltest_VALID_QUEUE_NAME,
			solace_META_msgSpoolUsageTarget: soltest_VALID_MSG_SPOOL_TARGET,
		},
		true,
	},
	{
		"#303 - Connect with invalid password",
		map[string]string{
			solace_META_brokerBaseUrl:       "",
			solace_META_brokerProtocol:      soltest_VALID_PROTOCOL,
			solace_META_brokerHostname:      soltest_VALID_HOSTNAME,
			solace_META_brokerPort:          soltest_VALID_PORT,
			solace_META_username:            soltest_VALID_USERNAME,
			solace_META_password:            "THIS_IS_NOT_MY_PASSWORD",
			solace_META_msgVpn:              soltest_VALID_VPN,
			solace_META_queueName:           soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget:      soltest_VALID_MSG_COUNT_TARGET,
			solace_META_msgSpoolUsageTarget: soltest_VALID_MSG_SPOOL_TARGET,
		},
		true,
	},
	// +Case - Should Connect (Derived Base URL)
	{
		"#304 - Connect with component URL + Env Var Auth",
		map[string]string{
			solace_META_brokerBaseUrl:  "",
			solace_META_brokerProtocol: soltest_VALID_PROTOCOL,
			solace_META_brokerHostname: soltest_VALID_HOSTNAME,
			solace_META_brokerPort:     soltest_VALID_PORT,
			solace_META_msgVpn:         soltest_VALID_VPN,
			solace_META_usernameEnv:    "",
			solace_META_passwordEnv:    "",
			solace_META_username:       soltest_VALID_USERNAME,
			solace_META_password:       soltest_VALID_PASSWORD,
			solace_META_queueName:      soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget: soltest_VALID_MSG_COUNT_TARGET,
		},
		false,
	},
	{
		"#305 - Connect with component URL + K8S Auth Record",
		map[string]string{
			solace_META_brokerBaseUrl:  "",
			solace_META_brokerProtocol: soltest_VALID_PROTOCOL,
			solace_META_brokerHostname: soltest_VALID_HOSTNAME,
			solace_META_brokerPort:     soltest_VALID_PORT,
			solace_META_msgVpn:         soltest_VALID_VPN,
			solace_META_usernameEnv:    "",
			solace_META_passwordEnv:    "",
			//		solace_META_username:              soltest_VALID_USERNAME,
			//		solace_META_password:              soltest_VALID_PASSWORD,
			solace_META_queueName:      soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget: soltest_VALID_MSG_COUNT_TARGET,
		},
		false,
	},
	// -Case - Solace VPN DNE
	{
		"#306 - Solace VPN DNE",
		map[string]string{
			solace_META_brokerBaseUrl:  "",
			solace_META_brokerProtocol: soltest_VALID_PROTOCOL,
			solace_META_brokerHostname: soltest_VALID_HOSTNAME,
			solace_META_brokerPort:     soltest_VALID_PORT,
			solace_META_msgVpn:         "THIS_VPN_DOES_NOT_EXIST",
			solace_META_usernameEnv:    "",
			solace_META_passwordEnv:    "",
			solace_META_username:       soltest_VALID_USERNAME,
			solace_META_password:       soltest_VALID_PASSWORD,
			solace_META_queueName:      soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget: soltest_VALID_MSG_COUNT_TARGET,
		},
		true,
	},
	// -Case - Queue DNE
	{
		"#307 - Solace Queue DNE",
		map[string]string{
			solace_META_brokerBaseUrl:  "",
			solace_META_brokerProtocol: soltest_VALID_PROTOCOL,
			solace_META_brokerHostname: soltest_VALID_HOSTNAME,
			solace_META_brokerPort:     soltest_VALID_PORT,
			solace_META_msgVpn:         soltest_VALID_VPN,
			solace_META_usernameEnv:    "",
			solace_META_passwordEnv:    "",
			solace_META_username:       soltest_VALID_USERNAME,
			solace_META_password:       soltest_VALID_PASSWORD,
			solace_META_queueName:      "THIS_QUEUE_DOES_NOT_EXIST",
			solace_META_msgCountTarget: soltest_VALID_MSG_COUNT_TARGET,
		},
		true,
	},
	// -Case - Bad baseUrl
	{
		"#308 - Bad Base URL",
		map[string]string{
			solace_META_brokerBaseUrl:  "http://not-a-real-server.nothing:999",
			solace_META_brokerProtocol: "",
			solace_META_brokerHostname: "",
			solace_META_brokerPort:     "",
			solace_META_msgVpn:         soltest_VALID_VPN,
			solace_META_usernameEnv:    "",
			solace_META_passwordEnv:    "",
			solace_META_username:       soltest_VALID_USERNAME,
			solace_META_password:       soltest_VALID_PASSWORD,
			solace_META_queueName:      soltest_VALID_QUEUE_NAME,
			solace_META_msgCountTarget: soltest_VALID_MSG_COUNT_TARGET,
		},
		true,
	},
}

func TestSolaceParseSolaceMetadata(t *testing.T) {
	for _, testData := range testParseSolaceMetadata {
		fmt.Print(testData.testId)
		_, err := parseSolaceMetadata(&ScalerConfig{ResolvedEnv: nil, TriggerMetadata: testData.metadata, AuthParams: nil})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error: ", err)
			fmt.Println(" --> FAIL")
		} else if testData.isError && err == nil {
			t.Error("Expected error but got success")
			fmt.Println(" --> FAIL")
		} else {
			fmt.Println(" --> PASS")
		}
	}
	for _, testData := range testSolaceEnvCreds {
		fmt.Print(testData.testId)
		_, err := parseSolaceMetadata(&ScalerConfig{ResolvedEnv: testDataSolaceResolvedEnvVALID, TriggerMetadata: testData.metadata, AuthParams: nil})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error: ", err)
			fmt.Println(" --> FAIL")
		} else if testData.isError && err == nil {
			t.Error("Expected error but got success")
			fmt.Println(" --> FAIL")
		} else {
			fmt.Println(" --> PASS")
		}
	}
	for _, testData := range testSolaceK8sSecretCreds {
		fmt.Print(testData.testId)
		_, err := parseSolaceMetadata(&ScalerConfig{ResolvedEnv: nil, TriggerMetadata: testData.metadata, AuthParams: testDataSolaceAuthParamsVALID})
		if err != nil && !testData.isError {
			t.Error("Expected success but result is error: ", err)
			fmt.Println(" --> FAIL")
		} else if testData.isError && err == nil {
			t.Error("Expected error but result is success")
			fmt.Println(" --> FAIL")
		} else {
			fmt.Println(" --> PASS")
		}
	}
}

func TestSolaceGetMetricSpec(t *testing.T) {
	for idx := 0; idx < len(testSolaceGetMetricSpecData); idx++ {

		testData := testSolaceGetMetricSpecData[idx]
		fmt.Print(testData.testId)
		var err error
		var solaceMeta *SolaceMetadata
		if idx == 0 {
			// The first instance will have nil ResolvedEnv and AuthParams -- for Clear Auth
			solaceMeta, err = parseSolaceMetadata(&ScalerConfig{ResolvedEnv: nil, TriggerMetadata: testData.metadata, AuthParams: nil})
		} else {
			// The first instance will have nil ResolvedEnv and AuthParams -- for Clear Auth
			solaceMeta, err = parseSolaceMetadata(&ScalerConfig{ResolvedEnv: testDataSolaceResolvedEnvVALID, TriggerMetadata: testData.metadata, AuthParams: testDataSolaceAuthParamsVALID})
		}
		if err != nil {
			fmt.Printf("\n       Failed to parse metadata: %v", err)
			//			t.Error("Failed to parse metadata: ", err)
		} else {

			//			fmt.Println("Here")
			//			fmt.Println(solaceMeta.brokerBaseUrl)
			//			fmt.Printf("%v\n", solaceMeta.msgCountTarget)

			//			var startTime metav1.Time = metav1.Now()
			//			startTime.Time.Add(time.Duration(-2) * time.Second)

			// DECLARE SCALER AND RUN METHOD TO GET METRICS
			testSolaceScaler := SolaceScaler{
				metadata:   solaceMeta,
				httpClient: http.DefaultClient,
			}

			var metric []v2beta2.MetricSpec
			if metric = testSolaceScaler.GetMetricSpecForScaling(); metric == nil || len(metric) == 0 {
				t.Error("Metric value not found")
			}
			metricName := metric[0].External.Metric.Name
			metricValue := metric[0].External.Target.AverageValue
			if _, ok := testSolaceExpectedMetricNames[metricName]; ok == false {
				t.Error("Expected Metric value not found")
			}

			fmt.Printf("\n       Found Metric: %s=%v", metricName, metricValue)
		}

		if testData.isError && err == nil {
			fmt.Println(" --> FAIL")
			t.Error("Expected to fail but passed", err)
		} else if !testData.isError && err != nil {
			t.Error("Expected success but failed", err)
		} else {
			fmt.Println(" --> PASS")
		}

	}
}

func TestSolaceSEMPConnection(t *testing.T) {
	for idx := 0; idx < len(testSolaceSEMPConnectionMetadata); idx++ {

		testData := testSolaceSEMPConnectionMetadata[idx]
		fmt.Print(testData.testId)
		var err error
		var solaceMeta *SolaceMetadata
		if idx < 3 {
			// The first instance will have nil ResolvedEnv and AuthParams -- for Clear Auth
			solaceMeta, err = parseSolaceMetadata(&ScalerConfig{ResolvedEnv: nil, TriggerMetadata: testData.metadata, AuthParams: nil})
		} else {
			// The first instance will have nil ResolvedEnv and AuthParams -- for Clear Auth
			solaceMeta, err = parseSolaceMetadata(&ScalerConfig{ResolvedEnv: testDataSolaceResolvedEnvVALID, TriggerMetadata: testData.metadata, AuthParams: testDataSolaceAuthParamsVALID})
		}
		if err != nil {
			t.Fatal("FAILED to parse metadata: ", err)
		}

		testSolaceScaler := SolaceScaler{
			metadata:   solaceMeta,
			httpClient: http.DefaultClient,
		}

		val, err := testSolaceScaler.getSolaceQueueMetricsFromSEMP()

		if testData.isError && err == nil {
			fmt.Println(" --> FAIL")
			t.Error("Expected to fail but passed", err)
		} else if !testData.isError && err != nil {
			fmt.Println(" --> FAIL: Connection to SEMP Failed")
			t.Error("Expected successful connection to SEMP Failed", err)
		} else {
			fmt.Printf(" --> PASS")
			if val.msgCount > 0 {
				fmt.Printf("; msgCount=%d", val.msgCount)
			}
			if val.msgSpoolUsage > 0 {
				fmt.Printf("; msgSpoolUsage=%d", val.msgSpoolUsage)
			}
			fmt.Println()
		}
	}
}
