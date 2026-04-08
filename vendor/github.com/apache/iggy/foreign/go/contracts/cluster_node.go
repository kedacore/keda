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
	"bytes"
	"encoding/binary"
	"fmt"

	ierror "github.com/apache/iggy/foreign/go/errors"
)

type ClusterNode struct {
	Name      string
	IP        string
	Endpoints TransportEndpoints
	Role      ClusterNodeRole
	Status    ClusterNodeStatus
}

func (n *ClusterNode) BufferSize() int {
	return 4 + len(n.Name) + 4 + len(n.IP) + n.Endpoints.GetBufferSize() + 1 + 1
}

func (n *ClusterNode) MarshalBinary() ([]byte, error) {
	size := n.BufferSize()
	buf := bytes.NewBuffer(make([]byte, 0, size))

	// name
	if err := binary.Write(buf, binary.LittleEndian, uint32(len(n.Name))); err != nil {
		return nil, err
	}
	if _, err := buf.WriteString(n.Name); err != nil {
		return nil, err
	}

	// ip
	if err := binary.Write(buf, binary.LittleEndian, uint32(len(n.IP))); err != nil {
		return nil, err
	}
	if _, err := buf.WriteString(n.IP); err != nil {
		return nil, err
	}

	// endpoints (use MarshalBinary on TransportEndpoints)
	epb, err := n.Endpoints.MarshalBinary()
	if err != nil {
		return nil, err
	}
	if _, err := buf.Write(epb); err != nil {
		return nil, err
	}

	// role
	rb, err := n.Role.MarshalBinary()
	if err != nil {
		return nil, err
	}
	if _, err := buf.Write(rb); err != nil {
		return nil, err
	}

	// status
	sb, err := n.Status.MarshalBinary()
	if err != nil {
		return nil, err
	}
	if _, err := buf.Write(sb); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (n *ClusterNode) UnmarshalBinary(b []byte) error {
	// Minimal size check: 4 (name_len) + 4 (ip_len) + 1 (role) + 1 (status)
	if len(b) < 10 {
		return ierror.ErrInvalidCommand
	}
	pos := 0

	// name length
	if len(b) < pos+4 {
		return ierror.ErrInvalidNumberEncoding
	}
	nameLen := int(binary.LittleEndian.Uint32(b[pos : pos+4]))
	pos += 4
	if len(b) < pos+nameLen {
		return ierror.ErrInvalidCommand
	}
	n.Name = string(b[pos : pos+nameLen])
	pos += nameLen

	// ip length
	if len(b) < pos+4 {
		return ierror.ErrInvalidNumberEncoding
	}
	ipLen := int(binary.LittleEndian.Uint32(b[pos : pos+4]))
	pos += 4
	if len(b) < pos+ipLen {
		return ierror.ErrInvalidCommand
	}
	n.IP = string(b[pos : pos+ipLen])
	pos += ipLen

	// endpoints: use BufferSize and UnmarshalBinary
	ep := TransportEndpoints{}
	epSize := ep.GetBufferSize()
	if len(b) < pos+epSize {
		return ierror.ErrInvalidCommand
	}
	if err := ep.UnmarshalBinary(b[pos : pos+epSize]); err != nil {
		return err
	}
	n.Endpoints = ep
	pos += epSize

	// role (1 byte)
	if len(b) < pos+1 {
		return ierror.ErrInvalidCommand
	}
	var r ClusterNodeRole
	if err := r.UnmarshalBinary(b[pos : pos+1]); err != nil {
		return err
	}
	n.Role = r
	pos++

	// status (1 byte)
	if len(b) < pos+1 {
		return ierror.ErrInvalidCommand
	}
	var s ClusterNodeStatus
	if err := s.UnmarshalBinary(b[pos : pos+1]); err != nil {
		return err
	}
	n.Status = s

	return nil
}

func (n *ClusterNode) String() string {
	return fmt.Sprintf("ClusterNode { name: %s, ip: %s, endpoints: %s, role: %s, status: %s }",
		n.Name, n.IP, n.Endpoints.String(), n.Role.String(), n.Status.String())
}
