// Copyright (C) 2017 Kale Blankenship
// Portions Copyright (c) Microsoft Corporation
package amqp

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp/internal/encoding"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp/internal/frames"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp/internal/log"
)

// SASL Mechanisms
const (
	saslMechanismPLAIN     encoding.Symbol = "PLAIN"
	saslMechanismANONYMOUS encoding.Symbol = "ANONYMOUS"
	saslMechanismEXTERNAL  encoding.Symbol = "EXTERNAL"
	saslMechanismXOAUTH2   encoding.Symbol = "XOAUTH2"
)

const (
	frameTypeAMQP = 0x0
	frameTypeSASL = 0x1
)

// SASLType represents a SASL configuration to use during authentication.
type SASLType func(c *conn) error

// ConnSASLPlain enables SASL PLAIN authentication for the connection.
//
// SASL PLAIN transmits credentials in plain text and should only be used
// on TLS/SSL enabled connection.
func SASLTypePlain(username, password string) SASLType {
	// TODO: how widely used is hostname? should it be supported
	return func(c *conn) error {
		// make handlers map if no other mechanism has
		if c.saslHandlers == nil {
			c.saslHandlers = make(map[encoding.Symbol]stateFunc)
		}

		// add the handler the the map
		c.saslHandlers[saslMechanismPLAIN] = func() (stateFunc, error) {
			// send saslInit with PLAIN payload
			init := &frames.SASLInit{
				Mechanism:       "PLAIN",
				InitialResponse: []byte("\x00" + username + "\x00" + password),
				Hostname:        "",
			}
			log.Debug(1, "TX (ConnSASLPlain): %s", init)
			err := c.writeFrame(frames.Frame{
				Type: frameTypeSASL,
				Body: init,
			})
			if err != nil {
				return nil, err
			}

			// go to c.saslOutcome to handle the server response
			return c.saslOutcome, nil
		}
		return nil
	}
}

// ConnSASLAnonymous enables SASL ANONYMOUS authentication for the connection.
func SASLTypeAnonymous() SASLType {
	return func(c *conn) error {
		// make handlers map if no other mechanism has
		if c.saslHandlers == nil {
			c.saslHandlers = make(map[encoding.Symbol]stateFunc)
		}

		// add the handler the the map
		c.saslHandlers[saslMechanismANONYMOUS] = func() (stateFunc, error) {
			init := &frames.SASLInit{
				Mechanism:       saslMechanismANONYMOUS,
				InitialResponse: []byte("anonymous"),
			}
			log.Debug(1, "TX (ConnSASLAnonymous): %s", init)
			err := c.writeFrame(frames.Frame{
				Type: frameTypeSASL,
				Body: init,
			})
			if err != nil {
				return nil, err
			}

			// go to c.saslOutcome to handle the server response
			return c.saslOutcome, nil
		}
		return nil
	}
}

// ConnSASLExternal enables SASL EXTERNAL authentication for the connection.
// The value for resp is dependent on the type of authentication (empty string is common for TLS).
// See https://datatracker.ietf.org/doc/html/rfc4422#appendix-A for additional info.
func SASLTypeExternal(resp string) SASLType {
	return func(c *conn) error {
		// make handlers map if no other mechanism has
		if c.saslHandlers == nil {
			c.saslHandlers = make(map[encoding.Symbol]stateFunc)
		}

		// add the handler the the map
		c.saslHandlers[saslMechanismEXTERNAL] = func() (stateFunc, error) {
			init := &frames.SASLInit{
				Mechanism:       saslMechanismEXTERNAL,
				InitialResponse: []byte(resp),
			}
			log.Debug(1, "TX (ConnSASLExternal): %s", init)
			err := c.writeFrame(frames.Frame{
				Type: frameTypeSASL,
				Body: init,
			})
			if err != nil {
				return nil, err
			}

			// go to c.saslOutcome to handle the server response
			return c.saslOutcome, nil
		}
		return nil
	}
}

