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
	"fmt"
	"os"
	"strconv"
	"time"

	"k8s.io/utils/ptr"
)

const RestrictSecretAccessEnvVar = "KEDA_RESTRICT_SECRET_ACCESS"
const BoundServiceAccountTokenExpiryEnvVar = "KEDA_BOUND_SERVICE_ACCOUNT_TOKEN_EXPIRY"

var clusterObjectNamespaceCache *string

func ResolveOsEnvBool(envName string, defaultValue bool) (bool, error) {
	valueStr, found := os.LookupEnv(envName)

	if found && valueStr != "" {
		return strconv.ParseBool(valueStr)
	}

	return defaultValue, nil
}

func ResolveOsEnvInt(envName string, defaultValue int) (int, error) {
	valueStr, found := os.LookupEnv(envName)

	if found && valueStr != "" {
		return strconv.Atoi(valueStr)
	}

	return defaultValue, nil
}

func ResolveOsEnvDuration(envName string) (*time.Duration, error) {
	valueStr, found := os.LookupEnv(envName)

	if found && valueStr != "" {
		value, err := time.ParseDuration(valueStr)
		return &value, err
	}

	return nil, nil
}

// GetClusterObjectNamespace retrieves the cluster object namespace of KEDA, default is the namespace of KEDA Operator & Metrics Server
func GetClusterObjectNamespace() (string, error) {
	// Check if a cached value is available.
	if clusterObjectNamespaceCache != nil {
		return *clusterObjectNamespaceCache, nil
	}
	env := os.Getenv("KEDA_CLUSTER_OBJECT_NAMESPACE")
	if env != "" {
		clusterObjectNamespaceCache = &env
		return env, nil
	}
	data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", err
	}
	strData := string(data)
	clusterObjectNamespaceCache = &strData
	return strData, nil
}

// GetPodNamespace returns the namespace for the pod
func GetPodNamespace() string {
	ns, found := os.LookupEnv("POD_NAMESPACE")
	if !found {
		return "keda"
	}
	return ns
}

// GetRestrictSecretAccess retrieves the value of the environment variable of KEDA_RESTRICT_SECRET_ACCESS
func GetRestrictSecretAccess() string {
	return os.Getenv(RestrictSecretAccessEnvVar)
}

// GetBoundServiceAccountTokenExpiry retrieves the value of the environment variable of KEDA_BOUND_SERVICE_ACCOUNT_TOKEN_EXPIRY
func GetBoundServiceAccountTokenExpiry() (*time.Duration, error) {
	expiry, err := ResolveOsEnvDuration(BoundServiceAccountTokenExpiryEnvVar)
	if err != nil {
		return nil, err
	}
	if expiry == nil {
		return ptr.To[time.Duration](time.Hour), nil // if blank, default to 1 hour
	}
	if *expiry < time.Hour || *expiry > 6*time.Hour {
		return nil, fmt.Errorf("invalid value for %s: %s, must be between 1h and 6h", BoundServiceAccountTokenExpiryEnvVar, expiry.String()) // Must be between 1 hour and 6 hours
	}
	return expiry, nil
}
