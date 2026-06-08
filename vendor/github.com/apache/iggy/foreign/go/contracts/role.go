/* Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package iggcon

import (
	ierror "github.com/apache/iggy/foreign/go/errors"
)

type ClusterNodeRole byte

const (
	RoleLeader   ClusterNodeRole = 0
	RoleFollower ClusterNodeRole = 1
)

// ClusterNodeRoleTryFrom validates a raw byte and returns the corresponding role.
func ClusterNodeRoleTryFrom(b byte) (ClusterNodeRole, error) {
	switch ClusterNodeRole(b) {
	case RoleLeader, RoleFollower:
		return ClusterNodeRole(b), nil
	default:
		return 0, ierror.ErrInvalidCommand
	}
}

func (r *ClusterNodeRole) MarshalBinary() ([]byte, error) {
	return []byte{byte(*r)}, nil
}

func (r *ClusterNodeRole) UnmarshalBinary(b []byte) error {
	if len(b) == 0 {
		return ierror.ErrInvalidCommand
	}
	v, err := ClusterNodeRoleTryFrom(b[0])
	if err != nil {
		return err
	}
	*r = v
	return nil
}

func (r *ClusterNodeRole) String() string {
	switch *r {
	case RoleLeader:
		return "leader"
	case RoleFollower:
		return "follower"
	default:
		return "unknown"
	}
}
