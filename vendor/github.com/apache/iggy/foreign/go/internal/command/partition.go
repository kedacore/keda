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

type CreatePartitions struct {
	StreamId        iggcon.Identifier `json:"streamId"`
	TopicId         iggcon.Identifier `json:"topicId"`
	PartitionsCount uint32            `json:"partitionsCount"`
}

func (c *CreatePartitions) Code() Code {
	return CreatePartitionsCode
}

func (c *CreatePartitions) MarshalBinary() ([]byte, error) {
	streamIdBytes, err := c.StreamId.MarshalBinary()
	if err != nil {
		return nil, err
	}
	topicIdBytes, err := c.TopicId.MarshalBinary()
	if err != nil {
		return nil, err
	}
	bytes := make([]byte, len(streamIdBytes)+len(topicIdBytes)+4)
	position := 0
	copy(bytes[position:], streamIdBytes)
	position += len(streamIdBytes)
	copy(bytes[position:], topicIdBytes)
	position += len(topicIdBytes)
	binary.LittleEndian.PutUint32(bytes[position:position+4], c.PartitionsCount)

	return bytes, nil
}

type DeletePartitions struct {
	StreamId        iggcon.Identifier `json:"streamId"`
	TopicId         iggcon.Identifier `json:"topicId"`
	PartitionsCount uint32            `json:"partitionsCount"`
}

func (d *DeletePartitions) Code() Code {
	return DeletePartitionsCode
}

func (d *DeletePartitions) MarshalBinary() ([]byte, error) {
	streamIdBytes, err := d.StreamId.MarshalBinary()
	if err != nil {
		return nil, err
	}
	topicIdBytes, err := d.TopicId.MarshalBinary()
	if err != nil {
		return nil, err
	}
	bytes := make([]byte, len(streamIdBytes)+len(topicIdBytes)+4)
	position := 0
	copy(bytes[position:], streamIdBytes)
	position += len(streamIdBytes)
	copy(bytes[position:], topicIdBytes)
	position += len(topicIdBytes)
	binary.LittleEndian.PutUint32(bytes[position:position+4], d.PartitionsCount)

	return bytes, nil
}
