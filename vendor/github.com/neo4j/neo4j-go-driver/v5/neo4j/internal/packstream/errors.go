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

// Package packstream handles serialization of data sent to database server and
// deserialization of data received from database server.
package packstream

type OverflowError struct {
	msg string
}

func (e *OverflowError) Error() string {
	return e.msg
}

type IoError struct{}

func (e *IoError) Error() string {
	return "IO error"
}

type UnpackError struct {
	msg string
}

func (e *UnpackError) Error() string {
	return e.msg
}
