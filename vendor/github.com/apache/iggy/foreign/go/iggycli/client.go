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

package iggycli

import (
	iggcon "github.com/apache/iggy/foreign/go/contracts"
)

type Client interface {
	// GetStream get the info about a specific stream by unique ID or name.
	// Authentication is required, and the permission to read the streams.
	GetStream(streamId iggcon.Identifier) (*iggcon.StreamDetails, error)

	// GetStreams get the info about all the streams.
	// Authentication is required, and the permission to read the streams.
	GetStreams() ([]iggcon.Stream, error)

    // CreateStream create a new stream.
    // Authentication is required, and the permission to manage the streams.
    CreateStream(name string) (*iggcon.StreamDetails, error)

	// UpdateStream update a stream by unique ID or name.
	// Authentication is required, and the permission to manage the streams.
	UpdateStream(streamId iggcon.Identifier, name string) error

	// DeleteStream delete a topic by unique ID or name.
	// Authentication is required, and the permission to manage the topics.
	DeleteStream(id iggcon.Identifier) error

	// GetTopic Get the info about a specific topic by unique ID or name.
	// Authentication is required, and the permission to read the topics.
	GetTopic(streamId, topicId iggcon.Identifier) (*iggcon.TopicDetails, error)

	// GetTopics get the info about all the topics.
	// Authentication is required, and the permission to read the topics.
	GetTopics(streamId iggcon.Identifier) ([]iggcon.Topic, error)

	// CreateTopic create a new topic.
	// Authentication is required, and the permission to manage the topics.
    CreateTopic(
        streamId iggcon.Identifier,
        name string,
        partitionsCount uint32,
        compressionAlgorithm iggcon.CompressionAlgorithm,
        messageExpiry iggcon.Duration,
        maxTopicSize uint64,
        replicationFactor *uint8,
    ) (*iggcon.TopicDetails, error)

	// UpdateTopic update a topic by unique ID or name.
	// Authentication is required, and the permission to manage the topics.
	UpdateTopic(
		streamId iggcon.Identifier,
		topicId iggcon.Identifier,
		name string,
		compressionAlgorithm iggcon.CompressionAlgorithm,
		messageExpiry iggcon.Duration,
		maxTopicSize uint64,
		replicationFactor *uint8,
	) error

	// DeleteTopic delete a topic by unique ID or name.
	// Authentication is required, and the permission to manage the topics.
	DeleteTopic(streamId, topicId iggcon.Identifier) error

	// SendMessages sends messages using specified partitioning strategy to the given stream and topic by unique IDs or names.
	// Authentication is required, and the permission to send the messages.
	SendMessages(
		streamId iggcon.Identifier,
		topicId iggcon.Identifier,
		partitioning iggcon.Partitioning,
		messages []iggcon.IggyMessage,
	) error

	// PollMessages poll given amount of messages using the specified consumer and strategy from the specified stream and topic by unique IDs or names.
	// Authentication is required, and the permission to poll the messages.
	PollMessages(
		streamId iggcon.Identifier,
		topicId iggcon.Identifier,
		consumer iggcon.Consumer,
		strategy iggcon.PollingStrategy,
		count uint32,
		autoCommit bool,
		partitionId *uint32,
	) (*iggcon.PolledMessage, error)

	// StoreConsumerOffset store the consumer offset for a specific consumer or consumer group for the given stream and topic by unique IDs or names.
	// Authentication is required, and the permission to poll the messages.
	StoreConsumerOffset(
		consumer iggcon.Consumer,
		streamId iggcon.Identifier,
		topicId iggcon.Identifier,
		offset uint64,
		partitionId *uint32,
	) error

	// GetConsumerOffset get the consumer offset for a specific consumer or consumer group for the given stream and topic by unique IDs or names.
	// Authentication is required, and the permission to poll the messages.
	GetConsumerOffset(
		consumer iggcon.Consumer,
		streamId iggcon.Identifier,
		topicId iggcon.Identifier,
		partitionId *uint32,
	) (*iggcon.ConsumerOffsetInfo, error)

	// GetConsumerGroups get the info about all the consumer groups for the given stream and topic by unique IDs or names.
	// Authentication is required, and the permission to read the streams or topics.
	GetConsumerGroups(streamId iggcon.Identifier, topicId iggcon.Identifier) ([]iggcon.ConsumerGroup, error)

	// DeleteConsumerOffset delete the consumer offset for a specific consumer or consumer group for the given stream and topic by unique IDs or names.
	// Authentication is required, and the permission to poll the messages.
	DeleteConsumerOffset(
		consumer iggcon.Consumer,
		streamId iggcon.Identifier,
		topicId iggcon.Identifier,
		partitionId *uint32,
	) error

	// GetConsumerGroup get the info about a specific consumer group by unique ID or name for the given stream and topic by unique IDs or names.
	// Authentication is required, and the permission to read the streams or topics.
	GetConsumerGroup(
		streamId iggcon.Identifier,
		topicId iggcon.Identifier,
		groupId iggcon.Identifier,
	) (*iggcon.ConsumerGroupDetails, error)

	// CreateConsumerGroup create a new consumer group for the given stream and topic by unique IDs or names.
	// Authentication is required, and the permission to manage the streams or topics.
    CreateConsumerGroup(
        streamId iggcon.Identifier,
        topicId iggcon.Identifier,
        name string,
    ) (*iggcon.ConsumerGroupDetails, error)

	// DeleteConsumerGroup delete a consumer group by unique ID or name for the given stream and topic by unique IDs or names.
	// Authentication is required, and the permission to manage the streams or topics.
	DeleteConsumerGroup(
		streamId iggcon.Identifier,
		topicId iggcon.Identifier,
		groupId iggcon.Identifier,
	) error

	// JoinConsumerGroup join a consumer group by unique ID or name for the given stream and topic by unique IDs or names.
	// Authentication is required, and the permission to read the streams or topics.
	JoinConsumerGroup(
		streamId iggcon.Identifier,
		topicId iggcon.Identifier,
		groupId iggcon.Identifier,
	) error

	// LeaveConsumerGroup leave a consumer group by unique ID or name for the given stream and topic by unique IDs or names.
	// Authentication is required, and the permission to read the streams or topics.
	LeaveConsumerGroup(
		streamId iggcon.Identifier,
		topicId iggcon.Identifier,
		groupId iggcon.Identifier,
	) error

	// CreatePartitions create new N partitions for a topic by unique ID or name.
	// For example, given a topic with 3 partitions, if you create 2 partitions, the topic will have 5 partitions (from 1 to 5).
	// Authentication is required, and the permission to manage the partitions.
	CreatePartitions(
		streamId iggcon.Identifier,
		topicId iggcon.Identifier,
		partitionsCount uint32,
	) error

	// DeletePartitions delete last N partitions for a topic by unique ID or name.
	// For example, given a topic with 5 partitions, if you delete 2 partitions, the topic will have 3 partitions left (from 1 to 3).
	// Authentication is required, and the permission to manage the partitions.
	DeletePartitions(
		streamId iggcon.Identifier,
		topicId iggcon.Identifier,
		partitionsCount uint32,
	) error

	// GetUser get the info about a specific user by unique ID or username.
	// Authentication is required, and the permission to read the users, unless the provided user ID is the same as the authenticated user.
	GetUser(identifier iggcon.Identifier) (*iggcon.UserInfoDetails, error)

	// GetUsers get the info about all the users.
	// Authentication is required, and the permission to read the users.
	GetUsers() ([]iggcon.UserInfo, error)

	// CreateUser create a new user.
	// Authentication is required, and the permission to manage the users.
	CreateUser(
		username string,
		password string,
		status iggcon.UserStatus,
		permissions *iggcon.Permissions,
	) (*iggcon.UserInfoDetails, error)

	// UpdateUser update a user by unique ID or username.
	// Authentication is required, and the permission to manage the users.
	UpdateUser(
		userID iggcon.Identifier,
		username *string,
		status *iggcon.UserStatus,
	) error

	// UpdatePermissions update the permissions of a user by unique ID or username.
	// Authentication is required, and the permission to manage the users.
	UpdatePermissions(userID iggcon.Identifier, permissions *iggcon.Permissions) error

	// ChangePassword change the password of a user by unique ID or username.
	// Authentication is required, and the permission to manage the users, unless the provided user ID is the same as the authenticated user.
	ChangePassword(
		userID iggcon.Identifier,
		currentPassword string,
		newPassword string,
	) error

	// DeleteUser delete a user by unique ID or username.
	// Authentication is required, and the permission to manage the users.
	DeleteUser(identifier iggcon.Identifier) error

	// CreatePersonalAccessToken create a new personal access token for the currently authenticated user.
	CreatePersonalAccessToken(name string, expiry uint32) (*iggcon.RawPersonalAccessToken, error)

	// DeletePersonalAccessToken delete a personal access token of the currently authenticated user by unique token name.
	DeletePersonalAccessToken(name string) error

	// GetPersonalAccessTokens get the info about all the personal access tokens of the currently authenticated user.
	GetPersonalAccessTokens() ([]iggcon.PersonalAccessTokenInfo, error)

	// LoginWithPersonalAccessToken login the user with the provided personal access token.
	LoginWithPersonalAccessToken(token string) (*iggcon.IdentityInfo, error)

	// LoginUser login a user by username and password.
	LoginUser(username string, password string) (*iggcon.IdentityInfo, error)

	// LogoutUser logout the currently authenticated user.
	LogoutUser() error

	// GetStats get the stats of the system such as PID, memory usage, streams count etc.
	// Authentication is required, and the permission to read the server info.
	GetStats() (*iggcon.Stats, error)

	// Ping the server to check if it's alive.
	Ping() error

	// GetClients get the info about all the currently connected clients (not to be confused with the users).
	// Authentication is required, and the permission to read the server info.
	GetClients() ([]iggcon.ClientInfo, error)

	// GetClient get the info about a specific client by unique ID (not to be confused with the user).
	// Authentication is required, and the permission to read the server info.
	GetClient(clientId uint32) (*iggcon.ClientInfoDetails, error)
}
