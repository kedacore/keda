//go:build e2e
// +build e2e

package utils

import (
	"fmt"
	"os"
	"testing"

	. "github.com/kedacore/keda/v2/tests/helper"
)

func TestMain(m *testing.M) {
	var err error
	KubeClient, err = GetKubernetesClientOrError()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error getting kubernetes client - %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
