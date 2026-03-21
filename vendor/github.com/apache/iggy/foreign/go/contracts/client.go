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

type Client interface {
	// Close closes the client and releases all the resources.
	Close() error

	// GetConnectionInfo returns the current connection information including protocol and server address
	GetConnectionInfo() *ConnectionInfo

	// GetClusterMetadata get the metadata of the cluster including node information, roles, and status.
	// Authentication is required.
	GetClusterMetadata() (*ClusterMetadata, error)

	// GetStream get the info about a specific stream by unique ID or name.
	// Authentication is required, and the permission to read the streams.
	GetStream(streamId Identifier) (*StreamDetails, error)

	// GetStreams get the info about all the streams.
	// Authentication is required, and the permission to read the streams.
	GetStreams() ([]Stream, error)

	// CreateStream create a new stream.
	// Authentication is required, and the permission to manage the streams.
	CreateStream(name string) (*StreamDetails, error)

	// UpdateStream update a stream by unique ID or name.
	// Authentication is required, and the permission to manage the streams.
	UpdateStream(streamId Identifier, name string) error

	// DeleteStream delete a topic by unique ID or name.
	// Authentication is required, and the permission to manage the topics.
	DeleteStream(id Identifier) error

	// GetTopic Get the info about a specific topic by unique ID or name.
	// Authentication is required, and the permission to read the topics.
	GetTopic(streamId, topicId Identifier) (*TopicDetails, error)

	// GetTopics get the info about all the topics.
	// Authentication is required, and the permission to read the topics.
	GetTopics(streamId Identifier) ([]Topic, error)

	// CreateTopic create a new topic.
	// Authentication is required, and the permission to manage the topics.
	CreateTopic(
		streamId Identifier,
		name string,
		partitionsCount uint32,
		compressionAlgorithm CompressionAlgorithm,
		messageExpiry Duration,
		maxTopicSize uint64,
		replicationFactor *uint8,
	) (*TopicDetails, error)

	// UpdateTopic update a topic by unique ID or name.
	// Authentication is required, and the permission to manage the topics.
	UpdateTopic(
		streamId Identifier,
		topicId Identifier,
		name string,
		compressionAlgorithm CompressionAlgorithm,
		messageExpiry Duration,
		maxTopicSize uint64,
		replicationFactor *uint8,
	) error

	// DeleteTopic delete a topic by unique ID or name.
	// Authentication is required, and the permission to manage the topics.
	DeleteTopic(streamId, topicId Identifier) error

	// SendMessages sends messages using specified partitioning strategy to the given stream and topic by unique IDs or names.
	// Authentication is required, and the permission to send the messages.
	SendMessages(
		streamId Identifier,
		topicId Identifier,
		partitioning Partitioning,
		messages []IggyMessage,
	) error

	// PollMessages poll given amount of messages using the specified consumer and strategy from the specified stream and topic by unique IDs or names.
	// Authentication is required, and the permission to poll the messages.
	PollMessages(
		streamId Identifier,
		topicId Identifier,
		consumer Consumer,
		strategy PollingStrategy,
		count uint32,
		autoCommit bool,
		partitionId *uint32,
	) (*PolledMessage, error)

	// StoreConsumerOffset store the consumer offset for a specific consumer or consumer group for the given stream and topic by unique IDs or names.
	// Authentication is required, and the permission to poll the messages.
	StoreConsumerOffset(
		consumer Consumer,
		streamId Identifier,
		topicId Identifier,
		offset uint64,
		partitionId *uint32,
	) error

	// GetConsumerOffset get the consumer offset for a specific consumer or consumer group for the given stream and topic by unique IDs or names.
	// Authentication is required, and the permission to poll the messages.
	GetConsumerOffset(
		consumer Consumer,
		streamId Identifier,
		topicId Identifier,
		partitionId *uint32,
	) (*ConsumerOffsetInfo, error)

	// GetConsumerGroups get the info about all the consumer groups for the given stream and topic by unique IDs or names.
	// Authentication is required, and the permission to read the streams or topics.
	GetConsumerGroups(streamId Identifier, topicId Identifier) ([]ConsumerGroup, error)

	// DeleteConsumerOffset delete the consumer offset for a specific consumer or consumer group for the given stream and topic by unique IDs or names.
	// Authentication is required, and the permission to poll the messages.
	DeleteConsumerOffset(
		consumer Consumer,
		streamId Identifier,
		topicId Identifier,
		partitionId *uint32,
	) error

	// GetConsumerGroup get the info about a specific consumer group by unique ID or name for the given stream and topic by unique IDs or names.
	// Authentication is required, and the permission to read the streams or topics.
	GetConsumerGroup(
		streamId Identifier,
		topicId Identifier,
		groupId Identifier,
	) (*ConsumerGroupDetails, error)

	// CreateConsumerGroup create a new consumer group for the given stream and topic by unique IDs or names.
	// Authentication is required, and the permission to manage the streams or topics.
	CreateConsumerGroup(
		streamId Identifier,
		topicId Identifier,
		name string,
	) (*ConsumerGroupDetails, error)

	// DeleteConsumerGroup delete a consumer group by unique ID or name for the given stream and topic by unique IDs or names.
	// Authentication is required, and the permission to manage the streams or topics.
	DeleteConsumerGroup(
		streamId Identifier,
		topicId Identifier,
		groupId Identifier,
	) error

	// JoinConsumerGroup join a consumer group by unique ID or name for the given stream and topic by unique IDs or names.
	// Authentication is required, and the permission to read the streams or topics.
	JoinConsumerGroup(
		streamId Identifier,
		topicId Identifier,
		groupId Identifier,
	) error

	// LeaveConsumerGroup leave a consumer group by unique ID or name for the given stream and topic by unique IDs or names.
	// Authentication is required, and the permission to read the streams or topics.
	LeaveConsumerGroup(
		streamId Identifier,
		topicId Identifier,
		groupId Identifier,
	) error

	// CreatePartitions create new N partitions for a topic by unique ID or name.
	// For example, given a topic with 3 partitions, if you create 2 partitions, the topic will have 5 partitions (from 1 to 5).
	// Authentication is required, and the permission to manage the partitions.
	CreatePartitions(
		streamId Identifier,
		topicId Identifier,
		partitionsCount uint32,
	) error

	// DeletePartitions delete last N partitions for a topic by unique ID or name.
	// For example, given a topic with 5 partitions, if you delete 2 partitions, the topic will have 3 partitions left (from 1 to 3).
	// Authentication is required, and the permission to manage the partitions.
	DeletePartitions(
		streamId Identifier,
		topicId Identifier,
		partitionsCount uint32,
	) error

	// GetUser get the info about a specific user by unique ID or username.
	// Authentication is required, and the permission to read the users, unless the provided user ID is the same as the authenticated user.
	GetUser(identifier Identifier) (*UserInfoDetails, error)

	// GetUsers get the info about all the users.
	// Authentication is required, and the permission to read the users.
	GetUsers() ([]UserInfo, error)

	// CreateUser create a new user.
	// Authentication is required, and the permission to manage the users.
	CreateUser(
		username string,
		password string,
		status UserStatus,
		permissions *Permissions,
	) (*UserInfoDetails, error)

	// UpdateUser update a user by unique ID or username.
	// Authentication is required, and the permission to manage the users.
	UpdateUser(
		userID Identifier,
		username *string,
		status *UserStatus,
	) error

	// UpdatePermissions update the permissions of a user by unique ID or username.
	// Authentication is required, and the permission to manage the users.
	UpdatePermissions(userID Identifier, permissions *Permissions) error

	// ChangePassword change the password of a user by unique ID or username.
	// Authentication is required, and the permission to manage the users, unless the provided user ID is the same as the authenticated user.
	ChangePassword(
		userID Identifier,
		currentPassword string,
		newPassword string,
	) error

	// DeleteUser delete a user by unique ID or username.
	// Authentication is required, and the permission to manage the users.
	DeleteUser(identifier Identifier) error

	// CreatePersonalAccessToken create a new personal access token for the currently authenticated user.
	CreatePersonalAccessToken(name string, expiry uint32) (*RawPersonalAccessToken, error)

	// DeletePersonalAccessToken delete a personal access token of the currently authenticated user by unique token name.
	DeletePersonalAccessToken(name string) error

	// GetPersonalAccessTokens get the info about all the personal access tokens of the currently authenticated user.
	GetPersonalAccessTokens() ([]PersonalAccessTokenInfo, error)

	// LoginWithPersonalAccessToken login the user with the provided personal access token.
	LoginWithPersonalAccessToken(token string) (*IdentityInfo, error)

	// LoginUser login a user by username and password.
	LoginUser(username string, password string) (*IdentityInfo, error)

	// LogoutUser logout the currently authenticated user.
	LogoutUser() error

	// GetStats get the stats of the system such as PID, memory usage, streams count etc.
	// Authentication is required, and the permission to read the server info.
	GetStats() (*Stats, error)

	// Ping the server to check if it's alive.
	Ping() error

	// GetClients get the info about all the currently connected clients (not to be confused with the users).
	// Authentication is required, and the permission to read the server info.
	GetClients() ([]ClientInfo, error)

	// GetClient get the info about a specific client by unique ID (not to be confused with the user).
	// Authentication is required, and the permission to read the server info.
	GetClient(clientId uint32) (*ClientInfoDetails, error)
}