// ConnSASLXOAUTH2 enables SASL XOAUTH2 authentication for the connection.
//
// The saslMaxFrameSizeOverride parameter allows the limit that governs the maximum frame size this client will allow
// itself to generate to be raised for the sasl-init frame only.  Set this when the size of the size of the SASL XOAUTH2
// initial client response (which contains the username and bearer token) would otherwise breach the 512 byte min-max-frame-size
// (http://docs.oasis-open.org/amqp/core/v1.0/os/amqp-core-transport-v1.0-os.html#definition-MIN-MAX-FRAME-SIZE). Pass -1
// to keep the default.
//
// SASL XOAUTH2 transmits the bearer in plain text and should only be used
// on TLS/SSL enabled connection.
func SASLTypeXOAUTH2(username, bearer string, saslMaxFrameSizeOverride uint32) SASLType {
	return func(c *conn) error {
		// make handlers map if no other mechanism has
		if c.saslHandlers == nil {
			c.saslHandlers = make(map[encoding.Symbol]stateFunc)
		}

		response, err := saslXOAUTH2InitialResponse(username, bearer)
		if err != nil {
			return err
		}

		handler := saslXOAUTH2Handler{
			conn:                 c,
			maxFrameSizeOverride: saslMaxFrameSizeOverride,
			response:             response,
		}
		// add the handler the the map
		c.saslHandlers[saslMechanismXOAUTH2] = handler.init
		return nil
	}
}

type saslXOAUTH2Handler struct {
	conn                 *conn
	maxFrameSizeOverride uint32
	response             []byte
	errorResponse        []byte // https://developers.google.com/gmail/imap/xoauth2-protocol#error_response
}

func (s saslXOAUTH2Handler) init() (stateFunc, error) {
	originalPeerMaxFrameSize := s.conn.PeerMaxFrameSize
	if s.maxFrameSizeOverride > s.conn.PeerMaxFrameSize {
		s.conn.PeerMaxFrameSize = s.maxFrameSizeOverride
	}
	err := s.conn.writeFrame(frames.Frame{
		Type: frameTypeSASL,
		Body: &frames.SASLInit{
			Mechanism:       saslMechanismXOAUTH2,
			InitialResponse: s.response,
		},
	})
	s.conn.PeerMaxFrameSize = originalPeerMaxFrameSize
	if err != nil {
		return nil, err
	}

	return s.step, nil
}

func (s saslXOAUTH2Handler) step() (stateFunc, error) {
	// read challenge or outcome frame
	fr, err := s.conn.readFrame()
	if err != nil {
		return nil, err
	}

	switch v := fr.Body.(type) {
	case *frames.SASLOutcome:
		// check if auth succeeded
		if v.Code != encoding.CodeSASLOK {
			return nil, fmt.Errorf("SASL XOAUTH2 auth failed with code %#00x: %s : %s",
				v.Code, v.AdditionalData, s.errorResponse)
		}

		// return to c.negotiateProto
		s.conn.saslComplete = true
		return s.conn.negotiateProto, nil
	case *frames.SASLChallenge:
		if s.errorResponse == nil {
			s.errorResponse = v.Challenge

			// The SASL protocol requires clients to send an empty response to this challenge.
			err := s.conn.writeFrame(frames.Frame{
				Type: frameTypeSASL,
				Body: &frames.SASLResponse{
					Response: []byte{},
				},
			})
			if err != nil {
				return nil, err
			}
			return s.step, nil
		} else {
			return nil, fmt.Errorf("SASL XOAUTH2 unexpected additional error response received during "+
				"exchange. Initial error response: %s, additional response: %s", s.errorResponse, v.Challenge)
		}
	default:
		return nil, fmt.Errorf("sasl: unexpected frame type %T", fr.Body)
	}
}

func saslXOAUTH2InitialResponse(username string, bearer string) ([]byte, error) {
	if len(bearer) == 0 {
		return []byte{}, fmt.Errorf("unacceptable bearer token")
	}
	for _, char := range bearer {
		if char < '\x20' || char > '\x7E' {
			return []byte{}, fmt.Errorf("unacceptable bearer token")
		}
	}
	for _, char := range username {
		if char == '\x01' {
			return []byte{}, fmt.Errorf("unacceptable username")
		}
	}
	return []byte("user=" + username + "\x01auth=Bearer " + bearer + "\x01\x01"), nil
}
