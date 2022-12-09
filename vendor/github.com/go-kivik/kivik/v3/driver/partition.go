package driver

import (
	"context"
	"encoding/json"
)

// PartitionedDB is an optional interface that may be satisfied by a DB to
// support querying partitoin-specific information.
type PartitionedDB interface {
	// PartitionStats returns information about the named partition.
	PartitionStats(ctx context.Context, name string) (*PartitionStats, error)
}

// PartitionStats contains partition statistics.
type PartitionStats struct {
	DBName          string
	DocCount        int64
	DeletedDocCount int64
	Partition       string
	ActiveSize      int64
	ExternalSize    int64
	RawResponse     json.RawMessage
}
