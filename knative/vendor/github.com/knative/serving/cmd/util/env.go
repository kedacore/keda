/*
Copyright 2018 The Knative Authors

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
	"os"
	"strconv"

	"go.uber.org/zap"
)

// GetRequiredEnvOrFatal tries to get the value of an environment variable.
// If the value is empty, it logs an error and calls os.Exit(1).
func GetRequiredEnvOrFatal(key string, logger *zap.SugaredLogger) string {
	value := os.Getenv(key)
	if value == "" {
		logger.Fatalf("No %v provided", key)
	}
	logger.Infof("%v=%v", key, value)
	return value
}

func MustParseIntEnvOrFatal(key string, logger *zap.SugaredLogger) int {
	value := GetRequiredEnvOrFatal(key, logger)
	i, err := strconv.Atoi(value)
	if err != nil {
		logger.Fatalf("Invalid %v provided: %v", key, value)
	}
	logger.Infof("%v=%v", key, i)
	return i
}
