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

type CreateStreamRequest struct {
	StreamId int    `json:"streamId"`
	Name     string `json:"name"`
}

type Stream struct {
	Id            uint32 `json:"id"`
	Name          string `json:"name"`
	SizeBytes     uint64 `json:"sizeBytes"`
	CreatedAt     uint64 `json:"createdAt"`
	MessagesCount uint64 `json:"messagesCount"`
	TopicsCount   uint32 `json:"topicsCount"`
}

type StreamDetails struct {
	Stream
	Topics []Topic `json:"topics,omitempty"`
}

type GetStreamRequest struct {
	StreamID Identifier
}
