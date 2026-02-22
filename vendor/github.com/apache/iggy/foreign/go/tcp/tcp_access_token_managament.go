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
)

func (tms *IggyTcpClient) CreatePersonalAccessToken(name string, expiry uint32) (*iggcon.RawPersonalAccessToken, error) {
	message := binaryserialization.SerializeCreatePersonalAccessToken(iggcon.CreatePersonalAccessTokenRequest{
		Name:   name,
		Expiry: expiry,
	})
	buffer, err := tms.sendAndFetchResponse(message, iggcon.CreateAccessTokenCode)
	if err != nil {
		return nil, err
	}

	return binaryserialization.DeserializeAccessToken(buffer)
}

func (tms *IggyTcpClient) DeletePersonalAccessToken(name string) error {
	message := binaryserialization.SerializeDeletePersonalAccessToken(iggcon.DeletePersonalAccessTokenRequest{
		Name: name,
	})
	_, err := tms.sendAndFetchResponse(message, iggcon.DeleteAccessTokenCode)
	return err
}

func (tms *IggyTcpClient) GetPersonalAccessTokens() ([]iggcon.PersonalAccessTokenInfo, error) {
	buffer, err := tms.sendAndFetchResponse([]byte{}, iggcon.GetAccessTokensCode)
	if err != nil {
		return nil, err
	}

	return binaryserialization.DeserializeAccessTokens(buffer)
}
