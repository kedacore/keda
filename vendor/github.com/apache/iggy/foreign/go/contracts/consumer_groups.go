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

type ConsumerGroup struct {
	Id              uint32 `json:"id"`
	Name            string `json:"name"`
	PartitionsCount uint32 `json:"partitionsCount"`
	MembersCount    uint32 `json:"membersCount"`
}

type ConsumerGroupDetails struct {
	ConsumerGroup
	Members []ConsumerGroupMember
}

type ConsumerGroupMember struct {
	ID              uint32
	PartitionsCount uint32
	Partitions      []uint32
}

type CreateConsumerGroupRequest struct {
	StreamId        Identifier `json:"streamId"`
	TopicId         Identifier `json:"topicId"`
	Name            string     `json:"name"`
}

type DeleteConsumerGroupRequest struct {
	StreamId        Identifier `json:"streamId"`
	TopicId         Identifier `json:"topicId"`
	ConsumerGroupId Identifier `json:"consumerGroupId"`
}

type JoinConsumerGroupRequest struct {
	StreamId        Identifier `json:"streamId"`
	TopicId         Identifier `json:"topicId"`
	ConsumerGroupId Identifier `json:"consumerGroupId"`
}

type LeaveConsumerGroupRequest struct {
	StreamId        Identifier `json:"streamId"`
	TopicId         Identifier `json:"topicId"`
	ConsumerGroupId Identifier `json:"consumerGroupId"`
}

type ConsumerGroupInfo struct {
	StreamId        uint32 `json:"streamId"`
	TopicId         uint32 `json:"topicId"`
	ConsumerGroupId uint32 `json:"consumerGroupId"`
}
