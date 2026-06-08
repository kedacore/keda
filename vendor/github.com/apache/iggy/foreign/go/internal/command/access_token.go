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

type CreatePersonalAccessToken struct {
	Name   string `json:"Name"`
	Expiry uint32 `json:"Expiry"`
}

func (c *CreatePersonalAccessToken) Code() Code {
	return CreateAccessTokenCode
}

func (c *CreatePersonalAccessToken) MarshalBinary() ([]byte, error) {
	length := 1 + len(c.Name) + 8
	bytes := make([]byte, length)
	bytes[0] = byte(len(c.Name))
	copy(bytes[1:], c.Name)
	binary.LittleEndian.PutUint32(bytes[len(bytes)-4:], c.Expiry)
	return bytes, nil
}

type GetPersonalAccessTokens struct{}

func (g *GetPersonalAccessTokens) Code() Code {
	return GetAccessTokensCode
}

func (g *GetPersonalAccessTokens) MarshalBinary() ([]byte, error) {
	return []byte{}, nil
}

type DeletePersonalAccessToken struct {
	Name string `json:"Name"`
}

func (d *DeletePersonalAccessToken) Code() Code {
	return DeleteAccessTokenCode
}

func (d *DeletePersonalAccessToken) MarshalBinary() ([]byte, error) {
	length := 1 + len(d.Name)
	bytes := make([]byte, length)
	bytes[0] = byte(len(d.Name))
	copy(bytes[1:], d.Name)
	return bytes, nil
}
