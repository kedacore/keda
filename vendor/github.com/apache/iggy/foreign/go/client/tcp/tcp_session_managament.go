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
	"time"

	binaryserialization "github.com/apache/iggy/foreign/go/binary_serialization"
	"github.com/apache/iggy/foreign/go/internal/util"

	iggcon "github.com/apache/iggy/foreign/go/contracts"
)

func (c *IggyTcpClient) LoginUser(username string, password string) (*iggcon.IdentityInfo, error) {
	serializedRequest := binaryserialization.TcpLogInRequest{
		Username: username,
		Password: password,
	}
	buffer, err := c.sendAndFetchResponse(serializedRequest.Serialize(), iggcon.LoginUserCode)
	if err != nil {
		return nil, err
	}

	identity := binaryserialization.DeserializeLogInResponse(buffer)
	shouldRedirect, err := c.HandleLeaderRedirection()
	if err != nil {
		return nil, err
	}
	if shouldRedirect {
		if err = c.connect(); err != nil {
			return nil, err
		}
		return c.LoginUser(username, password)
	}
	return identity, nil
}

func (c *IggyTcpClient) LoginWithPersonalAccessToken(token string) (*iggcon.IdentityInfo, error) {
	message := binaryserialization.SerializeLoginWithPersonalAccessToken(iggcon.LoginWithPersonalAccessTokenRequest{
		Token: token,
	})
	buffer, err := c.sendAndFetchResponse(message, iggcon.LoginWithAccessTokenCode)
	if err != nil {
		return nil, err
	}

	identity := binaryserialization.DeserializeLogInResponse(buffer)
	shouldRedirect, err := c.HandleLeaderRedirection()
	if err != nil {
		return nil, err
	}
	if shouldRedirect {
		if err = c.connect(); err != nil {
			return nil, err
		}
		return c.LoginWithPersonalAccessToken(token)
	}
	return identity, nil
}

func (c *IggyTcpClient) LogoutUser() error {
	_, err := c.sendAndFetchResponse([]byte{}, iggcon.LogoutUserCode)
	return err
}

func (c *IggyTcpClient) HandleLeaderRedirection() (bool, error) {
	// Clone current address
	c.mtx.Lock()
	currentAddress := c.currentServerAddress
	c.mtx.Unlock()

	leaderAddress, err := util.CheckAndRedirectToLeader(
		c,
		currentAddress,
		iggcon.Tcp,
	)
	if err != nil {
		return false, err
	}

	if leaderAddress == "" {
		// No leader redirection
		c.mtx.Lock()
		c.leaderRedirectionState.Reset()
		c.mtx.Unlock()

		return false, nil
	}

	c.mtx.Lock()
	if !c.leaderRedirectionState.CanRedirect() {
		c.mtx.Unlock()
		return false, nil
	}
	c.mtx.Unlock()

	if err = c.disconnect(); err != nil {
		return false, err
	}

	c.mtx.Lock()
	c.leaderRedirectionState.IncrementRedirect(leaderAddress)
	// Clear connectedAt to avoid reestablish delay during redirection
	c.connectedAt = time.Time{}
	c.currentServerAddress = leaderAddress
	c.mtx.Unlock()

	return true, nil
}
