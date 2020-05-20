package azure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const csharpSdkCheckpoint = `{
		"Epoch": 123456,
		"Offset": "test offset",
		"Owner": "test owner",
		"PartitionId": "test partitionId",
		"SequenceNumber": 12345
	}`

const pythonSdkCheckpoint = `{
		"epoch": 123456,
		"offset": "test offset",
		"owner": "test owner",
		"partition_id": "test partitionId",
		"sequence_number": 12345
	}`

func TestGetCheckpoint(t *testing.T) {
	cckp, err := getCheckpoint([]byte(csharpSdkCheckpoint))
	if err != nil {
		t.Error(err)
	}

	pckp, err := getCheckpoint([]byte(pythonSdkCheckpoint))
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, cckp, pckp)
}
