//go:build e2e
// +build e2e

package pulsar_non_partitioned_topic_test

import (
	"testing"

	pulsar "github.com/kedacore/keda/v2/tests/scalers/pulsar/helper"
)

const (
	testName      = "pulsar-non-partitioned-topic-test"
	numPartitions = 0
)

func TestScaler(t *testing.T) {
	pulsar.TestScalerWithConfig(t, testName, numPartitions)
}
