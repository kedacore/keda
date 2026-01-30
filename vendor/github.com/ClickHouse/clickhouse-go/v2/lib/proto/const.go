// Licensed to ClickHouse, Inc. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. ClickHouse, Inc. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package proto

// see https://github.com/ClickHouse/ClickHouse/blob/master/src/Core/Protocol.h
const (
	DBMS_MIN_REVISION_WITH_CLIENT_INFO                          = 54032
	DBMS_MIN_REVISION_WITH_SERVER_TIMEZONE                      = 54058
	DBMS_MIN_REVISION_WITH_QUOTA_KEY_IN_CLIENT_INFO             = 54060
	DBMS_MIN_REVISION_WITH_SERVER_DISPLAY_NAME                  = 54372
	DBMS_MIN_REVISION_WITH_VERSION_PATCH                        = 54401
	DBMS_MIN_REVISION_WITH_CLIENT_WRITE_INFO                    = 54420
	DBMS_MIN_REVISION_WITH_SETTINGS_SERIALIZED_AS_STRINGS       = 54429
	DBMS_MIN_REVISION_WITH_INTERSERVER_SECRET                   = 54441
	DBMS_MIN_REVISION_WITH_OPENTELEMETRY                        = 54442
	DBMS_MIN_PROTOCOL_VERSION_WITH_DISTRIBUTED_DEPTH            = 54448
	DBMS_MIN_PROTOCOL_VERSION_WITH_INITIAL_QUERY_START_TIME     = 54449
	DBMS_MIN_PROTOCOL_VERSION_WITH_INCREMENTAL_PROFILE_EVENTS   = 54451
	DBMS_MIN_REVISION_WITH_PARALLEL_REPLICAS                    = 54453
	DBMS_MIN_REVISION_WITH_CUSTOM_SERIALIZATION                 = 54454
	DBMS_MIN_PROTOCOL_VERSION_WITH_ADDENDUM                     = 54458
	DBMS_MIN_PROTOCOL_VERSION_WITH_QUOTA_KEY                    = 54458
	DBMS_MIN_PROTOCOL_VERSION_WITH_PARAMETERS                   = 54459
	DBMS_MIN_PROTOCOL_VERSION_WITH_SERVER_QUERY_TIME_IN_PROGRES = 54460
	DBMS_TCP_PROTOCOL_VERSION                                   = DBMS_MIN_PROTOCOL_VERSION_WITH_SERVER_QUERY_TIME_IN_PROGRES
)

const (
	ClientHello  = 0
	ClientQuery  = 1
	ClientData   = 2
	ClientCancel = 3
	ClientPing   = 4
)

const (
	ClientQueryNone      = 0
	ClientQueryInitial   = 1
	ClientQuerySecondary = 2
)

const (
	CompressEnable  uint64 = 1
	CompressDisable uint64 = 0
)

const (
	StateComplete = 2
)

const (
	ServerHello               = 0
	ServerData                = 1
	ServerException           = 2
	ServerProgress            = 3
	ServerPong                = 4
	ServerEndOfStream         = 5
	ServerProfileInfo         = 6
	ServerTotals              = 7
	ServerExtremes            = 8
	ServerTablesStatus        = 9
	ServerLog                 = 10
	ServerTableColumns        = 11
	ServerPartUUIDs           = 12
	ServerReadTaskRequest     = 13
	ServerProfileEvents       = 14
	ServerTreeReadTaskRequest = 15
)
