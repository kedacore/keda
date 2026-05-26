// Copyright (c) 2021 VMware, Inc. or its affiliates. All Rights Reserved.
// Copyright (c) 2012-2021, Sean Treadway, SoundCloud Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package amqp091

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"time"
)

func (w *writer) WriteFrameNoFlush(frame frame) (err error) {
	err = frame.write(w.w)
	return
}

func (w *writer) WriteFrame(frame frame) (err error) {
	if err = frame.write(w.w); err != nil {
		return
	}

	if buf, ok := w.w.(*bufio.Writer); ok {
		err = buf.Flush()
	}

	return
}

func (f *methodFrame) write(w io.Writer) (err error) {
	var payload bytes.Buffer

	if f.Method == nil {
		return errors.New("malformed frame: missing method")
	}

	class, method := f.Method.id()

	if err = binary.Write(&payload, binary.BigEndian, class); err != nil {
		return
	}

	if err = binary.Write(&payload, binary.BigEndian, method); err != nil {
		return
	}

	if err = f.Method.write(&payload); err != nil {
		return
	}

	return writeFrame(w, frameMethod, f.ChannelId, payload.Bytes())
}

// Heartbeat
//
// Payload is empty
func (f *heartbeatFrame) write(w io.Writer) (err error) {
	return writeFrame(w, frameHeartbeat, f.ChannelId, []byte{})
}

// CONTENT HEADER
// 0          2        4           12               14
// +----------+--------+-----------+----------------+------------- - -
// | class-id | weight | body size | property flags | property list...
// +----------+--------+-----------+----------------+------------- - -
//
//	short     short    long long       short        remainder...
func (f *headerFrame) write(w io.Writer) (err error) {
	var payload bytes.Buffer

	if err = binary.Write(&payload, binary.BigEndian, f.ClassId); err != nil {
		return
	}

	if err = binary.Write(&payload, binary.BigEndian, f.weight); err != nil {
		return
	}

	if err = binary.Write(&payload, binary.BigEndian, f.Size); err != nil {
		return
	}

	// First pass will build the mask to be serialized, second pass will serialize
	// each of the fields that appear in the mask.

	var mask uint16

	if f.Properties.ContentType != "" {
		mask |= flagContentType
	}
	if f.Properties.ContentEncoding != "" {
		mask |= flagContentEncoding
	}
	if len(f.Properties.Headers) > 0 {
		mask |= flagHeaders
	}
	if f.Properties.DeliveryMode > 0 {
		mask |= flagDeliveryMode
	}
	if f.Properties.Priority > 0 {
		mask |= flagPriority
	}
	if f.Properties.CorrelationId != "" {
		mask |= flagCorrelationId
	}
	if f.Properties.ReplyTo != "" {
		mask |= flagReplyTo
	}
	if f.Properties.Expiration != "" {
		mask |= flagExpiration
	}
	if f.Properties.MessageId != "" {
		mask |= flagMessageId
	}
	if !f.Properties.Timestamp.IsZero() {
		mask |= flagTimestamp
	}
	if f.Properties.Type != "" {
		mask |= flagType
	}
	if f.Properties.UserId != "" {
		mask |= flagUserId
	}
	if f.Properties.AppId != "" {
		mask |= flagAppId
	}

	if err = binary.Write(&payload, binary.BigEndian, mask); err != nil {
		return
	}

	if hasProperty(mask, flagContentType) {
		if err = writeShortstr(&payload, f.Properties.ContentType); err != nil {
			return
		}
	}
	if hasProperty(mask, flagContentEncoding) {
		if err = writeShortstr(&payload, f.Properties.ContentEncoding); err != nil {
			return
		}
	}
	if hasProperty(mask, flagHeaders) {
		if err = writeTable(&payload, f.Properties.Headers); err != nil {
			return
		}
	}
	if hasProperty(mask, flagDeliveryMode) {
		if err = binary.Write(&payload, binary.BigEndian, f.Properties.DeliveryMode); err != nil {
			return
		}
	}
	if hasProperty(mask, flagPriority) {
		if err = binary.Write(&payload, binary.BigEndian, f.Properties.Priority); err != nil {
			return
		}
	}
	if hasProperty(mask, flagCorrelationId) {
		if err = writeShortstr(&payload, f.Properties.CorrelationId); err != nil {
			return
		}
	}
	if hasProperty(mask, flagReplyTo) {
		if err = writeShortstr(&payload, f.Properties.ReplyTo); err != nil {
			return
		}
	}
	if hasProperty(mask, flagExpiration) {
		if err = writeShortstr(&payload, f.Properties.Expiration); err != nil {
			return
		}
	}
	if hasProperty(mask, flagMessageId) {
		if err = writeShortstr(&payload, f.Properties.MessageId); err != nil {
			return
		}
	}
	if hasProperty(mask, flagTimestamp) {
		if err = binary.Write(&payload, binary.BigEndian, uint64(f.Properties.Timestamp.Unix())); err != nil {
			return
		}
	}
	if hasProperty(mask, flagType) {
		if err = writeShortstr(&payload, f.Properties.Type); err != nil {
			return
		}
	}
	if hasProperty(mask, flagUserId) {
		if err = writeShortstr(&payload, f.Properties.UserId); err != nil {
			return
		}
	}
	if hasProperty(mask, flagAppId) {
		if err = writeShortstr(&payload, f.Properties.AppId); err != nil {
			return
		}
	}

	return writeFrame(w, frameHeader, f.ChannelId, payload.Bytes())
}

