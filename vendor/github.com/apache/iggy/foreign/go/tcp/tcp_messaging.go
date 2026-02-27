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

func (tms *IggyTcpClient) SendMessages(
	streamId iggcon.Identifier,
	topicId iggcon.Identifier,
	partitioning iggcon.Partitioning,
	messages []iggcon.IggyMessage,
) error {
	if len(partitioning.Value) > 255 ||
		(partitioning.Kind != iggcon.Balanced && len(partitioning.Value) == 0) {
		return ierror.ErrInvalidKeyValueLength
	}
	if len(messages) == 0 {
		return ierror.ErrInvalidMessagesCount
	}
	serializedRequest := binaryserialization.TcpSendMessagesRequest{
		StreamId:     streamId,
		TopicId:      topicId,
		Partitioning: partitioning,
		Messages:     messages,
	}
	_, err := tms.sendAndFetchResponse(serializedRequest.Serialize(tms.MessageCompression), iggcon.SendMessagesCode)
	return err
}

func (tms *IggyTcpClient) PollMessages(
	streamId iggcon.Identifier,
	topicId iggcon.Identifier,
	consumer iggcon.Consumer,
	strategy iggcon.PollingStrategy,
	count uint32,
	autoCommit bool,
	partitionId *uint32,
) (*iggcon.PolledMessage, error) {
	serializedRequest := binaryserialization.TcpFetchMessagesRequest{
		StreamId:    streamId,
		TopicId:     topicId,
		Consumer:    consumer,
		AutoCommit:  autoCommit,
		Strategy:    strategy,
		Count:       count,
		PartitionId: partitionId,
	}
	buffer, err := tms.sendAndFetchResponse(serializedRequest.Serialize(), iggcon.PollMessagesCode)
	if err != nil {
		return nil, err
	}

	return binaryserialization.DeserializeFetchMessagesResponse(buffer, tms.MessageCompression)
}
