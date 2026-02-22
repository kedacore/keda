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
	"encoding/binary"
	"errors"

	"github.com/google/uuid"
)

type PartitionContract struct {
	Id            uint32 `json:"id"`
	MessagesCount uint64 `json:"messagesCount"`
	CreatedAt     uint64 `json:"createdAt"`
	SegmentsCount uint32 `json:"segmentsCount"`
	CurrentOffset uint64 `json:"currentOffset"`
	SizeBytes     uint64 `json:"sizeBytes"`
}

type CreatePartitionsRequest struct {
	StreamId        Identifier `json:"streamId"`
	TopicId         Identifier `json:"topicId"`
	PartitionsCount uint32     `json:"partitionsCount"`
}

type DeletePartitionsRequest struct {
	StreamId        Identifier `json:"streamId"`
	TopicId         Identifier `json:"topicId"`
	PartitionsCount uint32     `json:"partitionsCount"`
}

type PartitioningKind int

const (
	Balanced        PartitioningKind = 1
	PartitionIdKind PartitioningKind = 2
	MessageKey      PartitioningKind = 3
)

type Partitioning struct {
	Kind   PartitioningKind
	Length int
	Value  []byte
}

func None() Partitioning {
	return Partitioning{
		Kind:   Balanced,
		Length: 0,
		Value:  make([]byte, 0),
	}
}

func PartitionId(value uint32) Partitioning {
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, value)

	return Partitioning{
		Kind:   PartitionIdKind,
		Length: 4,
		Value:  bytes,
	}
}

func EntityIdString(value string) (Partitioning, error) {
	if len(value) == 0 || len(value) > 255 {
		return Partitioning{}, errors.New("value has incorrect size, must be between 1 and 255")
	}

	return Partitioning{
		Kind:   MessageKey,
		Length: len(value),
		Value:  []byte(value),
	}, nil
}

func EntityIdBytes(value []byte) (Partitioning, error) {
	if len(value) == 0 || len(value) > 255 {
		return Partitioning{}, errors.New("value has incorrect size, must be between 1 and 255")
	}

	return Partitioning{
		Kind:   MessageKey,
		Length: len(value),
		Value:  value,
	}, nil
}

func EntityIdInt(value int) Partitioning {
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, uint32(value))
	return Partitioning{
		Kind:   MessageKey,
		Length: 4,
		Value:  bytes,
	}
}

func EntityIdUlong(value uint64) Partitioning {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, value)
	return Partitioning{
		Kind:   MessageKey,
		Length: 8,
		Value:  bytes,
	}
}

func EntityIdGuid(value uuid.UUID) Partitioning {
	bytes := value[:]
	return Partitioning{
		Kind:   MessageKey,
		Length: len(bytes),
		Value:  bytes,
	}
}
