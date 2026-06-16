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

import iggcon "github.com/apache/iggy/foreign/go/contracts"

type TopicPath struct {
	StreamId iggcon.Identifier
	TopicId  iggcon.Identifier
}

type CreateConsumerGroup struct {
	TopicPath
	Name string
}

func (c *CreateConsumerGroup) Code() Code {
	return CreateGroupCode
}

func (c *CreateConsumerGroup) MarshalBinary() ([]byte, error) {
	streamIdBytes, err := c.StreamId.MarshalBinary()
	if err != nil {
		return nil, err
	}
	topicIdBytes, err := c.TopicId.MarshalBinary()
	if err != nil {
		return nil, err
	}
	offset := len(streamIdBytes) + len(topicIdBytes)
	bytes := make([]byte, offset+1+len(c.Name))
	copy(bytes[0:len(streamIdBytes)], streamIdBytes)
	copy(bytes[len(streamIdBytes):offset], topicIdBytes)
	bytes[offset] = byte(len(c.Name))
	copy(bytes[offset+1:], c.Name)
	return bytes, nil
}

type GetConsumerGroup struct {
	TopicPath
	GroupId iggcon.Identifier
}

func (g *GetConsumerGroup) Code() Code {
	return GetGroupCode
}

func (g *GetConsumerGroup) MarshalBinary() ([]byte, error) {
	return iggcon.MarshalIdentifiers(g.StreamId, g.TopicId, g.GroupId)
}

type GetConsumerGroups struct {
	StreamId iggcon.Identifier
	TopicId  iggcon.Identifier
}

func (g *GetConsumerGroups) Code() Code {
	return GetGroupsCode
}

func (g *GetConsumerGroups) MarshalBinary() ([]byte, error) {
	return iggcon.MarshalIdentifiers(g.StreamId, g.TopicId)
}

type JoinConsumerGroup struct {
	TopicPath
	GroupId iggcon.Identifier
}

func (j *JoinConsumerGroup) Code() Code {
	return JoinGroupCode
}

func (j *JoinConsumerGroup) MarshalBinary() ([]byte, error) {
	return iggcon.MarshalIdentifiers(j.StreamId, j.TopicId, j.GroupId)
}

type LeaveConsumerGroup struct {
	TopicPath
	GroupId iggcon.Identifier
}

func (l *LeaveConsumerGroup) Code() Code {
	return LeaveGroupCode
}

func (l *LeaveConsumerGroup) MarshalBinary() ([]byte, error) {
	return iggcon.MarshalIdentifiers(l.StreamId, l.TopicId, l.GroupId)
}

type DeleteConsumerGroup struct {
	TopicPath
	GroupId iggcon.Identifier
}

func (d *DeleteConsumerGroup) Code() Code {
	return DeleteGroupCode
}

func (d *DeleteConsumerGroup) MarshalBinary() ([]byte, error) {
	return iggcon.MarshalIdentifiers(d.StreamId, d.TopicId, d.GroupId)
}
