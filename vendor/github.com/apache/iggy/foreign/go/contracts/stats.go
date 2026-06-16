// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package iggcon

import (
	"fmt"

	"github.com/apache/iggy/foreign/go/internal/codec"
)

// CacheMetrics holds cache hit/miss statistics for a single partition.
type CacheMetrics struct {
	StreamId    uint32  `json:"stream_id"`
	TopicId     uint32  `json:"topic_id"`
	PartitionId uint32  `json:"partition_id"`
	Hits        uint64  `json:"hits"`
	Misses      uint64  `json:"misses"`
	HitRatio    float32 `json:"hit_ratio"`
}

// cacheMetricsWireSize is the fixed size of a CacheMetrics entry on the wire:
// stream_id(4) + topic_id(4) + partition_id(4) + hits(8) + misses(8) + hit_ratio(4) = 32.
const cacheMetricsWireSize = 4 + 4 + 4 + 8 + 8 + 4

type Stats struct {
	ProcessId           uint32         `json:"process_id"`
	CpuUsage            float32        `json:"cpu_usage"`
	TotalCpuUsage       float32        `json:"total_cpu_usage"`
	MemoryUsage         uint64         `json:"memory_usage"`
	TotalMemory         uint64         `json:"total_memory"`
	AvailableMemory     uint64         `json:"available_memory"`
	RunTime             uint64         `json:"run_time"`
	StartTime           uint64         `json:"start_time"`
	ReadBytes           uint64         `json:"read_bytes"`
	WrittenBytes        uint64         `json:"written_bytes"`
	MessagesSizeBytes   uint64         `json:"messages_size_bytes"`
	StreamsCount        uint32         `json:"streams_count"`
	TopicsCount         uint32         `json:"topics_count"`
	PartitionsCount     uint32         `json:"partitions_count"`
	SegmentsCount       uint32         `json:"segments_count"`
	MessagesCount       uint64         `json:"messages_count"`
	ClientsCount        uint32         `json:"clients_count"`
	ConsumerGroupsCount uint32         `json:"consumer_groups_count"`
	Hostname            string         `json:"hostname"`
	OsName              string         `json:"os_name"`
	OsVersion           string         `json:"os_version"`
	KernelVersion       string         `json:"kernel_version"`
	IggyServerVersion   string         `json:"iggy_server_version"`
	IggyServerSemver    uint32         `json:"iggy_server_semver"`
	CacheMetrics        []CacheMetrics `json:"cache_metrics"`
	ThreadsCount        uint32         `json:"threads_count"`
	FreeDiskSpace       uint64         `json:"free_disk_space"`
	TotalDiskSpace      uint64         `json:"total_disk_space"`
}

func (cm *CacheMetrics) MarshalBinary() ([]byte, error) {
	w := codec.NewWriterCap(32)
	w.U32(cm.StreamId)
	w.U32(cm.TopicId)
	w.U32(cm.PartitionId)
	w.U64(cm.Hits)
	w.U64(cm.Misses)
	w.F32(cm.HitRatio)
	return w.Bytes(), w.Err()
}

func (cm *CacheMetrics) UnmarshalBinary(data []byte) error {
	r := codec.NewReader(data)
	cm.StreamId = r.U32()
	cm.TopicId = r.U32()
	cm.PartitionId = r.U32()
	cm.Hits = r.U64()
	cm.Misses = r.U64()
	cm.HitRatio = r.F32()
	return r.Err()
}

func (s *Stats) UnmarshalBinary(payload []byte) error {
	r := codec.NewReader(payload)
	s.ProcessId = r.U32()
	s.CpuUsage = r.F32()
	s.TotalCpuUsage = r.F32()
	s.MemoryUsage = r.U64()
	s.TotalMemory = r.U64()
	s.AvailableMemory = r.U64()
	s.RunTime = r.U64()
	s.StartTime = r.U64()
	s.ReadBytes = r.U64()
	s.WrittenBytes = r.U64()
	s.MessagesSizeBytes = r.U64()
	s.StreamsCount = r.U32()
	s.TopicsCount = r.U32()
	s.PartitionsCount = r.U32()
	s.SegmentsCount = r.U32()
	s.MessagesCount = r.U64()
	s.ClientsCount = r.U32()
	s.ConsumerGroupsCount = r.U32()
	s.Hostname = r.U32LenStr()
	s.OsName = r.U32LenStr()
	s.OsVersion = r.U32LenStr()
	s.KernelVersion = r.U32LenStr()
	s.IggyServerVersion = r.U32LenStr()
	s.IggyServerSemver = r.U32()
	cacheCount := int(r.U32())
	if r.Err() != nil {
		return r.Err()
	}

	if cacheCount > r.Remaining()/cacheMetricsWireSize {
		return fmt.Errorf("stats: cache metrics count %d exceeds remaining bytes %d", cacheCount, r.Remaining())
	}

	if cacheCount > 0 {
		s.CacheMetrics = make([]CacheMetrics, cacheCount)
		for i := range s.CacheMetrics {
			r.Obj(cacheMetricsWireSize, &s.CacheMetrics[i])
		}
	}

	s.ThreadsCount = r.U32()
	s.FreeDiskSpace = r.U64()
	s.TotalDiskSpace = r.U64()

	return r.Err()
}
