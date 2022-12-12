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

package log

// Logger implementation that throws away all log events.
type Void struct{}

func (l Void) Error(name, id string, err error) {
}

func (l Void) Infof(name, id string, msg string, args ...any) {
}

func (l Void) Warnf(name, id string, msg string, args ...any) {
}

func (l Void) Debugf(name, id string, msg string, args ...any) {
}