// Body
//
// Payload is one byterange from the full body who's size is declared in the
// Header frame
func (f *bodyFrame) write(w io.Writer) (err error) {
	return writeFrame(w, frameBody, f.ChannelId, f.Body)
}

func writeFrame(w io.Writer, typ uint8, channel uint16, payload []byte) (err error) {
	end := []byte{frameEnd}
	size := uint(len(payload))

	_, err = w.Write([]byte{
		typ,
		byte((channel & 0xff00) >> 8),
		byte((channel & 0x00ff) >> 0),
		byte((size & 0xff000000) >> 24),
		byte((size & 0x00ff0000) >> 16),
		byte((size & 0x0000ff00) >> 8),
		byte((size & 0x000000ff) >> 0),
	})
	if err != nil {
		return
	}

	if _, err = w.Write(payload); err != nil {
		return
	}

	if _, err = w.Write(end); err != nil {
		return
	}

	return
}

func writeShortstr(w io.Writer, s string) (err error) {
	b := []byte(s)

	length := uint8(len(b))

	if err = binary.Write(w, binary.BigEndian, length); err != nil {
		return
	}

	if _, err = w.Write(b[:length]); err != nil {
		return
	}

	return
}

func writeLongstr(w io.Writer, s string) (err error) {
	b := []byte(s)

	length := uint32(len(b))

	if err = binary.Write(w, binary.BigEndian, length); err != nil {
		return
	}

	if _, err = w.Write(b[:length]); err != nil {
		return
	}

	return
}

/*
'A': []any
'D': Decimal
'F': Table
'I': int32
'S': string
'T': time.Time
'V': nil
'b': int8
'B': byte
'd': float64
'f': float32
'l': int64
'i': uint32
's': int16
'u': uint16
't': bool
'x': []byte
*/
func writeField(w io.Writer, value any) (err error) {
	var buf [9]byte
	var enc []byte

	switch v := value.(type) {
	case bool:
		buf[0] = 't'
		if v {
			buf[1] = byte(1)
		} else {
			buf[1] = byte(0)
		}
		enc = buf[:2]

	case byte:
		buf[0] = 'B'
		buf[1] = v
		enc = buf[:2]

	case int8:
		buf[0] = 'b'
		buf[1] = uint8(v)
		enc = buf[:2]

	case int16:
		buf[0] = 's'
		binary.BigEndian.PutUint16(buf[1:3], uint16(v))
		enc = buf[:3]

	case int:
		buf[0] = 'I'
		binary.BigEndian.PutUint32(buf[1:5], uint32(v))
		enc = buf[:5]

	case int32:
		buf[0] = 'I'
		binary.BigEndian.PutUint32(buf[1:5], uint32(v))
		enc = buf[:5]

	case int64:
		buf[0] = 'l'
		binary.BigEndian.PutUint64(buf[1:9], uint64(v))
		enc = buf[:9]

	case uint16:
		buf[0] = 'u'
		binary.BigEndian.PutUint16(buf[1:3], v)
		enc = buf[:3]

	case uint32:
		buf[0] = 'i'
		binary.BigEndian.PutUint32(buf[1:5], v)
		enc = buf[:5]

	case float32:
		buf[0] = 'f'
		binary.BigEndian.PutUint32(buf[1:5], math.Float32bits(v))
		enc = buf[:5]

	case float64:
		buf[0] = 'd'
		binary.BigEndian.PutUint64(buf[1:9], math.Float64bits(v))
		enc = buf[:9]

	case Decimal:
		buf[0] = 'D'
		buf[1] = v.Scale
		binary.BigEndian.PutUint32(buf[2:6], uint32(v.Value))
		enc = buf[:6]

	case string:
		buf[0] = 'S'
		binary.BigEndian.PutUint32(buf[1:5], uint32(len(v)))
		enc = append(buf[:5], []byte(v)...)

	case []any: // field-array
		buf[0] = 'A'

		sec := new(bytes.Buffer)
		for _, val := range v {
			if err = writeField(sec, val); err != nil {
				return
			}
		}

		binary.BigEndian.PutUint32(buf[1:5], uint32(sec.Len()))
		if _, err = w.Write(buf[:5]); err != nil {
			return
		}

		if _, err = w.Write(sec.Bytes()); err != nil {
			return
		}

		return

	case time.Time:
		buf[0] = 'T'
		binary.BigEndian.PutUint64(buf[1:9], uint64(v.Unix()))
		enc = buf[:9]

	case Table:
		if _, err = w.Write([]byte{'F'}); err != nil {
			return
		}
		return writeTable(w, v)

	case []byte:
		buf[0] = 'x'
		binary.BigEndian.PutUint32(buf[1:5], uint32(len(v)))
		if _, err = w.Write(buf[0:5]); err != nil {
			return
		}
		if _, err = w.Write(v); err != nil {
			return
		}
		return

	case nil:
		buf[0] = 'V'
		enc = buf[:1]

	default:
		return ErrFieldType
	}

	_, err = w.Write(enc)

	return
}

// writeTable serializes a Table to the given writer.
// It writes each key-value pair and returns the serialized data as a longstr.
func writeTable(w io.Writer, table Table) (err error) {
	var buf bytes.Buffer

	for key, val := range table {
		if err = writeShortstr(&buf, key); err != nil {
			return fmt.Errorf("writing key %q: %w", key, err)
		}
		if err = writeField(&buf, val); err != nil {
			return fmt.Errorf("writing value for key %q: %w", key, err)
		}
	}

	if err := writeLongstr(w, buf.String()); err != nil {
		return fmt.Errorf("writing final long string: %w", err)
	}

	return nil
}
