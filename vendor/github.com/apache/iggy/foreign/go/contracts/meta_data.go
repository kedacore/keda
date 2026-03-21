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
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

const MaxNodesPerCluster = 64

type ClusterMetadata struct {
	Name  string
	Nodes []ClusterNode
}

// BufferSize returns total serialized size.
func (m *ClusterMetadata) BufferSize() int {
	nodesSize := 0
	for i := range m.Nodes {
		nodesSize += m.Nodes[i].BufferSize()
	}
	return 4 + len(m.Name) + 4 + nodesSize
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (m *ClusterMetadata) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, m.BufferSize()))
	nameBytes := []byte(m.Name)
	if err := binary.Write(buf, binary.LittleEndian, uint32(len(nameBytes))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(nameBytes); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.LittleEndian, uint32(len(m.Nodes))); err != nil {
		return nil, err
	}

	for i := range m.Nodes {
		data, err := m.Nodes[i].MarshalBinary()
		if err != nil {
			return nil, err
		}
		buf.Write(data)
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (m *ClusterMetadata) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return errors.New("data too short for ClusterMetadata")
	}
	r := bytes.NewReader(data)

	var nameLen uint32
	if err := binary.Read(r, binary.LittleEndian, &nameLen); err != nil {
		return err
	}
	if int(nameLen) > r.Len() {
		return errors.New("invalid name length")
	}
	nameb := make([]byte, nameLen)
	if _, err := r.Read(nameb); err != nil {
		return err
	}
	m.Name = string(nameb)

	var nodesCount uint32
	if err := binary.Read(r, binary.LittleEndian, &nodesCount); err != nil {
		return err
	}

	if nodesCount > MaxNodesPerCluster {
		return errors.New("invalid number of nodes per cluster")
	}

	m.Nodes = make([]ClusterNode, 0, nodesCount)
	for i := uint32(0); i < nodesCount; i++ {
		// remaining bytes for this node are unknown up-front; parse via ClusterNode.UnmarshalBinary on the remaining slice
		remaining := data[len(data)-r.Len():]
		var node ClusterNode
		if err := node.UnmarshalBinary(remaining); err != nil {
			return err
		}
		m.Nodes = append(m.Nodes, node)
		// advance reader by the node's serialized size
		if _, err := r.Seek(int64(node.BufferSize()), io.SeekCurrent); err != nil {
			return err
		}
	}
	return nil
}
