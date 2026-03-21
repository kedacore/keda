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

// TransportEndpoints represents four 16-bit ports.
type TransportEndpoints struct {
	Tcp       uint16
	Quic      uint16
	Http      uint16
	WebSocket uint16
}

func (t *TransportEndpoints) GetBufferSize() int {
	return 8
}

// NewTransportEndpoints constructs a TransportEndpoints value.
func NewTransportEndpoints(tcp, quic, http, websocket uint16) TransportEndpoints {
	return TransportEndpoints{
		Tcp:       tcp,
		Quic:      quic,
		Http:      http,
		WebSocket: websocket,
	}
}

func (t *TransportEndpoints) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, t.GetBufferSize()))
	_ = binary.Write(buf, binary.LittleEndian, t.Tcp)
	_ = binary.Write(buf, binary.LittleEndian, t.Quic)
	_ = binary.Write(buf, binary.LittleEndian, t.Http)
	_ = binary.Write(buf, binary.LittleEndian, t.WebSocket)
	return buf.Bytes(), nil
}

func (t *TransportEndpoints) UnmarshalBinary(b []byte) error {
	if len(b) < t.GetBufferSize() {
		return ierror.ErrInvalidCommand
	}

	*t = TransportEndpoints{
		Tcp:       binary.LittleEndian.Uint16(b[0:2]),
		Quic:      binary.LittleEndian.Uint16(b[2:4]),
		Http:      binary.LittleEndian.Uint16(b[4:6]),
		WebSocket: binary.LittleEndian.Uint16(b[6:8]),
	}
	return nil
}

func (t *TransportEndpoints) String() string {
	return fmt.Sprintf("tcp: %d, quic: %d, http: %d, websocket: %d", t.Tcp, t.Quic, t.Http, t.WebSocket)
}
