/*
 * Copyright (c) "Neo4j"
 * Neo4j Sweden AB [https://neo4j.com]
 *
 * This file is part of Neo4j.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

// Package log defines the logging interface used internally by the driver and
// provides default logging implementations.
package log

import (
	"strconv"
	"sync/atomic"
)

// Logger is used throughout the driver for logging purposes.
// Driver client can implement this interface and provide an implementation
// upon driver creation.
//
// All logging functions takes a name and id that corresponds to the name of
// the logging component and it's identity, for example "router" and "1" to
// indicate who is logging and what instance.
//
// Database connections takes to form of "bolt3" and "bolt-123@192.168.0.1:7687"
// where "bolt3" is the name of the protocol handler in use, "bolt-123" is the
// databases identity of the connection on server "192.168.0.1:7687".
type Logger interface {
	// Error is called whenever the driver encounters an error that might
	// or might not cause a retry operation which means that all logged
	// errors are not critical. Type of err might or might not be a publicly
	// exported type. The same root cause of an error might be reported
	// more than once by different components using same or different err types.
	Error(name string, id string, err error)
	Warnf(name string, id string, msg string, args ...any)
	Infof(name string, id string, msg string, args ...any)
	Debugf(name string, id string, msg string, args ...any)
}

// List of component names used as parameter to logger functions.
const (
	Bolt3   = "bolt3"
	Bolt4   = "bolt4"
	Bolt5   = "bolt5"
	Driver  = "driver"
	Pool    = "pool"
	Router  = "router"
	Session = "session"
)

// Last used component id
var id uint32

// NewId generates a new id atomically.
func NewId() string {
	return strconv.FormatUint(uint64(atomic.AddUint32(&id, 1)), 10)
}
