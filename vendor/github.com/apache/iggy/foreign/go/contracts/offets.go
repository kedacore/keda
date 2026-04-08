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

type StoreConsumerOffsetRequest struct {
	StreamId    Identifier `json:"streamId"`
	TopicId     Identifier `json:"topicId"`
	Consumer    Consumer   `json:"consumer"`
	PartitionId *uint32    `json:"partitionId"`
	Offset      uint64     `json:"offset"`
}

type GetConsumerOffsetRequest struct {
	StreamId    Identifier `json:"streamId"`
	TopicId     Identifier `json:"topicId"`
	Consumer    Consumer   `json:"consumer"`
	PartitionId *uint32    `json:"partitionId"`
}

type ConsumerOffsetInfo struct {
	PartitionId   uint32 `json:"partitionId"`
	CurrentOffset uint64 `json:"currentOffset"`
	StoredOffset  uint64 `json:"storedOffset"`
}

type DeleteConsumerOffsetRequest struct {
	Consumer    Consumer
	StreamId    Identifier
	TopicId     Identifier
	PartitionId *uint32
}
