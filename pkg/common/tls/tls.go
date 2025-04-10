package tls

import (
	ctls "crypto/tls"
	"fmt"
	"os"
)

type tlsVersion string

const (
	TLS10 tlsVersion = "TLS10"
	TLS11 tlsVersion = "TLS11"
	TLS12 tlsVersion = "TLS12"
	TLS13 tlsVersion = "TLS13"
)

type tlsEnvVariableName string

const (
	minHTTPTLSVersionEnv tlsEnvVariableName = "KEDA_HTTP_MIN_TLS_VERSION"
	minGrpcTLSVersionEnv tlsEnvVariableName = "KEDA_GRPC_MIN_TLS_VERSION"
)

const (
	defaultMinHTTPTLSVersion = TLS12
	defaultMinGrpcTLSVersion = TLS13
)

func getMinTLSVersion(envKey tlsEnvVariableName, defaultVal tlsVersion) (uint16, error) {
	version := string(defaultVal)
	if val, ok := os.LookupEnv(string(envKey)); ok {
		version = val
	}
	mapping := map[string]uint16{
		string(TLS10): ctls.VersionTLS10,
		string(TLS11): ctls.VersionTLS11,
		string(TLS12): ctls.VersionTLS12,
		string(TLS13): ctls.VersionTLS13,
	}
	if v, ok := mapping[version]; ok {
		return v, nil
	}
	fallback := mapping[string(defaultVal)]
	return fallback, fmt.Errorf("invalid TLS version: %s, using %s", version, defaultVal)
}

func GetMinHTTPTLSVersion() (uint16, error) {
	return getMinTLSVersion(minHTTPTLSVersionEnv, defaultMinHTTPTLSVersion)
}

func GetMinGrpcTLSVersion() (uint16, error) {
	return getMinTLSVersion(minGrpcTLSVersionEnv, defaultMinGrpcTLSVersion)
}
