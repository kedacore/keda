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

import (
	"encoding/binary"

	iggcon "github.com/apache/iggy/foreign/go/contracts"
)

type CreateUser struct {
	Username    string              `json:"username"`
	Password    string              `json:"Password"`
	Status      iggcon.UserStatus   `json:"Status"`
	Permissions *iggcon.Permissions `json:"Permissions,omitempty"`
}

func (c *CreateUser) Code() Code {
	return CreateUserCode
}

func (c *CreateUser) MarshalBinary() ([]byte, error) {
	capacity := 4 + len(c.Username) + len(c.Password)
	if c.Permissions != nil {
		capacity += 4 + c.Permissions.Size()
	}

	bytes := make([]byte, capacity)
	position := 0

	bytes[position] = byte(len(c.Username))
	position += 1
	copy(bytes[position:position+len(c.Username)], []byte(c.Username))
	position += len(c.Username)

	bytes[position] = byte(len(c.Password))
	position += 1
	copy(bytes[position:position+len(c.Password)], []byte(c.Password))
	position += len(c.Password)

	statusByte := byte(0)
	switch c.Status {
	case iggcon.Active:
		statusByte = byte(1)
	case iggcon.Inactive:
		statusByte = byte(2)
	}
	bytes[position] = statusByte
	position += 1

	if c.Permissions != nil {
		bytes[position] = byte(1)
		position += 1
		permissionsBytes, err := c.Permissions.MarshalBinary()
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint32(bytes[position:position+4], uint32(len(permissionsBytes)))
		position += 4
		copy(bytes[position:position+len(permissionsBytes)], permissionsBytes)
	} else {
		bytes[position] = byte(0)
	}

	return bytes, nil
}

type GetUser struct {
	Id iggcon.Identifier
}

func (c *GetUser) Code() Code {
	return GetUserCode
}

func (c *GetUser) MarshalBinary() ([]byte, error) {
	return c.Id.MarshalBinary()
}

type GetUsers struct{}

func (g *GetUsers) Code() Code {
	return GetUsersCode
}

func (g *GetUsers) MarshalBinary() ([]byte, error) {
	return []byte{}, nil
}

type UpdatePermissions struct {
	UserID      iggcon.Identifier   `json:"-"`
	Permissions *iggcon.Permissions `json:"Permissions,omitempty"`
}

func (u *UpdatePermissions) Code() Code {
	return UpdatePermissionsCode
}

func (u *UpdatePermissions) MarshalBinary() ([]byte, error) {
	userIdBytes, err := u.UserID.MarshalBinary()
	if err != nil {
		return nil, err
	}
	length := len(userIdBytes) + 1

	if u.Permissions != nil {
		length += 4 + u.Permissions.Size()
	}

	bytes := make([]byte, length)
	position := 0

	copy(bytes[position:position+len(userIdBytes)], userIdBytes)
	position += len(userIdBytes)

	if u.Permissions != nil {
		bytes[position] = 1
		position++
		permissionsBytes, err := u.Permissions.MarshalBinary()
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint32(bytes[position:position+4], uint32(len(permissionsBytes)))
		position += 4
		copy(bytes[position:position+len(permissionsBytes)], permissionsBytes)
	} else {
		bytes[position] = 0
	}

	return bytes, nil
}

type ChangePassword struct {
	UserID          iggcon.Identifier `json:"-"`
	CurrentPassword string            `json:"CurrentPassword"`
	NewPassword     string            `json:"NewPassword"`
}

func (c *ChangePassword) Code() Code {
	return ChangePasswordCode
}

func (c *ChangePassword) MarshalBinary() ([]byte, error) {
	userIdBytes, err := c.UserID.MarshalBinary()
	if err != nil {
		return nil, err
	}
	length := len(userIdBytes) + len(c.CurrentPassword) + len(c.NewPassword) + 2
	bytes := make([]byte, length)
	position := 0

	copy(bytes[position:position+len(userIdBytes)], userIdBytes)
	position += len(userIdBytes)

	bytes[position] = byte(len(c.CurrentPassword))
	position++
	copy(bytes[position:position+len(c.CurrentPassword)], c.CurrentPassword)
	position += len(c.CurrentPassword)

	bytes[position] = byte(len(c.NewPassword))
	position++
	copy(bytes[position:position+len(c.NewPassword)], c.NewPassword)

	return bytes, nil
}

type DeleteUser struct {
	Id iggcon.Identifier
}

func (d *DeleteUser) Code() Code {
	return DeleteUserCode
}

func (d *DeleteUser) MarshalBinary() ([]byte, error) {
	return d.Id.MarshalBinary()
}
