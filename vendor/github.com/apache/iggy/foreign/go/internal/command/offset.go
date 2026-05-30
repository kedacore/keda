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

package command

import (
	"encoding/binary"

	iggcon "github.com/apache/iggy/foreign/go/contracts"
)

type StoreConsumerOffsetRequest struct {
	StreamId    iggcon.Identifier `json:"streamId"`
	TopicId     iggcon.Identifier `json:"topicId"`
	Consumer    iggcon.Consumer   `json:"consumer"`
	PartitionId *uint32           `json:"partitionId"`
	Offset      uint64            `json:"offset"`
}

func (s *StoreConsumerOffsetRequest) Code() Code {
	return StoreOffsetCode
}

func (s *StoreConsumerOffsetRequest) MarshalBinary() ([]byte, error) {
	hasPartition := byte(0)
	var partition uint32 = 0
	if s.PartitionId != nil {
		hasPartition = 1
		partition = *s.PartitionId
	}
	consumerBytes, err := s.Consumer.MarshalBinary()
	if err != nil {
		return nil, err
	}
	streamIdBytes, err := s.StreamId.MarshalBinary()
	if err != nil {
		return nil, err
	}
	topicIdBytes, err := s.TopicId.MarshalBinary()
	if err != nil {
		return nil, err
	}
	// consumer + stream_id + topic_id + hasPartition(1) + partition(4) + offset(8)
	bytes := make([]byte, len(consumerBytes)+len(streamIdBytes)+len(topicIdBytes)+13)
	position := 0
	copy(bytes[position:], consumerBytes)
	position += len(consumerBytes)
	copy(bytes[position:], streamIdBytes)
	position += len(streamIdBytes)
	copy(bytes[position:], topicIdBytes)
	position += len(topicIdBytes)
	bytes[position] = hasPartition
	binary.LittleEndian.PutUint32(bytes[position+1:position+5], partition)
	binary.LittleEndian.PutUint64(bytes[position+5:position+13], s.Offset)
	return bytes, nil
}

type GetConsumerOffset struct {
	StreamId    iggcon.Identifier `json:"streamId"`
	TopicId     iggcon.Identifier `json:"topicId"`
	Consumer    iggcon.Consumer   `json:"consumer"`
	PartitionId *uint32           `json:"partitionId"`
}

func (g *GetConsumerOffset) Code() Code {
	return GetOffsetCode
}

func (g *GetConsumerOffset) MarshalBinary() ([]byte, error) {
	hasPartition := byte(0)
	var partition uint32 = 0
	if g.PartitionId != nil {
		hasPartition = 1
		partition = *g.PartitionId
	}
	consumerBytes, err := g.Consumer.MarshalBinary()
	if err != nil {
		return nil, err
	}
	streamIdBytes, err := g.StreamId.MarshalBinary()
	if err != nil {
		return nil, err
	}
	topicIdBytes, err := g.TopicId.MarshalBinary()
	if err != nil {
		return nil, err
	}
	// consumer + stream_id + topic_id + hasPartition(1) + partition(4)
	bytes := make([]byte, len(consumerBytes)+len(streamIdBytes)+len(topicIdBytes)+5)
	position := 0
	copy(bytes[position:], consumerBytes)
	position += len(consumerBytes)
	copy(bytes[position:], streamIdBytes)
	position += len(streamIdBytes)
	copy(bytes[position:], topicIdBytes)
	position += len(topicIdBytes)
	bytes[position] = hasPartition
	binary.LittleEndian.PutUint32(bytes[position+1:position+5], partition)
	return bytes, nil
}

type DeleteConsumerOffset struct {
	Consumer    iggcon.Consumer
	StreamId    iggcon.Identifier
	TopicId     iggcon.Identifier
	PartitionId *uint32
}

func (d *DeleteConsumerOffset) Code() Code {
	return DeleteConsumerOffsetCode
}

func (d *DeleteConsumerOffset) MarshalBinary() ([]byte, error) {
	hasPartition := byte(0)
	var partition uint32 = 0
	if d.PartitionId != nil {
		hasPartition = 1
		partition = *d.PartitionId
	}
	consumerBytes, err := d.Consumer.MarshalBinary()
	if err != nil {
		return nil, err
	}
	streamIdBytes, err := d.StreamId.MarshalBinary()
	if err != nil {
		return nil, err
	}
	topicIdBytes, err := d.TopicId.MarshalBinary()
	if err != nil {
		return nil, err
	}
	// consumer + stream_id + topic_id + hasPartition(1) + partition(4)
	bytes := make([]byte, len(consumerBytes)+len(streamIdBytes)+len(topicIdBytes)+5)
	position := 0
	copy(bytes[position:], consumerBytes)
	position += len(consumerBytes)
	copy(bytes[position:], streamIdBytes)
	position += len(streamIdBytes)
	copy(bytes[position:], topicIdBytes)
	position += len(topicIdBytes)
	bytes[position] = hasPartition
	binary.LittleEndian.PutUint32(bytes[position+1:position+5], partition)
	return bytes, nil
}
