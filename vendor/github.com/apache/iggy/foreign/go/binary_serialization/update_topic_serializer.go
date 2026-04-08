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

package binaryserialization

import (
	"encoding/binary"

	iggcon "github.com/apache/iggy/foreign/go/contracts"
)

type TcpUpdateTopicRequest struct {
	StreamId             iggcon.Identifier           `json:"streamId"`
	TopicId              iggcon.Identifier           `json:"topicId"`
	CompressionAlgorithm iggcon.CompressionAlgorithm `json:"compressionAlgorithm"`
	MessageExpiry        iggcon.Duration             `json:"messageExpiry"`
	MaxTopicSize         uint64                      `json:"maxTopicSize"`
	ReplicationFactor    *uint8                      `json:"replicationFactor"`
	Name                 string                      `json:"name"`
}

func (request *TcpUpdateTopicRequest) Serialize() []byte {
	if request.ReplicationFactor == nil {
		request.ReplicationFactor = new(uint8)
	}
	streamIdBytes := SerializeIdentifier(request.StreamId)
	topicIdBytes := SerializeIdentifier(request.TopicId)

	buffer := make([]byte, 19+len(streamIdBytes)+len(topicIdBytes)+len(request.Name))

	offset := 0

	offset += copy(buffer[offset:], streamIdBytes)
	offset += copy(buffer[offset:], topicIdBytes)

	buffer[offset] = byte(request.CompressionAlgorithm)
	offset++

	binary.LittleEndian.PutUint64(buffer[offset:], uint64(request.MessageExpiry))
	offset += 8

	binary.LittleEndian.PutUint64(buffer[offset:], request.MaxTopicSize)
	offset += 8

	buffer[offset] = *request.ReplicationFactor
	offset++

	buffer[offset] = uint8(len(request.Name))
	offset++

	copy(buffer[offset:], request.Name)

	return buffer
}
