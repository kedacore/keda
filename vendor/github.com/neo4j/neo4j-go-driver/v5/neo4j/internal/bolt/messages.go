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

package bolt

// Message struct tags
// Shared between bolt versions
const (
	msgReset      byte = 0x0f
	msgRun        byte = 0x10
	msgDiscardAll byte = 0x2f
	msgDiscardN        = msgDiscardAll // Different name >= 4.0
	msgPullAll    byte = 0x3f
	msgPullN           = msgPullAll // Different name >= 4.0
	msgRecord     byte = 0x71
	msgSuccess    byte = 0x70
	msgIgnored    byte = 0x7e
	msgFailure    byte = 0x7f
	msgHello      byte = 0x01
	msgGoodbye    byte = 0x02
	msgBegin      byte = 0x11
	msgCommit     byte = 0x12
	msgRollback   byte = 0x13
	msgRoute      byte = 0x66 // > 4.2
)
