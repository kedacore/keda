package tls

import (
	"crypto/tls"
	"fmt"
	"os"
	"testing"
)

type minTLSVersionTestData struct {
	name            string
	envSet          bool
	envValue        string
	expectedVersion uint16
	shouldError     bool
}

var minTLSVersionTestDatas = []minTLSVersionTestData{
	{
		name:            "Set to TLS10",
		envSet:          true,
		envValue:        "TLS10",
		expectedVersion: tls.VersionTLS10,
	},
	{
		name:            "Set to TLS11",
		envSet:          true,
		envValue:        "TLS11",
		expectedVersion: tls.VersionTLS11,
	},
	{
		name:            "Set to TLS12",
		envSet:          true,
		envValue:        "TLS12",
		expectedVersion: tls.VersionTLS12,
	},
	{
		name:            "Set to TLS13",
		envSet:          true,
		envValue:        "TLS13",
		expectedVersion: tls.VersionTLS13,
	},
	{
		name:   "No setting",
		envSet: false,
	},
	{
		name:        "Invalid settings",
		envSet:      true,
		envValue:    "TLS9",
		shouldError: true,
	},
}

func testResolveMinTLSVersion(t *testing.T, minVersionFunc func() (uint16, error), envName string, defaultVersion uint16) {
	defer os.Unsetenv(envName)
	for _, testData := range minTLSVersionTestDatas {
		name := fmt.Sprintf("%s: %s", envName, testData.name)
		t.Run(name, func(t *testing.T) {
			os.Unsetenv(envName)
			expectedVersion := defaultVersion
			if testData.expectedVersion != 0 {
				expectedVersion = testData.expectedVersion
			}
			if testData.envSet {
				os.Setenv(envName, testData.envValue)
			}
			minVersion, err := minVersionFunc()
			if testData.shouldError && err == nil {
				t.Error("Expected error but got none")
			}
			if expectedVersion != minVersion {
				t.Error("Failed to resolve minTLSVersion correctly", "wants", testData.expectedVersion, "got", minVersion)
			}
		})
	}
}
func TestResolveMinTLSVersion(t *testing.T) {
	testResolveMinTLSVersion(t, GetMinHTTPTLSVersion, "KEDA_HTTP_MIN_TLS_VERSION", tls.VersionTLS12)
	testResolveMinTLSVersion(t, GetMinGrpcTLSVersion, "KEDA_GRPC_MIN_TLS_VERSION", tls.VersionTLS13)
}
