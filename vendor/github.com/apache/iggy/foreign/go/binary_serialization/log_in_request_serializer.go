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

package binaryserialization

import (
	"encoding/binary"
)

type TcpLogInRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (request *TcpLogInRequest) Serialize() []byte {
	usernameBytes := []byte(request.Username)
	passwordBytes := []byte(request.Password)
	versionBytes := []byte("")
	contextBytes := []byte("")

	// Calculate total length
	totalLength := 2 + len(usernameBytes) + len(passwordBytes) +
		8 + len(versionBytes) + len(contextBytes)

	result := make([]byte, totalLength)
	position := 0

	// Username
	result[position] = byte(len(usernameBytes))
	position++
	copy(result[position:], usernameBytes)
	position += len(usernameBytes)

	// Password
	result[position] = byte(len(passwordBytes))
	position++
	copy(result[position:], passwordBytes)
	position += len(passwordBytes)

	// Version
	binary.LittleEndian.PutUint32(result[position:], uint32(len(versionBytes)))
	position += 4
	copy(result[position:], versionBytes)
	position += len(versionBytes)

	// Context
	binary.LittleEndian.PutUint32(result[position:], uint32(len(contextBytes)))
	position += 4
	copy(result[position:], contextBytes)

	return result
}
