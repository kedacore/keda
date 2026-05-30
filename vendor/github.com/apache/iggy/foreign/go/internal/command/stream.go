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

import iggcon "github.com/apache/iggy/foreign/go/contracts"

const (
	nameLengthOffset = 0
	payloadOffset    = 1
)

type CreateStream struct {
	Name string
}

func (request *CreateStream) Code() Code {
	return CreateStreamCode
}

func (request *CreateStream) MarshalBinary() ([]byte, error) {
	nameLength := len(request.Name)
	serialized := make([]byte, payloadOffset+nameLength)
	serialized[nameLengthOffset] = byte(nameLength)
	copy(serialized[payloadOffset:], request.Name)
	return serialized, nil
}

type GetStream struct {
	StreamId iggcon.Identifier
}

func (g *GetStream) Code() Code {
	return GetStreamCode
}

func (g *GetStream) MarshalBinary() ([]byte, error) {
	return g.StreamId.MarshalBinary()
}

type GetStreams struct{}

func (g *GetStreams) Code() Code {
	return GetStreamsCode
}

func (g *GetStreams) MarshalBinary() ([]byte, error) {
	return []byte{}, nil
}

type UpdateStream struct {
	StreamId iggcon.Identifier `json:"streamId"`
	Name     string            `json:"name"`
}

func (u *UpdateStream) Code() Code {
	return UpdateStreamCode
}

func (u *UpdateStream) MarshalBinary() ([]byte, error) {
	streamIdBytes, err := u.StreamId.MarshalBinary()
	if err != nil {
		return nil, err
	}
	nameLength := len(u.Name)
	bytes := make([]byte, len(streamIdBytes)+1+nameLength)
	copy(bytes[0:len(streamIdBytes)], streamIdBytes)
	position := len(streamIdBytes)
	bytes[position] = byte(nameLength)
	copy(bytes[position+1:], u.Name)
	return bytes, nil
}

type DeleteStream struct {
	StreamId iggcon.Identifier
}

func (d *DeleteStream) Code() Code {
	return DeleteStreamCode
}

func (d *DeleteStream) MarshalBinary() ([]byte, error) {
	return d.StreamId.MarshalBinary()
}
