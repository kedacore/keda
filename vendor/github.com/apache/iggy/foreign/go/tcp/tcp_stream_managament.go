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

package tcp

import (
	binaryserialization "github.com/apache/iggy/foreign/go/binary_serialization"
	iggcon "github.com/apache/iggy/foreign/go/contracts"
	ierror "github.com/apache/iggy/foreign/go/errors"
)

func (tms *IggyTcpClient) GetStreams() ([]iggcon.Stream, error) {
	buffer, err := tms.sendAndFetchResponse([]byte{}, iggcon.GetStreamsCode)
	if err != nil {
		return nil, err
	}

	return binaryserialization.DeserializeStreams(buffer), nil
}

func (tms *IggyTcpClient) GetStream(streamId iggcon.Identifier) (*iggcon.StreamDetails, error) {
	message := binaryserialization.SerializeIdentifier(streamId)
	buffer, err := tms.sendAndFetchResponse(message, iggcon.GetStreamCode)
	if err != nil {
		return nil, err
	}
	if len(buffer) == 0 {
		return nil, ierror.ErrStreamIdNotFound
	}

	stream, err := binaryserialization.DeserializeStream(buffer)
	if err != nil {
		return nil, err
	}

	return stream, nil
}

func (tms *IggyTcpClient) CreateStream(name string) (*iggcon.StreamDetails, error) {
	if len(name) == 0 || MaxStringLength < len(name) {
		return nil, ierror.ErrInvalidStreamName
	}
    serializedRequest := binaryserialization.TcpCreateStreamRequest{Name: name}
	buffer, err := tms.sendAndFetchResponse(serializedRequest.Serialize(), iggcon.CreateStreamCode)
	if err != nil {
		return nil, err
	}
	stream, err := binaryserialization.DeserializeStream(buffer)
	if err != nil {
		return nil, err
	}

	return stream, err
}

func (tms *IggyTcpClient) UpdateStream(streamId iggcon.Identifier, name string) error {
	if len(name) > MaxStringLength || len(name) == 0 {
		return ierror.ErrInvalidStreamName
	}
	serializedRequest := binaryserialization.TcpUpdateStreamRequest{StreamId: streamId, Name: name}
	_, err := tms.sendAndFetchResponse(serializedRequest.Serialize(), iggcon.UpdateStreamCode)
	return err
}

func (tms *IggyTcpClient) DeleteStream(id iggcon.Identifier) error {
	message := binaryserialization.SerializeIdentifier(id)
	_, err := tms.sendAndFetchResponse(message, iggcon.DeleteStreamCode)
	return err
}
