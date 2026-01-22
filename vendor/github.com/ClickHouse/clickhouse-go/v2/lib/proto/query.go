// Licensed to ClickHouse, Inc. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. ClickHouse, Inc. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package proto

import (
	stdbin "encoding/binary"
	"fmt"
	chproto "github.com/ClickHouse/ch-go/proto"
	"go.opentelemetry.io/otel/trace"
	"os"
	"strings"
)

var (
	osUser      = os.Getenv("USER")
	hostname, _ = os.Hostname()
)

type Query struct {
	ID                       string
	ClientName               string
	ClientVersion            Version
	ClientTCPProtocolVersion uint64
	Span                     trace.SpanContext
	Body                     string
	QuotaKey                 string
	Settings                 Settings
	Parameters               Parameters
	Compression              bool
	InitialUser              string
	InitialAddress           string
}

func (q *Query) Encode(buffer *chproto.Buffer, revision uint64) error {
	buffer.PutString(q.ID)
	// client_info
	if err := q.encodeClientInfo(buffer, revision); err != nil {
		return err
	}
	// settings
	if err := q.Settings.Encode(buffer, revision); err != nil {
		return err
	}
	buffer.PutString("") /* empty string is a marker of the end of setting */

	if revision >= DBMS_MIN_REVISION_WITH_INTERSERVER_SECRET {
		buffer.PutString("")
	}
	{
		buffer.PutByte(StateComplete)
		buffer.PutBool(q.Compression)
	}
	buffer.PutString(q.Body)

	if revision >= DBMS_MIN_PROTOCOL_VERSION_WITH_PARAMETERS {
		if err := q.Parameters.Encode(buffer, revision); err != nil {
			return err
		}
		buffer.PutString("") /* empty string is a marker of the end of parameters */
	}

	return nil
}

func swap64(b []byte) {
	for i := 0; i < len(b); i += 8 {
		u := stdbin.BigEndian.Uint64(b[i:])
		stdbin.LittleEndian.PutUint64(b[i:], u)
	}
}

func (q *Query) encodeClientInfo(buffer *chproto.Buffer, revision uint64) error {
	buffer.PutByte(ClientQueryInitial)
	buffer.PutString(q.InitialUser)    // initial_user
	buffer.PutString("")               // initial_query_id
	buffer.PutString(q.InitialAddress) // initial_address
	if revision >= DBMS_MIN_PROTOCOL_VERSION_WITH_INITIAL_QUERY_START_TIME {
		buffer.PutInt64(0) // initial_query_start_time_microseconds
	}
	buffer.PutByte(1) // interface [tcp - 1, http - 2]
	{
		buffer.PutString(osUser)
		buffer.PutString(hostname)
		buffer.PutString(q.ClientName)
		buffer.PutUVarInt(q.ClientVersion.Major)
		buffer.PutUVarInt(q.ClientVersion.Minor)
		buffer.PutUVarInt(q.ClientTCPProtocolVersion)
	}
	if revision >= DBMS_MIN_REVISION_WITH_QUOTA_KEY_IN_CLIENT_INFO {
		buffer.PutString(q.QuotaKey)
	}
	if revision >= DBMS_MIN_PROTOCOL_VERSION_WITH_DISTRIBUTED_DEPTH {
		buffer.PutUVarInt(0)
	}
	if revision >= DBMS_MIN_REVISION_WITH_VERSION_PATCH {
		buffer.PutUVarInt(0)
	}
	if revision >= DBMS_MIN_REVISION_WITH_OPENTELEMETRY {
		switch {
		case q.Span.IsValid():
			buffer.PutByte(1)
			{
				v := q.Span.TraceID()
				swap64(v[:]) // https://github.com/ClickHouse/ClickHouse/issues/34369
				buffer.PutRaw(v[:])
			}
			{
				v := q.Span.SpanID()
				swap64(v[:]) // https://github.com/ClickHouse/ClickHouse/issues/34369
				buffer.PutRaw(v[:])
			}
			buffer.PutString(q.Span.TraceState().String())
			buffer.PutByte(byte(q.Span.TraceFlags()))

		default:
			buffer.PutByte(0)
		}
	}
	if revision >= DBMS_MIN_REVISION_WITH_PARALLEL_REPLICAS {
		buffer.PutUVarInt(0) // collaborate_with_initiator
		buffer.PutUVarInt(0) // count_participating_replicas
		buffer.PutUVarInt(0) // number_of_current_replica
	}
	return nil
}

type Settings []Setting

type Setting struct {
	Key       string
	Value     any
	Important bool
	Custom    bool
}

const (
	settingFlagImportant = 0x01
	settingFlagCustom    = 0x02
)

func (s Settings) Encode(buffer *chproto.Buffer, revision uint64) error {
	for _, s := range s {
		if err := s.encode(buffer, revision); err != nil {
			return err
		}
	}
	return nil
}

func (s *Setting) encode(buffer *chproto.Buffer, revision uint64) error {
	buffer.PutString(s.Key)
	if revision <= DBMS_MIN_REVISION_WITH_SETTINGS_SERIALIZED_AS_STRINGS {
		var value uint64
		switch v := s.Value.(type) {
		case int:
			value = uint64(v)
		case bool:
			if value = 0; v {
				value = 1
			}
		default:
			return fmt.Errorf("query setting %s has unsupported data type", s.Key)
		}
		buffer.PutUVarInt(value)
		return nil
	}

	{
		var flags uint64
		if s.Important {
			flags |= settingFlagImportant
		}
		if s.Custom {
			flags |= settingFlagCustom
		}
		buffer.PutUVarInt(flags)
	}

	if s.Custom {
		fieldDump, err := encodeFieldDump(s.Value)
		if err != nil {
			return err
		}

		buffer.PutString(fieldDump)
	} else {
		buffer.PutString(fmt.Sprint(s.Value))
	}

	return nil
}

type Parameters []Parameter

type Parameter struct {
	Key   string
	Value string
}

func (s Parameters) Encode(buffer *chproto.Buffer, revision uint64) error {
	for _, s := range s {
		if err := s.encode(buffer, revision); err != nil {
			return err
		}
	}
	return nil
}

func (s *Parameter) encode(buffer *chproto.Buffer, revision uint64) error {
	buffer.PutString(s.Key)
	buffer.PutUVarInt(uint64(settingFlagCustom))

	fieldDump, err := encodeFieldDump(s.Value)
	if err != nil {
		return err
	}

	buffer.PutString(fieldDump)

	return nil
}

// encodes a field dump with an appropriate type format
// implements the same logic as in ClickHouse Field::restoreFromDump (https://github.com/ClickHouse/ClickHouse/blob/master/src/Core/Field.cpp#L312)
// currently, only string type is supported
func encodeFieldDump(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("'%v'", strings.ReplaceAll(v, "'", "\\'")), nil
	}

	return "", fmt.Errorf("unsupported field type %T", value)
}
