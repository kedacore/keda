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
	"github.com/apache/iggy/foreign/go/internal/command"
)

func (c *IggyTcpClient) GetConsumerOffset(consumer iggcon.Consumer, streamId iggcon.Identifier, topicId iggcon.Identifier, partitionId *uint32) (*iggcon.ConsumerOffsetInfo, error) {
	buffer, err := c.do(&command.GetConsumerOffset{
		StreamId:    streamId,
		TopicId:     topicId,
		Consumer:    consumer,
		PartitionId: partitionId,
	})
	if err != nil {
		return nil, err
	}

	return binaryserialization.DeserializeOffset(buffer), nil
}

func (c *IggyTcpClient) StoreConsumerOffset(consumer iggcon.Consumer, streamId iggcon.Identifier, topicId iggcon.Identifier, offset uint64, partitionId *uint32) error {
	_, err := c.do(&command.StoreConsumerOffsetRequest{
		StreamId:    streamId,
		TopicId:     topicId,
		Offset:      offset,
		Consumer:    consumer,
		PartitionId: partitionId,
	})
	return err
}

func (c *IggyTcpClient) DeleteConsumerOffset(consumer iggcon.Consumer, streamId iggcon.Identifier, topicId iggcon.Identifier, partitionId *uint32) error {
	_, err := c.do(&command.DeleteConsumerOffset{
		Consumer:    consumer,
		StreamId:    streamId,
		TopicId:     topicId,
		PartitionId: partitionId,
	})
	return err
}
