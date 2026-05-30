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

type Code int

const (
	PingCode                 Code = 1
	GetStatsCode             Code = 10
	GetSnapshotFileCode      Code = 11
	GetClusterMetadataCode   Code = 12
	GetMeCode                Code = 20
	GetClientCode            Code = 21
	GetClientsCode           Code = 22
	GetUserCode              Code = 31
	GetUsersCode             Code = 32
	CreateUserCode           Code = 33
	DeleteUserCode           Code = 34
	UpdateUserCode           Code = 35
	UpdatePermissionsCode    Code = 36
	ChangePasswordCode       Code = 37
	LoginUserCode            Code = 38
	LogoutUserCode           Code = 39
	GetAccessTokensCode      Code = 41
	CreateAccessTokenCode    Code = 42
	DeleteAccessTokenCode    Code = 43
	LoginWithAccessTokenCode Code = 44
	PollMessagesCode         Code = 100
	SendMessagesCode         Code = 101
	GetOffsetCode            Code = 120
	StoreOffsetCode          Code = 121
	DeleteConsumerOffsetCode Code = 122
	GetStreamCode            Code = 200
	GetStreamsCode           Code = 201
	CreateStreamCode         Code = 202
	DeleteStreamCode         Code = 203
	UpdateStreamCode         Code = 204
	GetTopicCode             Code = 300
	GetTopicsCode            Code = 301
	CreateTopicCode          Code = 302
	DeleteTopicCode          Code = 303
	UpdateTopicCode          Code = 304
	CreatePartitionsCode     Code = 402
	DeletePartitionsCode     Code = 403
	GetGroupCode             Code = 600
	GetGroupsCode            Code = 601
	CreateGroupCode          Code = 602
	DeleteGroupCode          Code = 603
	JoinGroupCode            Code = 604
	LeaveGroupCode           Code = 605
)

//    internal const int GET_PERSONAL_ACCESS_TOKENS_CODE = 41;
//    internal const int CREATE_PERSONAL_ACCESS_TOKEN_CODE = 42;
//    internal const int DELETE_PERSONAL_ACCESS_TOKEN_CODE = 43;
//    internal const int LOGIN_WITH_PERSONAL_ACCESS_TOKEN_CODE = 44;
