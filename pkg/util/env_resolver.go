package util

import (
	"os"
	"strconv"
	"time"
)

func ResolveOsEnvInt(envName string, defaultValue int) (int, error) {
	valueStr, found := os.LookupEnv(envName)

	if found && valueStr != "" {
		return strconv.Atoi(valueStr)
	}

	return defaultValue, nil
}

func ResolveOsEnvDuration(envName string, defaultValue time.Duration) (time.Duration, error) {
	valueStr, found := os.LookupEnv(envName)

	if found && valueStr != "" {
		return time.ParseDuration(valueStr)
	}

	return defaultValue, nil
}
