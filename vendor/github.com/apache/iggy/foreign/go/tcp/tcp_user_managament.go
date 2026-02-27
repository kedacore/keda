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

func (tms *IggyTcpClient) GetUser(identifier iggcon.Identifier) (*iggcon.UserInfoDetails, error) {
	message := binaryserialization.SerializeIdentifier(identifier)
	buffer, err := tms.sendAndFetchResponse(message, iggcon.GetUserCode)
	if err != nil {
		return nil, err
	}
	if len(buffer) == 0 {
		return nil, ierror.ErrResourceNotFound
	}

	return binaryserialization.DeserializeUser(buffer)
}

func (tms *IggyTcpClient) GetUsers() ([]iggcon.UserInfo, error) {
	buffer, err := tms.sendAndFetchResponse([]byte{}, iggcon.GetUsersCode)
	if err != nil {
		return nil, err
	}

	return binaryserialization.DeserializeUsers(buffer)
}

func (tms *IggyTcpClient) CreateUser(username string, password string, status iggcon.UserStatus, permissions *iggcon.Permissions) (*iggcon.UserInfoDetails, error) {
	message := binaryserialization.SerializeCreateUserRequest(iggcon.CreateUserRequest{
		Username:    username,
		Password:    password,
		Status:      status,
		Permissions: permissions,
	})
	buffer, err := tms.sendAndFetchResponse(message, iggcon.CreateUserCode)
	if err != nil {
		return nil, err
	}
	userInfo, err := binaryserialization.DeserializeUser(buffer)
	if err != nil {
		return nil, err
	}
	return userInfo, nil
}

func (tms *IggyTcpClient) UpdateUser(userID iggcon.Identifier, username *string, status *iggcon.UserStatus) error {
	message := binaryserialization.SerializeUpdateUser(iggcon.UpdateUserRequest{
		UserID:   userID,
		Username: username,
		Status:   status,
	})
	_, err := tms.sendAndFetchResponse(message, iggcon.UpdateUserCode)
	return err
}

func (tms *IggyTcpClient) DeleteUser(identifier iggcon.Identifier) error {
	message := binaryserialization.SerializeIdentifier(identifier)
	_, err := tms.sendAndFetchResponse(message, iggcon.DeleteUserCode)
	return err
}

func (tms *IggyTcpClient) UpdatePermissions(userID iggcon.Identifier, permissions *iggcon.Permissions) error {
	message := binaryserialization.SerializeUpdateUserPermissionsRequest(iggcon.UpdatePermissionsRequest{
		UserID:      userID,
		Permissions: permissions,
	})
	_, err := tms.sendAndFetchResponse(message, iggcon.UpdatePermissionsCode)
	return err
}

func (tms *IggyTcpClient) ChangePassword(userID iggcon.Identifier, currentPassword string, newPassword string) error {
	message := binaryserialization.SerializeChangePasswordRequest(iggcon.ChangePasswordRequest{
		UserID:          userID,
		CurrentPassword: currentPassword,
		NewPassword:     newPassword,
	})
	_, err := tms.sendAndFetchResponse(message, iggcon.ChangePasswordCode)
	return err
}
