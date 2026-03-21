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

package iggcon

import (
	ierror "github.com/apache/iggy/foreign/go/errors"
)

type ClusterNodeStatus uint8

const (
	// Healthy indicates node is healthy and responsive
	Healthy ClusterNodeStatus = 0
	// Starting indicates node is starting up
	Starting ClusterNodeStatus = 1
	// Stopping indicates node is shutting down
	Stopping ClusterNodeStatus = 2
	// Unreachable indicates node is unreachable
	Unreachable ClusterNodeStatus = 3
	// Maintenance indicates node is in maintenance mode
	Maintenance ClusterNodeStatus = 4
	// Unknown indicates node is unknown
	Unknown ClusterNodeStatus = 5
)

func TryFrom(b byte) (ClusterNodeStatus, error) {
	switch ClusterNodeStatus(b) {
	case Healthy, Starting, Stopping, Unreachable, Maintenance:
		return ClusterNodeStatus(b), nil
	default:
		return 0, ierror.ErrInvalidCommand
	}
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (s *ClusterNodeStatus) MarshalBinary() ([]byte, error) {
	return []byte{byte(*s)}, nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (s *ClusterNodeStatus) UnmarshalBinary(b []byte) error {
	if len(b) == 0 {
		return ierror.ErrInvalidCommand
	}
	v, err := TryFrom(b[0])
	if err != nil {
		return err
	}
	*s = v
	return nil
}

// String implements fmt.Stringer with lowercase names (matches serde/strum settings).
func (s *ClusterNodeStatus) String() string {
	switch *s {
	case Healthy:
		return "healthy"
	case Starting:
		return "starting"
	case Stopping:
		return "stopping"
	case Unreachable:
		return "unreachable"
	case Maintenance:
		return "maintenance"
	default:
		return "unknown"
	}
}
