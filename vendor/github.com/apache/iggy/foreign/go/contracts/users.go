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

import "encoding/binary"

type UserInfo struct {
	Id        uint32     `json:"Id"`
	CreatedAt uint64     `json:"CreatedAt"`
	Status    UserStatus `json:"Status"`
	Username  string     `json:"Username"`
}

type UserInfoDetails struct {
	UserInfo
	Permissions *Permissions `json:"Permissions"`
}

type UserStatus int

const (
	Active UserStatus = iota
	Inactive
)

type Permissions struct {
	Global  GlobalPermissions          `json:"Global"`
	Streams map[int]*StreamPermissions `json:"Streams,omitempty"`
}

func (p *Permissions) MarshalBinary() ([]byte, error) {
	size := p.Size()
	bytes := make([]byte, size)

	bytes[0] = boolToByte(p.Global.ManageServers)
	bytes[1] = boolToByte(p.Global.ReadServers)
	bytes[2] = boolToByte(p.Global.ManageUsers)
	bytes[3] = boolToByte(p.Global.ReadUsers)
	bytes[4] = boolToByte(p.Global.ManageStreams)
	bytes[5] = boolToByte(p.Global.ReadStreams)
	bytes[6] = boolToByte(p.Global.ManageTopics)
	bytes[7] = boolToByte(p.Global.ReadTopics)
	bytes[8] = boolToByte(p.Global.PollMessages)
	bytes[9] = boolToByte(p.Global.SendMessages)

	position := 10

	if len(p.Streams) > 0 {
		bytes[position] = byte(1)
		position += 1

		streamsCount := len(p.Streams)
		currentStream := 1
		for streamID, stream := range p.Streams {
			binary.LittleEndian.PutUint32(bytes[position:position+4], uint32(streamID))
			position += 4

			bytes[position] = boolToByte(stream.ManageStream)
			bytes[position+1] = boolToByte(stream.ReadStream)
			bytes[position+2] = boolToByte(stream.ManageTopics)
			bytes[position+3] = boolToByte(stream.ReadTopics)
			bytes[position+4] = boolToByte(stream.PollMessages)
			bytes[position+5] = boolToByte(stream.SendMessages)
			position += 6

			if len(stream.Topics) > 0 {
				bytes[position] = byte(1)
				position += 1

				topicsCount := len(stream.Topics)
				currentTopic := 1
				for topicID, topic := range stream.Topics {
					binary.LittleEndian.PutUint32(bytes[position:position+4], uint32(topicID))
					position += 4

					bytes[position] = boolToByte(topic.ManageTopic)
					bytes[position+1] = boolToByte(topic.ReadTopic)
					bytes[position+2] = boolToByte(topic.PollMessages)
					bytes[position+3] = boolToByte(topic.SendMessages)
					position += 4

					if currentTopic < topicsCount {
						currentTopic++
						bytes[position] = byte(1)
					} else {
						bytes[position] = byte(0)
					}
					position += 1
				}
			} else {
				bytes[position] = byte(0)
				position += 1
			}

			if currentStream < streamsCount {
				currentStream++
				bytes[position] = byte(1)
			} else {
				bytes[position] = byte(0)
			}
			position += 1
		}
	} else {
		bytes[position] = byte(0)
	}

	return bytes, nil
}

func (p *Permissions) Size() int {
	size := 10

	if p.Streams != nil {
		size += 1

		for _, stream := range p.Streams {
			size += 4
			size += 6
			size += 1

			if stream.Topics != nil {
				size += 1
				size += len(stream.Topics) * 9
			} else {
				size += 1
			}
		}
	} else {
		size += 1
	}

	return size
}

func boolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

type GlobalPermissions struct {
	ManageServers bool `json:"ManageServers"`
	ReadServers   bool `json:"ReadServers"`
	ManageUsers   bool `json:"ManageUsers"`
	ReadUsers     bool `json:"ReadUsers"`
	ManageStreams bool `json:"ManageStreams"`
	ReadStreams   bool `json:"ReadStreams"`
	ManageTopics  bool `json:"ManageTopics"`
	ReadTopics    bool `json:"ReadTopics"`
	PollMessages  bool `json:"PollMessages"`
	SendMessages  bool `json:"SendMessages"`
}

type StreamPermissions struct {
	ManageStream bool                      `json:"ManageStream"`
	ReadStream   bool                      `json:"ReadStream"`
	ManageTopics bool                      `json:"ManageTopics"`
	ReadTopics   bool                      `json:"ReadTopics"`
	PollMessages bool                      `json:"PollMessages"`
	SendMessages bool                      `json:"SendMessages"`
	Topics       map[int]*TopicPermissions `json:"Topics,omitempty"`
}

type TopicPermissions struct {
	ManageTopic  bool `json:"ManageTopic"`
	ReadTopic    bool `json:"ReadTopic"`
	PollMessages bool `json:"PollMessages"`
	SendMessages bool `json:"SendMessages"`
}
