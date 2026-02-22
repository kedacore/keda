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

package tcp

import (
	binaryserialization "github.com/apache/iggy/foreign/go/binary_serialization"
	iggcon "github.com/apache/iggy/foreign/go/contracts"
	ierror "github.com/apache/iggy/foreign/go/errors"
)

func (tms *IggyTcpClient) GetTopics(streamId iggcon.Identifier) ([]iggcon.Topic, error) {
	message := binaryserialization.SerializeIdentifier(streamId)
	buffer, err := tms.sendAndFetchResponse(message, iggcon.GetTopicsCode)
	if err != nil {
		return nil, err
	}

	return binaryserialization.DeserializeTopics(buffer)
}

func (tms *IggyTcpClient) GetTopic(streamId iggcon.Identifier, topicId iggcon.Identifier) (*iggcon.TopicDetails, error) {
	message := binaryserialization.SerializeIdentifiers(streamId, topicId)
	buffer, err := tms.sendAndFetchResponse(message, iggcon.GetTopicCode)
	if err != nil {
		return nil, err
	}
	if len(buffer) == 0 {
		return nil, ierror.ErrTopicIdNotFound
	}

	topic, err := binaryserialization.DeserializeTopic(buffer)
	if err != nil {
		return nil, err
	}

	return topic, nil
}

func (tms *IggyTcpClient) CreateTopic(
	streamId iggcon.Identifier,
	name string,
	partitionsCount uint32,
	compressionAlgorithm iggcon.CompressionAlgorithm,
	messageExpiry iggcon.Duration,
	maxTopicSize uint64,
    replicationFactor *uint8,
) (*iggcon.TopicDetails, error) {
	if len(name) == 0 || len(name) > MaxStringLength {
		return nil, ierror.ErrInvalidTopicName
	}
	if partitionsCount > MaxPartitionCount {
		return nil, ierror.ErrTooManyPartitions
	}
	if replicationFactor != nil && *replicationFactor == 0 {
		return nil, ierror.ErrInvalidReplicationFactor
	}

	serializedRequest := binaryserialization.TcpCreateTopicRequest{
		StreamId:             streamId,
		Name:                 name,
		PartitionsCount:      partitionsCount,
		CompressionAlgorithm: compressionAlgorithm,
		MessageExpiry:        messageExpiry,
		MaxTopicSize:         maxTopicSize,
		ReplicationFactor:    replicationFactor,
	}
	buffer, err := tms.sendAndFetchResponse(serializedRequest.Serialize(), iggcon.CreateTopicCode)
	if err != nil {
		return nil, err
	}
	topic, err := binaryserialization.DeserializeTopic(buffer)
	return topic, err
}

func (tms *IggyTcpClient) UpdateTopic(
	streamId iggcon.Identifier,
	topicId iggcon.Identifier,
	name string,
	compressionAlgorithm iggcon.CompressionAlgorithm,
	messageExpiry iggcon.Duration,
	maxTopicSize uint64,
	replicationFactor *uint8,
) error {
	if len(name) == 0 || len(name) > MaxStringLength {
		return ierror.ErrInvalidTopicName
	}
	if replicationFactor != nil && *replicationFactor == 0 {
		return ierror.ErrInvalidReplicationFactor
	}
	serializedRequest := binaryserialization.TcpUpdateTopicRequest{
		StreamId:             streamId,
		TopicId:              topicId,
		CompressionAlgorithm: compressionAlgorithm,
		MessageExpiry:        messageExpiry,
		MaxTopicSize:         maxTopicSize,
		ReplicationFactor:    replicationFactor,
		Name:                 name}
	_, err := tms.sendAndFetchResponse(serializedRequest.Serialize(), iggcon.UpdateTopicCode)
	return err
}

func (tms *IggyTcpClient) DeleteTopic(streamId, topicId iggcon.Identifier) error {
	message := binaryserialization.SerializeIdentifiers(streamId, topicId)
	_, err := tms.sendAndFetchResponse(message, iggcon.DeleteTopicCode)
	return err
}
