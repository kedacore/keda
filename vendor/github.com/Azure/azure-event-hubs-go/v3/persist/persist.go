// Package persist provides abstract structures for checkpoint persistence.
package persist

import (
	"path"
	"sync"
)

type (
	// CheckpointPersister provides persistence for the received offset for a given namespace, hub name, consumer group, partition Id and
	// offset so that if a receiver where to be interrupted, it could resume after the last consumed event.
	CheckpointPersister interface {
		Write(namespace, name, consumerGroup, partitionID string, checkpoint Checkpoint) error
		Read(namespace, name, consumerGroup, partitionID string) (Checkpoint, error)
	}

	// MemoryPersister is a default implementation of a Hub CheckpointPersister, which will persist offset information in
	// memory.
	MemoryPersister struct {
		values map[string]Checkpoint
		mu     sync.Mutex
	}
)

// NewMemoryPersister creates a new in-memory storage for checkpoints
//
// MemoryPersister is only intended to be shared with EventProcessorHosts within the same process. This implementation
// is a toy. You should probably use the Azure Storage implementation or any other that provides durable storage for
// checkpoints.
func NewMemoryPersister() *MemoryPersister {
	return &MemoryPersister{
		values: make(map[string]Checkpoint),
	}
}

func (p *MemoryPersister) Write(namespace, name, consumerGroup, partitionID string, checkpoint Checkpoint) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := getPersistenceKey(namespace, name, consumerGroup, partitionID)
	p.values[key] = checkpoint
	return nil
}

func (p *MemoryPersister) Read(namespace, name, consumerGroup, partitionID string) (Checkpoint, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := getPersistenceKey(namespace, name, consumerGroup, partitionID)
	if offset, ok := p.values[key]; ok {
		return offset, nil
	}
	return NewCheckpointFromStartOfStream(), nil
}

func getPersistenceKey(namespace, name, consumerGroup, partitionID string) string {
	return path.Join(namespace, name, consumerGroup, partitionID)
}
