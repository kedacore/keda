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
	"github.com/apache/iggy/foreign/go/internal/command"
)

func (c *IggyTcpClient) GetUser(identifier iggcon.Identifier) (*iggcon.UserInfoDetails, error) {
	buffer, err := c.do(&command.GetUser{Id: identifier})
	if err != nil {
		return nil, err
	}
	if len(buffer) == 0 {
		return nil, ierror.ErrResourceNotFound
	}

	return binaryserialization.DeserializeUser(buffer)
}

func (c *IggyTcpClient) GetUsers() ([]iggcon.UserInfo, error) {
	buffer, err := c.do(&command.GetUsers{})
	if err != nil {
		return nil, err
	}

	return binaryserialization.DeserializeUsers(buffer)
}

func (c *IggyTcpClient) CreateUser(username string, password string, status iggcon.UserStatus, permissions *iggcon.Permissions) (*iggcon.UserInfoDetails, error) {
	buffer, err := c.do(&command.CreateUser{
		Username:    username,
		Password:    password,
		Status:      status,
		Permissions: permissions,
	})
	if err != nil {
		return nil, err
	}
	userInfo, err := binaryserialization.DeserializeUser(buffer)
	if err != nil {
		return nil, err
	}
	return userInfo, nil
}

func (c *IggyTcpClient) UpdateUser(userID iggcon.Identifier, username *string, status *iggcon.UserStatus) error {
	_, err := c.do(&command.UpdateUser{
		UserID:   userID,
		Username: username,
		Status:   status,
	})
	return err
}

func (c *IggyTcpClient) DeleteUser(identifier iggcon.Identifier) error {
	_, err := c.do(&command.DeleteUser{
		Id: identifier,
	})
	return err
}

func (c *IggyTcpClient) UpdatePermissions(userID iggcon.Identifier, permissions *iggcon.Permissions) error {
	_, err := c.do(&command.UpdatePermissions{
		UserID:      userID,
		Permissions: permissions,
	})
	return err
}

func (c *IggyTcpClient) ChangePassword(userID iggcon.Identifier, currentPassword string, newPassword string) error {
	_, err := c.do(&command.ChangePassword{
		UserID:          userID,
		CurrentPassword: currentPassword,
		NewPassword:     newPassword,
	})
	return err
}
