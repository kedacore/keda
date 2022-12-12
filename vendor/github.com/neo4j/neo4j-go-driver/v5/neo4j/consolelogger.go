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

package neo4j

import (
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/log"
)

// LogLevel is the type that default logging implementations use for available
// log levels
type LogLevel int

const (
	// ERROR is the level that error messages are written
	ERROR LogLevel = 1
	// WARNING is the level that warning messages are written
	WARNING = 2
	// INFO is the level that info messages are written
	INFO = 3
	// DEBUG is the level that debug messages are written
	DEBUG = 4
)

func ConsoleLogger(level LogLevel) *log.Console {
	return &log.Console{
		Errors: level >= ERROR,
		Warns:  level >= WARNING,
		Infos:  level >= INFO,
		Debugs: level >= DEBUG,
	}
}

func ConsoleBoltLogger() *log.ConsoleBoltLogger {
	return &log.ConsoleBoltLogger{}
}
