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

import "time"

type CreateTopicRequest struct {
	StreamId             Identifier    `json:"streamId"`
	TopicId              uint32        `json:"topicId"`
	PartitionsCount      int           `json:"partitionsCount"`
	CompressionAlgorithm uint8         `json:"compressionAlgorithm"`
	MessageExpiry        time.Duration `json:"messageExpiry"`
	MaxTopicSize         uint64        `json:"maxTopicSize"`
	ReplicationFactor    uint8         `json:"replicationFactor"`
	Name                 string        `json:"name"`
}

type UpdateTopicRequest struct {
	StreamId             Identifier    `json:"streamId"`
	TopicId              Identifier    `json:"topicId"`
	CompressionAlgorithm uint8         `json:"compressionAlgorithm"`
	MessageExpiry        time.Duration `json:"messageExpiry"`
	MaxTopicSize         uint64        `json:"maxTopicSize"`
	ReplicationFactor    uint8         `json:"replicationFactor"`
	Name                 string        `json:"name"`
}

type Topic struct {
	Id                   uint32   `json:"id"`
	CreatedAt            uint64   `json:"createdAt"`
	Name                 string   `json:"name"`
	Size                 uint64   `json:"size"`
	MessageExpiry        Duration `json:"messageExpiry"`
	CompressionAlgorithm uint8    `json:"compressionAlgorithm"`
	MaxTopicSize         uint64   `json:"maxTopicSize"`
	ReplicationFactor    uint8    `json:"replicationFactor"`
	MessagesCount        uint64   `json:"messagesCount"`
	PartitionsCount      uint32   `json:"partitionsCount"`
}

type TopicDetails struct {
	Topic
	Partitions []PartitionContract `json:"partitions,omitempty"`
}
