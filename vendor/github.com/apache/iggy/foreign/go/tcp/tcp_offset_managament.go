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
)

func (tms *IggyTcpClient) GetConsumerOffset(consumer iggcon.Consumer, streamId iggcon.Identifier, topicId iggcon.Identifier, partitionId *uint32) (*iggcon.ConsumerOffsetInfo, error) {
	message := binaryserialization.GetOffset(iggcon.GetConsumerOffsetRequest{
		StreamId:    streamId,
		TopicId:     topicId,
		Consumer:    consumer,
		PartitionId: partitionId,
	})
	buffer, err := tms.sendAndFetchResponse(message, iggcon.GetOffsetCode)
	if err != nil {
		return nil, err
	}

	return binaryserialization.DeserializeOffset(buffer), nil
}

func (tms *IggyTcpClient) StoreConsumerOffset(consumer iggcon.Consumer, streamId iggcon.Identifier, topicId iggcon.Identifier, offset uint64, partitionId *uint32) error {
	message := binaryserialization.UpdateOffset(iggcon.StoreConsumerOffsetRequest{
		StreamId:    streamId,
		TopicId:     topicId,
		Offset:      offset,
		Consumer:    consumer,
		PartitionId: partitionId,
	})
	_, err := tms.sendAndFetchResponse(message, iggcon.StoreOffsetCode)
	return err
}

func (tms *IggyTcpClient) DeleteConsumerOffset(consumer iggcon.Consumer, streamId iggcon.Identifier, topicId iggcon.Identifier, partitionId *uint32) error {
	message := binaryserialization.DeleteOffset(iggcon.DeleteConsumerOffsetRequest{
		Consumer:    consumer,
		StreamId:    streamId,
		TopicId:     topicId,
		PartitionId: partitionId,
	})
	_, err := tms.sendAndFetchResponse(message, iggcon.DeleteConsumerOffsetCode)
	return err
}
