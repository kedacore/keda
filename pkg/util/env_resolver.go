package util

import (
	"os"
	"strconv"
)

func ResolveOsEnvInt(envName string, defaultValue int) (int, error) {
	valueStr, found := os.LookupEnv(envName)

	if found {
		return strconv.Atoi(valueStr)
	}

	return defaultValue, nil
}
