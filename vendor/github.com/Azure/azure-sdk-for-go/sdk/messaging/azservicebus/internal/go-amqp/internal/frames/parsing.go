// Copyright (C) 2017 Kale Blankenship
// Portions Copyright (c) Microsoft Corporation
package frames

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp/internal/buffer"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp/internal/encoding"
)

const HeaderSize = 8

// Frame structure:
//
//     header (8 bytes)
//       0-3: SIZE (total size, at least 8 bytes for header, uint32)
//       4:   DOFF (data offset,at least 2, count of 4 bytes words, uint8)
//       5:   TYPE (frame type)
//                0x0: AMQP
//                0x1: SASL
//       6-7: type dependent (channel for AMQP)
//     extended header (opt)
//     body (opt)

// Header in a structure appropriate for use with binary.Read()
type Header struct {
	// size: an unsigned 32-bit integer that MUST contain the total frame size of the frame header,
	// extended header, and frame body. The frame is malformed if the size is less than the size of
	// the frame header (8 bytes).
	Size uint32
	// doff: gives the position of the body within the frame. The value of the data offset is an
	// unsigned, 8-bit integer specifying a count of 4-byte words. Due to the mandatory 8-byte
	// frame header, the frame is malformed if the value is less than 2.
	DataOffset uint8
	FrameType  uint8
	Channel    uint16
}

// ParseHeader reads the header from r and returns the result.
//
// No validation is done.
func ParseHeader(r *buffer.Buffer) (Header, error) {
	buf, ok := r.Next(8)
	if !ok {
		return Header{}, errors.New("invalid frameHeader")
	}
	_ = buf[7]

	fh := Header{
		Size:       binary.BigEndian.Uint32(buf[0:4]),
		DataOffset: buf[4],
		FrameType:  buf[5],
		Channel:    binary.BigEndian.Uint16(buf[6:8]),
	}

	if fh.Size < HeaderSize {
		return fh, fmt.Errorf("received frame header with invalid size %d", fh.Size)
	}

	if fh.DataOffset < 2 {
		return fh, fmt.Errorf("received frame header with invalid data offset %d", fh.DataOffset)
	}

	return fh, nil
}

// ParseBody reads and unmarshals an AMQP frame.
func ParseBody(r *buffer.Buffer) (FrameBody, error) {
	payload := r.Bytes()

	if r.Len() < 3 || payload[0] != 0 || encoding.AMQPType(payload[1]) != encoding.TypeCodeSmallUlong {
		return nil, errors.New("invalid frame body header")
	}

	switch pType := encoding.AMQPType(payload[2]); pType {
	case encoding.TypeCodeOpen:
		t := new(PerformOpen)
		err := t.Unmarshal(r)
		return t, err
	case encoding.TypeCodeBegin:
		t := new(PerformBegin)
		err := t.Unmarshal(r)
		return t, err
	case encoding.TypeCodeAttach:
		t := new(PerformAttach)
		err := t.Unmarshal(r)
		return t, err
	case encoding.TypeCodeFlow:
		t := new(PerformFlow)
		err := t.Unmarshal(r)
		return t, err
	case encoding.TypeCodeTransfer:
		t := new(PerformTransfer)
		err := t.Unmarshal(r)
		return t, err
	case encoding.TypeCodeDisposition:
		t := new(PerformDisposition)
		err := t.Unmarshal(r)
		return t, err
	case encoding.TypeCodeDetach:
		t := new(PerformDetach)
		err := t.Unmarshal(r)
		return t, err
	case encoding.TypeCodeEnd:
		t := new(PerformEnd)
		err := t.Unmarshal(r)
		return t, err
	case encoding.TypeCodeClose:
		t := new(PerformClose)
		err := t.Unmarshal(r)
		return t, err
	case encoding.TypeCodeSASLMechanism:
		t := new(SASLMechanisms)
		err := t.Unmarshal(r)
		return t, err
	case encoding.TypeCodeSASLChallenge:
		t := new(SASLChallenge)
		err := t.Unmarshal(r)
		return t, err
	case encoding.TypeCodeSASLOutcome:
		t := new(SASLOutcome)
		err := t.Unmarshal(r)
		return t, err
	default:
		return nil, fmt.Errorf("unknown performative type %02x", pType)
	}
}
