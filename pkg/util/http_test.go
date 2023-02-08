/*
Copyright 2023 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"crypto/tls"
	"os"
	"testing"

	"github.com/go-logr/logr"
)

type minTLSVersionTestData struct {
	envSet          bool
	envValue        string
	expectedVersion uint16
}

var minTLSVersionTestDatas = []minTLSVersionTestData{
	{
		envSet:          true,
		envValue:        "TLS10",
		expectedVersion: tls.VersionTLS10,
	},
	{
		envSet:          true,
		envValue:        "TLS11",
		expectedVersion: tls.VersionTLS11,
	},
	{
		envSet:          true,
		envValue:        "TLS12",
		expectedVersion: tls.VersionTLS12,
	},
	{
		envSet:          true,
		envValue:        "TLS13",
		expectedVersion: tls.VersionTLS13,
	},
	{
		envSet:          false,
		expectedVersion: tls.VersionTLS12,
	},
}

func TestResolveMinTLSVersion(t *testing.T) {
     defer os.Unsetenv("KEDA_HTTP_MIN_TLS_VERSION")
	for _, testData := range minTLSVersionTestDatas {
		os.Unsetenv("KEDA_HTTP_MIN_TLS_VERSION")
		if testData.envSet {
			os.Setenv("KEDA_HTTP_MIN_TLS_VERSION", testData.envValue)
		}
		minVersion := initMinTLSVersion(logr.Discard())

		if testData.expectedVersion != minVersion {
			t.Error("Failed to resolve minTLSVersion correctly", "wants", testData.expectedVersion, "got", minVersion)
		}
	}
}
