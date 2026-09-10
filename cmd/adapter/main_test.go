package main

import (
	"flag"
	"testing"
)

func TestAdapterMetricsTLSServerFlags(t *testing.T) {
	// Test default values
	defaultFlags := flag.NewFlagSet("test-default", flag.ContinueOnError)
	var metricsCertDir string
	var enableMetricsTLSServer bool
	defaultFlags.StringVar(&metricsCertDir, "metrics-cert-dir", "/certs", "Metrics server TLS certificates dir")
	defaultFlags.BoolVar(&enableMetricsTLSServer, "enable-metrics-tls-server", false, "Enable TLS for the metrics server")

	err := defaultFlags.Parse([]string{})
	if err != nil {
		t.Fatalf("Failed to parse default flags: %v", err)
	}
	if enableMetricsTLSServer {
		t.Error("Default should be false (HTTP only)")
	}
	if metricsCertDir != "/certs" {
		t.Errorf("Default cert dir should be /certs, got: %s", metricsCertDir)
	}
}

func TestAdapterMetricsTLSServerFlagsEnabled(t *testing.T) {
	// Test enabled value
	enabledFlags := flag.NewFlagSet("test-enabled", flag.ContinueOnError)
	var metricsCertDir string
	var enableMetricsTLSServer bool
	enabledFlags.StringVar(&metricsCertDir, "metrics-cert-dir", "/certs", "Metrics server TLS certificates dir")
	enabledFlags.BoolVar(&enableMetricsTLSServer, "enable-metrics-tls-server", false, "Enable TLS for the metrics server")

	err := enabledFlags.Parse([]string{"--enable-metrics-tls-server", "--metrics-cert-dir", "/custom/certs"})
	if err != nil {
		t.Fatalf("Failed to parse enabled flags: %v", err)
	}
	if !enableMetricsTLSServer {
		t.Error("Should be true when flag is set")
	}
	if metricsCertDir != "/custom/certs" {
		t.Errorf("Should accept custom cert dir, got: %s", metricsCertDir)
	}
}
