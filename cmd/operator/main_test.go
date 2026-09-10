package main

import (
	"flag"
	"testing"
)

func TestMetricsTLSServerFlag(t *testing.T) {
	// Test default values
	defaultFlags := flag.NewFlagSet("test-default", flag.ContinueOnError)
	var enableMetricsTLSServer bool
	defaultFlags.BoolVar(&enableMetricsTLSServer, "enable-metrics-tls-server", false, "Enable TLS for the metrics server")

	err := defaultFlags.Parse([]string{})
	if err != nil {
		t.Fatalf("Failed to parse default flags: %v", err)
	}
	if enableMetricsTLSServer {
		t.Error("Default should be false (HTTP only)")
	}
}

func TestMetricsTLSServerFlagEnabled(t *testing.T) {
	// Test enabled value
	enabledFlags := flag.NewFlagSet("test-enabled", flag.ContinueOnError)
	var enableMetricsTLSServer bool
	enabledFlags.BoolVar(&enableMetricsTLSServer, "enable-metrics-tls-server", false, "Enable TLS for the metrics server")

	err := enabledFlags.Parse([]string{"--enable-metrics-tls-server"})
	if err != nil {
		t.Fatalf("Failed to parse enabled flags: %v", err)
	}
	if !enableMetricsTLSServer {
		t.Error("Should be true when flag is set")
	}
}
