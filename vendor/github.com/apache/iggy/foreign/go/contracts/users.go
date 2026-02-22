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

type ChangePasswordRequest struct {
	UserID          Identifier `json:"-"`
	CurrentPassword string     `json:"CurrentPassword"`
	NewPassword     string     `json:"NewPassword"`
}

type UpdatePermissionsRequest struct {
	UserID      Identifier   `json:"-"`
	Permissions *Permissions `json:"Permissions,omitempty"`
}

type UpdateUserRequest struct {
	UserID   Identifier  `json:"-"`
	Username *string     `json:"username"`
	Status   *UserStatus `json:"userStatus"`
}

type CreateUserRequest struct {
	Username    string       `json:"username"`
	Password    string       `json:"Password"`
	Status      UserStatus   `json:"Status"`
	Permissions *Permissions `json:"Permissions,omitempty"`
}

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
