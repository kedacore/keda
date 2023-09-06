// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package logger

import (
	"os"
	"strconv"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	CommandFailed             = "Command failed"
	CommandStarted            = "Command started"
	CommandSucceeded          = "Command succeeded"
	ConnectionPoolCreated     = "Connection pool created"
	ConnectionPoolReady       = "Connection pool ready"
	ConnectionPoolCleared     = "Connection pool cleared"
	ConnectionPoolClosed      = "Connection pool closed"
	ConnectionCreated         = "Connection created"
	ConnectionReady           = "Connection ready"
	ConnectionClosed          = "Connection closed"
	ConnectionCheckoutStarted = "Connection checkout started"
	ConnectionCheckoutFailed  = "Connection checkout failed"
	ConnectionCheckedOut      = "Connection checked out"
	ConnectionCheckedIn       = "Connection checked in"
)

const (
	KeyCommand            = "command"
	KeyCommandName        = "commandName"
	KeyDatabaseName       = "databaseName"
	KeyDriverConnectionID = "driverConnectionId"
	KeyDurationMS         = "durationMS"
	KeyError              = "error"
	KeyFailure            = "failure"
	KeyMaxConnecting      = "maxConnecting"
	KeyMaxIdleTimeMS      = "maxIdleTimeMS"
	KeyMaxPoolSize        = "maxPoolSize"
	KeyMessage            = "message"
	KeyMinPoolSize        = "minPoolSize"
	KeyOperationID        = "operationId"
	KeyReason             = "reason"
	KeyReply              = "reply"
	KeyRequestID          = "requestId"
	KeyServerConnectionID = "serverConnectionId"
	KeyServerHost         = "serverHost"
	KeyServerPort         = "serverPort"
	KeyServiceID          = "serviceId"
	KeyTimestamp          = "timestamp"
)

type KeyValues []interface{}

func (kvs *KeyValues) Add(key string, value interface{}) {
	*kvs = append(*kvs, key, value)
}

const (
	ReasonConnClosedStale              = "Connection became stale because the pool was cleared"
	ReasonConnClosedIdle               = "Connection has been available but unused for longer than the configured max idle time"
	ReasonConnClosedError              = "An error occurred while using the connection"
	ReasonConnClosedPoolClosed         = "Connection pool was closed"
	ReasonConnCheckoutFailedTimout     = "Wait queue timeout elapsed without a connection becoming available"
	ReasonConnCheckoutFailedError      = "An error occurred while trying to establish a new connection"
	ReasonConnCheckoutFailedPoolClosed = "Connection pool was closed"
)

// Component is an enumeration representing the "components" which can be
// logged against. A LogLevel can be configured on a per-component basis.
type Component int

const (
	// ComponentAll enables logging for all components.
	ComponentAll Component = iota

	// ComponentCommand enables command monitor logging.
	ComponentCommand

	// ComponentTopology enables topology logging.
	ComponentTopology

	// ComponentServerSelection enables server selection logging.
	ComponentServerSelection

	// ComponentConnection enables connection services logging.
	ComponentConnection
)

const (
	mongoDBLogAllEnvVar             = "MONGODB_LOG_ALL"
	mongoDBLogCommandEnvVar         = "MONGODB_LOG_COMMAND"
	mongoDBLogTopologyEnvVar        = "MONGODB_LOG_TOPOLOGY"
	mongoDBLogServerSelectionEnvVar = "MONGODB_LOG_SERVER_SELECTION"
	mongoDBLogConnectionEnvVar      = "MONGODB_LOG_CONNECTION"
)

var componentEnvVarMap = map[string]Component{
	mongoDBLogAllEnvVar:             ComponentAll,
	mongoDBLogCommandEnvVar:         ComponentCommand,
	mongoDBLogTopologyEnvVar:        ComponentTopology,
	mongoDBLogServerSelectionEnvVar: ComponentServerSelection,
	mongoDBLogConnectionEnvVar:      ComponentConnection,
}

// EnvHasComponentVariables returns true if the environment contains any of the
// component environment variables.
func EnvHasComponentVariables() bool {
	for envVar := range componentEnvVarMap {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	return false
}

// Command is a struct defining common fields that must be included in all
// commands.
type Command struct {
	// TODO(GODRIVER-2824): change the DriverConnectionID type to int64.
	DriverConnectionID uint64              // Driver's ID for the connection
	Name               string              // Command name
	Message            string              // Message associated with the command
	OperationID        int32               // Driver-generated operation ID
	RequestID          int64               // Driver-generated request ID
	ServerConnectionID *int64              // Server's ID for the connection used for the command
	ServerHost         string              // Hostname or IP address for the server
	ServerPort         string              // Port for the server
	ServiceID          *primitive.ObjectID // ID for the command  in load balancer mode
}

// SerializeCommand takes a command and a variable number of key-value pairs and
// returns a slice of interface{} that can be passed to the logger for
// structured logging.
func SerializeCommand(cmd Command, extraKeysAndValues ...interface{}) []interface{} {
	// Initialize the boilerplate keys and values.
	keysAndValues := KeyValues{
		KeyCommandName, cmd.Name,
		KeyDriverConnectionID, cmd.DriverConnectionID,
		KeyMessage, cmd.Message,
		KeyOperationID, cmd.OperationID,
		KeyRequestID, cmd.RequestID,
		KeyServerHost, cmd.ServerHost,
	}

	// Add the extra keys and values.
	for i := 0; i < len(extraKeysAndValues); i += 2 {
		keysAndValues.Add(extraKeysAndValues[i].(string), extraKeysAndValues[i+1])
	}

	port, err := strconv.ParseInt(cmd.ServerPort, 0, 32)
	if err == nil {
		keysAndValues.Add(KeyServerPort, port)
	}

	// Add the "serverConnectionId" if it is not nil.
	if cmd.ServerConnectionID != nil {
		keysAndValues.Add(KeyServerConnectionID, *cmd.ServerConnectionID)
	}

	// Add the "serviceId" if it is not nil.
	if cmd.ServiceID != nil {
		keysAndValues.Add(KeyServiceID, cmd.ServiceID.Hex())
	}

	return keysAndValues
}

// Connection contains data that all connection log messages MUST contain.
type Connection struct {
	Message    string // Message associated with the connection
	ServerHost string // Hostname or IP address for the server
	ServerPort string // Port for the server
}

// SerializeConnection serializes a ConnectionMessage into a slice of keys
// and values that can be passed to a logger.
func SerializeConnection(conn Connection, extraKeysAndValues ...interface{}) []interface{} {
	// Initialize the boilerplate keys and values.
	keysAndValues := KeyValues{
		KeyMessage, conn.Message,
		KeyServerHost, conn.ServerHost,
	}

	// Add the optional keys and values.
	for i := 0; i < len(extraKeysAndValues); i += 2 {
		keysAndValues.Add(extraKeysAndValues[i].(string), extraKeysAndValues[i+1])
	}

	port, err := strconv.ParseInt(conn.ServerPort, 0, 32)
	if err == nil {
		keysAndValues.Add(KeyServerPort, port)
	}

	return keysAndValues
}
