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

package command

import "encoding/binary"

type GetClient struct {
	ClientID uint32
}

func (c *GetClient) Code() Code {
	return GetClientCode
}

func (c *GetClient) MarshalBinary() ([]byte, error) {
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, c.ClientID)
	return bytes, nil
}

type GetClients struct{}

func (c *GetClients) Code() Code {
	return GetClientsCode
}

func (c *GetClients) MarshalBinary() ([]byte, error) {
	return []byte{}, nil
}

type GetClusterMetadata struct {
}

func (m *GetClusterMetadata) Code() Code {
	return GetClusterMetadataCode
}

func (m *GetClusterMetadata) MarshalBinary() ([]byte, error) {
	return []byte{}, nil
}

type GetStats struct{}

func (c *GetStats) Code() Code {
	return GetStatsCode
}

func (c *GetStats) MarshalBinary() ([]byte, error) {
	return []byte{}, nil
}

type Ping struct{}

func (p *Ping) Code() Code {
	return PingCode
}

func (p *Ping) MarshalBinary() ([]byte, error) {
	return []byte{}, nil
}
